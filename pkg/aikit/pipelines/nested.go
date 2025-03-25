package pipelines

import (
	"context"
	"fmt"
)

// NestedPipeline allows for the composition of multiple pipelines
type NestedPipeline struct {
	stages []Pipeline
}

func NewNestedPipeline(stages []Pipeline) *NestedPipeline {
	return &NestedPipeline{
		stages: stages,
	}
}

func (p *NestedPipeline) Execute(ctx context.Context, input string) (string, error) {
	if len(p.stages) == 0 {
		return "", fmt.Errorf("nested pipeline needs at least one stage")
	}

	current := input
	for i, stage := range p.stages {
		result, err := stage.Execute(ctx, current)
		if err != nil {
			return "", fmt.Errorf("pipeline stage %d failed: %w", i+1, err)
		}
		current = result
	}

	return current, nil
}
