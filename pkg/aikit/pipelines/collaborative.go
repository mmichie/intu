package pipelines

import (
	"context"
	"fmt"
	"strings"
)

// CollaborativePipeline allows providers to communicate and build on each other's responses
type CollaborativePipeline struct {
	providers []Provider
	rounds    int
}

// Discussion represents a multi-turn conversation between providers
type Discussion struct {
	Rounds   []Round
	Question string
}

// Round represents one round of the discussion
type Round struct {
	Number    int
	Responses []ProviderResponse
}

func NewCollaborativePipeline(providers []Provider, rounds int) *CollaborativePipeline {
	if rounds < 1 {
		rounds = 1
	}
	return &CollaborativePipeline{
		providers: providers,
		rounds:    rounds,
	}
}

func (p *CollaborativePipeline) Execute(ctx context.Context, input string) (string, error) {
	if len(p.providers) == 0 {
		return "", fmt.Errorf("collaborative pipeline needs at least one provider")
	}

	// Initialize the discussion
	discussion := Discussion{
		Question: input,
		Rounds:   make([]Round, 0, p.rounds),
	}

	// First round: get initial responses from all providers
	initialPrompt := fmt.Sprintf(`You are participating in a collaborative discussion to solve the following problem:

%s

Provide your initial thoughts and approach to this problem. Be clear and concise.`, input)

	firstRound := Round{
		Number:    1,
		Responses: make([]ProviderResponse, len(p.providers)),
	}

	for i, provider := range p.providers {
		response, err := provider.GenerateResponse(ctx, initialPrompt)
		if err != nil {
			return "", fmt.Errorf("provider %s failed in round 1: %w", provider.Name(), err)
		}
		firstRound.Responses[i] = ProviderResponse{
			ProviderName: provider.Name(),
			Content:      response,
		}
	}
	discussion.Rounds = append(discussion.Rounds, firstRound)

	// Additional rounds: each provider builds on previous round
	for r := 2; r <= p.rounds; r++ {
		currentRound := Round{
			Number:    r,
			Responses: make([]ProviderResponse, len(p.providers)),
		}

		for i, provider := range p.providers {
			// Format the conversation history
			var conversationHistory string
			for _, prevRound := range discussion.Rounds {
				conversationHistory += fmt.Sprintf("--- Round %d ---\n", prevRound.Number)
				for _, resp := range prevRound.Responses {
					conversationHistory += fmt.Sprintf("%s: %s\n\n", resp.ProviderName, resp.Content)
				}
			}

			// Build collaborative prompt
			collaborativePrompt := fmt.Sprintf(`You are participating in a collaborative discussion to solve the following problem:

Question: %s

Here is the conversation so far:

%s

You are %s. It's now round %d of the discussion.

Review the previous contributions and build on them. You can:
1. Add new insights
2. Improve existing ideas
3. Address gaps or limitations
4. Suggest a synthesis of the best ideas

Be constructive and focus on advancing the solution.`, input, conversationHistory, provider.Name(), r)

			response, err := provider.GenerateResponse(ctx, collaborativePrompt)
			if err != nil {
				return "", fmt.Errorf("provider %s failed in round %d: %w", provider.Name(), r, err)
			}
			currentRound.Responses[i] = ProviderResponse{
				ProviderName: provider.Name(),
				Content:      response,
			}
		}
		discussion.Rounds = append(discussion.Rounds, currentRound)
	}

	// Format the final result
	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Collaborative Discussion: %s ===\n\n", truncateQuestion(input)))

	for _, round := range discussion.Rounds {
		result.WriteString(fmt.Sprintf("--- Round %d ---\n", round.Number))
		for _, response := range round.Responses {
			result.WriteString(fmt.Sprintf("%s: %s\n\n", response.ProviderName, response.Content))
		}
	}

	// Final synthesis in a real implementation would use a provider to summarize the discussion

	return result.String(), nil
}

// Helper function to truncate a long question for display
func truncateQuestion(question string) string {
	if len(question) <= 50 {
		return question
	}
	return question[:47] + "..."
}
