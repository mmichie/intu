package commands

import (
	"fmt"
	"io/ioutil"

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

// InitCodeReviewCommand initializes and adds the codereview command to the given root command
func InitCodeReviewCommand(rootCmd *cobra.Command) {
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
	codeReviewPrompt, ok := prompts.GetPrompt("codereview")
	if !ok {
		return fmt.Errorf("code review prompt not found")
	}

	// Format the prompt with the file content
	formattedPrompt, err := codeReviewPrompt.Format(content)
	if err != nil {
		return fmt.Errorf("error formatting prompt: %w", err)
	}

	// Generate the code review
	result, err := client.ProcessWithAI(cmd.Context(), content, formattedPrompt)
	if err != nil {
		return fmt.Errorf("error generating code review: %w", err)
	}

	// Extract the review comments from the result
	reviewComments, err := intu.ParseReviewComments(result)
	if err != nil {
		return fmt.Errorf("error parsing review comments: %w", err)
	}

	// Print the generated code review comments
	fmt.Println(reviewComments)
	return nil
}

func readFile(filename string) (string, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
