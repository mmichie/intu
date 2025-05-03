package aikit

import (
	"context"
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/providers"
)

// Provider interface for AI providers
type Provider interface {
	// Core methods
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	Name() string
	GetSupportedModels() []string

	// Streaming capabilities
	SupportsStreaming() bool
	GenerateStreamingResponse(ctx context.Context, prompt string, handler providers.StreamHandler) error

	// Function calling capabilities
	SupportsFunctionCalling() bool
	RegisterFunction(def providers.FunctionDefinition) error
	RegisterFunctions(functions []providers.FunctionDefinition)
	GenerateResponseWithFunctions(
		ctx context.Context,
		prompt string,
		functionExecutor providers.FunctionExecutorFunc,
	) (string, error)

	// Streaming with function calls
	GenerateStreamingResponseWithFunctions(
		ctx context.Context,
		prompt string,
		functionExecutor providers.FunctionExecutorFunc,
		handler providers.StreamHandler,
	) error
}

// BaseProvider is now just a facade for backward compatibility
type BaseProvider struct {
	providers.BaseProvider
}

// NewProvider creates a new provider based on the name
func NewProvider(name string) (Provider, error) {
	switch name {
	case "claude":
		return providers.NewClaudeAIProvider()
	case "openai":
		return providers.NewOpenAIProvider()
	case "gemini":
		return providers.NewGeminiProvider()
	case "grok":
		return providers.NewGrokProvider()
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

// GetAvailableProviders returns a list of all available provider names
func GetAvailableProviders() []string {
	// Return all providers that support function calling
	return []string{"claude", "openai", "gemini", "grok"}
}

// GetProviderModels returns a map of provider names to their supported models
func GetProviderModels() (map[string][]string, error) {
	result := make(map[string][]string)

	for _, providerName := range GetAvailableProviders() {
		provider, err := NewProvider(providerName)
		if err != nil {
			// Skip providers that we can't initialize (likely due to missing API keys)
			continue
		}

		result[providerName] = provider.GetSupportedModels()
	}

	return result, nil
}
