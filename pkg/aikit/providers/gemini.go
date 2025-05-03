package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mmichie/intu/pkg/httputil"
	"github.com/pkg/errors"
)

// SupportedGeminiModels is a list of supported Gemini models with feature capabilities
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

type GeminiProvider struct {
	BaseProvider
	registeredFunctions map[string]FunctionDefinition
}

func NewGeminiProvider() (*GeminiProvider, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}
	provider := &GeminiProvider{
		BaseProvider: BaseProvider{
			APIKey: apiKey,
			URL:    "https://generativelanguage.googleapis.com/v1beta/models",
		},
		registeredFunctions: make(map[string]FunctionDefinition),
	}

	// Get model from environment or use default, and validate it
	modelFromEnv := provider.GetEnvOrDefault("GEMINI_MODEL", "gemini-1.5-pro")
	provider.SetModel(modelFromEnv)
	return provider, nil
}

// SetModel sets the model and validates it
func (p *GeminiProvider) SetModel(model string) bool {
	modelInfo, exists := SupportedGeminiModels[model]
	if exists && modelInfo.Supported {
		p.Model = model
		return true
	}

	// Default to a modern model with function calling
	p.Model = "gemini-1.5-pro"
	return false
}

// SupportsFunctionCalling returns whether the current model supports function calling
func (p *GeminiProvider) SupportsFunctionCalling() bool {
	modelInfo, exists := SupportedGeminiModels[p.Model]
	return exists && modelInfo.FunctionCalling
}

// RegisterFunction adds a function to the available functions
// RegisterFunctions registers multiple functions with Gemini
func (p *GeminiProvider) RegisterFunctions(functions []FunctionDefinition) {
	for _, fn := range functions {
		_ = p.RegisterFunction(fn) // Ignore errors for now
	}
}

func (p *GeminiProvider) RegisterFunction(def FunctionDefinition) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("invalid function definition: %w", err)
	}

	p.registeredFunctions[def.Name] = def
	return nil
}

func (p *GeminiProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	// Construct the full URL with model name and API key
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", p.URL, p.Model, p.APIKey)

	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 2048,
			"temperature":     0.7,
		},
	}

	details := httputil.RequestDetails{
		URL:         url,
		APIKey:      p.APIKey,
		RequestBody: requestBody,
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

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	err = json.Unmarshal(responseBody, &geminiResp)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling Gemini response")
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no text content in Gemini response")
	}

	return strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text), nil
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

// GetSupportedModels returns a list of supported models for this provider
func (p *GeminiProvider) GetSupportedModels() []string {
	models := make([]string, 0, len(SupportedGeminiModels))
	for model, info := range SupportedGeminiModels {
		if info.Supported {
			models = append(models, model)
		}
	}
	return models
}

// GenerateResponseWithFunctions sends a prompt to Gemini with function calling
func (p *GeminiProvider) GenerateResponseWithFunctions(
	ctx context.Context,
	prompt string,
	functionExecutor FunctionExecutorFunc,
) (string, error) {
	if !p.SupportsFunctionCalling() {
		return "", fmt.Errorf("model %s does not support function calling", p.Model)
	}

	if len(p.registeredFunctions) == 0 {
		return "", fmt.Errorf("no functions registered for function calling")
	}

	// Construct the full URL with model name and API key
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", p.URL, p.Model, p.APIKey)

	// Convert registered functions to Gemini tools format
	var tools []map[string]interface{}
	for _, fn := range p.registeredFunctions {
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

	// Create initial request
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 2048,
			"temperature":     0.7,
		},
		"tools": tools,
	}

	details := httputil.RequestDetails{
		URL:         url,
		APIKey:      p.APIKey,
		RequestBody: requestBody,
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
	var geminiResponse struct {
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
	}

	err = json.Unmarshal(responseBody, &geminiResponse)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling Gemini response")
	}

	if len(geminiResponse.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in Gemini response")
	}

	var messages []map[string]interface{}
	var textResponse strings.Builder

	// Add the initial user message
	messages = append(messages, map[string]interface{}{
		"role": "user",
		"parts": []map[string]interface{}{
			{"text": prompt},
		},
	})

	// Process candidate content parts
	for _, part := range geminiResponse.Candidates[0].Content.Parts {
		if part.Text != "" {
			textResponse.WriteString(part.Text)
			textResponse.WriteString("\n")
		} else if part.FunctionCall != nil {
			// Extract function call details
			fnCall := FunctionCall{
				Name:       part.FunctionCall.Name,
				Parameters: part.FunctionCall.Args,
			}

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

			// Add the function response as a new message
			messages = append(messages, map[string]interface{}{
				"role": "model",
				"parts": []map[string]interface{}{
					{
						"functionCall": map[string]interface{}{
							"name": fnCall.Name,
							"args": json.RawMessage(fnCall.Parameters),
						},
					},
				},
			})

			messages = append(messages, map[string]interface{}{
				"role": "function",
				"parts": []map[string]interface{}{
					{
						"functionResponse": map[string]interface{}{
							"name":     fnCall.Name,
							"response": json.RawMessage(fnResponseJSON),
						},
					},
				},
			})

			// Continue the conversation with the function response
			continuationRequestBody := map[string]interface{}{
				"contents": messages,
				"generationConfig": map[string]interface{}{
					"maxOutputTokens": 2048,
					"temperature":     0.7,
				},
				"tools": tools,
			}

			details.RequestBody = continuationRequestBody
			continuationResponseBody, err := httputil.SendRequest(ctx, details, options)
			if err != nil {
				return textResponse.String(), errors.Wrap(err, "error sending continuation request")
			}

			// Parse continuation response
			var continuationResponse struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text,omitempty"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
			}

			err = json.Unmarshal(continuationResponseBody, &continuationResponse)
			if err != nil {
				return textResponse.String(), errors.Wrap(err, "error unmarshaling continuation response")
			}

			// Append continuation response text to our result
			if len(continuationResponse.Candidates) > 0 {
				for _, part := range continuationResponse.Candidates[0].Content.Parts {
					if part.Text != "" {
						textResponse.WriteString(part.Text)
						textResponse.WriteString("\n")
					}
				}
			}
		}
	}

	return strings.TrimSpace(textResponse.String()), nil
}
