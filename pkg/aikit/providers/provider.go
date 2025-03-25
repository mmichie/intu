package providers

import (
	"context"
	"os"
)

// Provider interface
type Provider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	Name() string

	// Optional methods - implementing providers can support these
	// for enhanced model handling
	GetSupportedModels() []string // Returns supported models
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

// GetSupportedModels is a default implementation that returns an empty slice
// Providers should override this method to return their supported models
func (p *BaseProvider) GetSupportedModels() []string {
	return []string{}
}

// SetModel sets the model for this provider if it's supported
// Returns true if the model was set, false if it wasn't supported
func (p *BaseProvider) SetModel(model string, supportedModels map[string]bool, defaultModel string) bool {
	if supportedModels[model] {
		p.Model = model
		return true
	}
	p.Model = defaultModel
	return false
}
