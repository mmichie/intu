package providers

import (
	"context"
	"fmt"
	"time"
)

// SupportsStreaming returns whether the provider supports streaming responses
func (p *OpenAIProvider) SupportsStreaming() bool {
	return true
}

// GenerateStreamingResponse generates a streaming response from OpenAI
func (p *OpenAIProvider) GenerateStreamingResponse(ctx context.Context, prompt string, handler StreamHandler) error {
	// Create a separate context with a slightly shorter timeout
	// This will help prevent the "context canceled" errors that happen when
	// one part of the code cancels while another is still using the context
	innerCtx, cancel := context.WithTimeout(ctx, 85*time.Second)
	defer cancel()

	// Handle context cancellation before we even start
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Get the full response first (non-streaming)
	response, err := p.GenerateResponse(innerCtx, prompt)
	if err != nil {
		// If the context was canceled, return a more specific error
		if innerCtx.Err() != nil {
			return fmt.Errorf("streaming operation interrupted: %w", innerCtx.Err())
		}
		return err
	}

	// If context was canceled while getting response, return early
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Use the helper to simulate streaming with the original context
	return SimulateStreamingResponse(ctx, response, handler)
}

// GenerateStreamingResponseWithFunctions streams a response with function calling
func (p *OpenAIProvider) GenerateStreamingResponseWithFunctions(
	ctx context.Context,
	prompt string,
	functionExecutor FunctionExecutorFunc,
	handler StreamHandler,
) error {
	// Create a separate context with a slightly shorter timeout
	innerCtx, cancel := context.WithTimeout(ctx, 85*time.Second)
	defer cancel()

	// Handle context cancellation before we even start
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Get the full response with function calls
	response, err := p.GenerateResponseWithFunctions(innerCtx, prompt, functionExecutor)
	if err != nil {
		// If the context was canceled, return a more specific error
		if innerCtx.Err() != nil {
			return fmt.Errorf("streaming operation interrupted: %w", innerCtx.Err())
		}
		return err
	}

	// If context was canceled while getting response, return early
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Simulate streaming with the original context
	return SimulateStreamingResponse(ctx, response, handler)
}
