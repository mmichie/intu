package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mmichie/intu/pkg/httputil"
	"github.com/pkg/errors"
)

// SupportedOpenAIModels is a list of supported OpenAI models
var SupportedOpenAIModels = map[string]bool{
	"gpt-4":         true,
	"gpt-4-turbo":   true,
	"gpt-4o":        true,
	"gpt-3.5-turbo": true,
}

type OpenAIProvider struct {
	BaseProvider
}

func NewOpenAIProvider() (*OpenAIProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}
	provider := &OpenAIProvider{
		BaseProvider: BaseProvider{
			APIKey: apiKey,
			URL:    "https://api.openai.com/v1/chat/completions",
		},
	}

	// Get model from environment or use default, and validate it
	modelFromEnv := provider.GetEnvOrDefault("OPENAI_MODEL", "gpt-4")
	provider.SetModel(modelFromEnv, SupportedOpenAIModels, "gpt-4")
	return provider, nil
}

func (p *OpenAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	details := httputil.RequestDetails{
		URL:         p.URL,
		APIKey:      p.APIKey,
		RequestBody: requestBody,
	}

	options := httputil.ClientOptions{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	responseBody, err := httputil.SendRequest(ctx, details, options)
	if err != nil {
		return "", err
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err = json.Unmarshal(responseBody, &openAIResp)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling OpenAI response")
	}
	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}
	return openAIResp.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

// GetSupportedModels returns a list of supported models for this provider
func (p *OpenAIProvider) GetSupportedModels() []string {
	models := make([]string, 0, len(SupportedOpenAIModels))
	for model := range SupportedOpenAIModels {
		models = append(models, model)
	}
	return models
}
