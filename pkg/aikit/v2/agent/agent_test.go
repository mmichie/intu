package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/mmichie/intu/pkg/aikit/v2/function"
	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// mockProvider is a test provider
type mockProvider struct {
	name            string
	model           string
	capabilities    []string
	generateError   error
	streamError     error
	lastRequest     provider.Request
	responseContent string
	streamingChunks []string
}

func (p *mockProvider) GenerateResponse(ctx context.Context, request provider.Request) (provider.Response, error) {
	p.lastRequest = request
	if p.generateError != nil {
		return provider.Response{}, p.generateError
	}

	return provider.Response{
		Content:  p.responseContent,
		Model:    p.model,
		Provider: p.name,
	}, nil
}

func (p *mockProvider) GenerateStreamingResponse(ctx context.Context, request provider.Request, handler provider.StreamHandler) error {
	p.lastRequest = request
	if p.streamError != nil {
		return p.streamError
	}

	// Simulate streaming chunks
	for _, chunk := range p.streamingChunks {
		err := handler(provider.ResponseChunk{
			Content: chunk,
			IsFinal: false,
		})
		if err != nil {
			return err
		}
	}

	// Send final chunk
	return handler(provider.ResponseChunk{
		Content: "",
		IsFinal: true,
	})
}

func (p *mockProvider) Name() string {
	return p.name
}

func (p *mockProvider) Model() string {
	return p.model
}

func (p *mockProvider) Capabilities() []string {
	return p.capabilities
}

func TestAgent(t *testing.T) {
	// Create a mock provider
	mockProv := &mockProvider{
		name:            "test-provider",
		model:           "test-model",
		capabilities:    []string{"test-cap"},
		responseContent: "Test response",
	}

	// Create agent
	agent := New(mockProv)

	// Test basic processing
	ctx := context.Background()
	response, err := agent.Process(ctx, "Test prompt")
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}
	if response != "Test response" {
		t.Errorf("Expected 'Test response', got '%s'", response)
	}
	if mockProv.lastRequest.Prompt != "Test prompt" {
		t.Errorf("Expected prompt 'Test prompt', got '%s'", mockProv.lastRequest.Prompt)
	}

	// Test with system prompt
	agentWithSystem := New(mockProv, WithSystemPrompt("System: You are helpful"))
	response, err = agentWithSystem.Process(ctx, "User prompt")
	if err != nil {
		t.Errorf("Process with system prompt failed: %v", err)
	}
	expectedPrompt := "System: You are helpful\n\nUser prompt"
	if mockProv.lastRequest.Prompt != expectedPrompt {
		t.Errorf("Expected prompt '%s', got '%s'", expectedPrompt, mockProv.lastRequest.Prompt)
	}

	// Test with input processing
	response, err = agent.ProcessWithInput(ctx, "Input data", "Process this")
	if err != nil {
		t.Errorf("ProcessWithInput failed: %v", err)
	}
	expectedPrompt = "Process this\n\nInput: Input data"
	if mockProv.lastRequest.Prompt != expectedPrompt {
		t.Errorf("Expected prompt '%s', got '%s'", expectedPrompt, mockProv.lastRequest.Prompt)
	}
}

func TestAgentOptions(t *testing.T) {
	mockProv := &mockProvider{
		name:            "test-provider",
		model:           "test-model",
		responseContent: "Test response",
	}

	// Test with custom config
	cfg := Config{
		Temperature: 0.5,
		MaxTokens:   1000,
		Parameters: map[string]interface{}{
			"custom": "value",
		},
	}

	agent := New(mockProv, WithConfig(cfg))

	ctx := context.Background()
	_, err := agent.Process(ctx, "Test")
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if mockProv.lastRequest.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", mockProv.lastRequest.Temperature)
	}
	if mockProv.lastRequest.MaxTokens != 1000 {
		t.Errorf("Expected max tokens 1000, got %d", mockProv.lastRequest.MaxTokens)
	}

	// Test request options override
	_, err = agent.Process(ctx, "Test", WithTemperature(0.9), WithMaxTokens(500))
	if err != nil {
		t.Errorf("Process with options failed: %v", err)
	}

	if mockProv.lastRequest.Temperature != 0.9 {
		t.Errorf("Expected temperature 0.9, got %f", mockProv.lastRequest.Temperature)
	}
	if mockProv.lastRequest.MaxTokens != 500 {
		t.Errorf("Expected max tokens 500, got %d", mockProv.lastRequest.MaxTokens)
	}
}

func TestAgentStreaming(t *testing.T) {
	mockProv := &mockProvider{
		name:            "test-provider",
		model:           "test-model",
		streamingChunks: []string{"Hello", " ", "world"},
	}

	agent := New(mockProv)

	ctx := context.Background()
	var receivedChunks []string
	var finalChunkReceived bool

	err := agent.ProcessStreaming(ctx, "Test prompt", func(chunk StreamChunk) error {
		if chunk.IsFinal {
			finalChunkReceived = true
		} else {
			receivedChunks = append(receivedChunks, chunk.Content)
		}
		return nil
	})

	if err != nil {
		t.Errorf("ProcessStreaming failed: %v", err)
	}

	if len(receivedChunks) != 3 {
		t.Errorf("Expected 3 chunks (excluding final), got %d", len(receivedChunks))
	}

	if !finalChunkReceived {
		t.Error("Final chunk not received")
	}

	// Test streaming error
	mockProv.streamError = errors.New("stream error")
	err = agent.ProcessStreaming(ctx, "Test", func(chunk StreamChunk) error {
		return nil
	})
	if err == nil {
		t.Error("Expected streaming error")
	}
}

func TestAgentFunctions(t *testing.T) {
	mockProv := &mockProvider{
		name:            "test-provider",
		model:           "test-model",
		responseContent: "Function result",
	}

	registry := function.NewRegistry()
	agent := New(mockProv, WithFunctionRegistry(registry))

	// Register a test function
	err := agent.RegisterFunction(function.FunctionDefinition{
		Name:        "test_func",
		Description: "Test function",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{"type": "string"},
			},
		},
	})

	if err != nil {
		t.Errorf("RegisterFunction failed: %v", err)
	}

	// Test function executor
	agent.SetFunctionExecutor(func(call function.FunctionCall) (function.FunctionResponse, error) {
		return function.FunctionResponse{
			Name:    call.Name,
			Content: "Function executed",
		}, nil
	})

	ctx := context.Background()
	_, err = agent.Process(ctx, "Call test function")
	if err != nil {
		t.Errorf("Process with functions failed: %v", err)
	}

	// Verify function registry was passed
	if mockProv.lastRequest.FunctionRegistry == nil {
		t.Error("Function registry not passed to provider")
	}
	if mockProv.lastRequest.FunctionExecutor == nil {
		t.Error("Function executor not passed to provider")
	}
}

func TestAgentInfo(t *testing.T) {
	mockProv := &mockProvider{
		name:         "test-provider",
		model:        "test-model",
		capabilities: []string{"cap1", "cap2"},
	}

	agent := New(mockProv)
	info := agent.ProviderInfo()

	if info.ProviderName != "test-provider" {
		t.Errorf("Expected provider name 'test-provider', got '%s'", info.ProviderName)
	}
	if info.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", info.Model)
	}
	if len(info.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(info.Capabilities))
	}
	if info.HasFunctions {
		t.Error("Expected HasFunctions to be false")
	}

	// Test with functions
	registry := function.NewRegistry()
	err := registry.Register(function.FunctionDefinition{
		Name:        "test",
		Description: "Test function",
		Parameters: map[string]interface{}{
			"type": "object",
		},
	})
	if err != nil {
		t.Errorf("Failed to register function: %v", err)
	}

	agentWithFuncs := New(mockProv, WithFunctionRegistry(registry))
	info = agentWithFuncs.ProviderInfo()

	if !info.HasFunctions {
		t.Errorf("Expected HasFunctions to be true, registry has %d functions", registry.Count())
	}
}
