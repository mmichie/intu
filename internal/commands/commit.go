package commands

import (
	"fmt"

	"github.com/mmichie/intu/pkg/ai"
	"github.com/mmichie/intu/pkg/prompts"
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

	agent := ai.NewAIAgent(provider)
	commitPrompt, ok := prompts.GetPrompt("commit")
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

	fmt.Print(commitMessage)
	return nil
}
