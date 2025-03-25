package pipelines

import (
	"context"
)

// Provider interface for pipeline use
type Provider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
	Name() string
}

// ProviderResponse pairs the provider name with its response
type ProviderResponse struct {
	ProviderName string
	Content      string
}

// ResultCombiner defines how multiple results should be combined
type ResultCombiner interface {
	Combine(ctx context.Context, results []ProviderResponse) (string, error)
}

// Pipeline represents a sequence of AI operations
type Pipeline interface {
	Execute(ctx context.Context, input string) (string, error)
}
