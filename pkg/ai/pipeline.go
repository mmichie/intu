package ai

import (
	"context"
	"fmt"
	"sync"
)

// ProviderResponse pairs the provider name with its response
type ProviderResponse struct {
	ProviderName string
	Content      string
}

// Pipeline represents a sequence of AI operations
type Pipeline interface {
	Execute(ctx context.Context, input string) (string, error)
}

// ParallelPipeline executes multiple providers and combines/selects results
type ParallelPipeline struct {
	providers []Provider
	combiner  ResultCombiner
}

// SerialPipeline passes output from one provider to the next
type SerialPipeline struct {
	providers []Provider
}

// ResultCombiner defines how multiple results should be combined
type ResultCombiner interface {
	Combine(ctx context.Context, results []ProviderResponse) (string, error)
}

// BestPickerCombiner uses an AI provider to select the best result
type BestPickerCombiner struct {
	picker Provider
}

// ConcatCombiner simply concatenates all results
type ConcatCombiner struct {
	separator string
}

func NewParallelPipeline(providers []Provider, combiner ResultCombiner) *ParallelPipeline {
	return &ParallelPipeline{
		providers: providers,
		combiner:  combiner,
	}
}

func NewSerialPipeline(providers []Provider) *SerialPipeline {
	return &SerialPipeline{
		providers: providers,
	}
}

func (p *ParallelPipeline) Execute(ctx context.Context, input string) (string, error) {
	results := make([]ProviderResponse, len(p.providers))
	errors := make([]error, len(p.providers))
	var wg sync.WaitGroup

	for i, provider := range p.providers {
		wg.Add(1)
		go func(idx int, prov Provider) {
			defer wg.Done()
			result, err := prov.GenerateResponse(ctx, input)
			results[idx] = ProviderResponse{
				ProviderName: prov.Name(),
				Content:      result,
			}
			errors[idx] = err
		}(i, provider)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return "", fmt.Errorf("provider %d (%s) failed: %w", i, p.providers[i].Name(), err)
		}
	}

	return p.combiner.Combine(ctx, results)
}

func (p *SerialPipeline) Execute(ctx context.Context, input string) (string, error) {
	current := input
	for i, provider := range p.providers {
		result, err := provider.GenerateResponse(ctx, current)
		if err != nil {
			return "", fmt.Errorf("provider %d (%s) failed: %w", i, provider.Name(), err)
		}
		current = result
	}
	return current, nil
}

// BestPickerCombiner implementation
func NewBestPickerCombiner(picker Provider) *BestPickerCombiner {
	return &BestPickerCombiner{picker: picker}
}

func (c *BestPickerCombiner) Combine(ctx context.Context, results []ProviderResponse) (string, error) {
	// First show all responses with provider names
	var allResponses string
	for i, result := range results {
		allResponses += fmt.Sprintf("\n=== Response %d (%s) ===\n%s\n\n", i+1, result.ProviderName, result.Content)
	}

	// Then get AI to evaluate them
	prompt := "Given these responses, select the best one and explain why:\n\n"
	for i, result := range results {
		prompt += fmt.Sprintf("Response %d (%s):\n%s\n\n", i+1, result.ProviderName, result.Content)
	}

	evaluation, err := c.picker.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s\n=== Evaluation ===\n%s", allResponses, evaluation), nil
}

// ConcatCombiner implementation
func NewConcatCombiner(separator string) *ConcatCombiner {
	return &ConcatCombiner{separator: separator}
}

func (c *ConcatCombiner) Combine(_ context.Context, results []ProviderResponse) (string, error) {
	var combined string
	for i, result := range results {
		if i > 0 {
			combined += c.separator
		}
		combined += fmt.Sprintf("=== %s ===\n%s", result.ProviderName, result.Content)
	}
	return combined, nil
}
