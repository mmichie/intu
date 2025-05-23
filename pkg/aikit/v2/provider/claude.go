package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
	aierrors "github.com/mmichie/intu/pkg/aikit/v2/errors"
	"github.com/mmichie/intu/pkg/aikit/v2/function"
	"github.com/mmichie/intu/pkg/httputil"
)

const (
	defaultClaudeURL       = "https://api.anthropic.com/v1/messages"
	defaultClaudeModel     = "claude-3-5-sonnet-20240620"
	defaultClaudeMaxTokens = 4096
)

// SupportedClaudeModels defines capabilities for Claude models
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

// ClaudeProvider implements the Provider interface for Anthropic's Claude
type ClaudeProvider struct {
	apiKey  string
	model   string
	baseURL string

	registry *function.Registry
	executor function.FunctionExecutor
}

// ClaudeFactory creates Claude providers
type ClaudeFactory struct{}

// Name returns the provider name
func (f *ClaudeFactory) Name() string {
	return "claude"
}

// Create returns a new Claude provider
func (f *ClaudeFactory) Create(cfg config.Config) (Provider, error) {
	// Check for API key
	if cfg.APIKey == "" {
		return nil, aierrors.New("claude", "create", aierrors.ErrInvalidConfig)
	}

	// Determine model to use
	model := cfg.Model
	if model == "" {
		model = defaultClaudeModel
	}

	// Validate model
	if _, exists := SupportedClaudeModels[model]; !exists {
		model = defaultClaudeModel
	}

	// Determine base URL
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultClaudeURL
	}

	// Create provider
	return &ClaudeProvider{
		apiKey:   cfg.APIKey,
		model:    model,
		baseURL:  baseURL,
		registry: function.NewRegistry(),
	}, nil
}

// GetAvailableModels returns supported Claude models
func (f *ClaudeFactory) GetAvailableModels() []string {
	models := make([]string, 0, len(SupportedClaudeModels))
	for model, info := range SupportedClaudeModels {
		if info.Supported {
			models = append(models, model)
		}
	}
	return models
}

// GetCapabilities returns Claude capabilities
func (f *ClaudeFactory) GetCapabilities() []string {
	return []string{
		"function_calling",
		"streaming",
		"vision",
		"multimodal",
	}
}

// Name returns the provider name
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// Model returns the current model
func (p *ClaudeProvider) Model() string {
	return p.model
}

// Capabilities returns supported capabilities for the current model
func (p *ClaudeProvider) Capabilities() []string {
	capabilities := []string{"streaming"}

	// Add model-specific capabilities
	if info, exists := SupportedClaudeModels[p.model]; exists {
		if info.FunctionCalling {
			capabilities = append(capabilities, "function_calling")
		}
		if info.VisionCapable {
			capabilities = append(capabilities, "vision", "multimodal")
		}
	}

	return capabilities
}

// supportsFunctionCalling checks if current model supports function calling
func (p *ClaudeProvider) supportsFunctionCalling() bool {
	info, exists := SupportedClaudeModels[p.model]
	return exists && info.FunctionCalling
}

// GenerateResponse sends a request to Claude and returns the response
func (p *ClaudeProvider) GenerateResponse(ctx context.Context, request Request) (Response, error) {
	// Check for function calling support if requested
	if request.FunctionRegistry != nil && !p.supportsFunctionCalling() {
		return Response{}, aierrors.New("claude", "generate_response",
			fmt.Errorf("model %s does not support function calling", p.model))
	}

	// Prepare Claude request structure
	maxTokens := defaultClaudeMaxTokens
	if request.MaxTokens > 0 {
		maxTokens = request.MaxTokens
	}

	temperature := 0.7
	if request.Temperature > 0 {
		temperature = request.Temperature
	}

	claudeReq := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": request.Prompt},
		},
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}

	// Add tools if function registry is provided
	if request.FunctionRegistry != nil {
		functions := request.FunctionRegistry.List()
		if len(functions) > 0 {
			var tools []map[string]interface{}
			for _, fn := range functions {
				tools = append(tools, map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":        fn.Name,
						"description": fn.Description,
						"parameters":  fn.Parameters,
					},
				})
			}
			claudeReq["tools"] = tools
		}
	}

	// Configure HTTP request
	details := httputil.RequestDetails{
		URL:    p.baseURL,
		APIKey: p.apiKey,
		AdditionalHeaders: map[string]string{
			"x-api-key":         p.apiKey,
			"anthropic-version": "2023-06-01",
		},
		RequestBody: claudeReq,
	}

	options := httputil.ClientOptions{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	// Send the request
	responseBody, err := httputil.SendRequest(ctx, details, options)
	if err != nil {
		return Response{}, aierrors.New("claude", "generate_response", err)
	}

	// Parse the response
	var claudeResp struct {
		Content []struct {
			Type         string `json:"type"`
			Text         string `json:"text,omitempty"`
			FunctionCall *struct {
				Name       string          `json:"name"`
				Parameters json.RawMessage `json:"parameters"`
			} `json:"function_call,omitempty"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	err = json.Unmarshal(responseBody, &claudeResp)
	if err != nil {
		return Response{}, aierrors.New("claude", "parse_response", err)
	}

	// Build the provider-agnostic response
	response := Response{
		Model:    p.model,
		Provider: "claude",
		Usage: &UsageInfo{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}

	// Extract content and function calls
	var textContent strings.Builder
	for _, content := range claudeResp.Content {
		if content.Type == "text" {
			textContent.WriteString(content.Text)
		} else if content.Type == "function_call" && content.FunctionCall != nil {
			// Save the function call
			response.FunctionCall = &function.FunctionCall{
				Name:       content.FunctionCall.Name,
				Parameters: content.FunctionCall.Parameters,
			}

			// If we have a function executor, execute it
			if request.FunctionExecutor != nil {
				fnResponse, err := request.FunctionExecutor(*response.FunctionCall)
				if err != nil {
					textContent.WriteString(fmt.Sprintf("Error executing function %s: %s\n",
						response.FunctionCall.Name, err.Error()))
					continue
				}

				// Add a textual representation of the function response
				if fnResponse.Error != "" {
					textContent.WriteString(fmt.Sprintf("Function %s returned error: %s\n",
						fnResponse.Name, fnResponse.Error))
				} else {
					// Add simple summary of function result
					textContent.WriteString(fmt.Sprintf("Function %s executed successfully.\n",
						fnResponse.Name))

					// Convert result to JSON string for representation
					resultJSON, _ := json.MarshalIndent(fnResponse.Content, "", "  ")
					textContent.WriteString(fmt.Sprintf("Result: %s\n", resultJSON))
				}
			}
		}
	}

	response.Content = strings.TrimSpace(textContent.String())
	return response, nil
}

// GenerateStreamingResponse streams a response from Claude
func (p *ClaudeProvider) GenerateStreamingResponse(ctx context.Context, request Request, handler StreamHandler) error {
	// Check for function calling support if requested
	if request.FunctionRegistry != nil && !p.supportsFunctionCalling() {
		return aierrors.New("claude", "generate_streaming_response",
			fmt.Errorf("model %s does not support function calling", p.model))
	}

	// Claude doesn't properly support streaming with function calls
	if request.FunctionRegistry != nil && request.FunctionExecutor != nil {
		// Fall back to non-streaming for function calls
		resp, err := p.GenerateResponse(ctx, request)
		if err != nil {
			return err
		}

		// Simulate streaming
		return SimulateStreaming(ctx, resp.Content, func(chunk ResponseChunk) error {
			return handler(chunk)
		})
	}

	// Prepare regular streaming request
	maxTokens := defaultClaudeMaxTokens
	if request.MaxTokens > 0 {
		maxTokens = request.MaxTokens
	}

	temperature := 0.7
	if request.Temperature > 0 {
		temperature = request.Temperature
	}

	claudeReq := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": request.Prompt},
		},
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"stream":      true,
	}

	// Add tools if function registry is provided
	if request.FunctionRegistry != nil {
		functions := request.FunctionRegistry.List()
		if len(functions) > 0 {
			var tools []map[string]interface{}
			for _, fn := range functions {
				tools = append(tools, map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":        fn.Name,
						"description": fn.Description,
						"parameters":  fn.Parameters,
					},
				})
			}
			claudeReq["tools"] = tools
		}
	}

	// Configure HTTP request
	details := httputil.RequestDetails{
		URL:    p.baseURL,
		APIKey: p.apiKey,
		AdditionalHeaders: map[string]string{
			"x-api-key":         p.apiKey,
			"anthropic-version": "2023-06-01",
		},
		RequestBody: claudeReq,
		Stream:      true,
	}

	options := httputil.ClientOptions{
		Timeout:       90 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	// Process the streaming response
	textStreamHandler := func(data string) error {
		// Skip empty chunks and "[DONE]" messages
		if data == "" || data == "[DONE]" {
			return nil
		}

		// Remove the "data: " prefix if present
		if strings.HasPrefix(data, "data: ") {
			data = strings.TrimPrefix(data, "data: ")
		}

		// Parse the JSON chunk
		var streamResp struct {
			Type         string                 `json:"type"`
			Delta        *struct{ Text string } `json:"delta"`
			ContentBlock *struct{ Text string } `json:"content_block"`
		}

		err := json.Unmarshal([]byte(data), &streamResp)
		if err != nil {
			return errors.New("error parsing stream chunk: " + err.Error())
		}

		// Extract the text content based on the response format
		var text string
		if streamResp.Type == "content_block_delta" && streamResp.Delta != nil {
			text = streamResp.Delta.Text
		} else if streamResp.Type == "content_block_start" && streamResp.ContentBlock != nil {
			text = streamResp.ContentBlock.Text
		}

		// If we have text, send it to the handler
		if text != "" {
			return handler(ResponseChunk{
				Content: text,
				IsFinal: false,
			})
		}

		return nil
	}

	err := httputil.SendTextStreamingRequest(ctx, details, options, textStreamHandler)
	if err != nil {
		return aierrors.New("claude", "generate_streaming_response", err)
	}

	// Send final chunk
	return handler(ResponseChunk{
		Content: "",
		IsFinal: true,
	})
}

// Register the Claude factory
func init() {
	RegisterFactory(&ClaudeFactory{})
}
