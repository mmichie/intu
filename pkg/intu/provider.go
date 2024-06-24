package intu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Provider represents an AI provider
type Provider interface {
	GenerateResponse(prompt string) (string, error)
	ValidateConfig() error
}

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	APIKey string
	Model  string
}

// ClaudeAIProvider implements the Provider interface for Claude AI
type ClaudeAIProvider struct {
	APIKey string
	Model  string
}

type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type claudeAIRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type claudeAIResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func NewOpenAIProvider() (*OpenAIProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4" // Default model if not specified
	}

	return &OpenAIProvider{
		APIKey: apiKey,
		Model:  model,
	}, nil
}

func NewClaudeAIProvider() (*ClaudeAIProvider, error) {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("CLAUDE_API_KEY environment variable is not set")
	}

	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "claude-3-5-sonnet-20240620" // Default to Claude-3.5-Sonnet model
	}

	return &ClaudeAIProvider{
		APIKey: apiKey,
		Model:  model,
	}, nil
}

func (p *OpenAIProvider) ValidateConfig() error {
	if p.APIKey == "" {
		return fmt.Errorf("OpenAI API key is not set")
	}
	if p.Model == "" {
		return fmt.Errorf("OpenAI model is not set")
	}
	return nil
}

func (p *ClaudeAIProvider) ValidateConfig() error {
	if p.APIKey == "" {
		return fmt.Errorf("Claude AI API key is not set")
	}
	if p.Model == "" {
		return fmt.Errorf("Claude AI model is not set")
	}
	return nil
}

func (p *OpenAIProvider) GenerateResponse(prompt string) (string, error) {
	if err := p.ValidateConfig(); err != nil {
		return "", err
	}

	requestBody := openAIRequest{
		Model: p.Model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	var openAIResp openAIResponse
	err = json.Unmarshal(body, &openAIResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

func (p *ClaudeAIProvider) GenerateResponse(prompt string) (string, error) {
	if err := p.ValidateConfig(); err != nil {
		return "", err
	}

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
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	var claudeAIResp claudeAIResponse
	err = json.Unmarshal(body, &claudeAIResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	if len(claudeAIResp.Content) == 0 {
		return "", fmt.Errorf("no content in Claude AI response")
	}

	return strings.TrimSpace(claudeAIResp.Content[0].Text), nil
}
