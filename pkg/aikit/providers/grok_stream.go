package providers

import (
	"context"
)

// SupportsStreaming returns whether the provider supports streaming responses
func (p *GrokProvider) SupportsStreaming() bool {
	return true
}

// GenerateStreamingResponse generates a streaming response from Grok
func (p *GrokProvider) GenerateStreamingResponse(ctx context.Context, prompt string, handler StreamHandler) error {
	// For now, simulate streaming using the regular response
	response, err := p.GenerateResponse(ctx, prompt)
	if err != nil {
		return err
	}

	// Use the helper to simulate streaming
	return SimulateStreamingResponse(ctx, response, handler)
}

// GenerateStreamingResponseWithFunctions streams a response with function calling
func (p *GrokProvider) GenerateStreamingResponseWithFunctions(
	ctx context.Context,
	prompt string,
	functionExecutor FunctionExecutorFunc,
	handler StreamHandler,
) error {
	// Get the full response with function calls
	response, err := p.GenerateResponseWithFunctions(ctx, prompt, functionExecutor)
	if err != nil {
		return err
	}

	// Simulate streaming
	return SimulateStreamingResponse(ctx, response, handler)
}
