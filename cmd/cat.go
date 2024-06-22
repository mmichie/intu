package cmd

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var catCmd = &cobra.Command{
	Use:   "cat [file...]",
	Short: "Concatenate and display file contents",
	Run:   runCatCommand,
}

func init() {
	rootCmd.AddCommand(catCmd)
	catCmd.Flags().BoolP("recursive", "r", false, "Recursively search for files")
	catCmd.Flags().BoolP("json", "j", false, "Output in JSON format")
	catCmd.Flags().StringP("pattern", "p", "", "File pattern to match (e.g., \"*.go\")")
}

func runCatCommand(cmd *cobra.Command, args []string) {
	recursive, _ := cmd.Flags().GetBool("recursive")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	pattern, _ := cmd.Flags().GetString("pattern")

	// If no pattern is provided via flag, use the first argument as pattern
	if pattern == "" && len(args) > 0 {
		pattern = args[0]
	}

	// If still no pattern, default to "*"
	if pattern == "" {
		pattern = "*"
	}

	fmt.Printf("Debug: Pattern = %s, Recursive = %v\n", pattern, recursive)

	client := intu.NewIntuClient(&intu.OpenAIProvider{APIKey: viper.GetString("OPENAI_API_KEY")})
	result, err := client.CatFiles(pattern, recursive)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Debug: Found %d files\n", len(result))

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
