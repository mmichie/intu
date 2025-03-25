package pipelines

import (
	"context"
)

// PipelineFunc is a function type that implements the Pipeline interface
type PipelineFunc func(ctx context.Context, input string) (string, error)

// PipelineAdapter adapts a function to the Pipeline interface
type PipelineAdapter struct {
	fn PipelineFunc
}

// NewPipelineAdapter creates a new PipelineAdapter
func NewPipelineAdapter(fn PipelineFunc) *PipelineAdapter {
	return &PipelineAdapter{fn: fn}
}

// Execute implements the Pipeline interface
func (p *PipelineAdapter) Execute(ctx context.Context, input string) (string, error) {
	return p.fn(ctx, input)
}
