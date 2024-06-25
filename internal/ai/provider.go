package ai

import "context"

type Provider interface {
	GenerateResponse(ctx context.Context, prompt string) (string, error)
}
