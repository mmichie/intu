package pipeline

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// ConcatCombiner concatenates all results with a separator
type ConcatCombiner struct {
	Separator string
}

// NewConcatCombiner creates a new concatenation combiner
func NewConcatCombiner(separator string) *ConcatCombiner {
	if separator == "" {
		separator = "\n\n"
	}
	return &ConcatCombiner{Separator: separator}
}

// Combine concatenates all results
func (c *ConcatCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	if len(results) == 0 {
		return provider.Response{}, fmt.Errorf("no results to combine")
	}

	var contents []string
	var totalPromptTokens, totalCompletionTokens int

	for _, result := range results {
		if result.Content != "" {
			contents = append(contents, result.Content)
		}
		if result.Usage != nil {
			totalPromptTokens += result.Usage.PromptTokens
			totalCompletionTokens += result.Usage.CompletionTokens
		}
	}

	combined := strings.Join(contents, c.Separator)

	return provider.Response{
		Content:  combined,
		Provider: "concat_combiner",
		Model:    "combined",
		Usage: &provider.UsageInfo{
			PromptTokens:     totalPromptTokens,
			CompletionTokens: totalCompletionTokens,
			TotalTokens:      totalPromptTokens + totalCompletionTokens,
		},
		Metadata: map[string]interface{}{
			"source_count": len(results),
			"sources":      getSourceProviders(results),
		},
	}, nil
}

// MajorityVoteCombiner selects the most common response
type MajorityVoteCombiner struct{}

// NewMajorityVoteCombiner creates a new majority vote combiner
func NewMajorityVoteCombiner() *MajorityVoteCombiner {
	return &MajorityVoteCombiner{}
}

// Combine selects the most common response
func (m *MajorityVoteCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	if len(results) == 0 {
		return provider.Response{}, fmt.Errorf("no results to combine")
	}

	// Count occurrences of each response
	votes := make(map[string][]provider.Response)
	for _, result := range results {
		normalized := strings.TrimSpace(result.Content)
		votes[normalized] = append(votes[normalized], result)
	}

	// Find the response with the most votes
	var bestResponse provider.Response
	maxVotes := 0

	for _, responses := range votes {
		if len(responses) > maxVotes {
			maxVotes = len(responses)
			bestResponse = responses[0] // Use the first occurrence
		}
	}

	// Add voting metadata
	if bestResponse.Metadata == nil {
		bestResponse.Metadata = make(map[string]interface{})
	}
	bestResponse.Metadata["votes"] = maxVotes
	bestResponse.Metadata["total_responses"] = len(results)
	bestResponse.Metadata["consensus_ratio"] = float64(maxVotes) / float64(len(results))

	return bestResponse, nil
}

// FirstSuccessfulCombiner returns the first non-empty result
type FirstSuccessfulCombiner struct{}

// NewFirstSuccessfulCombiner creates a combiner that returns the first successful result
func NewFirstSuccessfulCombiner() *FirstSuccessfulCombiner {
	return &FirstSuccessfulCombiner{}
}

// Combine returns the first non-empty result
func (f *FirstSuccessfulCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	for _, result := range results {
		if result.Content != "" {
			return result, nil
		}
	}
	return provider.Response{}, fmt.Errorf("no successful results")
}

// LongestResponseCombiner selects the longest response
type LongestResponseCombiner struct{}

// NewLongestResponseCombiner creates a combiner that selects the longest response
func NewLongestResponseCombiner() *LongestResponseCombiner {
	return &LongestResponseCombiner{}
}

// Combine selects the longest response
func (l *LongestResponseCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	if len(results) == 0 {
		return provider.Response{}, fmt.Errorf("no results to combine")
	}

	longest := results[0]
	for _, result := range results[1:] {
		if len(result.Content) > len(longest.Content) {
			longest = result
		}
	}

	return longest, nil
}

// RoundRobinCombiner distributes requests in round-robin fashion
type RoundRobinCombiner struct {
	counter uint64
}

// NewRoundRobinCombiner creates a round-robin combiner for load balancing
func NewRoundRobinCombiner() *RoundRobinCombiner {
	return &RoundRobinCombiner{}
}

// Combine selects one result in round-robin fashion
func (r *RoundRobinCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	if len(results) == 0 {
		return provider.Response{}, fmt.Errorf("no results to combine")
	}

	// Increment counter atomically and select result
	index := atomic.AddUint64(&r.counter, 1) % uint64(len(results))
	return results[index], nil
}

// RandomCombiner randomly selects a result
type RandomCombiner struct{}

// NewRandomCombiner creates a combiner that randomly selects a result
func NewRandomCombiner() *RandomCombiner {
	return &RandomCombiner{}
}

// Combine randomly selects one result
func (r *RandomCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	if len(results) == 0 {
		return provider.Response{}, fmt.Errorf("no results to combine")
	}

	index := rand.Intn(len(results))
	return results[index], nil
}

// ConsensusCombiner uses an evaluator to find consensus
type ConsensusCombiner struct {
	Evaluator provider.Provider
}

// NewConsensusCombiner creates a combiner that uses an AI provider to evaluate consensus
func NewConsensusCombiner(evaluator provider.Provider) *ConsensusCombiner {
	return &ConsensusCombiner{Evaluator: evaluator}
}

// Combine finds consensus among results using the evaluator
func (c *ConsensusCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	if len(results) == 0 {
		return provider.Response{}, fmt.Errorf("no results to combine")
	}

	if len(results) == 1 {
		return results[0], nil
	}

	// Build a prompt for the evaluator
	var prompt strings.Builder
	prompt.WriteString("Multiple AI providers have given the following responses. ")
	prompt.WriteString("Please synthesize these into a single, consensus response that captures ")
	prompt.WriteString("the key points where they agree and notes any significant disagreements:\n\n")

	for i, result := range results {
		prompt.WriteString(fmt.Sprintf("Response %d (from %s):\n%s\n\n",
			i+1, result.Provider, result.Content))
	}

	prompt.WriteString("Synthesized consensus response:")

	// Use the evaluator to generate consensus
	consensusReq := provider.Request{
		Prompt:    prompt.String(),
		MaxTokens: 2048,
	}

	consensusResp, err := c.Evaluator.GenerateResponse(ctx, consensusReq)
	if err != nil {
		return provider.Response{}, fmt.Errorf("failed to generate consensus: %w", err)
	}

	// Add metadata about the consensus process
	consensusResp.Metadata = map[string]interface{}{
		"consensus_from": len(results),
		"sources":        getSourceProviders(results),
		"method":         "ai_synthesis",
	}

	return consensusResp, nil
}

// WeightedCombiner combines results based on weights
type WeightedCombiner struct {
	Weights map[string]float64
}

// NewWeightedCombiner creates a combiner that weights results by provider
func NewWeightedCombiner(weights map[string]float64) *WeightedCombiner {
	return &WeightedCombiner{Weights: weights}
}

// Combine selects result based on provider weights
func (w *WeightedCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	if len(results) == 0 {
		return provider.Response{}, fmt.Errorf("no results to combine")
	}

	// Sort results by weight
	type weightedResult struct {
		response provider.Response
		weight   float64
	}

	weighted := make([]weightedResult, 0, len(results))
	for _, result := range results {
		weight := w.Weights[result.Provider]
		if weight == 0 {
			weight = 1.0 // Default weight
		}
		weighted = append(weighted, weightedResult{result, weight})
	}

	// Sort by weight descending
	sort.Slice(weighted, func(i, j int) bool {
		return weighted[i].weight > weighted[j].weight
	})

	// Return the highest weighted result
	bestResult := weighted[0].response
	if bestResult.Metadata == nil {
		bestResult.Metadata = make(map[string]interface{})
	}
	bestResult.Metadata["selected_weight"] = weighted[0].weight

	return bestResult, nil
}

// QualityScoreCombiner selects based on response quality metrics
type QualityScoreCombiner struct {
	MinLength int
}

// NewQualityScoreCombiner creates a combiner that scores responses by quality
func NewQualityScoreCombiner(minLength int) *QualityScoreCombiner {
	if minLength <= 0 {
		minLength = 50
	}
	return &QualityScoreCombiner{MinLength: minLength}
}

// Combine selects the highest quality response
func (q *QualityScoreCombiner) Combine(ctx context.Context, results []provider.Response) (provider.Response, error) {
	if len(results) == 0 {
		return provider.Response{}, fmt.Errorf("no results to combine")
	}

	type scoredResult struct {
		response provider.Response
		score    float64
	}

	scored := make([]scoredResult, 0, len(results))
	for _, result := range results {
		score := q.scoreResponse(result)
		scored = append(scored, scoredResult{result, score})
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Return the highest scored result
	bestResult := scored[0].response
	if bestResult.Metadata == nil {
		bestResult.Metadata = make(map[string]interface{})
	}
	bestResult.Metadata["quality_score"] = scored[0].score

	return bestResult, nil
}

// scoreResponse calculates a quality score for a response
func (q *QualityScoreCombiner) scoreResponse(resp provider.Response) float64 {
	score := 0.0

	// Length score (up to 100 points)
	length := len(resp.Content)
	if length >= q.MinLength {
		score += 50.0
		// Additional points for substantial content
		if length >= q.MinLength*2 {
			score += 25.0
		}
		if length >= q.MinLength*4 {
			score += 25.0
		}
	} else {
		// Partial credit for short responses
		score += float64(length) / float64(q.MinLength) * 50.0
	}

	// Structure score (up to 50 points)
	// Check for paragraphs
	paragraphs := strings.Count(resp.Content, "\n\n") + 1
	if paragraphs > 1 {
		score += 20.0
	}

	// Check for lists or formatting
	if strings.Contains(resp.Content, "\n- ") || strings.Contains(resp.Content, "\n* ") ||
		strings.Contains(resp.Content, "\n1. ") {
		score += 15.0
	}

	// Check for code blocks
	if strings.Contains(resp.Content, "```") {
		score += 15.0
	}

	return score
}

// Helper functions

func getSourceProviders(results []provider.Response) []string {
	providers := make([]string, len(results))
	for i, result := range results {
		providers[i] = result.Provider
	}
	return providers
}
