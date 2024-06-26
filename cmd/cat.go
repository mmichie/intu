package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mmichie/intu/internal/fileops"
	"github.com/mmichie/intu/internal/filters"
	"github.com/spf13/cobra"
)

var (
	listFilters      bool
	ignorePatterns   []string
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
	catCmd.Flags().BoolP("recursive", "r", false, "Recursively search for files")
	catCmd.Flags().BoolP("json", "j", false, "Output in JSON format")
	catCmd.Flags().StringP("pattern", "p", "", `File pattern to match (e.g., "*.go")`)
	catCmd.Flags().StringSliceP("filters", "f", []string{}, "List of filters to apply (comma-separated)")
	catCmd.Flags().StringSliceVarP(&ignorePatterns, "ignore", "i", []string{}, "Patterns to ignore (can be specified multiple times)")
	catCmd.Flags().BoolVarP(&listFilters, "list-filters", "l", false, "List all available filters")
	catCmd.Flags().BoolVarP(&extendedMetadata, "extended", "e", false, "Display extended metadata")
}

func runCatCommand(cmd *cobra.Command, args []string) error {
	if listFilters {
		return listAvailableFilters()
	}

	recursive, _ := cmd.Flags().GetBool("recursive")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	pattern, _ := cmd.Flags().GetString("pattern")
	filterNames, _ := cmd.Flags().GetStringSlice("filters")

	// If no pattern is provided via flag, use the first argument as pattern
	if pattern == "" && len(args) > 0 {
		pattern = args[0]
	}

	// If still no pattern, default to "*"
	if pattern == "" {
		pattern = "*"
	}

	fileOps := fileops.NewFileOperator()

	// Create a slice to hold the filters
	var appliedFilters []filters.Filter
	for _, name := range filterNames {
		if filter := filters.Get(name); filter != nil {
			appliedFilters = append(appliedFilters, filter)
		} else {
			fmt.Printf("Warning: No filter found with name '%s'\n", name)
		}
	}

	options := fileops.Options{
		Recursive: recursive,
		Extended:  extendedMetadata,
		Ignore:    ignorePatterns,
	}

	results, err := processFiles(context.Background(), fileOps, pattern, options, appliedFilters)
	if err != nil {
		return fmt.Errorf("error processing files: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No files found matching the pattern.")
		return nil
	}

	if jsonOutput {
		return outputJSON(results)
	}
	outputText(results)
	return nil
}

func processFiles(ctx context.Context, fileOps fileops.FileOperator, pattern string, options fileops.Options, filters []filters.Filter) ([]fileops.FileInfo, error) {
	files, err := fileOps.FindFiles(pattern, options)
	if err != nil {
		return nil, fmt.Errorf("error finding files: %w", err)
	}

	var results []fileops.FileInfo
	for _, file := range files {
		content, err := fileOps.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("error reading file %s: %w", file, err)
		}

		for _, filter := range filters {
			content = filter.Process(content)
		}

		var info fileops.FileInfo
		if options.Extended {
			info, err = fileOps.GetExtendedFileInfo(file, content)
		} else {
			info, err = fileOps.GetBasicFileInfo(file, content)
		}
		if err != nil {
			return nil, fmt.Errorf("error getting file info for %s: %w", file, err)
		}

		results = append(results, info)
	}

	return results, nil
}

func outputJSON(results []fileops.FileInfo) error {
	jsonResult, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("error converting to JSON: %w", err)
	}
	fmt.Println(string(jsonResult))
	return nil
}

func outputText(results []fileops.FileInfo) {
	for _, info := range results {
		fmt.Printf("--- File Metadata ---\n")
		fmt.Printf("Filename: %s\n", info.Filename)
		fmt.Printf("Relative Path: %s\n", info.RelativePath)
		fmt.Printf("File Type: %s\n", info.FileType)
		if info.FileSize > 0 {
			fmt.Printf("File Size: %d bytes\n", info.FileSize)
		}
		if !info.LastModified.IsZero() {
			fmt.Printf("Last Modified: %s\n", info.LastModified)
		}
		if info.LineCount > 0 {
			fmt.Printf("Line Count: %d\n", info.LineCount)
		}
		if info.MD5Checksum != "" {
			fmt.Printf("MD5 Checksum: %s\n", info.MD5Checksum)
		}
		fmt.Printf("--- File Contents ---\n")
		fmt.Println(info.Content)
		fmt.Println()
	}
}

func listAvailableFilters() error {
	fmt.Println("Available Filters:")
	for name := range filters.Registry {
		fmt.Printf("- %s\n", name)
	}
	return nil
}
