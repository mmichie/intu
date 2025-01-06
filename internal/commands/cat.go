package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mmichie/intu/internal/fileops"
	"github.com/mmichie/intu/internal/filters"
	"github.com/mmichie/intu/internal/rle"
	"github.com/spf13/cobra"
)

var (
	recursive, jsonOutput, rleOutput, listFilters, extendedMetadata bool
	pattern                                                         string
	filterNames, ignorePatterns                                     []string
)

// CatCommandError is a custom error type for the cat command
type CatCommandError struct {
	MainError error
	Pattern   string
	FileCount int
}

func (e *CatCommandError) Error() string {
	var msg strings.Builder
	if e.FileCount == 0 {
		msg.WriteString(fmt.Sprintf("no files were successfully processed for pattern '%s'", e.Pattern))
	} else {
		msg.WriteString(fmt.Sprintf("some files could not be processed for pattern '%s'", e.Pattern))
	}
	if e.MainError != nil {
		msg.WriteString(fmt.Sprintf(": %v", e.MainError))
	}
	return msg.String()
}

// FileInfoJSON is a struct for JSON output that conditionally includes fields
type FileInfoJSON struct {
	Filename     string     `json:"filename,omitempty"`
	RelativePath string     `json:"relative_path"`
	FileType     string     `json:"file_type,omitempty"`
	Content      string     `json:"content"`
	FileSize     int64      `json:"file_size,omitempty"`
	LastModified *time.Time `json:"last_modified,omitempty"`
	LineCount    int        `json:"line_count,omitempty"`
	MD5Checksum  string     `json:"md5_checksum,omitempty"`
}

var catCmd = &cobra.Command{
	Use:   "cat [file...] or [pattern]",
	Short: "Concatenate and display file contents",
	Long:  `Display contents of files with optional filters applied to transform the text. Supports full regex and path patterns.`,
	RunE:  runCatCommand,
}

func InitCatCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(catCmd)
	catCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively search for files")
	catCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")
	catCmd.Flags().BoolVarP(&rleOutput, "rle", "c", false, "Output in RLE compressed format")
	catCmd.Flags().StringVarP(&pattern, "pattern", "p", "", `File pattern to match (supports full regex and paths)`)
	catCmd.Flags().StringSliceVarP(&filterNames, "filters", "f", nil, "List of filters to apply (comma-separated)")
	catCmd.Flags().StringSliceVarP(&ignorePatterns, "ignore", "i", nil, "Patterns to ignore (can be specified multiple times)")
	catCmd.Flags().BoolVarP(&listFilters, "list-filters", "l", false, "List all available filters")
	catCmd.Flags().BoolVarP(&extendedMetadata, "extended", "e", false, "Display extended metadata")
}

func runCatCommand(cmd *cobra.Command, args []string) error {
	if listFilters {
		return listAvailableFilters(os.Stdout)
	}

	fileOps := fileops.NewFileOperator()
	appliedFilters := getAppliedFilters(filterNames)

	options := fileops.Options{
		Recursive: recursive,
		Extended:  extendedMetadata,
		Ignore:    ignorePatterns,
	}

	var results []fileops.FileInfo
	var err error

	if pattern == "" && len(args) == 1 {
		// Single file mode
		file := args[0]
		info, err := processFile(cmd.Context(), fileOps, file, options.Extended, appliedFilters)
		if err != nil {
			return fmt.Errorf("error processing file '%s': %w", file, err)
		}
		results = []fileops.FileInfo{info}
	} else {
		// Pattern or multiple files mode
		if pattern == "" && len(args) > 0 {
			pattern = args[0]
		}
		if pattern == "" {
			pattern = "*"
		}
		results, err = processFiles(cmd.Context(), fileOps, pattern, options, appliedFilters)
		if err != nil {
			return &CatCommandError{
				MainError: err,
				Pattern:   pattern,
				FileCount: len(results),
			}
		}
	}

	if len(results) == 0 {
		return fmt.Errorf("no files found matching the pattern '%s'", pattern)
	}

	switch {
	case jsonOutput:
		if err := outputJSON(os.Stdout, results); err != nil {
			return fmt.Errorf("error outputting JSON: %w", err)
		}
	case rleOutput:
		if err := outputRLE(os.Stdout, results); err != nil {
			return fmt.Errorf("error outputting RLE: %w", err)
		}
	default:
		if err := outputText(os.Stdout, results); err != nil {
			return fmt.Errorf("error outputting text: %w", err)
		}
	}

	return nil
}

func processFiles(ctx context.Context, fileOps fileops.FileOperator, pattern string, options fileops.Options, filters []filters.Filter) ([]fileops.FileInfo, error) {
	files, err := findFilesWithRegex(ctx, pattern, options)
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
			info, err := processFile(ctx, fileOps, file, options.Extended, filters)
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

func findFilesWithRegex(ctx context.Context, pattern string, options fileops.Options) ([]string, error) {
	var files []string
	var regexPattern *regexp.Regexp
	var err error

	// If the pattern is not a full path, treat it as a regex
	if !filepath.IsAbs(pattern) && !strings.Contains(pattern, string(os.PathSeparator)) {
		regexPattern, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !options.Recursive && info.IsDir() && path != "." {
			return filepath.SkipDir
		}

		for _, ignore := range options.Ignore {
			if matched, _ := filepath.Match(ignore, filepath.Base(path)); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			return nil
		}

		// If it's a full path pattern, use filepath.Match
		if filepath.IsAbs(pattern) || strings.Contains(pattern, string(os.PathSeparator)) {
			matched, err := filepath.Match(pattern, path)
			if err != nil {
				return fmt.Errorf("error matching path: %w", err)
			}
			if matched {
				files = append(files, path)
			}
		} else if regexPattern != nil {
			// Use regex for matching if it's not a full path
			if regexPattern.MatchString(path) {
				files = append(files, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return files, nil
}

func processFile(ctx context.Context, fileOps fileops.FileOperator, file string, extended bool, filters []filters.Filter) (fileops.FileInfo, error) {
	content, err := fileOps.ReadFile(ctx, file)
	if err != nil {
		return fileops.FileInfo{}, fmt.Errorf("error reading file %s: %w", file, err)
	}

	for _, filter := range filters {
		content = filter.Process(content)
	}

	if extended {
		return fileOps.GetExtendedFileInfo(ctx, file, content)
	}
	return fileOps.GetBasicFileInfo(ctx, file, content)
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
	var jsonResults []FileInfoJSON
	for _, info := range results {
		jsonInfo := FileInfoJSON{
			RelativePath: info.RelativePath,
			Content:      info.Content,
		}
		if extendedMetadata {
			jsonInfo.Filename = info.Filename
			jsonInfo.FileType = info.FileType
			jsonInfo.FileSize = info.FileSize
			jsonInfo.LastModified = info.LastModified
			jsonInfo.LineCount = info.LineCount
			jsonInfo.MD5Checksum = info.MD5Checksum
		}
		jsonResults = append(jsonResults, jsonInfo)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(jsonResults); err != nil {
		return fmt.Errorf("error encoding JSON: %w", err)
	}
	return nil
}

func outputRLE(w io.Writer, results []fileops.FileInfo) error {
	var files []rle.FileOutput

	for _, info := range results {
		files = append(files, rle.CompressFile(info.RelativePath, info.Content))
	}

	output := rle.NewBatchOutput(files)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("error encoding RLE output: %w", err)
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
			{"Relative Path", info.RelativePath, true},
			{"Filename", info.Filename, extendedMetadata},
			{"File Type", info.FileType, extendedMetadata},
			{"File Size", fmt.Sprintf("%d bytes", info.FileSize), info.FileSize > 0},
			{"Last Modified", func() string {
				if info.LastModified != nil {
					return info.LastModified.Format(time.RFC3339)
				}
				return ""
			}(), info.LastModified != nil},
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
