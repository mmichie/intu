package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
)

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
	provider.Model = provider.GetEnvOrDefault("GROK_MODEL", "grok-beta")
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
