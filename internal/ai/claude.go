package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type ClaudeAIProvider struct {
	APIKey string
	Model  string
}

type claudeAIRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
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
		APIKey: apiKey,
		Model:  model,
	}, nil
}

func (p *ClaudeAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	requestBody := claudeAIRequest{
		Model: p.Model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	var claudeAIResp claudeAIResponse
	err = json.Unmarshal(body, &claudeAIResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	if len(claudeAIResp.Content) == 0 {
		return "", fmt.Errorf("no content in Claude AI response")
	}

	return strings.TrimSpace(claudeAIResp.Content[0].Text), nil
}
