package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Constants for provider names and environment variables
const (
	ProviderOpenAI = "openai"
	ProviderClaude = "claude"

	EnvOpenAIAPIKey = "OPENAI_API_KEY"
	EnvClaudeAPIKey = "CLAUDE_API_KEY"

	DefaultOpenAIModel = "gpt-4"
	DefaultClaudeModel = "claude-3-5-sonnet-20240620"
)

// Provider defines the interface for AI providers
type Provider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
}

// SelectProvider chooses and initializes the appropriate AI provider based on the given name
func SelectProvider(providerName string) (Provider, error) {
	switch strings.ToLower(providerName) {
	case ProviderOpenAI:
		return NewOpenAIProvider()
	case ProviderClaude:
		return NewClaudeAIProvider()
	case "":
		// If no provider is specified, use the one from config or default to OpenAI
		configProvider := viper.GetString("default_provider")
		if configProvider == "" {
			configProvider = ProviderOpenAI
		}
		return SelectProvider(configProvider)
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}

// InitProviderConfig initializes the configuration for AI providers
func InitProviderConfig() {
	viper.SetDefault("openai_model", DefaultOpenAIModel)
	viper.SetDefault("claude_model", DefaultClaudeModel)
	viper.SetDefault("default_provider", ProviderOpenAI)

	// Bind environment variables
	viper.BindEnv("openai_api_key", EnvOpenAIAPIKey)
	viper.BindEnv("claude_api_key", EnvClaudeAPIKey)
}

// GetProviderAPIKey retrieves the API key for the specified provider
func GetProviderAPIKey(provider string) (string, error) {
	var apiKey string
	switch provider {
	case ProviderOpenAI:
		apiKey = viper.GetString("openai_api_key")
		if apiKey == "" {
			apiKey = os.Getenv(EnvOpenAIAPIKey)
		}
	case ProviderClaude:
		apiKey = viper.GetString("claude_api_key")
		if apiKey == "" {
			apiKey = os.Getenv(EnvClaudeAPIKey)
		}
	default:
		return "", fmt.Errorf("unknown provider: %s", provider)
	}

	if apiKey == "" {
		return "", fmt.Errorf("API key not found for provider: %s", provider)
	}

	return apiKey, nil
}

// GetProviderModel retrieves the model name for the specified provider
func GetProviderModel(provider string) string {
	switch provider {
	case ProviderOpenAI:
		return viper.GetString("openai_model")
	case ProviderClaude:
		return viper.GetString("claude_model")
	default:
		return ""
	}
}
