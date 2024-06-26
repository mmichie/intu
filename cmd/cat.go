package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

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
		return fmt.Errorf("error processing files: %w", err)
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No files found matching the pattern.")
		return nil
	}

	if jsonOutput {
		return outputJSON(os.Stdout, results)
	}
	outputText(os.Stdout, results)
	return nil
}

func processFiles(ctx context.Context, fileOps fileops.FileOperator, pattern string, options fileops.Options, filters []filters.Filter) ([]fileops.FileInfo, error) {
	files, err := fileOps.FindFiles(pattern, options)
	if err != nil {
		return nil, fmt.Errorf("error finding files: %w", err)
	}

	var results []fileops.FileInfo
	for _, file := range files {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
			info, err := processFile(fileOps, file, options.Extended, filters)
			if err != nil {
				return nil, err
			}
			results = append(results, info)
		}
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
	return encoder.Encode(results)
}

func outputText(w io.Writer, results []fileops.FileInfo) {
	for _, info := range results {
		fmt.Fprintf(w, "--- File Metadata ---\n")
		fmt.Fprintf(w, "Filename: %s\n", info.Filename)
		fmt.Fprintf(w, "Relative Path: %s\n", info.RelativePath)
		fmt.Fprintf(w, "File Type: %s\n", info.FileType)
		if info.FileSize > 0 {
			fmt.Fprintf(w, "File Size: %d bytes\n", info.FileSize)
		}
		if !info.LastModified.IsZero() {
			fmt.Fprintf(w, "Last Modified: %s\n", info.LastModified)
		}
		if info.LineCount > 0 {
			fmt.Fprintf(w, "Line Count: %d\n", info.LineCount)
		}
		if info.MD5Checksum != "" {
			fmt.Fprintf(w, "MD5 Checksum: %s\n", info.MD5Checksum)
		}
		fmt.Fprintf(w, "--- File Contents ---\n")
		fmt.Fprintln(w, info.Content)
		fmt.Fprintln(w)
	}
}

func listAvailableFilters(w io.Writer) error {
	if len(filters.Registry) == 0 {
		return errors.New("no filters available")
	}
	fmt.Fprintln(w, "Available Filters:")
	for name := range filters.Registry {
		fmt.Fprintf(w, "- %s\n", name)
	}
	return nil
}
