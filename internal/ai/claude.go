package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
)

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
	provider.Model = provider.GetEnvOrDefault("CLAUDE_MODEL", "claude-3-5-sonnet-20240620")

	return provider, nil
}

func (p *ClaudeAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  1000,
		"temperature": 0.7,
	}

	headers := map[string]string{
		"x-api-key":         p.APIKey,
		"anthropic-version": "2023-06-01",
	}

	responseBody, err := sendRequest(ctx, p.URL, "", requestBody, headers)
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
