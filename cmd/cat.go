package cmd

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/mmichie/intu/pkg/filters"
	"github.com/mmichie/intu/pkg/intu"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	listFilters    bool
	ignorePatterns []string
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

	client := intu.NewIntuClient(&intu.OpenAIProvider{APIKey: viper.GetString("OPENAI_API_KEY")})

	// Load filters based on the provided names
	for _, name := range filterNames {
		if filter := filters.Get(name); filter != nil {
			client.ActiveFilters = append(client.ActiveFilters, filter)
		} else {
			fmt.Printf("Warning: No filter found with name '%s'\n", name)
		}
	}

	result, err := client.CatFiles(pattern, recursive, ignorePatterns)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if len(result) == 0 {
		fmt.Println("No files found matching the pattern.")
		return
	}

	if jsonOutput {
		jsonResult, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatalf("Error converting to JSON: %v", err)
		}
		fmt.Println(string(jsonResult))
	} else {
		for _, info := range result {
			fmt.Printf("--- File Metadata ---\n")
			fmt.Printf("Filename: %s\n", info.Filename)
			fmt.Printf("Relative Path: %s\n", info.RelativePath)
			fmt.Printf("File Size: %d bytes\n", info.FileSize)
			fmt.Printf("Last Modified: %s\n", info.LastModified)
			fmt.Printf("File Type: %s\n", info.FileType)
			fmt.Printf("Line Count: %d\n", info.LineCount)
			fmt.Printf("File Extension: %s\n", info.FileExtension)
			fmt.Printf("MD5 Checksum: %s\n", info.MD5Checksum)
			fmt.Printf("--- File Contents ---\n")
			fmt.Println(info.Content)
			fmt.Println()
		}
	}
}

// listAvailableFilters prints all filters registered in the system.
func listAvailableFilters() {
	fmt.Println("Available Filters:")
	for name := range filters.Registry {
		fmt.Printf("- %s\n", name)
	}
}
