package providers

import (
	"context"
	"os"
)

// Provider interface
type Provider interface {
	// Core methods
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	Name() string
	GetSupportedModels() []string // Returns supported models

	// Function calling capabilities
	SupportsFunctionCalling() bool
	RegisterFunction(def FunctionDefinition) error
	RegisterFunctions(functions []FunctionDefinition)
	GenerateResponseWithFunctions(
		ctx context.Context,
		prompt string,
		functionExecutor FunctionExecutorFunc,
	) (string, error)
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
