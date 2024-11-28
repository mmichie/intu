package commands

import (
	"fmt"

	"github.com/mmichie/intu/pkg/ai"
	"github.com/mmichie/intu/pkg/prompts"
	"github.com/spf13/cobra"
)

func runCodeReviewCommand(cmd *cobra.Command, args []string) error {
	content, err := getContent(args)
	if err != nil {
		return err
	}

	if content == "" {
		fmt.Println("No input received. Please provide a file or pipe content to stdin.")
		fmt.Println("Usage: intu codereview [file]")
		fmt.Println("   or: cat file | intu codereview")
		return nil
	}

	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}

	agent := ai.NewAIAgent(provider)
	codeReviewPrompt, ok := prompts.GetPrompt("codereview")
	if !ok {
		return fmt.Errorf("code review prompt not found")
	}

	formattedPrompt, err := codeReviewPrompt.Format(content)
	if err != nil {
		return fmt.Errorf("error formatting prompt: %w", err)
	}

	result, err := agent.Process(cmd.Context(), content, formattedPrompt)
	if err != nil {
		return fmt.Errorf("error generating code review: %w", err)
	}

	reviewComments, err := ParseReviewComments(result)
	if err != nil {
		return fmt.Errorf("error parsing review comments: %w", err)
	}

	fmt.Println(reviewComments)
	return nil
}

func getContent(args []string) (string, error) {
	if len(args) > 0 {
		content, err := readFileContent(args[0])
		if err != nil {
			return "", fmt.Errorf("error reading file: %w", err)
		}
		return content, nil
	}

	content, err := readInput(args)
	if err != nil {
		return "", fmt.Errorf("error reading input: %w", err)
	}
	return content, nil
}
