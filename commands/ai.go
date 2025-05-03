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

	// Create an adapter to interface AIAgent with UI Agent
	uiAgent := &uiAgentAdapter{agent: agent}

	// Use the TUI with streaming support
	return ui.StartTUI(cmd.Context(), uiAgent, width, height)
}

// uiAgentAdapter adapts AIAgent to UI Agent
type uiAgentAdapter struct {
	agent *aikit.AIAgent
}

func (a *uiAgentAdapter) Process(ctx context.Context, input, prompt string) (string, error) {
	return a.agent.Process(ctx, input, prompt)
}

func (a *uiAgentAdapter) SupportsStreaming() bool {
	return a.agent.SupportsStreaming()
}

func (a *uiAgentAdapter) ProcessStreaming(ctx context.Context, input, prompt string, handler ui.StreamHandler) error {
	return a.agent.ProcessStreaming(ctx, input, prompt, handler)
}
