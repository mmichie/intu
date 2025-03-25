package aikit

import (
	"context"
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

// ResultCombiner defines how multiple results should be combined
type ResultCombiner interface {
	Combine(ctx context.Context, results []ProviderResponse) (string, error)
}