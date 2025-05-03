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

// SupportedOpenAIModels is a list of supported OpenAI models with feature capabilities
var SupportedOpenAIModels = map[string]struct {
	Supported        bool
	FunctionCalling  bool
	VisionCapable    bool
	MaxContextTokens int
}{
	"gpt-4o":               {true, true, true, 128000},
	"gpt-4-turbo":          {true, true, true, 128000},
	"gpt-4-vision-preview": {true, true, true, 128000},
	"gpt-4":                {true, true, false, 8192},
	"gpt-4-32k":            {true, true, false, 32768},
	"gpt-3.5-turbo":        {true, true, false, 16385},
	"gpt-3.5-turbo-16k":    {true, true, false, 16385},
}

type OpenAIProvider struct {
	BaseProvider
	registeredFunctions map[string]FunctionDefinition
}

func NewOpenAIProvider() (*OpenAIProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}
	provider := &OpenAIProvider{
		BaseProvider: BaseProvider{
			APIKey: apiKey,
			URL:    "https://api.openai.com/v1/chat/completions",
		},
		registeredFunctions: make(map[string]FunctionDefinition),
	}

	// Get model from environment or use default, and validate it
	modelFromEnv := provider.GetEnvOrDefault("OPENAI_MODEL", "gpt-4o")
	provider.SetModel(modelFromEnv)
	return provider, nil
}

// SetModel sets the model and validates it
func (p *OpenAIProvider) SetModel(model string) bool {
	modelInfo, exists := SupportedOpenAIModels[model]
	if exists && modelInfo.Supported {
		p.Model = model
		return true
	}

	// Default to a modern model with function calling
	p.Model = "gpt-4o"
	return false
}

// SupportsFunctionCalling returns whether the current model supports function calling
func (p *OpenAIProvider) SupportsFunctionCalling() bool {
	modelInfo, exists := SupportedOpenAIModels[p.Model]
	return exists && modelInfo.FunctionCalling
}

// RegisterFunction adds a function to the available functions
// RegisterFunctions registers multiple functions with OpenAI
func (p *OpenAIProvider) RegisterFunctions(functions []FunctionDefinition) {
	for _, fn := range functions {
		_ = p.RegisterFunction(fn) // Ignore errors for now
	}
}

func (p *OpenAIProvider) RegisterFunction(def FunctionDefinition) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("invalid function definition: %w", err)
	}

	p.registeredFunctions[def.Name] = def
	return nil
}

func (p *OpenAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  2048,
		"temperature": 0.7,
	}

	details := httputil.RequestDetails{
		URL:         p.URL,
		APIKey:      p.APIKey,
		RequestBody: requestBody,
		AdditionalHeaders: map[string]string{
			"Authorization": "Bearer " + p.APIKey,
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

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err = json.Unmarshal(responseBody, &openAIResp)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling OpenAI response")
	}
	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}
	return strings.TrimSpace(openAIResp.Choices[0].Message.Content), nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

// GetSupportedModels returns a list of supported models for this provider
func (p *OpenAIProvider) GetSupportedModels() []string {
	models := make([]string, 0, len(SupportedOpenAIModels))
	for model, info := range SupportedOpenAIModels {
		if info.Supported {
			models = append(models, model)
		}
	}
	return models
}

// GenerateResponseWithFunctions sends a prompt to OpenAI with function calling
func (p *OpenAIProvider) GenerateResponseWithFunctions(
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

	// Convert registered functions to OpenAI tools format
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
		"max_tokens":  2048,
		"temperature": 0.7,
		"tools":       tools,
	}

	details := httputil.RequestDetails{
		URL:         p.URL,
		APIKey:      p.APIKey,
		RequestBody: requestBody,
		AdditionalHeaders: map[string]string{
			"Authorization": "Bearer " + p.APIKey,
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
	var openAIResponse struct {
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
		} `json:"choices"`
	}

	err = json.Unmarshal(responseBody, &openAIResponse)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling OpenAI response")
	}

	if len(openAIResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}

	// Handle function calls and continue the conversation
	var messages []map[string]interface{}
	var textResponse strings.Builder

	// Add the initial user message
	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": prompt,
	})

	// Add the assistant's response
	messages = append(messages, map[string]interface{}{
		"role":       "assistant",
		"content":    openAIResponse.Choices[0].Message.Content,
		"tool_calls": openAIResponse.Choices[0].Message.ToolCalls,
	})

	// If there's no tool calls, just return the text
	if len(openAIResponse.Choices[0].Message.ToolCalls) == 0 {
		return strings.TrimSpace(openAIResponse.Choices[0].Message.Content), nil
	}

	// Process tool calls
	for _, toolCall := range openAIResponse.Choices[0].Message.ToolCalls {
		if toolCall.Type == "function" {
			// Extract function call details
			fnCall := FunctionCall{
				Name:       toolCall.Function.Name,
				Parameters: toolCall.Function.Arguments,
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

			// Add the function response to the conversation
			messages = append(messages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": toolCall.ID,
				"content":      string(fnResponseJSON),
			})
		}
	}

	// Continue the conversation with the function response
	continuationRequestBody := map[string]interface{}{
		"model":       p.Model,
		"messages":    messages,
		"max_tokens":  2048,
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
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	err = json.Unmarshal(continuationResponseBody, &continuationResponse)
	if err != nil {
		return textResponse.String(), errors.Wrap(err, "error unmarshaling continuation response")
	}

	// Append continuation response text to our result
	if len(continuationResponse.Choices) > 0 {
		textResponse.WriteString(continuationResponse.Choices[0].Message.Content)
	}

	return strings.TrimSpace(textResponse.String()), nil
}
