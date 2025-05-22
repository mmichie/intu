package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// BestPickerCombiner selects the best response from multiple providers
// using a designated judge provider
type BestPickerCombiner struct {
	Judge provider.Provider
}

// NewBestPickerCombiner creates a new best picker combiner
func NewBestPickerCombiner(judge provider.Provider) *BestPickerCombiner {
	return &BestPickerCombiner{
		Judge: judge,
	}
}

// Combine uses the judge to select the best response
func (c *BestPickerCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	if len(results) == 0 {
		return provider.Response{}, errors.New("no results to combine")
	}

	// If there is only one result, return it directly
	if len(results) == 1 {
		return results[0], nil
	}

	// Construct a prompt for the judge to evaluate the responses
	var promptBuilder strings.Builder
	promptBuilder.WriteString("Please evaluate the following responses and select the best one. ")
	promptBuilder.WriteString("Return ONLY the number of the best response (1, 2, etc.) without explanation.\n\n")

	// Add each response to the prompt
	for i, result := range results {
		providerName := result.Provider
		if providerName == "" {
			providerName = fmt.Sprintf("Provider %d", i+1)
		}

		promptBuilder.WriteString(fmt.Sprintf("Response %d (from %s):\n%s\n\n",
			i+1, providerName, result.Content))
	}

	// Create a request for the judge
	judgeRequest := provider.Request{
		Prompt:      promptBuilder.String(),
		Temperature: 0.1, // Low temperature for more decisive judgment
	}

	// Ask the judge to evaluate
	judgeResponse, err := c.Judge.GenerateResponse(ctx, judgeRequest)
	if err != nil {
		return provider.Response{}, fmt.Errorf("error asking judge to evaluate: %w", err)
	}

	// Parse the judge's response to determine the best result
	bestIndex := -1
	judgment := strings.TrimSpace(judgeResponse.Content)

	// Try to find a number in the response
	for i := 1; i <= len(results); i++ {
		if strings.Contains(judgment, fmt.Sprintf("%d", i)) {
			bestIndex = i - 1
			break
		}
	}

	// If no valid index was found, use the first result
	if bestIndex < 0 || bestIndex >= len(results) {
		bestIndex = 0
	}

	// Return the selected response
	selected := results[bestIndex]

	// Add judgment metadata
	if selected.Metadata == nil {
		selected.Metadata = make(map[string]interface{})
	}
	selected.Metadata["judgment"] = judgment
	selected.Metadata["judge_provider"] = c.Judge.Name()
	selected.Metadata["total_responses"] = len(results)
	selected.Metadata["selected_index"] = bestIndex + 1

	return selected, nil
}
