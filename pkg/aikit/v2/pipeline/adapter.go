package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// ProcessFunc is a function that processes text
type ProcessFunc func(ctx context.Context, input string) (string, error)

// RequestProcessFunc is a function that processes a full request
type RequestProcessFunc func(ctx context.Context, request provider.Request) (provider.Response, error)

// FunctionAdapter wraps a function as a pipeline
type FunctionAdapter struct {
	Name      string
	ProcessFn ProcessFunc
	RequestFn RequestProcessFunc
	BasePipeline
}

// NewFunctionAdapter creates a pipeline from a processing function
func NewFunctionAdapter(name string, fn ProcessFunc, opts ...Option) *FunctionAdapter {
	options := PipelineOptions{
		MaxRetries: 1,
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &FunctionAdapter{
		Name:      name,
		ProcessFn: fn,
		BasePipeline: BasePipeline{
			options: options,
		},
	}
}

// NewRequestAdapter creates a pipeline from a request processing function
func NewRequestAdapter(name string, fn RequestProcessFunc, opts ...Option) *FunctionAdapter {
	options := PipelineOptions{
		MaxRetries: 1,
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &FunctionAdapter{
		Name:      name,
		RequestFn: fn,
		BasePipeline: BasePipeline{
			options: options,
		},
	}
}

// Execute processes input through the function
func (f *FunctionAdapter) Execute(ctx context.Context, input string) (string, error) {
	if f.ProcessFn != nil {
		return f.ProcessFn(ctx, input)
	}

	if f.RequestFn != nil {
		// Convert to request and back
		request := provider.Request{Prompt: input}
		response, err := f.RequestFn(ctx, request)
		if err != nil {
			return "", err
		}
		return response.Content, nil
	}

	return "", fmt.Errorf("no processing function configured")
}

// ExecuteWithRequest processes a full request
func (f *FunctionAdapter) ExecuteWithRequest(ctx context.Context, request provider.Request) (provider.Response, error) {
	if f.RequestFn != nil {
		return f.RequestFn(ctx, request)
	}

	if f.ProcessFn != nil {
		// Use the simple function with just the prompt
		result, err := f.ProcessFn(ctx, request.Prompt)
		if err != nil {
			return provider.Response{}, err
		}

		return provider.Response{
			Content:  result,
			Provider: f.Name,
			Model:    "function",
			Metadata: map[string]interface{}{
				"adapter_type": "function",
			},
		}, nil
	}

	return provider.Response{}, fmt.Errorf("no processing function configured")
}

// WithOptions returns a new pipeline with options applied
func (f *FunctionAdapter) WithOptions(opts ...Option) Pipeline {
	newOptions := f.options
	for _, opt := range opts {
		opt(&newOptions)
	}

	return &FunctionAdapter{
		Name:      f.Name,
		ProcessFn: f.ProcessFn,
		RequestFn: f.RequestFn,
		BasePipeline: BasePipeline{
			options: newOptions,
		},
	}
}

// TransformAdapter applies a transformation to pipeline input/output
type TransformAdapter struct {
	Pipeline        Pipeline
	InputTransform  ProcessFunc
	OutputTransform ProcessFunc
	Name            string
	BasePipeline
}

// NewTransformAdapter creates a pipeline that transforms input and output
func NewTransformAdapter(p Pipeline, inputTransform, outputTransform ProcessFunc, opts ...Option) *TransformAdapter {
	options := PipelineOptions{
		MaxRetries: 1,
	}

	for _, opt := range opts {
		opt(&options)
	}

	name := "transform"
	if p != nil {
		name = fmt.Sprintf("transform(%s)", getpipelineName(p))
	}

	return &TransformAdapter{
		Pipeline:        p,
		InputTransform:  inputTransform,
		OutputTransform: outputTransform,
		Name:            name,
		BasePipeline: BasePipeline{
			options: options,
		},
	}
}

// Execute applies transformations around the pipeline execution
func (t *TransformAdapter) Execute(ctx context.Context, input string) (string, error) {
	if t.Pipeline == nil {
		return "", fmt.Errorf("no pipeline configured")
	}

	// Apply input transformation if provided
	processInput := input
	if t.InputTransform != nil {
		transformed, err := t.InputTransform(ctx, input)
		if err != nil {
			return "", fmt.Errorf("input transformation failed: %w", err)
		}
		processInput = transformed
	}

	// Execute the pipeline
	result, err := t.Pipeline.Execute(ctx, processInput)
	if err != nil {
		return "", err
	}

	// Apply output transformation if provided
	if t.OutputTransform != nil {
		transformed, err := t.OutputTransform(ctx, result)
		if err != nil {
			return "", fmt.Errorf("output transformation failed: %w", err)
		}
		result = transformed
	}

	return result, nil
}

// ExecuteWithRequest applies transformations to request processing
func (t *TransformAdapter) ExecuteWithRequest(ctx context.Context, request provider.Request) (provider.Response, error) {
	if t.Pipeline == nil {
		return provider.Response{}, fmt.Errorf("no pipeline configured")
	}

	// Apply input transformation if provided
	processRequest := request
	if t.InputTransform != nil {
		transformed, err := t.InputTransform(ctx, request.Prompt)
		if err != nil {
			return provider.Response{}, fmt.Errorf("input transformation failed: %w", err)
		}
		processRequest.Prompt = transformed
	}

	// Execute the pipeline
	response, err := t.Pipeline.ExecuteWithRequest(ctx, processRequest)
	if err != nil {
		return response, err
	}

	// Apply output transformation if provided
	if t.OutputTransform != nil {
		transformed, err := t.OutputTransform(ctx, response.Content)
		if err != nil {
			return response, fmt.Errorf("output transformation failed: %w", err)
		}
		response.Content = transformed
	}

	// Add transformation metadata
	if response.Metadata == nil {
		response.Metadata = make(map[string]interface{})
	}
	response.Metadata["transformed"] = true
	response.Metadata["transform_adapter"] = t.Name

	return response, nil
}

// WithOptions returns a new pipeline with options applied
func (t *TransformAdapter) WithOptions(opts ...Option) Pipeline {
	newOptions := t.options
	for _, opt := range opts {
		opt(&newOptions)
	}

	return &TransformAdapter{
		Pipeline:        t.Pipeline,
		InputTransform:  t.InputTransform,
		OutputTransform: t.OutputTransform,
		Name:            t.Name,
		BasePipeline: BasePipeline{
			options: newOptions,
		},
	}
}

// Common transformation functions

// TemplateTransform creates a transform function that applies a template
func TemplateTransform(template string) ProcessFunc {
	return func(ctx context.Context, input string) (string, error) {
		// Simple template replacement
		result := strings.ReplaceAll(template, "{{input}}", input)
		result = strings.ReplaceAll(result, "{{.Input}}", input)
		return result, nil
	}
}

// PrefixTransform adds a prefix to the input
func PrefixTransform(prefix string) ProcessFunc {
	return func(ctx context.Context, input string) (string, error) {
		return prefix + input, nil
	}
}

// SuffixTransform adds a suffix to the input
func SuffixTransform(suffix string) ProcessFunc {
	return func(ctx context.Context, input string) (string, error) {
		return input + suffix, nil
	}
}

// WrapTransform wraps the input with prefix and suffix
func WrapTransform(prefix, suffix string) ProcessFunc {
	return func(ctx context.Context, input string) (string, error) {
		return prefix + input + suffix, nil
	}
}

// JSONExtractTransform extracts a field from JSON output
func JSONExtractTransform(field string) ProcessFunc {
	return func(ctx context.Context, input string) (string, error) {
		// This is a simplified version - in production you'd use proper JSON parsing
		// For now, we'll just return the input if it fails
		return input, nil
	}
}

// Helper function to get pipeline name
func getpipelineName(p Pipeline) string {
	switch v := p.(type) {
	case *SimplePipeline:
		if v.Provider != nil {
			return v.Provider.Name()
		}
	case *SerialPipeline:
		return "serial"
	case *ParallelPipeline:
		return "parallel"
	case *CollaborativePipeline:
		return "collaborative"
	case *NestedPipeline:
		return "nested"
	case *FunctionAdapter:
		return v.Name
	case *TransformAdapter:
		return v.Name
	}
	return "unknown"
}
