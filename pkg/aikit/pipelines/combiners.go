package pipelines

import (
	"context"
	"fmt"
	"strings"
)

// BestPickerCombiner uses an AI provider to select the best result
type BestPickerCombiner struct {
	picker Provider
}

// ConcatCombiner simply concatenates all results
type ConcatCombiner struct {
	separator string
}

// JuryCombiner uses multiple AI providers to vote on the best response
type JuryCombiner struct {
	jurors       []Provider
	votingMethod string // "majority", "consensus", "weighted"
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

// JuryCombiner implementation
func NewJuryCombiner(jurors []Provider, votingMethod string) *JuryCombiner {
	if votingMethod == "" {
		votingMethod = "majority"
	}
	return &JuryCombiner{
		jurors:       jurors,
		votingMethod: votingMethod,
	}
}

func (c *JuryCombiner) Combine(ctx context.Context, results []ProviderResponse) (string, error) {
	if len(results) == 0 {
		return "", fmt.Errorf("no results to combine")
	}

	// Format all responses for the jury to review
	var allResponses string
	for i, result := range results {
		allResponses += fmt.Sprintf("\n=== Response %d (%s) ===\n%s\n\n", i+1, result.ProviderName, result.Content)
	}

	// Prepare the voting prompt
	votingPrompt := fmt.Sprintf(`You are a member of an AI jury tasked with evaluating responses to a question or task.

The following responses were provided by different AI systems:

%s

Your task:
1. Carefully review each response
2. Evaluate them based on accuracy, completeness, clarity, and relevance
3. Provide your vote for the BEST response (just the number)
4. Explain your reasoning in 2-3 sentences

Format your answer as:
VOTE: [response number]
REASON: [your explanation]`, allResponses)

	// Collect votes from all jurors
	type Vote struct {
		JurorName string
		Response  int
		Reason    string
	}

	votes := make([]Vote, 0, len(c.jurors))
	for _, juror := range c.jurors {
		voteText, err := juror.GenerateResponse(ctx, votingPrompt)
		if err != nil {
			return "", fmt.Errorf("juror %s failed to vote: %w", juror.Name(), err)
		}

		// Parse the vote
		lines := strings.Split(voteText, "\n")
		var voteNumber int
		var reason string

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToUpper(line), "VOTE:") {
				_, err := fmt.Sscanf(line, "VOTE: %d", &voteNumber)
				if err != nil {
					voteNumber = 0
				}
			} else if strings.HasPrefix(strings.ToUpper(line), "REASON:") {
				reason = strings.TrimPrefix(strings.TrimPrefix(line, "REASON:"), "reason:")
				reason = strings.TrimSpace(reason)
			}
		}

		// If we couldn't parse the vote properly, default to first response
		if voteNumber < 1 || voteNumber > len(results) {
			voteNumber = 1
		}

		votes = append(votes, Vote{
			JurorName: juror.Name(),
			Response:  voteNumber,
			Reason:    reason,
		})
	}

	// Count votes
	counts := make(map[int]int)
	for _, vote := range votes {
		counts[vote.Response]++
	}

	// Determine winner based on voting method
	var winner int
	var winnerCount int
	var deliberation string

	switch c.votingMethod {
	case "consensus":
		// Check if all votes are for the same response
		consensus := true
		firstVote := votes[0].Response
		for _, vote := range votes[1:] {
			if vote.Response != firstVote {
				consensus = false
				break
			}
		}

		if consensus {
			winner = firstVote
			deliberation = "The jury reached consensus."
		} else {
			// If no consensus, pick the response with most votes
			for response, count := range counts {
				if count > winnerCount {
					winner = response
					winnerCount = count
				}
			}
			deliberation = fmt.Sprintf("The jury failed to reach consensus. The response with the most votes (%d/%d) was selected.", winnerCount, len(votes))
		}

	case "weighted":
		// In a real implementation, weights would be assigned to jurors
		// For now, this is the same as majority
		fallthrough

	default: // majority
		for response, count := range counts {
			if count > winnerCount {
				winner = response
				winnerCount = count
			}
		}
		deliberation = fmt.Sprintf("The jury selected the response with the most votes (%d/%d).", winnerCount, len(votes))
	}

	// Format the final result
	var finalResult string
	finalResult += fmt.Sprintf("=== Jury Deliberation ===\n%s\n\n", deliberation)

	finalResult += "=== Votes ===\n"
	for _, vote := range votes {
		finalResult += fmt.Sprintf("%s voted for Response %d: %s\n", vote.JurorName, vote.Response, vote.Reason)
	}

	finalResult += fmt.Sprintf("\n=== Winning Response (%s) ===\n%s\n",
		results[winner-1].ProviderName, results[winner-1].Content)

	return finalResult, nil
}
