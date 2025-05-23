package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// ParallelPipeline executes providers in parallel and combines their responses
type ParallelPipeline struct {
	Providers []provider.Provider
	Combiner  ResultCombiner
	BasePipeline
}

// ResultCombiner defines how multiple results should be combined
type ResultCombiner interface {
	Combine(ctx context.Context, results []provider.Response) (provider.Response, error)
}

// NewParallelPipeline creates a new parallel pipeline
func NewParallelPipeline(providers []provider.Provider, combiner ResultCombiner, opts ...Option) *ParallelPipeline {
	options := PipelineOptions{
		MaxRetries: 1,
	}

	// Apply options
	for _, opt := range opts {
		opt(&options)
	}

	return &ParallelPipeline{
		Providers: providers,
		Combiner:  combiner,
		BasePipeline: BasePipeline{
			options: options,
		},
	}
}

// Execute processes a text input through multiple providers in parallel
func (p *ParallelPipeline) Execute(ctx context.Context, input string) (string, error) {
	if len(p.Providers) == 0 {
		return "", errors.New("no providers configured for parallel pipeline")
	}

	// Create basic request
	request := provider.Request{
		Prompt: input,
	}

	response, err := p.ExecuteWithRequest(ctx, request)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// ExecuteWithRequest processes a full request through multiple providers in parallel
func (p *ParallelPipeline) ExecuteWithRequest(ctx context.Context, request provider.Request) (provider.Response, error) {
	if len(p.Providers) == 0 {
		return provider.Response{}, errors.New("no providers configured for parallel pipeline")
	}

	if p.Combiner == nil {
		return provider.Response{}, errors.New("no combiner configured for parallel pipeline")
	}

	// Create channels for results and errors
	type result struct {
		response provider.Response
		err      error
		index    int
	}

	resultsChan := make(chan result, len(p.Providers))

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Launch goroutines for each provider
	var wg sync.WaitGroup
	for i, prov := range p.Providers {
		wg.Add(1)
		go func(idx int, p provider.Provider) {
			defer wg.Done()

			// Generate response
			resp, err := p.GenerateResponse(ctx, request)

			select {
			case resultsChan <- result{response: resp, err: err, index: idx}:
			case <-ctx.Done():
				// Context cancelled, exit
			}
		}(i, prov)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var responses []provider.Response
	var errs []error

	for res := range resultsChan {
		if res.err != nil {
			errs = append(errs, fmt.Errorf("provider %d: %w", res.index, res.err))
			continue
		}
		responses = append(responses, res.response)
	}

	// If no successful responses, return aggregated error
	if len(responses) == 0 {
		if len(errs) > 0 {
			return provider.Response{}, fmt.Errorf("all providers failed: %v", errs)
		}
		return provider.Response{}, errors.New("no responses received")
	}

	// Combine results
	combined, err := p.Combiner.Combine(ctx, responses)
	if err != nil {
		return provider.Response{}, fmt.Errorf("failed to combine results: %w", err)
	}

	return combined, nil
}

// parallelProviderResponse wraps a provider response with timing info
type parallelProviderResponse struct {
	Response provider.Response
	Provider provider.Provider
	Duration int64 // in milliseconds
	Index    int
}

// executionMetrics tracks execution metrics for parallel execution
type executionMetrics struct {
	TotalDuration   int64
	FastestDuration int64
	SlowestDuration int64
	FailureCount    int
	SuccessCount    int
}

// ExecuteWithMetrics executes providers in parallel and returns metrics
func (p *ParallelPipeline) ExecuteWithMetrics(ctx context.Context, request provider.Request) (provider.Response, *executionMetrics, error) {
	// This is similar to ExecuteWithRequest but also tracks metrics
	// Implementation would track timing and success/failure rates

	// For now, just delegate to ExecuteWithRequest
	response, err := p.ExecuteWithRequest(ctx, request)

	metrics := &executionMetrics{
		SuccessCount: len(p.Providers),
	}

	return response, metrics, err
}

// WithOptions returns a new pipeline with the options applied
func (p *ParallelPipeline) WithOptions(opts ...Option) Pipeline {
	// Copy current options
	newOptions := p.options

	// Apply new options
	for _, opt := range opts {
		opt(&newOptions)
	}

	// Create a new pipeline with the updated options
	return &ParallelPipeline{
		Providers: p.Providers,
		Combiner:  p.Combiner,
		BasePipeline: BasePipeline{
			options: newOptions,
		},
	}
}
