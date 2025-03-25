package aikit

import (
	"context"

	"github.com/mmichie/intu/pkg/aikit/pipelines"
)

// NewSerialPipeline creates a new serial pipeline
func NewSerialPipeline(providers []Provider) Pipeline {
	return pipelines.NewSerialPipeline(providersToPipeline(providers))
}

// NewParallelPipeline creates a new parallel pipeline
func NewParallelPipeline(providers []Provider, combiner ResultCombiner) Pipeline {
	return pipelines.NewParallelPipeline(providersToPipeline(providers), combinerToPipeline(combiner))
}

// NewBestPickerCombiner creates a new best picker combiner
func NewBestPickerCombiner(picker Provider) ResultCombiner {
	return &bestPickerCombinerAdapter{
		internal: pipelines.NewBestPickerCombiner(pipelineProvider{picker}),
	}
}

// NewConcatCombiner creates a new concatenation combiner
func NewConcatCombiner(separator string) ResultCombiner {
	return &concatCombinerAdapter{
		internal: pipelines.NewConcatCombiner(separator),
	}
}

// Adapter types to convert between interfaces
type pipelineProvider struct {
	Provider
}

type pipelinePipeline struct {
	internal pipelines.Pipeline
}

func (p pipelinePipeline) Execute(ctx context.Context, input string) (string, error) {
	return p.internal.Execute(ctx, input)
}

type bestPickerCombinerAdapter struct {
	internal *pipelines.BestPickerCombiner
}

func (b *bestPickerCombinerAdapter) Combine(ctx context.Context, results []ProviderResponse) (string, error) {
	pipelineResults := make([]pipelines.ProviderResponse, len(results))
	for i, r := range results {
		pipelineResults[i] = pipelines.ProviderResponse{
			ProviderName: r.ProviderName,
			Content:      r.Content,
		}
	}
	return b.internal.Combine(ctx, pipelineResults)
}

type concatCombinerAdapter struct {
	internal *pipelines.ConcatCombiner
}

func (c *concatCombinerAdapter) Combine(ctx context.Context, results []ProviderResponse) (string, error) {
	pipelineResults := make([]pipelines.ProviderResponse, len(results))
	for i, r := range results {
		pipelineResults[i] = pipelines.ProviderResponse{
			ProviderName: r.ProviderName,
			Content:      r.Content,
		}
	}
	return c.internal.Combine(ctx, pipelineResults)
}

func providersToPipeline(providers []Provider) []pipelines.Provider {
	result := make([]pipelines.Provider, len(providers))
	for i, p := range providers {
		result[i] = pipelineProvider{p}
	}
	return result
}

func combinerToPipeline(combiner ResultCombiner) pipelines.ResultCombiner {
	switch c := combiner.(type) {
	case *bestPickerCombinerAdapter:
		return c.internal
	case *concatCombinerAdapter:
		return c.internal
	default:
		return &resultCombinerAdapter{combiner}
	}
}

type resultCombinerAdapter struct {
	ResultCombiner
}

func (r *resultCombinerAdapter) Combine(ctx context.Context, results []pipelines.ProviderResponse) (string, error) {
	adapterResults := make([]ProviderResponse, len(results))
	for i, result := range results {
		adapterResults[i] = ProviderResponse{
			ProviderName: result.ProviderName,
			Content:      result.Content,
		}
	}
	return r.ResultCombiner.Combine(ctx, adapterResults)
}
