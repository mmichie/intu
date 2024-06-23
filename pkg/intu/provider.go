package intu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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

type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
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

func (p *OpenAIProvider) ValidateConfig() error {
	if p.APIKey == "" {
		return fmt.Errorf("OpenAI API key is not set")
	}
	if p.Model == "" {
		return fmt.Errorf("OpenAI model is not set")
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
			{Role: "system", Content: "You are a helpful assistant that generates concise git commit messages."},
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
