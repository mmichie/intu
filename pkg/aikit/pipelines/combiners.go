package pipelines

import (
	"context"
	"fmt"
)

// BestPickerCombiner uses an AI provider to select the best result
type BestPickerCombiner struct {
	picker Provider
}

// ConcatCombiner simply concatenates all results
type ConcatCombiner struct {
	separator string
}

// BestPickerCombiner implementation
func NewBestPickerCombiner(picker Provider) *BestPickerCombiner {
	return &BestPickerCombiner{picker: picker}
}

func (c *BestPickerCombiner) Combine(ctx context.Context, results []ProviderResponse) (string, error) {
	// First show all responses with provider names
	var allResponses string
	for i, result := range results {
		allResponses += fmt.Sprintf("\n=== Response %d (%s) ===\n%s\n\n", i+1, result.ProviderName, result.Content)
	}

	// Then get AI to evaluate them
	prompt := "Given these responses, select the best one and explain why:\n\n"
	for i, result := range results {
		prompt += fmt.Sprintf("Response %d (%s):\n%s\n\n", i+1, result.ProviderName, result.Content)
	}

	evaluation, err := c.picker.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s\n=== Evaluation ===\n%s", allResponses, evaluation), nil
}

// ConcatCombiner implementation
func NewConcatCombiner(separator string) *ConcatCombiner {
	return &ConcatCombiner{separator: separator}
}

func (c *ConcatCombiner) Combine(_ context.Context, results []ProviderResponse) (string, error) {
	var combined string
	for i, result := range results {
		if i > 0 {
			combined += c.separator
		}
		combined += fmt.Sprintf("=== %s ===\n%s", result.ProviderName, result.Content)
	}
	return combined, nil
}