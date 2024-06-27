package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/mmichie/intu/pkg/prompts"
	"github.com/spf13/cobra"
)

var codeReviewCmd = &cobra.Command{
	Use:   "codereview [file]",
	Short: "Generate a code review for a given file",
	Long:  `Analyze the provided code file and generate constructive code review comments.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCodeReviewCommand,
}

func init() {
	rootCmd.AddCommand(codeReviewCmd)
}

func runCodeReviewCommand(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Read the content of the file
	content, err := readFile(filename)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Select the AI provider
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}

	// Create the client
	client := intu.NewClient(provider)

	// Get the code review prompt
	codeReviewPrompt := prompts.CodeReview

	// Generate the code review
	result, err := client.ProcessWithAI(cmd.Context(), content, codeReviewPrompt.Format(content))
	if err != nil {
		return fmt.Errorf("error generating code review: %w", err)
	}

	// Extract the review comments from the result
	reviewComments := extractReviewComments(result)

	// Print the generated code review comments
	fmt.Println(reviewComments)
	return nil
}

func extractReviewComments(result string) string {
	start := strings.Index(result, "<review_comments>")
	end := strings.Index(result, "</review_comments>")
	if start != -1 && end != -1 {
		return strings.TrimSpace(result[start+len("<review_comments>") : end])
	}
	return result // Return the full result if tags are not found
}

func readFile(filename string) (string, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
