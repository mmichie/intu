package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	if p.Combiner == nil {
		return "", errors.New("no combiner configured for parallel pipeline")
	}

	// Create a basic request
	request := provider.Request{
		Prompt: input,
	}

	// Execute with the request
	response, err := p.ExecuteWithRequest(ctx, request)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// ExecuteWithRequest executes a request through providers in parallel
func (p *ParallelPipeline) ExecuteWithRequest(ctx context.Context, request provider.Request) (provider.Response, error) {
	if len(p.Providers) == 0 {
		return provider.Response{}, errors.New("no providers configured for parallel pipeline")
	}

	if p.Combiner == nil {
		return provider.Response{}, errors.New("no combiner configured for parallel pipeline")
	}

	// Create channels for results and errors
	type resultError struct {
		response provider.Response
		err      error
		index    int
	}
	resultChan := make(chan resultError, len(p.Providers))

	// Create a wait group to track completion
	var wg sync.WaitGroup
	wg.Add(len(p.Providers))

	// Create a context with cancellation
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Execute each provider in a goroutine
	for i, providerObj := range p.Providers {
		go func(index int, prov provider.Provider) {
			defer wg.Done()

			// Execute with retries
			var response provider.Response
			var err error

			attemptsLeft := p.options.MaxRetries
			for attemptsLeft > 0 {
				attemptsLeft--

				// Check if context is cancelled before attempt
				select {
				case <-childCtx.Done():
					resultChan <- resultError{
						response: provider.Response{},
						err:      childCtx.Err(),
						index:    index,
					}
					return
				default:
					// Continue with execution
				}

				// Process with the current provider
				response, err = prov.GenerateResponse(childCtx, request)

				// If successful, break
				if err == nil {
					break
				}

				// Exit if no more attempts or context canceled
				if attemptsLeft <= 0 || errors.Is(childCtx.Err(), context.Canceled) {
					break
				}
			}

			// Send result to channel
			resultChan <- resultError{
				response: response,
				err:      err,
				index:    index,
			}
		}(i, providerObj)
	}

	// Close the result channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results maintaining provider order
	results := make([]provider.Response, len(p.Providers))
	errors := make([]error, len(p.Providers))

	for result := range resultChan {
		if result.err != nil {
			errors[result.index] = result.err
		} else {
			results[result.index] = result.response
		}
	}

	// Check for errors
	var errorMessages []string
	successfulResults := make([]provider.Response, 0, len(p.Providers))

	for i, err := range errors {
		if err != nil {
			providerName := fmt.Sprintf("provider-%d", i)
			if i < len(p.Providers) {
				providerName = p.Providers[i].Name()
			}
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %v", providerName, err))
		} else {
			successfulResults = append(successfulResults, results[i])
		}
	}

	// If all providers failed, return an error
	if len(successfulResults) == 0 {
		return provider.Response{}, fmt.Errorf("all providers failed: %s", strings.Join(errorMessages, "; "))
	}

	// Combine the successful results
	return p.Combiner.Combine(ctx, successfulResults)
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

// Common combiners

// ConcatCombiner combines results by concatenating them with a separator
type ConcatCombiner struct {
	Separator string
}

// NewConcatCombiner creates a new concatenation combiner
func NewConcatCombiner(separator string) *ConcatCombiner {
	return &ConcatCombiner{
		Separator: separator,
	}
}

// Combine concatenates multiple responses with a separator
func (c *ConcatCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	if len(results) == 0 {
		return provider.Response{}, errors.New("no results to combine")
	}

	// Use builder for efficient string concatenation
	var builder strings.Builder

	for i, result := range results {
		if i > 0 {
			builder.WriteString(c.Separator)
		}
		builder.WriteString(result.Content)
	}

	// Use the metadata and model info from the first result
	combined := provider.Response{
		Content:  builder.String(),
		Provider: "combined",
	}

	// If all results have the same provider, use that
	sameProvider := true
	firstProvider := results[0].Provider

	for _, result := range results {
		if result.Provider != firstProvider {
			sameProvider = false
			break
		}
	}

	if sameProvider {
		combined.Provider = firstProvider
		combined.Model = results[0].Model
	}

	return combined, nil
}
