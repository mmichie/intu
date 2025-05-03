package aikit

import (
	"context"
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/providers"
)

// Use providers.StreamHandler as our StreamHandler type

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

// SupportsStreaming returns whether the agent supports streaming responses
func (a *AIAgent) SupportsStreaming() bool {
	return a.provider.SupportsStreaming()
}

// ProcessStreaming processes an input with a streaming response
func (a *AIAgent) ProcessStreaming(ctx context.Context, input, prompt string, handler providers.StreamHandler) error {
	fullPrompt := input
	if prompt != "" && input != "" {
		fullPrompt = fmt.Sprintf("%s\n\nInput: %s", prompt, input)
	} else if prompt != "" {
		fullPrompt = prompt
	}

	// Delegate to the provider's streaming implementation
	return a.provider.GenerateStreamingResponse(ctx, fullPrompt, func(chunk string) error {
		return handler(chunk)
	})
}

// ProcessWithFunctions processes an input with function calling
func (a *AIAgent) ProcessWithFunctions(
	ctx context.Context,
	input,
	prompt string,
	functionExecutor providers.FunctionExecutorFunc,
) (string, error) {
	fullPrompt := input
	if prompt != "" && input != "" {
		fullPrompt = fmt.Sprintf("%s\n\nInput: %s", prompt, input)
	} else if prompt != "" {
		fullPrompt = prompt
	}

	return a.provider.GenerateResponseWithFunctions(ctx, fullPrompt, functionExecutor)
}

// ProcessStreamingWithFunctions processes an input with streaming and function calling
func (a *AIAgent) ProcessStreamingWithFunctions(
	ctx context.Context,
	input,
	prompt string,
	functionExecutor providers.FunctionExecutorFunc,
	handler providers.StreamHandler,
) error {
	fullPrompt := input
	if prompt != "" && input != "" {
		fullPrompt = fmt.Sprintf("%s\n\nInput: %s", prompt, input)
	} else if prompt != "" {
		fullPrompt = prompt
	}

	return a.provider.GenerateStreamingResponseWithFunctions(ctx, fullPrompt, functionExecutor, handler)
}
