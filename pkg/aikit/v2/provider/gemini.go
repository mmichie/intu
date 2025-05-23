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
	defaultGeminiURL       = "https://generativelanguage.googleapis.com/v1beta/models"
	defaultGeminiModel     = "gemini-1.5-pro"
	defaultGeminiMaxTokens = 2048
)

// SupportedGeminiModels defines capabilities for Gemini models
var SupportedGeminiModels = map[string]struct {
	Supported        bool
	FunctionCalling  bool
	VisionCapable    bool
	MaxContextTokens int
}{
	"gemini-1.5-pro":        {true, true, true, 1000000},
	"gemini-1.5-flash":      {true, true, true, 1000000},
	"gemini-1.0-pro":        {true, false, true, 32760},
	"gemini-1.0-pro-vision": {true, false, true, 32760},
}

// GeminiProvider implements the Provider interface for Google's Gemini
type GeminiProvider struct {
	apiKey  string
	model   string
	baseURL string

	registry *function.Registry
	executor function.FunctionExecutor
}

// GeminiFactory creates Gemini providers
type GeminiFactory struct{}

// Name returns the provider name
func (f *GeminiFactory) Name() string {
	return "gemini"
}

// Create returns a new Gemini provider
func (f *GeminiFactory) Create(cfg config.Config) (Provider, error) {
	// Check for API key
	if cfg.APIKey == "" {
		return nil, aierrors.New("gemini", "create", aierrors.ErrInvalidConfig)
	}

	// Determine model to use
	model := cfg.Model
	if model == "" {
		model = defaultGeminiModel
	}

	// Validate model
	if _, exists := SupportedGeminiModels[model]; !exists {
		model = defaultGeminiModel
	}

	// Determine base URL
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultGeminiURL
	}

	// Create provider
	return &GeminiProvider{
		apiKey:   cfg.APIKey,
		model:    model,
		baseURL:  baseURL,
		registry: function.NewRegistry(),
	}, nil
}

// GetAvailableModels returns supported Gemini models
func (f *GeminiFactory) GetAvailableModels() []string {
	models := make([]string, 0, len(SupportedGeminiModels))
	for model, info := range SupportedGeminiModels {
		if info.Supported {
			models = append(models, model)
		}
	}
	return models
}

// GetCapabilities returns Gemini capabilities
func (f *GeminiFactory) GetCapabilities() []string {
	return []string{
		"function_calling",
		"streaming",
		"vision",
		"multimodal",
	}
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// Model returns the current model
func (p *GeminiProvider) Model() string {
	return p.model
}

// Capabilities returns supported capabilities for the current model
func (p *GeminiProvider) Capabilities() []string {
	capabilities := []string{"streaming"}

	// Add model-specific capabilities
	if info, exists := SupportedGeminiModels[p.model]; exists {
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
func (p *GeminiProvider) supportsFunctionCalling() bool {
	info, exists := SupportedGeminiModels[p.model]
	return exists && info.FunctionCalling
}

// GenerateResponse sends a request to Gemini and returns the response
func (p *GeminiProvider) GenerateResponse(ctx context.Context, request Request) (Response, error) {
	// Check for function calling support if requested
	if request.FunctionRegistry != nil && !p.supportsFunctionCalling() {
		return Response{}, aierrors.New("gemini", "generate_response",
			fmt.Errorf("model %s does not support function calling", p.model))
	}

	// Construct the full URL with model name and API key
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)

	// Prepare Gemini request structure
	maxTokens := defaultGeminiMaxTokens
	if request.MaxTokens > 0 {
		maxTokens = request.MaxTokens
	}

	temperature := 0.7
	if request.Temperature > 0 {
		temperature = request.Temperature
	}

	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": request.Prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": maxTokens,
			"temperature":     temperature,
		},
	}

	// Add tools if function registry is provided
	if request.FunctionRegistry != nil {
		functions := request.FunctionRegistry.List()
		if len(functions) > 0 {
			var tools []map[string]interface{}
			for _, fn := range functions {
				tools = append(tools, map[string]interface{}{
					"functionDeclarations": []map[string]interface{}{
						{
							"name":        fn.Name,
							"description": fn.Description,
							"parameters":  fn.Parameters,
						},
					},
				})
			}
			geminiReq["tools"] = tools
		}
	}

	// Configure HTTP request
	details := httputil.RequestDetails{
		URL:         url,
		APIKey:      p.apiKey,
		RequestBody: geminiReq,
	}

	options := httputil.ClientOptions{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	// Send the request
	responseBody, err := httputil.SendRequest(ctx, details, options)
	if err != nil {
		return Response{}, aierrors.New("gemini", "generate_response", err)
	}

	// Parse the response
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text,omitempty"`
					FunctionCall *struct {
						Name string          `json:"name"`
						Args json.RawMessage `json:"args"`
					} `json:"functionCall,omitempty"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	err = json.Unmarshal(responseBody, &geminiResp)
	if err != nil {
		return Response{}, aierrors.New("gemini", "parse_response", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return Response{}, aierrors.New("gemini", "empty_response",
			errors.New("no candidates returned from API"))
	}

	// Build the provider-agnostic response
	response := Response{
		Model:    p.model,
		Provider: "gemini",
		Usage: &UsageInfo{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		},
	}

	// Extract content and function calls
	var textContent strings.Builder
	var messages []map[string]interface{}

	// Add the initial user message for potential continuation
	messages = append(messages, map[string]interface{}{
		"role": "user",
		"parts": []map[string]interface{}{
			{"text": request.Prompt},
		},
	})

	// Process candidate content parts
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		if part.Text != "" {
			textContent.WriteString(part.Text)
		} else if part.FunctionCall != nil {
			// Save the function call
			response.FunctionCall = &function.FunctionCall{
				Name:       part.FunctionCall.Name,
				Parameters: part.FunctionCall.Args,
			}

			// If we have a function executor, execute it and continue conversation
			if request.FunctionExecutor != nil {
				fnResponse, err := request.FunctionExecutor(*response.FunctionCall)
				if err != nil {
					textContent.WriteString(fmt.Sprintf("Error executing function %s: %s\n",
						response.FunctionCall.Name, err.Error()))
					continue
				}

				// Convert function response to JSON
				fnResponseJSON, err := json.Marshal(fnResponse.Content)
				if err != nil {
					textContent.WriteString(fmt.Sprintf("Error serializing function response: %s\n", err.Error()))
					continue
				}

				// Add the function call and response to messages for continuation
				messages = append(messages, map[string]interface{}{
					"role": "model",
					"parts": []map[string]interface{}{
						{
							"functionCall": map[string]interface{}{
								"name": response.FunctionCall.Name,
								"args": response.FunctionCall.Parameters,
							},
						},
					},
				})

				messages = append(messages, map[string]interface{}{
					"role": "function",
					"parts": []map[string]interface{}{
						{
							"functionResponse": map[string]interface{}{
								"name":     response.FunctionCall.Name,
								"response": json.RawMessage(fnResponseJSON),
							},
						},
					},
				})

				// Continue the conversation with the function response
				continuationReq := map[string]interface{}{
					"contents": messages,
					"generationConfig": map[string]interface{}{
						"maxOutputTokens": maxTokens,
						"temperature":     temperature,
					},
				}

				// Add tools for continuation
				if request.FunctionRegistry != nil {
					functions := request.FunctionRegistry.List()
					if len(functions) > 0 {
						var tools []map[string]interface{}
						for _, fn := range functions {
							tools = append(tools, map[string]interface{}{
								"functionDeclarations": []map[string]interface{}{
									{
										"name":        fn.Name,
										"description": fn.Description,
										"parameters":  fn.Parameters,
									},
								},
							})
						}
						continuationReq["tools"] = tools
					}
				}

				details.RequestBody = continuationReq
				continuationResponseBody, err := httputil.SendRequest(ctx, details, options)
				if err != nil {
					return response, aierrors.New("gemini", "continuation_request", err)
				}

				// Parse continuation response
				var continuationResp struct {
					Candidates []struct {
						Content struct {
							Parts []struct {
								Text string `json:"text,omitempty"`
							} `json:"parts"`
						} `json:"content"`
					} `json:"candidates"`
					UsageMetadata struct {
						PromptTokenCount     int `json:"promptTokenCount"`
						CandidatesTokenCount int `json:"candidatesTokenCount"`
						TotalTokenCount      int `json:"totalTokenCount"`
					} `json:"usageMetadata"`
				}

				err = json.Unmarshal(continuationResponseBody, &continuationResp)
				if err != nil {
					return response, aierrors.New("gemini", "parse_continuation", err)
				}

				// Append continuation response text
				if len(continuationResp.Candidates) > 0 {
					for _, part := range continuationResp.Candidates[0].Content.Parts {
						if part.Text != "" {
							textContent.WriteString(part.Text)
						}
					}
				}

				// Update usage metrics
				response.Usage.PromptTokens += continuationResp.UsageMetadata.PromptTokenCount
				response.Usage.CompletionTokens += continuationResp.UsageMetadata.CandidatesTokenCount
				response.Usage.TotalTokens += continuationResp.UsageMetadata.TotalTokenCount
			}
		}
	}

	response.Content = strings.TrimSpace(textContent.String())
	return response, nil
}

// GenerateStreamingResponse streams a response from Gemini
func (p *GeminiProvider) GenerateStreamingResponse(ctx context.Context, request Request, handler StreamHandler) error {
	// Check for function calling support if requested
	if request.FunctionRegistry != nil && !p.supportsFunctionCalling() {
		return aierrors.New("gemini", "generate_streaming_response",
			fmt.Errorf("model %s does not support function calling", p.model))
	}

	// Gemini doesn't properly support streaming with function calls
	if request.FunctionRegistry != nil && request.FunctionExecutor != nil {
		// Fall back to non-streaming for function calls
		resp, err := p.GenerateResponse(ctx, request)
		if err != nil {
			return err
		}

		// Simulate streaming
		return SimulateStreaming(ctx, resp.Content, handler)
	}

	// Construct the full URL with model name and API key for streaming
	url := fmt.Sprintf("%s/%s:streamGenerateContent?key=%s", p.baseURL, p.model, p.apiKey)

	// Prepare regular streaming request
	maxTokens := defaultGeminiMaxTokens
	if request.MaxTokens > 0 {
		maxTokens = request.MaxTokens
	}

	temperature := 0.7
	if request.Temperature > 0 {
		temperature = request.Temperature
	}

	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": request.Prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": maxTokens,
			"temperature":     temperature,
		},
	}

	// Configure HTTP request
	details := httputil.RequestDetails{
		URL:         url,
		APIKey:      p.apiKey,
		RequestBody: geminiReq,
		Stream:      true,
	}

	options := httputil.ClientOptions{
		Timeout:       90 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	// Process the streaming response
	textStreamHandler := func(data string) error {
		// Skip empty chunks
		if data == "" {
			return nil
		}

		// Remove the "data: " prefix if present
		if strings.HasPrefix(data, "data: ") {
			data = strings.TrimPrefix(data, "data: ")
		}

		// Parse the JSON chunk
		var streamResp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		err := json.Unmarshal([]byte(data), &streamResp)
		if err != nil {
			// Skip non-JSON data
			return nil
		}

		// Extract text from candidates
		if len(streamResp.Candidates) > 0 && len(streamResp.Candidates[0].Content.Parts) > 0 {
			text := streamResp.Candidates[0].Content.Parts[0].Text
			if text != "" {
				return handler(ResponseChunk{
					Content: text,
					IsFinal: false,
				})
			}
		}

		return nil
	}

	err := httputil.SendTextStreamingRequest(ctx, details, options, textStreamHandler)
	if err != nil {
		return aierrors.New("gemini", "generate_streaming_response", err)
	}

	// Send final chunk
	return handler(ResponseChunk{
		Content: "",
		IsFinal: true,
	})
}

// Register the Gemini factory
func init() {
	RegisterFactory(&GeminiFactory{})
}
