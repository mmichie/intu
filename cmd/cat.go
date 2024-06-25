package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mmichie/intu/internal/ai"
	"github.com/mmichie/intu/internal/fileutils"
	"github.com/mmichie/intu/internal/filters"
	"github.com/mmichie/intu/pkg/intu"
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
	Run:   runCatCommand,
}

func init() {
	rootCmd.AddCommand(catCmd)
	catCmd.Flags().BoolP("recursive", "r", false, "Recursively search for files")
	catCmd.Flags().BoolP("json", "j", false, "Output in JSON format")
	catCmd.Flags().StringP("pattern", "p", "", "File pattern to match (e.g., \"*.go\")")
	catCmd.Flags().StringSliceP("filters", "f", []string{}, "List of filters to apply (comma-separated)")
	catCmd.Flags().StringSliceVarP(&ignorePatterns, "ignore", "i", []string{}, "Patterns to ignore (can be specified multiple times)")
	catCmd.Flags().BoolVarP(&listFilters, "list-filters", "l", false, "List all available filters")
	catCmd.Flags().BoolVarP(&extendedMetadata, "extended", "e", false, "Display extended metadata")
}

func runCatCommand(cmd *cobra.Command, args []string) {
	if listFilters {
		listAvailableFilters()
		return
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

	provider, err := selectProvider(cmd)
	if err != nil {
		log.Fatalf("Error creating AI provider: %v", err)
	}

	client := intu.NewClient(provider)

	// Add filters to the client
	for _, name := range filterNames {
		if filter := filters.Get(name); filter != nil {
			client.AddFilter(filter)
		} else {
			fmt.Printf("Warning: No filter found with name '%s'\n", name)
		}
	}

	options := fileutils.Options{
		Recursive: recursive,
		Extended:  extendedMetadata,
		Ignore:    ignorePatterns,
	}

	results, err := client.CatFiles(context.Background(), pattern, options)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if len(results) == 0 {
		fmt.Println("No files found matching the pattern.")
		return
	}

	if jsonOutput {
		outputJSON(results)
	} else {
		outputText(results)
	}
}

func selectProvider(cmd *cobra.Command) (ai.Provider, error) {
	providerName, _ := cmd.Flags().GetString("provider")
	switch providerName {
	case "openai":
		return ai.NewOpenAIProvider()
	case "claude":
		return ai.NewClaudeAIProvider()
	default:
		return ai.NewOpenAIProvider()
	}
}

func outputJSON(results []fileutils.FileInfo) {
	jsonResult, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("Error converting to JSON: %v", err)
	}
	fmt.Println(string(jsonResult))
}

func outputText(results []fileutils.FileInfo) {
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

func listAvailableFilters() {
	fmt.Println("Available Filters:")
	for name := range filters.Registry {
		fmt.Printf("- %s\n", name)
	}
}
