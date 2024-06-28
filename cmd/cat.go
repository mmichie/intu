package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/mmichie/intu/internal/fileops"
	"github.com/mmichie/intu/internal/filters"
	"github.com/spf13/cobra"
)

var (
	recursive        bool
	jsonOutput       bool
	pattern          string
	filterNames      []string
	ignorePatterns   []string
	listFilters      bool
	extendedMetadata bool
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
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		if len(results) == 0 {
			return fmt.Errorf("no files were successfully processed")
		}
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No files found matching the pattern.")
		return nil
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
		return nil, fmt.Errorf("error finding files: %w", err)
	}

	var results []fileops.FileInfo
	var resultError error

	for _, file := range files {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
			info, err := processFile(fileOps, file, options.Extended, filters)
			if err != nil {
				resultError = multierror.Append(resultError, fmt.Errorf("error processing %s: %w", file, err))
			} else {
				results = append(results, info)
			}
		}
	}

	return results, resultError
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
		if _, err := fmt.Fprintf(w, "--- File Metadata ---\n"); err != nil {
			return fmt.Errorf("error writing to output: %w", err)
		}
		if _, err := fmt.Fprintf(w, "Filename: %s\n", info.Filename); err != nil {
			return fmt.Errorf("error writing to output: %w", err)
		}
		if _, err := fmt.Fprintf(w, "Relative Path: %s\n", info.RelativePath); err != nil {
			return fmt.Errorf("error writing to output: %w", err)
		}
		if _, err := fmt.Fprintf(w, "File Type: %s\n", info.FileType); err != nil {
			return fmt.Errorf("error writing to output: %w", err)
		}
		if info.FileSize > 0 {
			if _, err := fmt.Fprintf(w, "File Size: %d bytes\n", info.FileSize); err != nil {
				return fmt.Errorf("error writing to output: %w", err)
			}
		}
		if !info.LastModified.IsZero() {
			if _, err := fmt.Fprintf(w, "Last Modified: %s\n", info.LastModified); err != nil {
				return fmt.Errorf("error writing to output: %w", err)
			}
		}
		if info.LineCount > 0 {
			if _, err := fmt.Fprintf(w, "Line Count: %d\n", info.LineCount); err != nil {
				return fmt.Errorf("error writing to output: %w", err)
			}
		}
		if info.MD5Checksum != "" {
			if _, err := fmt.Fprintf(w, "MD5 Checksum: %s\n", info.MD5Checksum); err != nil {
				return fmt.Errorf("error writing to output: %w", err)
			}
		}
		if _, err := fmt.Fprintf(w, "--- File Contents ---\n"); err != nil {
			return fmt.Errorf("error writing to output: %w", err)
		}
		if _, err := fmt.Fprintln(w, info.Content); err != nil {
			return fmt.Errorf("error writing to output: %w", err)
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("error writing to output: %w", err)
		}
	}
	return nil
}

func listAvailableFilters(w io.Writer) error {
	if len(filters.Registry) == 0 {
		return errors.New("no filters available")
	}
	if _, err := fmt.Fprintln(w, "Available Filters:"); err != nil {
		return fmt.Errorf("error writing to output: %w", err)
	}
	for name := range filters.Registry {
		if _, err := fmt.Fprintf(w, "- %s\n", name); err != nil {
			return fmt.Errorf("error writing to output: %w", err)
		}
	}
	return nil
}
