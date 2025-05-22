// Package provider defines the core provider interface for AI services
package provider

import (
	"context"
	"errors"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
	aierrors "github.com/mmichie/intu/pkg/aikit/v2/errors"
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

// Registry of provider factories
var factories = make(map[string]ProviderFactory)

// RegisterFactory adds a provider factory to the registry
func RegisterFactory(factory ProviderFactory) error {
	name := factory.Name()
	if name == "" {
		return errors.New("provider factory must have a name")
	}

	if _, exists := factories[name]; exists {
		return errors.New("provider factory already registered: " + name)
	}

	factories[name] = factory
	return nil
}

// GetFactory returns a provider factory by name
func GetFactory(name string) (ProviderFactory, error) {
	factory, exists := factories[name]
	if !exists {
		return nil, aierrors.New("registry", "get_factory", errors.New("provider factory not found: "+name))
	}
	return factory, nil
}

// GetAvailableProviders returns a list of all registered provider names
func GetAvailableProviders() []string {
	result := make([]string, 0, len(factories))
	for name := range factories {
		result = append(result, name)
	}
	return result
}

// IsProviderAvailable checks if a provider is available
func IsProviderAvailable(name string) bool {
	_, exists := factories[name]
	return exists
}
