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