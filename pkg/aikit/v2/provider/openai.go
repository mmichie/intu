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
	defaultOpenAIURL       = "https://api.openai.com/v1/chat/completions"
	defaultOpenAIModel     = "gpt-4o"
	defaultOpenAIMaxTokens = 4096
)

// SupportedOpenAIModels defines capabilities for OpenAI models
var SupportedOpenAIModels = map[string]struct {
	Supported        bool
	FunctionCalling  bool
	VisionCapable    bool
	MaxContextTokens int
}{
	"gpt-4o":            {true, true, true, 128000},
	"gpt-4o-mini":       {true, true, true, 128000},
	"gpt-4-turbo":       {true, true, true, 128000},
	"gpt-4":             {true, true, false, 8192},
	"gpt-3.5-turbo":     {true, true, false, 16385},
	"gpt-3.5-turbo-16k": {true, true, false, 16385},
}

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	apiKey  string
	model   string
	baseURL string

	registry *function.Registry
	executor function.FunctionExecutor
}

// OpenAIFactory creates OpenAI providers
type OpenAIFactory struct{}

// Name returns the provider name
func (f *OpenAIFactory) Name() string {
	return "openai"
}

// Create returns a new OpenAI provider
func (f *OpenAIFactory) Create(cfg config.Config) (Provider, error) {
	// Check for API key
	if cfg.APIKey == "" {
		return nil, aierrors.New("openai", "create", aierrors.ErrInvalidConfig)
	}

	// Determine model to use
	model := cfg.Model
	if model == "" {
		model = defaultOpenAIModel
	}

	// Validate model
	if _, exists := SupportedOpenAIModels[model]; !exists {
		model = defaultOpenAIModel
	}

	// Determine base URL
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenAIURL
	}

	// Create provider
	return &OpenAIProvider{
		apiKey:   cfg.APIKey,
		model:    model,
		baseURL:  baseURL,
		registry: function.NewRegistry(),
	}, nil
}

// GetAvailableModels returns supported OpenAI models
func (f *OpenAIFactory) GetAvailableModels() []string {
	models := make([]string, 0, len(SupportedOpenAIModels))
	for model, info := range SupportedOpenAIModels {
		if info.Supported {
			models = append(models, model)
		}
	}
	return models
}

// GetCapabilities returns OpenAI capabilities
func (f *OpenAIFactory) GetCapabilities() []string {
	return []string{
		"function_calling",
		"streaming",
		"vision",
		"multimodal",
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Model returns the current model
func (p *OpenAIProvider) Model() string {
	return p.model
}

// Capabilities returns supported capabilities for the current model
func (p *OpenAIProvider) Capabilities() []string {
	capabilities := []string{"streaming", "function_calling"}

	// Add model-specific capabilities
	if info, exists := SupportedOpenAIModels[p.model]; exists {
		if info.VisionCapable {
			capabilities = append(capabilities, "vision", "multimodal")
		}
	}

	return capabilities
}

// supportsFunctionCalling checks if current model supports function calling
func (p *OpenAIProvider) supportsFunctionCalling() bool {
	info, exists := SupportedOpenAIModels[p.model]
	return exists && info.FunctionCalling
}

// GenerateResponse sends a request to OpenAI and returns the response
func (p *OpenAIProvider) GenerateResponse(ctx context.Context, request Request) (Response, error) {
	// Check for function calling support if requested
	if request.FunctionRegistry != nil && !p.supportsFunctionCalling() {
		return Response{}, aierrors.New("openai", "generate_response",
			fmt.Errorf("model %s does not support function calling", p.model))
	}

	// Prepare OpenAI request structure
	maxTokens := defaultOpenAIMaxTokens
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

	openaiReq := map[string]interface{}{
		"model":       p.model,
		"messages":    messages,
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}

	// Add functions if function registry is provided
	if request.FunctionRegistry != nil {
		functions := request.FunctionRegistry.List()
		if len(functions) > 0 {
			// Convert to OpenAI function format
			funcs := make([]map[string]interface{}, len(functions))
			for i, fn := range functions {
				funcs[i] = map[string]interface{}{
					"name":        fn.Name,
					"description": fn.Description,
					"parameters":  fn.Parameters,
				}
			}
			openaiReq["functions"] = funcs
			openaiReq["function_call"] = "auto"
		}
	}

	// Configure HTTP request
	details := httputil.RequestDetails{
		URL:    p.baseURL,
		APIKey: p.apiKey,
		AdditionalHeaders: map[string]string{
			"Authorization": "Bearer " + p.apiKey,
		},
		RequestBody: openaiReq,
	}

	options := httputil.ClientOptions{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	// Send the request
	responseBody, err := httputil.SendRequest(ctx, details, options)
	if err != nil {
		return Response{}, aierrors.New("openai", "generate_response", err)
	}

	// Parse the response
	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content      string `json:"content"`
				FunctionCall *struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function_call"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	err = json.Unmarshal(responseBody, &openaiResp)
	if err != nil {
		return Response{}, aierrors.New("openai", "parse_response", err)
	}

	// Check for valid response
	if len(openaiResp.Choices) == 0 {
		return Response{}, aierrors.New("openai", "empty_response",
			errors.New("no choices returned from API"))
	}

	// Build the provider-agnostic response
	response := Response{
		Model:    p.model,
		Provider: "openai",
		Usage: &UsageInfo{
			PromptTokens:     openaiResp.Usage.PromptTokens,
			CompletionTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:      openaiResp.Usage.TotalTokens,
		},
	}

	// Extract content or function call
	choice := openaiResp.Choices[0]
	if choice.Message.FunctionCall != nil {
		// Parse the function call arguments
		functionCall := &function.FunctionCall{
			Name:       choice.Message.FunctionCall.Name,
			Parameters: []byte(choice.Message.FunctionCall.Arguments),
		}
		response.FunctionCall = functionCall

		// If we have a function executor, execute it
		if request.FunctionExecutor != nil {
			fnResponse, err := request.FunctionExecutor(*functionCall)
			if err != nil {
				response.Content = fmt.Sprintf("Error executing function %s: %s",
					functionCall.Name, err.Error())
			} else {
				// Add a textual representation of the function response
				if fnResponse.Error != "" {
					response.Content = fmt.Sprintf("Function %s returned error: %s",
						fnResponse.Name, fnResponse.Error)
				} else {
					// Add simple summary of function result
					response.Content = fmt.Sprintf("Function %s executed successfully.\n",
						fnResponse.Name)

					// Convert result to JSON string for representation
					resultJSON, _ := json.MarshalIndent(fnResponse.Content, "", "  ")
					response.Content += fmt.Sprintf("Result: %s", resultJSON)
				}
			}
		}
	} else {
		response.Content = choice.Message.Content
	}

	return response, nil
}

// GenerateStreamingResponse streams a response from OpenAI
func (p *OpenAIProvider) GenerateStreamingResponse(ctx context.Context, request Request, handler StreamHandler) error {
	// Check for function calling support if requested
	if request.FunctionRegistry != nil && !p.supportsFunctionCalling() {
		return aierrors.New("openai", "generate_streaming_response",
			fmt.Errorf("model %s does not support function calling", p.model))
	}

	// OpenAI supports streaming but has limitations with function calls
	// For simplicity, we'll use non-streaming for function calls
	if request.FunctionRegistry != nil && request.FunctionExecutor != nil {
		// Fall back to non-streaming for function calls
		resp, err := p.GenerateResponse(ctx, request)
		if err != nil {
			return err
		}

		// Simulate streaming
		return simulateStreaming(ctx, resp.Content, func(chunk ResponseChunk) error {
			return handler(chunk)
		})
	}

	// Prepare regular streaming request
	maxTokens := defaultOpenAIMaxTokens
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

	openaiReq := map[string]interface{}{
		"model":       p.model,
		"messages":    messages,
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"stream":      true,
	}

	// Add functions if function registry is provided
	if request.FunctionRegistry != nil {
		functions := request.FunctionRegistry.List()
		if len(functions) > 0 {
			// Convert to OpenAI function format
			funcs := make([]map[string]interface{}, len(functions))
			for i, fn := range functions {
				funcs[i] = map[string]interface{}{
					"name":        fn.Name,
					"description": fn.Description,
					"parameters":  fn.Parameters,
				}
			}
			openaiReq["functions"] = funcs
			openaiReq["function_call"] = "auto"
		}
	}

	// Configure HTTP request
	details := httputil.RequestDetails{
		URL:    p.baseURL,
		APIKey: p.apiKey,
		AdditionalHeaders: map[string]string{
			"Authorization": "Bearer " + p.apiKey,
		},
		RequestBody: openaiReq,
		Stream:      true,
	}

	options := httputil.ClientOptions{
		Timeout:       90 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	// Process the streaming response
	var contentBuilder strings.Builder
	var functionCallBuilder *struct {
		Name      string
		Arguments strings.Builder
	}

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
					Content      string `json:"content"`
					FunctionCall *struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function_call"`
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

		// Check for function call
		if choice.Delta.FunctionCall != nil {
			// Initialize function call builder if needed
			if functionCallBuilder == nil {
				functionCallBuilder = &struct {
					Name      string
					Arguments strings.Builder
				}{
					Name: choice.Delta.FunctionCall.Name,
				}
			}

			// If it has a name, update it
			if choice.Delta.FunctionCall.Name != "" {
				functionCallBuilder.Name = choice.Delta.FunctionCall.Name
			}

			// Append any arguments
			if choice.Delta.FunctionCall.Arguments != "" {
				functionCallBuilder.Arguments.WriteString(choice.Delta.FunctionCall.Arguments)
			}

			// Don't send anything to handler yet for function calls
			return nil
		}

		// For regular content, append to builder and send to handler
		if choice.Delta.Content != "" {
			contentBuilder.WriteString(choice.Delta.Content)
			return handler(ResponseChunk{
				Content: choice.Delta.Content,
				IsFinal: false,
			})
		}

		// Check for finish reason
		if choice.FinishReason != "" {
			if functionCallBuilder != nil {
				// We have a complete function call, create the response chunk
				functionCall := &function.FunctionCall{
					Name:       functionCallBuilder.Name,
					Parameters: []byte(functionCallBuilder.Arguments.String()),
				}

				// Send the function call
				err := handler(ResponseChunk{
					FunctionCall: functionCall,
					IsFinal:      true,
				})

				if err != nil {
					return err
				}

				// If we have a function executor, execute it and stream the result
				if request.FunctionExecutor != nil {
					fnResponse, execErr := request.FunctionExecutor(*functionCall)
					if execErr != nil {
						errMsg := fmt.Sprintf("Error executing function %s: %s",
							functionCall.Name, execErr.Error())
						return handler(ResponseChunk{
							Content: errMsg,
							IsFinal: true,
							Error:   execErr,
						})
					}

					// Format function response for text output
					var resultText string
					if fnResponse.Error != "" {
						resultText = fmt.Sprintf("Function %s returned error: %s",
							fnResponse.Name, fnResponse.Error)
					} else {
						resultText = fmt.Sprintf("Function %s executed successfully.\n",
							fnResponse.Name)
						resultJSON, _ := json.MarshalIndent(fnResponse.Content, "", "  ")
						resultText += fmt.Sprintf("Result: %s", resultJSON)
					}

					// Simulate streaming the function result
					return simulateStreaming(ctx, resultText, handler)
				}
			} else {
				// Normal content finished
				return handler(ResponseChunk{
					IsFinal: true,
				})
			}
		}

		return nil
	}

	err := httputil.SendTextStreamingRequest(ctx, details, options, textStreamHandler)
	if err != nil {
		return aierrors.New("openai", "generate_streaming_response", err)
	}

	return nil
}

// Register the OpenAI factory
func init() {
	RegisterFactory(&OpenAIFactory{})
}
