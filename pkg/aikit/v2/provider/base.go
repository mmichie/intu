// Package provider contains AI provider implementations
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mmichie/intu/pkg/aikit/v2/errors"
	"github.com/mmichie/intu/pkg/aikit/v2/function"
)

// ModelCapabilities defines what a model can do
type ModelCapabilities struct {
	Supported        bool
	FunctionCalling  bool
	VisionCapable    bool
	MaxContextTokens int
	DefaultMaxTokens int
}

// BaseProvider contains common fields and methods for all providers
type BaseProvider struct {
	apiKey   string
	model    string
	baseURL  string
	registry *function.Registry
	executor function.FunctionExecutor

	// Provider-specific configuration
	providerName    string
	defaultModel    string
	supportedModels map[string]ModelCapabilities
}

// NewBaseProvider creates a new base provider instance
func NewBaseProvider(providerName, apiKey, model, baseURL, defaultModel string, supportedModels map[string]ModelCapabilities) *BaseProvider {
	return &BaseProvider{
		apiKey:          apiKey,
		model:           model,
		baseURL:         baseURL,
		providerName:    providerName,
		defaultModel:    defaultModel,
		supportedModels: supportedModels,
	}
}

// SetFunctionRegistry sets the function registry for this provider
func (b *BaseProvider) SetFunctionRegistry(registry *function.Registry) {
	b.registry = registry
}

// SetFunctionExecutor sets the function executor for this provider
func (b *BaseProvider) SetFunctionExecutor(executor function.FunctionExecutor) {
	b.executor = executor
}

// GetModel returns the current model
func (b *BaseProvider) GetModel() string {
	return b.model
}

// ValidateAndSetModel validates the model and sets it with fallback to default
func (b *BaseProvider) ValidateAndSetModel(model string) error {
	if model == "" {
		b.model = b.defaultModel
		return nil
	}

	capabilities, exists := b.supportedModels[model]
	if !exists || !capabilities.Supported {
		return errors.New(b.providerName, "validate_model",
			fmt.Errorf("unsupported model: %s", model))
	}

	b.model = model
	return nil
}

// GetModelCapabilities returns the capabilities for the current model
func (b *BaseProvider) GetModelCapabilities() (ModelCapabilities, bool) {
	capabilities, exists := b.supportedModels[b.model]
	return capabilities, exists
}

// SupportsFunctionCalling checks if the current model supports function calling
func (b *BaseProvider) SupportsFunctionCalling() bool {
	capabilities, exists := b.GetModelCapabilities()
	return exists && capabilities.FunctionCalling
}

// SupportsVision checks if the current model supports vision
func (b *BaseProvider) SupportsVision() bool {
	capabilities, exists := b.GetModelCapabilities()
	return exists && capabilities.VisionCapable
}

// ValidateFunctionSupport checks if functions are supported and returns an error if not
func (b *BaseProvider) ValidateFunctionSupport(request Request) error {
	if request.FunctionRegistry != nil && !b.SupportsFunctionCalling() {
		return errors.New(b.providerName, "validate_function_support",
			fmt.Errorf("model %s does not support function calling", b.model))
	}
	return nil
}

// GetDefaultMaxTokens returns the default max tokens for the current model
func (b *BaseProvider) GetDefaultMaxTokens() int {
	capabilities, exists := b.GetModelCapabilities()
	if exists && capabilities.DefaultMaxTokens > 0 {
		return capabilities.DefaultMaxTokens
	}
	return 4096 // Fallback default
}

// PrepareRequestDefaults applies default values to a request
func (b *BaseProvider) PrepareRequestDefaults(request *Request) {
	if request.Temperature == 0 {
		request.Temperature = 0.7
	}

	if request.MaxTokens == 0 {
		request.MaxTokens = b.GetDefaultMaxTokens()
	}
}

// HTTPOptions contains common HTTP configuration
type HTTPOptions struct {
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
}

// GetDefaultHTTPOptions returns standard HTTP options
func (b *BaseProvider) GetDefaultHTTPOptions() HTTPOptions {
	return HTTPOptions{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// GetStreamingHTTPOptions returns streaming-specific HTTP options
func (b *BaseProvider) GetStreamingHTTPOptions() HTTPOptions {
	return HTTPOptions{
		Timeout:    90 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// ExecuteFunctionCall executes a function call and returns a formatted response
func (b *BaseProvider) ExecuteFunctionCall(ctx context.Context, funcCall function.FunctionCall) (string, error) {
	if b.executor == nil {
		return "", errors.New(b.providerName, "execute_function",
			fmt.Errorf("no function executor configured"))
	}

	// Execute the function
	funcResp, err := b.executor(funcCall)
	if err != nil {
		return b.formatFunctionError(funcCall.Name, err), nil
	}

	// Format the response
	return b.formatFunctionResponse(funcResp)
}

// formatFunctionError formats a function execution error
func (b *BaseProvider) formatFunctionError(funcName string, err error) string {
	errorResp := map[string]interface{}{
		"function": funcName,
		"error":    err.Error(),
		"status":   "failed",
	}

	jsonBytes, _ := json.MarshalIndent(errorResp, "", "  ")
	return fmt.Sprintf("Function '%s' failed with error: %s\n\nError Details:\n%s",
		funcName, err.Error(), string(jsonBytes))
}

// formatFunctionResponse formats a successful function response
func (b *BaseProvider) formatFunctionResponse(resp function.FunctionResponse) (string, error) {
	// Create a JSON representation
	jsonResp := map[string]interface{}{
		"function": resp.Name,
		"result":   resp.Content,
		"status":   "success",
	}

	if resp.Metadata != nil {
		jsonResp["metadata"] = resp.Metadata
	}

	jsonBytes, err := json.MarshalIndent(jsonResp, "", "  ")
	if err != nil {
		return "", errors.New(b.providerName, "format_function_response",
			fmt.Errorf("failed to marshal function response: %w", err))
	}

	// Create a readable summary
	summary := fmt.Sprintf("Successfully executed function '%s'\n\nResult:\n%s",
		resp.Name, string(jsonBytes))

	return summary, nil
}

// HandleStreamingFallback handles the common pattern of falling back to non-streaming for function calls
func (b *BaseProvider) HandleStreamingFallback(ctx context.Context, request Request, generateFunc func(context.Context, Request) (Response, error), stream StreamHandler) error {
	if request.FunctionRegistry != nil && request.FunctionExecutor != nil {
		// Fall back to non-streaming for function calling
		resp, err := generateFunc(ctx, request)
		if err != nil {
			return err
		}

		// Simulate streaming by splitting the content
		return SimulateStreaming(ctx, resp.Content, stream)
	}

	return errors.New(b.providerName, "streaming_fallback",
		fmt.Errorf("no fallback logic implemented"))
}

// WrapProviderError wraps an error with provider context
func (b *BaseProvider) WrapProviderError(operation string, err error) error {
	return errors.New(b.providerName, operation, err)
}

// ParseJSONResponse is a generic helper for parsing JSON responses
func (b *BaseProvider) ParseJSONResponse(data []byte, target interface{}) error {
	if err := json.Unmarshal(data, target); err != nil {
		return b.WrapProviderError("parse_response",
			fmt.Errorf("failed to parse JSON response: %w", err))
	}
	return nil
}
