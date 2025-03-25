package providers

import (
	"context"
	"os"
)

// Provider interface
type Provider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	Name() string
}

// BaseProvider contains common provider fields and methods
type BaseProvider struct {
	APIKey string
	Model  string
	URL    string
}

// GetEnvOrDefault retrieves an environment variable or returns a default
func (p *BaseProvider) GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}