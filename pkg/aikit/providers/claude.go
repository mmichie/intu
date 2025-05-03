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

// SupportedClaudeModels is a list of supported Claude models with feature capabilities
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

type ClaudeAIProvider struct {
	BaseProvider
	registeredFunctions map[string]FunctionDefinition
}

func NewClaudeAIProvider() (*ClaudeAIProvider, error) {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("CLAUDE_API_KEY environment variable is not set")
	}
	provider := &ClaudeAIProvider{
		BaseProvider: BaseProvider{
			APIKey: apiKey,
			URL:    "https://api.anthropic.com/v1/messages",
		},
		registeredFunctions: make(map[string]FunctionDefinition),
	}

	// Get model from environment or use default, and validate it
	modelFromEnv := provider.GetEnvOrDefault("CLAUDE_MODEL", "claude-3-5-sonnet-20240620")
	provider.SetModel(modelFromEnv)
	return provider, nil
}

// SetModel sets the model and validates it
func (p *ClaudeAIProvider) SetModel(model string) bool {
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
func (p *ClaudeAIProvider) SupportsFunctionCalling() bool {
	modelInfo, exists := SupportedClaudeModels[p.Model]
	return exists && modelInfo.FunctionCalling
}

// RegisterFunction adds a function to the available functions
// RegisterFunctions registers multiple functions with Claude
func (p *ClaudeAIProvider) RegisterFunctions(functions []FunctionDefinition) {
	for _, fn := range functions {
		_ = p.RegisterFunction(fn) // Ignore errors for now
	}
}

func (p *ClaudeAIProvider) RegisterFunction(def FunctionDefinition) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("invalid function definition: %w", err)
	}

	p.registeredFunctions[def.Name] = def
	return nil
}

func (p *ClaudeAIProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
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
			Text string `json:"text"`
		} `json:"content"`
	}
	err = json.Unmarshal(responseBody, &claudeAIResp)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshaling Claude AI response")
	}
	if len(claudeAIResp.Content) == 0 {
		return "", fmt.Errorf("no content in Claude AI response")
	}
	return strings.TrimSpace(claudeAIResp.Content[0].Text), nil
}

func (p *ClaudeAIProvider) Name() string {
	return "claude"
}

// SupportsStreaming returns whether the provider supports streaming responses
func (p *ClaudeAIProvider) SupportsStreaming() bool {
	return true
}

// GenerateStreamingResponse generates a streaming response from Claude
func (p *ClaudeAIProvider) GenerateStreamingResponse(ctx context.Context, prompt string, handler StreamHandler) error {
	requestBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  4096,
		"temperature": 0.7,
		"stream":      true,
	}

	details := httputil.RequestDetails{
		URL:         p.URL,
		APIKey:      p.APIKey,
		RequestBody: requestBody,
		AdditionalHeaders: map[string]string{
			"x-api-key":         p.APIKey,
			"anthropic-version": "2023-06-01",
		},
		Stream: true,
	}

	options := httputil.ClientOptions{
		Timeout:       90 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	// Process the streaming response
	streamHandler := func(chunk []byte) error {
		// Skip empty chunks and "[DONE]" messages
		if len(chunk) == 0 || string(chunk) == "[DONE]" {
			return nil
		}

		// Remove the "data: " prefix if present
		data := string(chunk)
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
			return errors.Wrap(err, "error parsing stream chunk")
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
			return handler(text)
		}

		return nil
	}

	err := httputil.SendStreamingRequest(ctx, details, options, streamHandler)
	if err != nil {
		return errors.Wrap(err, "error in streaming request")
	}

	return nil
}

// GenerateStreamingResponseWithFunctions streams a response with function calling support
func (p *ClaudeAIProvider) GenerateStreamingResponseWithFunctions(
	ctx context.Context,
	prompt string,
	functionExecutor FunctionExecutorFunc,
	handler StreamHandler,
) error {
	// Claude doesn't support streaming with function calls directly,
	// so we'll simulate it by collecting function calls and then streaming the final response
	response, err := p.GenerateResponseWithFunctions(ctx, prompt, functionExecutor)
	if err != nil {
		return err
	}

	// Simulate streaming by sending small chunks
	chunks := splitIntoChunks(response, 15)
	for _, chunk := range chunks {
		if err := handler(chunk); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// splitIntoChunks splits a string into chunks of approximately the given size,
// but tries to split at word boundaries
func splitIntoChunks(text string, chunkSize int) []string {
	var chunks []string
	runes := []rune(text)

	for i := 0; i < len(runes); {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		} else {
			// Try to find a word boundary
			for j := end - 1; j > i; j-- {
				if j < len(runes) && (runes[j] == ' ' || runes[j] == '\n') {
					end = j + 1
					break
				}
			}
		}

		chunks = append(chunks, string(runes[i:end]))
		i = end
	}

	return chunks
}

// GetSupportedModels returns a list of supported models for this provider
func (p *ClaudeAIProvider) GetSupportedModels() []string {
	models := make([]string, 0, len(SupportedClaudeModels))
	for model, info := range SupportedClaudeModels {
		if info.Supported {
			models = append(models, model)
		}
	}
	return models
}

// GenerateResponseWithFunctions sends a prompt to Claude with function calling
func (p *ClaudeAIProvider) GenerateResponseWithFunctions(
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
			fnCall := FunctionCall{
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
