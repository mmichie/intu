package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mmichie/intu/pkg/httputil"
	"github.com/pkg/errors"
)

// SupportedClaudeModels is a list of supported Claude models
var SupportedClaudeModels = map[string]bool{
	"claude-3-5-sonnet-20240620": true,
	"claude-3-opus-20240229":     true,
	"claude-3-sonnet-20240229":   true,
	"claude-3-haiku-20240307":    true,
	"claude-2.1":                 true,
	"claude-2.0":                 true,
}

type ClaudeAIProvider struct {
	BaseProvider
}

func NewClaudeAIProvider() (*ClaudeAIProvider, error) {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("CLAUDE_API_KEY environment variable is not set")
	}
	provider := &ClaudeAIProvider{
		BaseProvider: BaseProvider{
			APIKey: apiKey,
			URL:    "https://api.anthropic.com/v1/messages",
		},
	}

	// Get model from environment or use default, and validate it
	modelFromEnv := provider.GetEnvOrDefault("CLAUDE_MODEL", "claude-3-5-sonnet-20240620")
	provider.SetModel(modelFromEnv, SupportedClaudeModels, "claude-3-5-sonnet-20240620")
	return provider, nil
}

func (p *ClaudeAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  4096,
		"temperature": 0.7,
	}

	details := httputil.RequestDetails{
		URL:         p.URL,
		APIKey:      p.APIKey,
		RequestBody: requestBody,
		AdditionalHeaders: map[string]string{
			"x-api-key":         p.APIKey,
			"anthropic-version": "2023-06-01",
		},
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

	var claudeAIResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	err = json.Unmarshal(responseBody, &claudeAIResp)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling Claude AI response")
	}
	if len(claudeAIResp.Content) == 0 {
		return "", fmt.Errorf("no content in Claude AI response")
	}
	return strings.TrimSpace(claudeAIResp.Content[0].Text), nil
}

func (p *ClaudeAIProvider) Name() string {
	return "claude"
}

// GetSupportedModels returns a list of supported models for this provider
func (p *ClaudeAIProvider) GetSupportedModels() []string {
	models := make([]string, 0, len(SupportedClaudeModels))
	for model := range SupportedClaudeModels {
		models = append(models, model)
	}
	return models
}
