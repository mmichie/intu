package aikit

import (
	"context"
	"fmt"
)

type AIAgent struct {
	provider Provider
}

func NewAIAgent(provider Provider) *AIAgent {
	return &AIAgent{provider: provider}
}

func (a *AIAgent) Process(ctx context.Context, input, prompt string) (string, error) {
	fullPrompt := input
	if prompt != "" && input != "" {
		fullPrompt = fmt.Sprintf("%s\n\nInput: %s", prompt, input)
	} else if prompt != "" {
		fullPrompt = prompt
	}
	return a.provider.GenerateResponse(ctx, fullPrompt)
}
