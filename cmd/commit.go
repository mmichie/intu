package cmd

import (
	"fmt"
	"strings"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/mmichie/intu/pkg/prompts"
	"github.com/spf13/cobra"
)

func runCommitCommand(cmd *cobra.Command, args []string) error {
	// Read input from stdin
	diffOutput, err := readInput(args)
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
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
		return fmt.Errorf("failed to select AI provider: %w", err)
	}

	// Create the client
	client := intu.NewClient(provider)

	// Get the commit prompt
	commitPrompt := prompts.Commit

	// Generate the commit message
	result, err := client.ProcessWithAI(cmd.Context(), diffOutput, commitPrompt.Format(diffOutput))
	if err != nil {
		return fmt.Errorf("error generating commit message: %w", err)
	}

	// Extract the commit message from the result
	commitMessage := extractCommitMessage(result)

	// Print the generated commit message
	fmt.Println(commitMessage)
	return nil
}

func extractCommitMessage(result string) string {
	start := strings.Index(result, "<commit_message>")
	end := strings.Index(result, "</commit_message>")
	if start != -1 && end != -1 {
		return strings.TrimSpace(result[start+len("<commit_message>") : end])
	}
	return result // Return the full result if tags are not found
}
