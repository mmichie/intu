package pipelines

import (
	"context"
	"fmt"
)

// SerialPipeline passes output from one provider to the next
type SerialPipeline struct {
	providers []Provider
}

func NewSerialPipeline(providers []Provider) *SerialPipeline {
	return &SerialPipeline{
		providers: providers,
	}
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
