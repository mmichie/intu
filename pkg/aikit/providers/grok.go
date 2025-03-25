package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
)

// SupportedGrokModels is a list of supported Grok models
var SupportedGrokModels = map[string]bool{
	"grok-1":      true,
	"grok-beta":   true,
	"grok-1-mini": true,
}

type GrokProvider struct {
	BaseProvider
}

func NewGrokProvider() (*GrokProvider, error) {
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("XAI_API_KEY environment variable is not set")
	}
	provider := &GrokProvider{
		BaseProvider: BaseProvider{
			APIKey: apiKey,
			URL:    "https://api.x.ai/v1/chat/completions",
		},
	}

	// Get model from environment or use default, and validate it
	modelFromEnv := provider.GetEnvOrDefault("GROK_MODEL", "grok-1")
	provider.SetModel(modelFromEnv, SupportedGrokModels, "grok-1")
	return provider, nil
}

func (p *GrokProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	}

	details := RequestDetails{
		URL:         p.URL,
		APIKey:      p.APIKey,
		RequestBody: requestBody,
	}

	options := ClientOptions{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	responseBody, err := SendRequest(ctx, details, options)
	if err != nil {
		return "", err
	}

	var grokResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err = json.Unmarshal(responseBody, &grokResp); err != nil {
		return "", errors.Wrap(err, "error unmarshaling Grok response")
	}
	if len(grokResp.Choices) == 0 {
		return "", fmt.Errorf("no response from Grok")
	}
	return grokResp.Choices[0].Message.Content, nil
}

func (p *GrokProvider) Name() string {
	return "grok"
}

// GetSupportedModels returns a list of supported models for this provider
func (p *GrokProvider) GetSupportedModels() []string {
	models := make([]string, 0, len(SupportedGrokModels))
	for model := range SupportedGrokModels {
		models = append(models, model)
	}
	return models
}
