package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func processWithAI(ctx context.Context, input, promptText string) error {
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}

	agent := aikit.NewAIAgent(provider)
	result, err := agent.Process(ctx, input, promptText)
	if err != nil {
		return fmt.Errorf("error processing with AI: %w", err)
	}

	fmt.Println(result)
	return nil
}

func runTUICommand(cmd *cobra.Command, args []string) error {
	provider, err := selectProvider()
	if err != nil {
		return err
	}
	agent := aikit.NewAIAgent(provider)

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 80, 24
	}

	// Use the enhanced TUI with streaming support
	return ui.StartTUIEnhanced(cmd.Context(), agent, width, height)
}
