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
	provider.Model = provider.GetEnvOrDefault("OPENAI_MODEL", "gpt-4")

	return provider, nil
}

func (p *OpenAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	responseBody, err := sendRequest(ctx, p.URL, p.APIKey, requestBody)
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
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}
