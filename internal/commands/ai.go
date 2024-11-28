package commands

import (
	"context"
	"fmt"
	"os"

	tui "github.com/mmichie/intu/internal/ui"
	"github.com/mmichie/intu/pkg/ai"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func processWithAI(ctx context.Context, input, promptText string) error {
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}

	agent := ai.NewAIAgent(provider)
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
	agent := ai.NewAIAgent(provider)

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 80, 24
	}

	return tui.StartTUI(cmd.Context(), agent, width, height)
}
