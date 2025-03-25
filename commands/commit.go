package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/prompt"
	"github.com/spf13/cobra"
)

func runCommitCommand(cmd *cobra.Command, args []string) error {
	diffOutput, err := readInput(args)
	if err != nil {
		return fmt.Errorf("error reading git diff input: %w", err)
	}

	if diffOutput == "" {
		fmt.Println("No input received. Please provide git diff output.")
		fmt.Println("Usage: git diff --staged | intu commit")
		return nil
	}

	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}

	agent := aikit.NewAIAgent(provider)
	commitPrompt, ok := prompt.GetPrompt("commit")
	if !ok {
		return fmt.Errorf("commit prompt not found")
	}

	formattedPrompt, err := commitPrompt.Format(diffOutput)
	if err != nil {
		return fmt.Errorf("error formatting commit prompt: %w", err)
	}

	result, err := agent.Process(cmd.Context(), diffOutput, formattedPrompt)
	if err != nil {
		return fmt.Errorf("error generating commit message: %w", err)
	}

	commitMessage, err := ParseCommitMessage(result)
	if err != nil {
		return fmt.Errorf("error parsing generated commit message: %w", err)
	}

	if _, err := io.WriteString(os.Stdout, commitMessage); err != nil {
		return fmt.Errorf("error writing commit message: %w", err)
	}

	// Force write a final newline if not present
	if len(commitMessage) == 0 || commitMessage[len(commitMessage)-1] != '\n' {
		if _, err := io.WriteString(os.Stdout, "\n"); err != nil {
			return fmt.Errorf("error writing final newline: %w", err)
		}
	}

	return nil
}
