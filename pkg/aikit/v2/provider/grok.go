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
	defaultGrokURL       = "https://api.xai.com/v1/chat/completions"
	defaultGrokModel     = "grok-1"
	defaultGrokMaxTokens = 2048
)

// SupportedGrokModels defines capabilities for Grok models
var SupportedGrokModels = map[string]struct {
	Supported        bool
	FunctionCalling  bool
	VisionCapable    bool
	MaxContextTokens int
}{
	"grok-1":      {true, true, false, 65536},
	"grok-1-mini": {true, false, false, 32768},
}

// GrokProvider implements the Provider interface for xAI's Grok
type GrokProvider struct {
	apiKey  string
	model   string
	baseURL string

	registry *function.Registry
	executor function.FunctionExecutor
}

// GrokFactory creates Grok providers
type GrokFactory struct{}

// Name returns the provider name
func (f *GrokFactory) Name() string {
	return "grok"
}

// Create returns a new Grok provider
func (f *GrokFactory) Create(cfg config.Config) (Provider, error) {
	// Check for API key
	if cfg.APIKey == "" {
		return nil, aierrors.New("grok", "create", aierrors.ErrInvalidConfig)
	}

	// Determine model to use
	model := cfg.Model
	if model == "" {
		model = defaultGrokModel
	}

	// Validate model
	if _, exists := SupportedGrokModels[model]; !exists {
		model = defaultGrokModel
	}

	// Determine base URL
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultGrokURL
	}

	// Create provider
	return &GrokProvider{
		apiKey:   cfg.APIKey,
		model:    model,
		baseURL:  baseURL,
		registry: function.NewRegistry(),
	}, nil
}

// GetAvailableModels returns supported Grok models
func (f *GrokFactory) GetAvailableModels() []string {
	models := make([]string, 0, len(SupportedGrokModels))
	for model, info := range SupportedGrokModels {
		if info.Supported {
			models = append(models, model)
		}
	}
	return models
}

// GetCapabilities returns Grok capabilities
func (f *GrokFactory) GetCapabilities() []string {
	return []string{
		"function_calling",
		"streaming",
	}
}

// Name returns the provider name
func (p *GrokProvider) Name() string {
	return "grok"
}

// Model returns the current model
func (p *GrokProvider) Model() string {
	return p.model
}

// Capabilities returns supported capabilities for the current model
func (p *GrokProvider) Capabilities() []string {
	capabilities := []string{"streaming"}

	// Add model-specific capabilities
	if info, exists := SupportedGrokModels[p.model]; exists {
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
func (p *GrokProvider) supportsFunctionCalling() bool {
	info, exists := SupportedGrokModels[p.model]
	return exists && info.FunctionCalling
}

// GenerateResponse sends a request to Grok and returns the response
func (p *GrokProvider) GenerateResponse(ctx context.Context, request Request) (Response, error) {
	// Check for function calling support if requested
	if request.FunctionRegistry != nil && !p.supportsFunctionCalling() {
		return Response{}, aierrors.New("grok", "generate_response",
			fmt.Errorf("model %s does not support function calling", p.model))
	}

	// Prepare Grok request structure
	maxTokens := defaultGrokMaxTokens
	if request.MaxTokens > 0 {
		maxTokens = request.MaxTokens
	}

	temperature := 0.7
	if request.Temperature > 0 {
		temperature = request.Temperature
	}

	// Build messages array
	messages := []map[string]interface{}{
		{"role": "user", "content": request.Prompt},
	}

	grokReq := map[string]interface{}{
		"model":       p.model,
		"messages":    messages,
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
			grokReq["tools"] = tools
		}
	}

	// Configure HTTP request
	details := httputil.RequestDetails{
		URL:    p.baseURL,
		APIKey: p.apiKey,
		AdditionalHeaders: map[string]string{
			"Authorization": "Bearer " + p.apiKey,
		},
		RequestBody: grokReq,
	}

	options := httputil.ClientOptions{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	// Send the request
	responseBody, err := httputil.SendRequest(ctx, details, options)
	if err != nil {
		return Response{}, aierrors.New("grok", "generate_response", err)
	}

	// Parse the response
	var grokResp struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls,omitempty"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	err = json.Unmarshal(responseBody, &grokResp)
	if err != nil {
		return Response{}, aierrors.New("grok", "parse_response", err)
	}

	// Check for valid response
	if len(grokResp.Choices) == 0 {
		return Response{}, aierrors.New("grok", "empty_response",
			errors.New("no choices returned from API"))
	}

	// Build the provider-agnostic response
	response := Response{
		Model:    p.model,
		Provider: "grok",
		Usage: &UsageInfo{
			PromptTokens:     grokResp.Usage.PromptTokens,
			CompletionTokens: grokResp.Usage.CompletionTokens,
			TotalTokens:      grokResp.Usage.TotalTokens,
		},
	}

	// Extract content or handle tool calls
	choice := grokResp.Choices[0]
	var textContent strings.Builder

	// If there's no tool calls, just return the text
	if len(choice.Message.ToolCalls) == 0 {
		response.Content = strings.TrimSpace(choice.Message.Content)
		return response, nil
	}

	// Process tool calls
	messages = append(messages, map[string]interface{}{
		"role":       "assistant",
		"content":    choice.Message.Content,
		"tool_calls": choice.Message.ToolCalls,
	})

	// Process each tool call
	for _, toolCall := range choice.Message.ToolCalls {
		if toolCall.Type == "function" {
			// Save the function call (we only handle the first one for compatibility)
			if response.FunctionCall == nil {
				response.FunctionCall = &function.FunctionCall{
					Name:       toolCall.Function.Name,
					Parameters: toolCall.Function.Arguments,
				}
			}

			// If we have a function executor, execute it
			if request.FunctionExecutor != nil {
				fnCall := function.FunctionCall{
					Name:       toolCall.Function.Name,
					Parameters: toolCall.Function.Arguments,
				}

				fnResponse, err := request.FunctionExecutor(fnCall)
				if err != nil {
					textContent.WriteString(fmt.Sprintf("Error executing function %s: %s\n",
						fnCall.Name, err.Error()))
					continue
				}

				// Convert function response to JSON
				fnResponseJSON, err := json.Marshal(fnResponse.Content)
				if err != nil {
					textContent.WriteString(fmt.Sprintf("Error serializing function response: %s\n", err.Error()))
					continue
				}

				// Add the function response to the conversation
				messages = append(messages, map[string]interface{}{
					"role":         "tool",
					"tool_call_id": toolCall.ID,
					"content":      string(fnResponseJSON),
				})
			}
		}
	}

	// If we have function responses, continue the conversation
	if len(messages) > 1 && request.FunctionExecutor != nil {
		// Continue the conversation with the function response
		continuationReq := map[string]interface{}{
			"model":       p.model,
			"messages":    messages,
			"max_tokens":  maxTokens,
			"temperature": temperature,
		}

		// Re-add tools for continuation
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
				continuationReq["tools"] = tools
			}
		}

		details.RequestBody = continuationReq
		continuationResponseBody, err := httputil.SendRequest(ctx, details, options)
		if err != nil {
			return response, aierrors.New("grok", "continuation_request", err)
		}

		// Parse continuation response
		var continuationResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}

		err = json.Unmarshal(continuationResponseBody, &continuationResp)
		if err != nil {
			return response, aierrors.New("grok", "parse_continuation", err)
		}

		// Append continuation response text
		if len(continuationResp.Choices) > 0 {
			textContent.WriteString(continuationResp.Choices[0].Message.Content)
		}

		// Update usage metrics
		response.Usage.PromptTokens += continuationResp.Usage.PromptTokens
		response.Usage.CompletionTokens += continuationResp.Usage.CompletionTokens
		response.Usage.TotalTokens += continuationResp.Usage.TotalTokens
	} else {
		// If no function executor, just return the original content
		textContent.WriteString(choice.Message.Content)
	}

	response.Content = strings.TrimSpace(textContent.String())
	return response, nil
}

// GenerateStreamingResponse streams a response from Grok
func (p *GrokProvider) GenerateStreamingResponse(ctx context.Context, request Request, handler StreamHandler) error {
	// Check for function calling support if requested
	if request.FunctionRegistry != nil && !p.supportsFunctionCalling() {
		return aierrors.New("grok", "generate_streaming_response",
			fmt.Errorf("model %s does not support function calling", p.model))
	}

	// Grok supports streaming but has limitations with function calls
	// For simplicity, we'll use non-streaming for function calls
	if request.FunctionRegistry != nil && request.FunctionExecutor != nil {
		// Fall back to non-streaming for function calls
		resp, err := p.GenerateResponse(ctx, request)
		if err != nil {
			return err
		}

		// Simulate streaming
		return SimulateStreaming(ctx, resp.Content, handler)
	}

	// Prepare regular streaming request
	maxTokens := defaultGrokMaxTokens
	if request.MaxTokens > 0 {
		maxTokens = request.MaxTokens
	}

	temperature := 0.7
	if request.Temperature > 0 {
		temperature = request.Temperature
	}

	// Build messages array
	messages := []map[string]string{
		{"role": "user", "content": request.Prompt},
	}

	grokReq := map[string]interface{}{
		"model":       p.model,
		"messages":    messages,
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"stream":      true,
	}

	// Configure HTTP request
	details := httputil.RequestDetails{
		URL:    p.baseURL,
		APIKey: p.apiKey,
		AdditionalHeaders: map[string]string{
			"Authorization": "Bearer " + p.apiKey,
		},
		RequestBody: grokReq,
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
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		err := json.Unmarshal([]byte(data), &streamResp)
		if err != nil {
			// Skip non-JSON data like "[DONE]"
			return nil
		}

		if len(streamResp.Choices) == 0 {
			return nil
		}

		choice := streamResp.Choices[0]

		// For regular content, send to handler
		if choice.Delta.Content != "" {
			return handler(ResponseChunk{
				Content: choice.Delta.Content,
				IsFinal: false,
			})
		}

		// Check for finish reason
		if choice.FinishReason != "" {
			return handler(ResponseChunk{
				IsFinal: true,
			})
		}

		return nil
	}

	err := httputil.SendTextStreamingRequest(ctx, details, options, textStreamHandler)
	if err != nil {
		return aierrors.New("grok", "generate_streaming_response", err)
	}

	return nil
}

// Register the Grok factory
func init() {
	RegisterFactory(&GrokFactory{})
}
