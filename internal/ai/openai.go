package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type OpenAIProvider struct {
	BaseProvider
}

type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewOpenAIProvider() (*OpenAIProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4"
	}

	return &OpenAIProvider{
		BaseProvider: BaseProvider{
			APIKey: apiKey,
			Model:  model,
			URL:    "https://api.openai.com/v1/chat/completions",
		},
	}, nil
}

func (p *OpenAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	requestBody := openAIRequest{
		Model: p.Model,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}

	headers := map[string]string{
		"Authorization": "Bearer " + p.APIKey,
	}

	responseBody, err := p.sendRequest(ctx, requestBody, headers)
	if err != nil {
		return "", err
	}

	var openAIResp openAIResponse
	err = json.Unmarshal(responseBody, &openAIResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}
