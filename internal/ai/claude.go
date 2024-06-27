package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ClaudeAIProvider struct {
	BaseProvider
}

type claudeAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type claudeAIResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func NewClaudeAIProvider() (*ClaudeAIProvider, error) {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("CLAUDE_API_KEY environment variable is not set")
	}

	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "claude-3-5-sonnet-20240620"
	}

	return &ClaudeAIProvider{
		BaseProvider: BaseProvider{
			APIKey: apiKey,
			Model:  model,
			URL:    "https://api.anthropic.com/v1/messages",
		},
	}, nil
}

func (p *ClaudeAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	requestBody := claudeAIRequest{
		Model: p.Model,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	headers := map[string]string{
		"x-api-key":         p.APIKey,
		"anthropic-version": "2023-06-01",
	}

	responseBody, err := p.sendRequest(ctx, requestBody, headers)
	if err != nil {
		return "", err
	}

	var claudeAIResp claudeAIResponse
	err = json.Unmarshal(responseBody, &claudeAIResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	if len(claudeAIResp.Content) == 0 {
		return "", fmt.Errorf("no content in Claude AI response")
	}

	return strings.TrimSpace(claudeAIResp.Content[0].Text), nil
}
