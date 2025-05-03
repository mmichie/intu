package providers

import (
	"context"
	"os"

	"github.com/mmichie/intu/pkg/aikit"
)

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

// Provider interface for AI providers
type Provider interface {
	// Core methods
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	Name() string
	GetSupportedModels() []string

	// Function calling capabilities
	SupportsFunctionCalling() bool
	RegisterFunction(def aikit.FunctionDefinition) error
	GenerateResponseWithFunctions(
		ctx context.Context,
		prompt string,
		functionExecutor aikit.FunctionExecutorFunc,
	) (string, error)
}

// ProviderCapability represents a capability that a provider may support
type ProviderCapability string

const (
	// CapabilityFunctionCalling indicates support for function calling
	CapabilityFunctionCalling ProviderCapability = "function_calling"

	// CapabilityVision indicates support for vision/image input
	CapabilityVision ProviderCapability = "vision"

	// CapabilityStreaming indicates support for streaming responses
	CapabilityStreaming ProviderCapability = "streaming"

	// CapabilityMultimodal indicates support for multiple input modalities
	CapabilityMultimodal ProviderCapability = "multimodal"
)

// ProviderFactory creates Provider instances
type ProviderFactory interface {
	// Create returns a new Provider instance
	Create() (Provider, error)

	// GetAvailableModels returns a list of available models for this provider
	GetAvailableModels() []string

	// GetCapabilities returns a list of capabilities supported by this provider
	GetCapabilities() []ProviderCapability
}
