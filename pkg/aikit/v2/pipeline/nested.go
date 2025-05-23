package pipeline

import (
	"context"
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// NestedPipeline chains multiple pipelines together
type NestedPipeline struct {
	Stages []Pipeline
	BasePipeline
}

// NewNestedPipeline creates a pipeline that chains multiple pipelines
func NewNestedPipeline(stages []Pipeline, opts ...Option) *NestedPipeline {
	options := PipelineOptions{
		MaxRetries: 1,
	}

	// Apply options
	for _, opt := range opts {
		opt(&options)
	}

	return &NestedPipeline{
		Stages: stages,
		BasePipeline: BasePipeline{
			options: options,
		},
	}
}

// Execute processes input through all pipeline stages sequentially
func (n *NestedPipeline) Execute(ctx context.Context, input string) (string, error) {
	if len(n.Stages) == 0 {
		return "", fmt.Errorf("no stages configured for nested pipeline")
	}

	current := input
	for i, stage := range n.Stages {
		result, err := stage.Execute(ctx, current)
		if err != nil {
			return "", fmt.Errorf("stage %d failed: %w", i, err)
		}
		current = result
	}

	return current, nil
}

// ExecuteWithRequest processes a request through all stages
func (n *NestedPipeline) ExecuteWithRequest(ctx context.Context, request provider.Request) (provider.Response, error) {
	if len(n.Stages) == 0 {
		return provider.Response{}, fmt.Errorf("no stages configured for nested pipeline")
	}

	// First stage processes the original request
	response, err := n.Stages[0].ExecuteWithRequest(ctx, request)
	if err != nil {
		return response, fmt.Errorf("stage 0 failed: %w", err)
	}

	// Subsequent stages process the output as new prompts
	for i := 1; i < len(n.Stages); i++ {
		// Create a new request with the previous response content
		nextRequest := provider.Request{
			Prompt:           response.Content,
			Temperature:      request.Temperature,
			MaxTokens:        request.MaxTokens,
			FunctionRegistry: request.FunctionRegistry,
			FunctionExecutor: request.FunctionExecutor,
			Parameters:       request.Parameters,
		}

		response, err = n.Stages[i].ExecuteWithRequest(ctx, nextRequest)
		if err != nil {
			return response, fmt.Errorf("stage %d failed: %w", i, err)
		}
	}

	// Add metadata about the pipeline execution
	if response.Metadata == nil {
		response.Metadata = make(map[string]interface{})
	}
	response.Metadata["pipeline_stages"] = len(n.Stages)
	response.Metadata["pipeline_type"] = "nested"

	return response, nil
}

// WithOptions returns a new pipeline with options applied
func (n *NestedPipeline) WithOptions(opts ...Option) Pipeline {
	newOptions := n.options
	for _, opt := range opts {
		opt(&newOptions)
	}

	return &NestedPipeline{
		Stages: n.Stages,
		BasePipeline: BasePipeline{
			options: newOptions,
		},
	}
}

// AddStage adds a new stage to the pipeline
func (n *NestedPipeline) AddStage(stage Pipeline) *NestedPipeline {
	n.Stages = append(n.Stages, stage)
	return n
}

// StageCount returns the number of stages
func (n *NestedPipeline) StageCount() int {
	return len(n.Stages)
}

// GetStage returns a specific stage by index
func (n *NestedPipeline) GetStage(index int) (Pipeline, error) {
	if index < 0 || index >= len(n.Stages) {
		return nil, fmt.Errorf("stage index %d out of range", index)
	}
	return n.Stages[index], nil
}
