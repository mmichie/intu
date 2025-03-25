package commands

import (
	"fmt"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/prompt"
	"github.com/spf13/cobra"
)

func runSecurityReviewCommand(cmd *cobra.Command, args []string) error {
	content, err := getContent(args)
	if err != nil {
		return err
	}

	if content == "" {
		fmt.Println("No input received. Please provide a file or pipe content to stdin.")
		fmt.Println("Usage: intu securityreview [file]")
		fmt.Println("   or: cat file | intu securityreview")
		return nil
	}

	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}

	agent := aikit.NewAIAgent(provider)
	securityReviewPrompt, ok := prompt.GetPrompt("security_review")
	if !ok {
		return fmt.Errorf("security review prompt not found")
	}

	formattedPrompt, err := securityReviewPrompt.Format(content)
	if err != nil {
		return fmt.Errorf("error formatting prompt: %w", err)
	}

	result, err := agent.Process(cmd.Context(), content, formattedPrompt)
	if err != nil {
		return fmt.Errorf("error generating security review: %w", err)
	}

	securityReview, err := ParseSecurityReview(result)
	if err != nil {
		return fmt.Errorf("error parsing security review: %w", err)
	}

	fmt.Println(securityReview)
	return nil
}
