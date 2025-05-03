package providers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// StreamHandler is a callback function for streaming responses
type StreamHandler func(chunk string) error

// Provider interface
type Provider interface {
	// Core methods
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	Name() string
	GetSupportedModels() []string // Returns supported models

	// Streaming capabilities
	SupportsStreaming() bool
	GenerateStreamingResponse(ctx context.Context, prompt string, handler StreamHandler) error

	// Function calling capabilities
	SupportsFunctionCalling() bool
	RegisterFunction(def FunctionDefinition) error
	RegisterFunctions(functions []FunctionDefinition)
	GenerateResponseWithFunctions(
		ctx context.Context,
		prompt string,
		functionExecutor FunctionExecutorFunc,
	) (string, error)

	// Streaming with function calls
	GenerateStreamingResponseWithFunctions(
		ctx context.Context,
		prompt string,
		functionExecutor FunctionExecutorFunc,
		handler StreamHandler,
	) error
}

// BaseProvider contains common provider fields and methods
type BaseProvider struct {
	APIKey string
	Model  string
	URL    string
}

// SupportsStreaming is a default implementation
// Most providers support streaming, so default to true
func (p *BaseProvider) SupportsStreaming() bool {
	return true
}

// Note: BaseProvider doesn't implement GenerateResponse and GenerateResponseWithFunctions
// Each provider must implement these methods.

// We only add helper methods for streaming simulation in BaseProvider.

// SimulateStreamingResponse is a helper to simulate streaming from a full response
func SimulateStreamingResponse(ctx context.Context, fullResponse string, handler StreamHandler) error {
	// Split into chunks and stream
	chunks := splitTextIntoChunks(fullResponse, 15)

	// Create a timer for the simulation delay
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for i, chunk := range chunks {
		// Check for context cancellation before each chunk
		select {
		case <-ctx.Done():
			// If this is the last chunk, try to send it anyway before returning
			if i == len(chunks)-1 {
				_ = handler(chunk)
			}
			return ctx.Err()
		default:
			// Continue processing
		}

		// Send chunk
		if err := handler(chunk); err != nil {
			// If handler returns error, check if it's related to context
			if strings.Contains(err.Error(), "context") {
				return fmt.Errorf("streaming interrupted: %w", err)
			}
			return err
		}

		// Wait for ticker or context cancellation
		if i < len(chunks)-1 { // No need to wait after the last chunk
			select {
			case <-ticker.C:
				// Time for next chunk
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// splitTextIntoChunks splits a string into chunks of approximately the given size,
// but tries to split at word boundaries
func splitTextIntoChunks(text string, chunkSize int) []string {
	var chunks []string
	runes := []rune(text)

	for i := 0; i < len(runes); {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		} else {
			// Try to find a word boundary
			for j := end - 1; j > i; j-- {
				if j < len(runes) && (runes[j] == ' ' || runes[j] == '\n') {
					end = j + 1
					break
				}
			}
		}

		chunks = append(chunks, string(runes[i:end]))
		i = end
	}

	return chunks
}

// GetEnvOrDefault retrieves an environment variable or returns a default
func (p *BaseProvider) GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetSupportedModels is a default implementation that returns an empty slice
// Providers should override this method to return their supported models
func (p *BaseProvider) GetSupportedModels() []string {
	return []string{}
}

// SetModel sets the model for this provider if it's supported
// Returns true if the model was set, false if it wasn't supported
func (p *BaseProvider) SetModel(model string, supportedModels map[string]bool, defaultModel string) bool {
	if supportedModels[model] {
		p.Model = model
		return true
	}
	p.Model = defaultModel
	return false
}
