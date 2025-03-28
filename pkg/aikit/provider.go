package aikit

import (
	"context"
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/providers"
)

// Provider interface for AI providers
type Provider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	Name() string

	// Optional methods for enhanced model handling
	GetSupportedModels() []string
}

// BaseProvider is now just a facade for backward compatibility
type BaseProvider struct {
	providers.BaseProvider
}

// NewProvider creates a new provider based on the name
func NewProvider(name string) (Provider, error) {
	switch name {
	case "openai":
		return providers.NewOpenAIProvider()
	case "claude":
		return providers.NewClaudeAIProvider()
	case "gemini":
		return providers.NewGeminiAIProvider()
	case "grok":
		return providers.NewGrokProvider()
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

// GetAvailableProviders returns a list of all available provider names
func GetAvailableProviders() []string {
	return []string{"openai", "claude", "gemini", "grok"}
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
