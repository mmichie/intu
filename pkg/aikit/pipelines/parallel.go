package pipelines

import (
	"context"
	"fmt"
	"sync"
)

// ParallelPipeline executes multiple providers and combines/selects results
type ParallelPipeline struct {
	providers []Provider
	combiner  ResultCombiner
}

func NewParallelPipeline(providers []Provider, combiner ResultCombiner) *ParallelPipeline {
	return &ParallelPipeline{
		providers: providers,
		combiner:  combiner,
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
