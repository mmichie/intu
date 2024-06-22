package intu

import (
	"fmt"
)

// Provider represents an AI provider
type Provider interface {
	GenerateResponse(prompt string) (string, error)
}

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	APIKey string
}

func (p *OpenAIProvider) GenerateResponse(prompt string) (string, error) {
	// TODO: Implement OpenAI API call here
	// This is a placeholder implementation
	return fmt.Sprintf("OpenAI response to: %s", prompt), nil
}
