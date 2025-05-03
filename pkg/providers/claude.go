package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/httputil"
	"github.com/pkg/errors"
)

// Updated supported Claude models with function calling support information
var SupportedClaudeModels = map[string]struct {
	Supported        bool
	FunctionCalling  bool
	VisionCapable    bool
	MaxContextTokens int
}{
	"claude-3-5-sonnet-20240620": {true, true, true, 200000},
	"claude-3-opus-20240229":     {true, true, true, 200000},
	"claude-3-sonnet-20240229":   {true, true, true, 200000},
	"claude-3-haiku-20240307":    {true, true, true, 200000},
	"claude-2.1":                 {true, false, false, 100000},
	"claude-2.0":                 {true, false, false, 100000},
}

// ClaudeAdapter implements the Provider interface for Anthropic's Claude
type ClaudeAdapter struct {
	BaseProvider
	registeredFunctions map[string]aikit.FunctionDefinition
}

// NewClaudeAdapter creates a new Claude provider adapter
func NewClaudeAdapter() (*ClaudeAdapter, error) {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("CLAUDE_API_KEY environment variable is not set")
	}

	provider := &ClaudeAdapter{
		BaseProvider: BaseProvider{
			APIKey: apiKey,
			URL:    "https://api.anthropic.com/v1/messages",
		},
		registeredFunctions: make(map[string]aikit.FunctionDefinition),
	}

	// Get model from environment or use default, and validate it
	modelFromEnv := provider.GetEnvOrDefault("CLAUDE_MODEL", "claude-3-5-sonnet-20240620")
	provider.SetModel(modelFromEnv)

	return provider, nil
}

// SetModel sets the model and validates it
func (p *ClaudeAdapter) SetModel(model string) bool {
	modelInfo, exists := SupportedClaudeModels[model]
	if exists && modelInfo.Supported {
		p.Model = model
		return true
	}

	// Default to a modern model with function calling
	p.Model = "claude-3-5-sonnet-20240620"
	return false
}

// SupportsFunctionCalling returns whether the current model supports function calling
func (p *ClaudeAdapter) SupportsFunctionCalling() bool {
	modelInfo, exists := SupportedClaudeModels[p.Model]
	return exists && modelInfo.FunctionCalling
}

// RegisterFunction adds a function to the available functions
func (p *ClaudeAdapter) RegisterFunction(def aikit.FunctionDefinition) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("invalid function definition: %w", err)
	}

	p.registeredFunctions[def.Name] = def
	return nil
}

// GenerateResponse sends a prompt to Claude and returns the response
func (p *ClaudeAdapter) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  4096,
		"temperature": 0.7,
	}

	details := httputil.RequestDetails{
		URL:         p.URL,
		APIKey:      p.APIKey,
		RequestBody: requestBody,
		AdditionalHeaders: map[string]string{
			"x-api-key":         p.APIKey,
			"anthropic-version": "2023-06-01",
		},
	}

	options := httputil.ClientOptions{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	responseBody, err := httputil.SendRequest(ctx, details, options)
	if err != nil {
		return "", err
	}

	var claudeAIResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	err = json.Unmarshal(responseBody, &claudeAIResp)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling Claude response")
	}

	if len(claudeAIResp.Content) == 0 {
		return "", fmt.Errorf("no content in Claude response")
	}

	var textContent strings.Builder
	for _, content := range claudeAIResp.Content {
		if content.Type == "text" {
			textContent.WriteString(content.Text)
		}
	}

	return strings.TrimSpace(textContent.String()), nil
}

// GenerateResponseWithFunctions sends a prompt to Claude with function calling
func (p *ClaudeAdapter) GenerateResponseWithFunctions(
	ctx context.Context,
	prompt string,
	functionExecutor aikit.FunctionExecutorFunc,
) (string, error) {
	if !p.SupportsFunctionCalling() {
		return "", fmt.Errorf("model %s does not support function calling", p.Model)
	}

	if len(p.registeredFunctions) == 0 {
		return "", fmt.Errorf("no functions registered for function calling")
	}

	// Convert registered functions to Claude tools format
	var tools []map[string]interface{}
	for _, fn := range p.registeredFunctions {
		tools = append(tools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        fn.Name,
				"description": fn.Description,
				"parameters":  fn.Parameters,
			},
		})
	}

	// Create initial request
	requestBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]interface{}{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  4096,
		"temperature": 0.7,
		"tools":       tools,
	}

	details := httputil.RequestDetails{
		URL:         p.URL,
		APIKey:      p.APIKey,
		RequestBody: requestBody,
		AdditionalHeaders: map[string]string{
			"x-api-key":         p.APIKey,
			"anthropic-version": "2023-06-01",
		},
	}

	options := httputil.ClientOptions{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	responseBody, err := httputil.SendRequest(ctx, details, options)
	if err != nil {
		return "", err
	}

	// Parse the response
	var claudeResponse struct {
		ID      string `json:"id"`
		Content []struct {
			Type         string `json:"type"`
			Text         string `json:"text,omitempty"`
			FunctionCall *struct {
				Name       string          `json:"name"`
				Parameters json.RawMessage `json:"parameters"`
			} `json:"function_call,omitempty"`
		} `json:"content"`
	}

	err = json.Unmarshal(responseBody, &claudeResponse)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling Claude response")
	}

	// Handle function calls and continue the conversation
	var messages []map[string]interface{}
	var textResponse strings.Builder

	// Add the initial user message
	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": prompt,
	})

	// Process the response content
	for _, content := range claudeResponse.Content {
		if content.Type == "text" {
			textResponse.WriteString(content.Text)
			textResponse.WriteString("\n")
		} else if content.Type == "function_call" && content.FunctionCall != nil {
			// Extract function call details
			fnCall := aikit.FunctionCall{
				Name:       content.FunctionCall.Name,
				Parameters: content.FunctionCall.Parameters,
			}

			// Add the assistant's function call to the conversation
			messages = append(messages, map[string]interface{}{
				"role": "assistant",
				"content": []map[string]interface{}{
					{
						"type": "function_call",
						"function_call": map[string]interface{}{
							"name":       fnCall.Name,
							"parameters": json.RawMessage(fnCall.Parameters),
						},
					},
				},
			})

			// Execute function and get result
			fnResponse, err := functionExecutor(fnCall)
			if err != nil {
				// If function execution fails, include the error in the message
				textResponse.WriteString(fmt.Sprintf("Error executing function %s: %s\n", fnCall.Name, err.Error()))
				continue
			}

			// Convert function response to JSON
			fnResponseJSON, err := json.Marshal(fnResponse.Content)
			if err != nil {
				textResponse.WriteString(fmt.Sprintf("Error serializing function response: %s\n", err.Error()))
				continue
			}

			// Add the function response to the conversation
			messages = append(messages, map[string]interface{}{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "function_response",
						"function_response": map[string]interface{}{
							"name":    fnCall.Name,
							"content": json.RawMessage(fnResponseJSON),
						},
					},
				},
			})

			// Continue the conversation with the function response
			continuationRequestBody := map[string]interface{}{
				"model":       p.Model,
				"messages":    messages,
				"max_tokens":  4096,
				"temperature": 0.7,
				"tools":       tools,
			}

			details.RequestBody = continuationRequestBody
			continuationResponseBody, err := httputil.SendRequest(ctx, details, options)
			if err != nil {
				return textResponse.String(), errors.Wrap(err, "error sending continuation request")
			}

			// Parse continuation response
			var continuationResponse struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				} `json:"content"`
			}

			err = json.Unmarshal(continuationResponseBody, &continuationResponse)
			if err != nil {
				return textResponse.String(), errors.Wrap(err, "error unmarshaling continuation response")
			}

			// Append continuation response text to our result
			for _, content := range continuationResponse.Content {
				if content.Type == "text" {
					textResponse.WriteString(content.Text)
					textResponse.WriteString("\n")
				}
			}
		}
	}

	return strings.TrimSpace(textResponse.String()), nil
}

// Name returns the provider name
func (p *ClaudeAdapter) Name() string {
	return "claude"
}

// GetSupportedModels returns a list of supported models
func (p *ClaudeAdapter) GetSupportedModels() []string {
	models := make([]string, 0, len(SupportedClaudeModels))
	for model, info := range SupportedClaudeModels {
		if info.Supported {
			models = append(models, model)
		}
	}
	return models
}

// ClaudeProviderFactory creates Claude provider instances
type ClaudeProviderFactory struct{}

// NewClaudeProviderFactory creates a new Claude provider factory
func NewClaudeProviderFactory() *ClaudeProviderFactory {
	return &ClaudeProviderFactory{}
}

// Create returns a new Claude provider instance
func (f *ClaudeProviderFactory) Create() (Provider, error) {
	return NewClaudeAdapter()
}

// GetAvailableModels returns a list of available models for this provider
func (f *ClaudeProviderFactory) GetAvailableModels() []string {
	models := make([]string, 0, len(SupportedClaudeModels))
	for model, info := range SupportedClaudeModels {
		if info.Supported {
			models = append(models, model)
		}
	}
	return models
}

// GetCapabilities returns a list of capabilities supported by this provider
func (f *ClaudeProviderFactory) GetCapabilities() []ProviderCapability {
	// Get the model from environment or use default
	envModel := os.Getenv("CLAUDE_MODEL")
	if envModel == "" {
		envModel = "claude-3-5-sonnet-20240620"
	}

	// Get the model info
	modelInfo, exists := SupportedClaudeModels[envModel]
	if !exists {
		// Default to a model we know
		modelInfo = SupportedClaudeModels["claude-3-5-sonnet-20240620"]
	}

	// Build capabilities list based on model features
	capabilities := []ProviderCapability{}

	if modelInfo.FunctionCalling {
		capabilities = append(capabilities, CapabilityFunctionCalling)
	}

	if modelInfo.VisionCapable {
		capabilities = append(capabilities, CapabilityVision)
		capabilities = append(capabilities, CapabilityMultimodal)
	}

	// All Claude models support streaming
	capabilities = append(capabilities, CapabilityStreaming)

	return capabilities
}
