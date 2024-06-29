package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mmichie/intu/internal/fileops"
	"github.com/mmichie/intu/internal/filters"
	"github.com/spf13/cobra"
)

var (
	recursive, jsonOutput, listFilters, extendedMetadata bool
	pattern                                              string
	filterNames, ignorePatterns                          []string
)

var catCmd = &cobra.Command{
	Use:   "cat [file...]",
	Short: "Concatenate and display file contents",
	Long:  `Display contents of files with optional filters applied to transform the text.`,
	RunE:  runCatCommand,
}

func init() {
	rootCmd.AddCommand(catCmd)
	catCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively search for files")
	catCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")
	catCmd.Flags().StringVarP(&pattern, "pattern", "p", "", `File pattern to match (e.g., "*.go")`)
	catCmd.Flags().StringSliceVarP(&filterNames, "filters", "f", nil, "List of filters to apply (comma-separated)")
	catCmd.Flags().StringSliceVarP(&ignorePatterns, "ignore", "i", nil, "Patterns to ignore (can be specified multiple times)")
	catCmd.Flags().BoolVarP(&listFilters, "list-filters", "l", false, "List all available filters")
	catCmd.Flags().BoolVarP(&extendedMetadata, "extended", "e", false, "Display extended metadata")
}

func runCatCommand(cmd *cobra.Command, args []string) error {
	if listFilters {
		return listAvailableFilters(os.Stdout)
	}

	if pattern == "" && len(args) > 0 {
		pattern = args[0]
	}
	if pattern == "" {
		pattern = "*"
	}

	fileOps := fileops.NewFileOperator()
	appliedFilters := getAppliedFilters(filterNames)

	options := fileops.Options{
		Recursive: recursive,
		Extended:  extendedMetadata,
		Ignore:    ignorePatterns,
	}

	results, err := processFiles(cmd.Context(), fileOps, pattern, options, appliedFilters)
	if err != nil {
		var errList []error
		if len(results) == 0 {
			errList = append(errList, fmt.Errorf("no files were successfully processed for pattern '%s'", pattern))
		} else {
			errList = append(errList, fmt.Errorf("some files could not be processed"))
		}
		errList = append(errList, err)
		return errors.Join(errList...)
	}

	if len(results) == 0 {
		return fmt.Errorf("no files found matching the pattern '%s'", pattern)
	}

	if jsonOutput {
		if err := outputJSON(os.Stdout, results); err != nil {
			return fmt.Errorf("error outputting JSON: %w", err)
		}
	} else {
		if err := outputText(os.Stdout, results); err != nil {
			return fmt.Errorf("error outputting text: %w", err)
		}
	}
	return nil
}

func processFiles(ctx context.Context, fileOps fileops.FileOperator, pattern string, options fileops.Options, filters []filters.Filter) ([]fileops.FileInfo, error) {
	files, err := fileOps.FindFiles(pattern, options)
	if err != nil {
		return nil, fmt.Errorf("error finding files with pattern '%s': %w", pattern, err)
	}

	var results []fileops.FileInfo
	var errs []error

	for _, file := range files {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
			info, err := processFile(fileOps, file, options.Extended, filters)
			if err != nil {
				errs = append(errs, fmt.Errorf("error processing file '%s': %w", file, err))
			} else {
				results = append(results, info)
			}
		}
		time.Sleep(time.Millisecond) // Prevent tight looping
	}

	if len(errs) > 0 {
		return results, errors.Join(errs...)
	}
	return results, nil
}

func processFile(fileOps fileops.FileOperator, file string, extended bool, filters []filters.Filter) (fileops.FileInfo, error) {
	content, err := fileOps.ReadFile(file)
	if err != nil {
		return fileops.FileInfo{}, fmt.Errorf("error reading file %s: %w", file, err)
	}

	for _, filter := range filters {
		content = filter.Process(content)
	}

	if extended {
		return fileOps.GetExtendedFileInfo(file, content)
	}
	return fileOps.GetBasicFileInfo(file, content)
}

func getAppliedFilters(names []string) []filters.Filter {
	var appliedFilters []filters.Filter
	for _, name := range names {
		if filter := filters.Get(name); filter != nil {
			appliedFilters = append(appliedFilters, filter)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: No filter found with name '%s'\n", name)
		}
	}
	return appliedFilters
}

func outputJSON(w io.Writer, results []fileops.FileInfo) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("error encoding JSON: %w", err)
	}
	return nil
}

func outputText(w io.Writer, results []fileops.FileInfo) error {
	for _, info := range results {
		if err := writeSection(w, "File Metadata"); err != nil {
			return err
		}

		fields := []struct {
			name  string
			value interface{}
			cond  bool
		}{
			{"Filename", info.Filename, true},
			{"Relative Path", info.RelativePath, true},
			{"File Type", info.FileType, true},
			{"File Size", fmt.Sprintf("%d bytes", info.FileSize), info.FileSize > 0},
			{"Last Modified", info.LastModified.Format(time.RFC3339), !info.LastModified.IsZero()},
			{"Line Count", info.LineCount, info.LineCount > 0},
			{"MD5 Checksum", info.MD5Checksum, info.MD5Checksum != ""},
		}

		for _, field := range fields {
			if field.cond {
				if err := writeField(w, field.name, field.value); err != nil {
					return err
				}
			}
		}

		if err := writeSection(w, "File Contents"); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(w, info.Content); err != nil {
			return fmt.Errorf("error writing content to output: %w", err)
		}

		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("error writing newline to output: %w", err)
		}
	}
	return nil
}

func writeSection(w io.Writer, sectionName string) error {
	_, err := fmt.Fprintf(w, "--- %s ---\n", sectionName)
	if err != nil {
		return fmt.Errorf("error writing section header to output: %w", err)
	}
	return nil
}

func writeField(w io.Writer, fieldName string, fieldValue interface{}) error {
	_, err := fmt.Fprintf(w, "%s: %v\n", fieldName, fieldValue)
	if err != nil {
		return fmt.Errorf("error writing field to output: %w", err)
	}
	return nil
}

func listAvailableFilters(w io.Writer) error {
	if len(filters.Registry) == 0 {
		return errors.New("no filters available")
	}
	if err := writeSection(w, "Available Filters"); err != nil {
		return err
	}
	for name := range filters.Registry {
		if err := writeField(w, "-", name); err != nil {
			return err
		}
	}
	return nil
}
