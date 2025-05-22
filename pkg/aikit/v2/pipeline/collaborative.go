package pipeline

import (
	"context"
	"errors"
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// CollaborativePipeline implements a pipeline where providers collaborate
// in discussion rounds
type CollaborativePipeline struct {
	Providers []provider.Provider
	Rounds    int
	BasePipeline
}

// NewCollaborativePipeline creates a new collaborative pipeline
func NewCollaborativePipeline(providers []provider.Provider, rounds int, opts ...Option) *CollaborativePipeline {
	if rounds <= 0 {
		rounds = 3 // Default to 3 rounds
	}

	options := PipelineOptions{
		MaxRetries: 1,
	}

	// Apply options
	for _, opt := range opts {
		opt(&options)
	}

	return &CollaborativePipeline{
		Providers: providers,
		Rounds:    rounds,
		BasePipeline: BasePipeline{
			options: options,
		},
	}
}

// Execute implements the collaborative discussion pipeline
func (p *CollaborativePipeline) Execute(ctx context.Context, input string) (string, error) {
	if len(p.Providers) == 0 {
		return "", errors.New("no providers configured for collaborative pipeline")
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

// ExecuteWithRequest processes a collaborative discussion
func (p *CollaborativePipeline) ExecuteWithRequest(ctx context.Context, request provider.Request) (provider.Response, error) {
	if len(p.Providers) == 0 {
		return provider.Response{}, errors.New("no providers configured for collaborative pipeline")
	}

	// Initialize the discussion with the input prompt
	discussion := "Topic: " + request.Prompt + "\n\n"

	// For each round
	for round := 1; round <= p.Rounds; round++ {
		// For each provider in this round
		for _, providerObj := range p.Providers {
			// Build a prompt with the current discussion and ask for contribution
			roundPrompt := fmt.Sprintf(
				"%s\n\nRound %d, %s's turn: Please add your thoughts on this topic.",
				discussion,
				round,
				providerObj.Name(),
			)

			// Create a request for this round
			roundRequest := provider.Request{
				Prompt:           roundPrompt,
				FunctionRegistry: request.FunctionRegistry,
				FunctionExecutor: request.FunctionExecutor,
				Temperature:      request.Temperature,
				MaxTokens:        request.MaxTokens,
			}

			// Execute with retries
			var response provider.Response
			var err error

			attemptsLeft := p.options.MaxRetries
			for attemptsLeft > 0 {
				attemptsLeft--

				// Process with the current provider
				response, err = providerObj.GenerateResponse(ctx, roundRequest)

				// If successful, break
				if err == nil {
					break
				}

				// Exit if no more attempts or context canceled
				if attemptsLeft <= 0 || errors.Is(ctx.Err(), context.Canceled) {
					break
				}
			}

			// Check for error after all attempts
			if err != nil {
				return provider.Response{}, fmt.Errorf("provider %s failed in round %d: %w",
					providerObj.Name(), round, err)
			}

			// Add the response to the discussion
			discussion += fmt.Sprintf("\n\n%s (Round %d): %s",
				providerObj.Name(), round, response.Content)
		}

		// Increment the round counter
		round++

		// Check if context is cancelled before starting a new round
		select {
		case <-ctx.Done():
			return provider.Response{}, ctx.Err()
		default:
			// Continue to next round
		}
	}

	// Create a summary request if needed
	if len(p.Providers) > 0 {
		finalProvider := p.Providers[0] // Use first provider for summary
		summaryPrompt := fmt.Sprintf(
			"%s\n\nPlease provide a concise summary of this discussion, highlighting the key points and areas of agreement or disagreement.",
			discussion,
		)

		summaryRequest := provider.Request{
			Prompt:      summaryPrompt,
			Temperature: 0.3, // Lower temperature for more focused summary
			MaxTokens:   request.MaxTokens,
		}

		// Generate summary
		summary, err := finalProvider.GenerateResponse(ctx, summaryRequest)
		if err != nil {
			// If summary fails, just return the full discussion
			return provider.Response{
				Content:  discussion,
				Provider: "collaborative",
			}, nil
		}

		// Return the summary
		return provider.Response{
			Content:  summary.Content,
			Provider: "collaborative",
			Metadata: map[string]interface{}{
				"full_discussion": discussion,
				"rounds":          p.Rounds,
				"providers":       providerNames(p.Providers),
			},
		}, nil
	}

	// Return the full discussion if no summary
	return provider.Response{
		Content:  discussion,
		Provider: "collaborative",
	}, nil
}

// WithOptions returns a new pipeline with the options applied
func (p *CollaborativePipeline) WithOptions(opts ...Option) Pipeline {
	// Copy current options
	newOptions := p.options

	// Apply new options
	for _, opt := range opts {
		opt(&newOptions)
	}

	// Create a new pipeline with the updated options
	return &CollaborativePipeline{
		Providers: p.Providers,
		Rounds:    p.Rounds,
		BasePipeline: BasePipeline{
			options: newOptions,
		},
	}
}

// Helper to get provider names
func providerNames(providers []provider.Provider) []string {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	return names
}
