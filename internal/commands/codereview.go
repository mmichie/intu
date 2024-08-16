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
	Short: "Generate a code review for a given file or stdin input",
	Long:  `Analyze the provided code file or stdin input and generate constructive code review comments.`,
	RunE:  runCodeReviewCommand,
}

// InitCodeReviewCommand initializes and adds the codereview command to the given root command
func InitCodeReviewCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(codeReviewCmd)
}

func runCodeReviewCommand(cmd *cobra.Command, args []string) error {
	var content string
	var err error

	if len(args) > 0 {
		// Read from file if a filename is provided
		content, err = readFileContent(args[0])
		if err != nil {
			return fmt.Errorf("error reading file: %w", err)
		}
	} else {
		// Read from stdin if no filename is provided
		content, err = readInput(args)
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}
	}

	// If there's no input, inform the user and exit
	if content == "" {
		fmt.Println("No input received. Please provide a file or pipe content to stdin.")
		fmt.Println("Usage: intu codereview [file]")
		fmt.Println("   or: cat file | intu codereview")
		return nil
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

// readFileContent reads the content of a file
func readFileContent(filename string) (string, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
