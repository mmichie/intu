// Package provider defines the core provider interface for AI services
package provider

import (
	"context"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
	"github.com/mmichie/intu/pkg/aikit/v2/function"
)

// Provider represents an AI model provider service
type Provider interface {
	// Core capabilities
	GenerateResponse(ctx context.Context, request Request) (Response, error)

	// Streaming support
	GenerateStreamingResponse(ctx context.Context, request Request, handler StreamHandler) error

	// Information methods
	Name() string
	Model() string
	Capabilities() []string
}

// Request contains all parameters for a generation request
type Request struct {
	// Prompt is the text prompt or query
	Prompt string

	// FunctionRegistry is an optional registry of available functions
	FunctionRegistry *function.Registry

	// FunctionExecutor is an optional function executor
	FunctionExecutor function.FunctionExecutor

	// Temperature controls randomness (0.0-1.0)
	Temperature float64

	// MaxTokens limits the response length
	MaxTokens int

	// Stream indicates whether to stream the response
	Stream bool

	// Additional provider-specific parameters
	Parameters map[string]interface{}
}

// Response contains the output from an AI provider
type Response struct {
	// Content is the text response
	Content string

	// FunctionCall contains function call information if applicable
	FunctionCall *function.FunctionCall

	// Usage contains token usage information
	Usage *UsageInfo

	// Model identifies the model used
	Model string

	// Provider identifies the provider used
	Provider string

	// Additional provider-specific information
	Metadata map[string]interface{}
}

// UsageInfo contains token usage statistics
type UsageInfo struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// StreamHandler processes chunks from streaming responses
type StreamHandler func(chunk ResponseChunk) error

// ResponseChunk represents a piece of a streaming response
type ResponseChunk struct {
	// Content is the text chunk
	Content string

	// FunctionCall contains partial function call information if applicable
	FunctionCall *function.FunctionCall

	// IsFinal indicates whether this is the last chunk
	IsFinal bool

	// Error contains an error if one occurred during streaming
	Error error
}

// ProviderFactory creates Provider instances
type ProviderFactory interface {
	// Name returns the name of this provider factory
	Name() string

	// Create returns a new Provider instance
	Create(cfg config.Config) (Provider, error)

	// GetAvailableModels returns a list of available models for this provider
	GetAvailableModels() []string

	// GetCapabilities returns a list of capabilities supported by this provider
	GetCapabilities() []string
}

// RegisterFactory adds a provider factory to the global registry
// This is a convenience function that delegates to the global registry
func RegisterFactory(factory ProviderFactory) error {
	return Register(factory)
}

// GetFactory returns a provider factory by name from the global registry
// This is kept for backward compatibility
func GetFactory(name string) (ProviderFactory, error) {
	return Get(name)
}

// GetAvailableProviders returns a list of all registered provider names
// This is kept for backward compatibility
func GetAvailableProviders() []string {
	return List()
}

// IsProviderAvailable checks if a provider is available
// This is kept for backward compatibility
func IsProviderAvailable(name string) bool {
	_, err := Get(name)
	return err == nil
}
