package commands

import (
	"fmt"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/mmichie/intu/pkg/prompts"

	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate a commit message",
	Long:  `Generate a commit message based on the git diff output.`,
	RunE:  runCommitCommand,
}

// InitCommitCommand initializes and adds the commit command to the given root command
func InitCommitCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(commitCmd)
}

func runCommitCommand(cmd *cobra.Command, args []string) error {
	// Read input from stdin
	diffOutput, err := readInput(args)
	if err != nil {
		return fmt.Errorf("error reading git diff input for commit command: %w", err)
	}

	// If there's no input, inform the user and exit
	if diffOutput == "" {
		fmt.Println("No input received. Please provide git diff output.")
		fmt.Println("Usage: git diff --staged | intu commit")
		return nil
	}

	// Select the AI provider
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider for commit command: %w", err)
	}

	// Create the client
	client := intu.NewClient(provider)

	// Get the commit prompt
	commitPrompt, ok := prompts.GetPrompt("commit")
	if !ok {
		return fmt.Errorf("commit prompt not found in available prompts")
	}

	// Format the prompt with the diff output
	formattedPrompt, err := commitPrompt.Format(diffOutput)
	if err != nil {
		return fmt.Errorf("error formatting commit prompt: %w", err)
	}

	// Generate the commit message
	result, err := client.ProcessWithAI(cmd.Context(), diffOutput, formattedPrompt)
	if err != nil {
		return fmt.Errorf("error generating commit message: %w", err)
	}

	// Extract the commit message from the result
	commitMessage, err := intu.ParseCommitMessage(result)
	if err != nil {
		return fmt.Errorf("error parsing generated commit message: %w", err)
	}

	// Print only the parsed commit message
	fmt.Print(commitMessage)

	return nil
}
