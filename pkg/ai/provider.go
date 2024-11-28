package ai

import (
	"context"
	"fmt"
	"os"
)

type Provider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	Name() string
}

type BaseProvider struct {
	APIKey string
	Model  string
	URL    string
}

func (p *BaseProvider) GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewProvider(name string) (Provider, error) {
	switch name {
	case "openai":
		return NewOpenAIProvider()
	case "claude":
		return NewClaudeAIProvider()
	case "gemini":
		return NewGeminiAIProvider()
	case "grok":
		return NewGrokProvider()
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
