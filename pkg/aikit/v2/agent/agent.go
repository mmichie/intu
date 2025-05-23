// Package agent provides a high-level abstraction for AI interactions
package agent

import (
	"context"
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
	aierrors "github.com/mmichie/intu/pkg/aikit/v2/errors"
	"github.com/mmichie/intu/pkg/aikit/v2/function"
	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// Agent represents a high-level AI agent that can process prompts
type Agent struct {
	provider         provider.Provider
	functionRegistry *function.Registry
	functionExecutor function.FunctionExecutor
	defaultConfig    Config
}

// Config contains agent-specific configuration
type Config struct {
	// Temperature controls randomness (0.0-1.0)
	Temperature float64

	// MaxTokens limits the response length
	MaxTokens int

	// SystemPrompt sets a default system prompt
	SystemPrompt string

	// Additional provider-specific parameters
	Parameters map[string]interface{}
}

// New creates a new AI agent with the specified provider
func New(p provider.Provider, opts ...Option) *Agent {
	agent := &Agent{
		provider:         p,
		functionRegistry: function.NewRegistry(),
		defaultConfig: Config{
			Temperature: 0.7,
			MaxTokens:   2048,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(agent)
	}

	return agent
}

// Option is a functional option for configuring an Agent
type Option func(*Agent)

// WithFunctionRegistry sets the function registry for the agent
func WithFunctionRegistry(registry *function.Registry) Option {
	return func(a *Agent) {
		a.functionRegistry = registry
	}
}

// WithFunctionExecutor sets the function executor for the agent
func WithFunctionExecutor(executor function.FunctionExecutor) Option {
	return func(a *Agent) {
		a.functionExecutor = executor
	}
}

// WithConfig sets the default configuration for the agent
func WithConfig(cfg Config) Option {
	return func(a *Agent) {
		a.defaultConfig = cfg
	}
}

// WithSystemPrompt sets a system prompt for the agent
func WithSystemPrompt(prompt string) Option {
	return func(a *Agent) {
		a.defaultConfig.SystemPrompt = prompt
	}
}

// Process sends a prompt to the AI and returns the response
func (a *Agent) Process(ctx context.Context, prompt string, opts ...RequestOption) (string, error) {
	req := a.buildRequest(prompt, opts...)

	resp, err := a.provider.GenerateResponse(ctx, req)
	if err != nil {
		return "", aierrors.New("agent", "process", err)
	}

	return resp.Content, nil
}

// ProcessWithInput processes an input with an optional instruction prompt
func (a *Agent) ProcessWithInput(ctx context.Context, input, instruction string, opts ...RequestOption) (string, error) {
	prompt := a.formatPromptWithInput(input, instruction)
	return a.Process(ctx, prompt, opts...)
}

// ProcessStreaming sends a prompt and streams the response
func (a *Agent) ProcessStreaming(ctx context.Context, prompt string, handler StreamHandler, opts ...RequestOption) error {
	req := a.buildRequest(prompt, opts...)

	// Convert the handler to provider.StreamHandler
	providerHandler := func(chunk provider.ResponseChunk) error {
		return handler(StreamChunk{
			Content: chunk.Content,
			IsFinal: chunk.IsFinal,
			Error:   chunk.Error,
		})
	}

	err := a.provider.GenerateStreamingResponse(ctx, req, providerHandler)
	if err != nil {
		return aierrors.New("agent", "process_streaming", err)
	}

	return nil
}

// ProcessStreamingWithInput processes input with streaming response
func (a *Agent) ProcessStreamingWithInput(ctx context.Context, input, instruction string, handler StreamHandler, opts ...RequestOption) error {
	prompt := a.formatPromptWithInput(input, instruction)
	return a.ProcessStreaming(ctx, prompt, handler, opts...)
}

// Chat sends a message in a conversational context
func (a *Agent) Chat(ctx context.Context, message string, opts ...RequestOption) (string, error) {
	// For now, this is the same as Process, but could be extended
	// to maintain conversation history
	return a.Process(ctx, message, opts...)
}

// CompleteTask processes a task-oriented prompt
func (a *Agent) CompleteTask(ctx context.Context, task, context string, opts ...RequestOption) (string, error) {
	prompt := fmt.Sprintf("Task: %s\n\nContext: %s", task, context)
	return a.Process(ctx, prompt, opts...)
}

// Provider returns the underlying provider
func (a *Agent) Provider() provider.Provider {
	return a.provider
}

// RegisterFunction registers a function that can be called by the AI
func (a *Agent) RegisterFunction(def function.FunctionDefinition) error {
	return a.functionRegistry.Register(def)
}

// SetFunctionExecutor sets the function executor
func (a *Agent) SetFunctionExecutor(executor function.FunctionExecutor) {
	a.functionExecutor = executor
}

// buildRequest builds a provider request from agent configuration
func (a *Agent) buildRequest(prompt string, opts ...RequestOption) provider.Request {
	// Start with defaults
	req := provider.Request{
		Prompt:      a.formatPrompt(prompt),
		Temperature: a.defaultConfig.Temperature,
		MaxTokens:   a.defaultConfig.MaxTokens,
		Parameters:  a.defaultConfig.Parameters,
	}

	// Add function support if available
	if a.functionRegistry != nil && a.functionRegistry.Count() > 0 {
		req.FunctionRegistry = a.functionRegistry
		req.FunctionExecutor = a.functionExecutor
	}

	// Apply request options
	for _, opt := range opts {
		opt(&req)
	}

	return req
}

// formatPrompt formats the prompt with system prompt if configured
func (a *Agent) formatPrompt(prompt string) string {
	if a.defaultConfig.SystemPrompt == "" {
		return prompt
	}
	return fmt.Sprintf("%s\n\n%s", a.defaultConfig.SystemPrompt, prompt)
}

// formatPromptWithInput formats a prompt with input and instruction
func (a *Agent) formatPromptWithInput(input, instruction string) string {
	if instruction != "" && input != "" {
		return fmt.Sprintf("%s\n\nInput: %s", instruction, input)
	} else if instruction != "" {
		return instruction
	}
	return input
}

// RequestOption modifies a request
type RequestOption func(*provider.Request)

// WithTemperature sets the temperature for a request
func WithTemperature(temp float64) RequestOption {
	return func(r *provider.Request) {
		r.Temperature = temp
	}
}

// WithMaxTokens sets the max tokens for a request
func WithMaxTokens(tokens int) RequestOption {
	return func(r *provider.Request) {
		r.MaxTokens = tokens
	}
}

// WithParameter sets a provider-specific parameter
func WithParameter(key string, value interface{}) RequestOption {
	return func(r *provider.Request) {
		if r.Parameters == nil {
			r.Parameters = make(map[string]interface{})
		}
		r.Parameters[key] = value
	}
}

// StreamHandler processes streaming chunks
type StreamHandler func(chunk StreamChunk) error

// StreamChunk represents a piece of a streaming response
type StreamChunk struct {
	Content string
	IsFinal bool
	Error   error
}

// ProviderInfo returns information about the agent's provider
func (a *Agent) ProviderInfo() AgentInfo {
	return AgentInfo{
		ProviderName: a.provider.Name(),
		Model:        a.provider.Model(),
		Capabilities: a.provider.Capabilities(),
		HasFunctions: a.functionRegistry != nil && a.functionRegistry.Count() > 0,
	}
}

// AgentInfo contains information about an agent
type AgentInfo struct {
	ProviderName string   `json:"provider_name"`
	Model        string   `json:"model"`
	Capabilities []string `json:"capabilities"`
	HasFunctions bool     `json:"has_functions"`
}

// NewFromConfig creates an agent from a configuration
func NewFromConfig(providerName string, cfg config.Config, opts ...Option) (*Agent, error) {
	// Create the provider
	p, err := provider.Create(providerName, cfg)
	if err != nil {
		return nil, aierrors.New("agent", "new_from_config", err)
	}

	return New(p, opts...), nil
}

// NewDefault creates an agent with the default provider
func NewDefault(cfg config.Config, opts ...Option) (*Agent, error) {
	// Create the default provider
	p, err := provider.CreateDefault(cfg)
	if err != nil {
		return nil, aierrors.New("agent", "new_default", err)
	}

	return New(p, opts...), nil
}
