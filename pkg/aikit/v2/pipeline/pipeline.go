// Package pipeline provides composable AI processing chains
package pipeline

import (
	"context"
	"errors"

	aierrors "github.com/mmichie/intu/pkg/aikit/v2/errors"
	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// Pipeline represents a composable AI processing chain
type Pipeline interface {
	// Execute processes an input string through the pipeline
	Execute(ctx context.Context, input string) (string, error)

	// ExecuteWithRequest processes a full request through the pipeline
	ExecuteWithRequest(ctx context.Context, request provider.Request) (provider.Response, error)

	// WithOptions returns a new pipeline with the options applied
	WithOptions(opts ...Option) Pipeline
}

// Option modifies pipeline behavior
type Option func(*PipelineOptions)

// PipelineOptions contains configurable pipeline settings
type PipelineOptions struct {
	// Cache enables response caching
	Cache bool

	// CacheTTL specifies cache lifetime
	CacheTTL int64

	// FallbackProvider specifies a fallback provider if the primary fails
	FallbackProvider provider.Provider

	// MaxRetries specifies retry attempts for transient errors
	MaxRetries int
}

// Common pipeline options

// WithCache enables caching with the given TTL in seconds
func WithCache(ttlSeconds int64) Option {
	return func(o *PipelineOptions) {
		o.Cache = true
		o.CacheTTL = ttlSeconds
	}
}

// WithFallback sets a fallback provider if the primary one fails
func WithFallback(fallback provider.Provider) Option {
	return func(o *PipelineOptions) {
		o.FallbackProvider = fallback
	}
}

// WithRetries sets max retry attempts for transient errors
func WithRetries(maxRetries int) Option {
	return func(o *PipelineOptions) {
		o.MaxRetries = maxRetries
	}
}

// BasePipeline provides common pipeline functionality
type BasePipeline struct {
	options PipelineOptions
}

// New creates a new pipeline with the given provider
func New(p provider.Provider, opts ...Option) Pipeline {
	return NewSimplePipeline(p, opts...)
}

// NewSimplePipeline creates a basic pipeline with a single provider
func NewSimplePipeline(p provider.Provider, opts ...Option) *SimplePipeline {
	options := PipelineOptions{
		MaxRetries: 1, // Default to 1 attempt (0 retries)
	}

	// Apply options
	for _, opt := range opts {
		opt(&options)
	}

	return &SimplePipeline{
		Provider: p,
		BasePipeline: BasePipeline{
			options: options,
		},
	}
}

// SimplePipeline is a basic pipeline with a single provider
type SimplePipeline struct {
	Provider provider.Provider
	BasePipeline
}

// Execute processes a text input and returns a text response
func (p *SimplePipeline) Execute(ctx context.Context, input string) (string, error) {
	if p.Provider == nil {
		return "", errors.New("no provider configured")
	}

	// Create a basic request
	request := provider.Request{
		Prompt: input,
	}

	// Use the full request execution
	response, err := p.ExecuteWithRequest(ctx, request)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// ExecuteWithRequest processes a full provider request
func (p *SimplePipeline) ExecuteWithRequest(ctx context.Context, request provider.Request) (provider.Response, error) {
	if p.Provider == nil {
		return provider.Response{}, aierrors.New("pipeline", "execute", errors.New("no provider configured"))
	}

	var response provider.Response
	var err error

	// Handle retries for transient errors
	attemptsLeft := p.options.MaxRetries
	for attemptsLeft > 0 {
		attemptsLeft--

		// Generate response
		response, err = p.Provider.GenerateResponse(ctx, request)

		// If successful, return
		if err == nil {
			return response, nil
		}

		// Only retry on rate limit or timeout errors
		if !errors.Is(err, aierrors.ErrRateLimit) && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			break
		}

		// Exit if no more attempts or context cancelled
		if attemptsLeft <= 0 || errors.Is(ctx.Err(), context.Canceled) {
			break
		}
	}

	// If all attempts failed but we have a fallback, try that
	if err != nil && p.options.FallbackProvider != nil {
		response, fallbackErr := p.options.FallbackProvider.GenerateResponse(ctx, request)
		if fallbackErr == nil {
			return response, nil
		}
		// Keep the original error
	}

	return response, aierrors.New("pipeline", "execute", err)
}

// WithOptions returns a new pipeline with the options applied
func (p *SimplePipeline) WithOptions(opts ...Option) Pipeline {
	// Copy current options
	newOptions := p.options

	// Apply new options
	for _, opt := range opts {
		opt(&newOptions)
	}

	// Create a new pipeline with the updated options
	return &SimplePipeline{
		Provider: p.Provider,
		BasePipeline: BasePipeline{
			options: newOptions,
		},
	}
}
