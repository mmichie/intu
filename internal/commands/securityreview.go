package commands

import (
	"fmt"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/mmichie/intu/pkg/prompts"

	"github.com/spf13/cobra"
)

var securityReviewCmd = &cobra.Command{
	Use:   "securityreview [file]",
	Short: "Perform a security review for a given file or stdin input",
	Long:  `Analyze the provided code file or stdin input and generate a comprehensive security review.`,
	RunE:  runSecurityReviewCommand,
}

// InitSecurityReviewCommand initializes and adds the securityreview command to the given root command
func InitSecurityReviewCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(securityReviewCmd)
}

func runSecurityReviewCommand(cmd *cobra.Command, args []string) error {
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
		fmt.Println("Usage: intu securityreview [file]")
		fmt.Println("   or: cat file | intu securityreview")
		return nil
	}

	// Select the AI provider
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}

	// Create the client
	client := intu.NewClient(provider)

	// Get the security review prompt
	securityReviewPrompt, ok := prompts.GetPrompt("security_review")
	if !ok {
		return fmt.Errorf("security review prompt not found")
	}

	// Format the prompt with the file content
	formattedPrompt, err := securityReviewPrompt.Format(content)
	if err != nil {
		return fmt.Errorf("error formatting prompt: %w", err)
	}

	// Generate the security review
	result, err := client.ProcessWithAI(cmd.Context(), content, formattedPrompt)
	if err != nil {
		return fmt.Errorf("error generating security review: %w", err)
	}

	// Extract the security review from the result
	securityReview, err := intu.ParseSecurityReview(result)
	if err != nil {
		return fmt.Errorf("error parsing security review: %w", err)
	}

	// Print the generated security review
	fmt.Println(securityReview)
	return nil
}
