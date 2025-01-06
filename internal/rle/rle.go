package rle

import (
	"fmt"
	"sort"
	"strings"
)

// FileOutput holds RLE output for a single file.
type FileOutput struct {
	RelativePath string `json:"relative_path"`
	RLEContent   string `json:"rle_content"`
}

// BatchOutput aggregates RLE info for multiple files.
type BatchOutput struct {
	Format      string            `json:"format"`
	Example     string            `json:"example"`
	Description string            `json:"description"`
	Tokens      map[string]string `json:"tokens"`
	Files       []FileOutput      `json:"files"`
}

// SubstringPattern holds a repeated substring plus its frequency and compression score.
type SubstringPattern struct {
	Text  string // the repeated substring
	Count int    // how many times it appears
	Score int    // cost-benefit metric, e.g. (Count * len(Text)) - overhead
}

// FindGlobalPatterns scans the concatenated text for repeated substrings within [minLen..maxLen],
// computing a frequency map, scoring them, then choosing the top patterns while skipping those
// contained in previously chosen patterns. This is a naive demonstration that can be optimized
// with advanced data structures (suffix array, suffix automaton, etc.).
func FindGlobalPatterns(allTexts []string, minLen, maxLen int) []SubstringPattern {
	// 1. Concatenate all text into one big chunk (optional).
	joined := strings.Join(allTexts, "\n\n")
	n := len(joined)
	if n == 0 || minLen > maxLen {
		return nil
	}

	// 2. Gather frequencies of all substrings of length L in [maxLen..minLen].
	//    This naive approach can be slow for large data; consider suffix arrays for production.
	freqMap := make(map[string]int)
	for L := maxLen; L >= minLen; L-- {
		if L > n {
			continue
		}
		for start := 0; start+L <= n; start++ {
			sub := joined[start : start+L]
			freqMap[sub]++
		}
	}

	// 3. Build candidate patterns with a minimal threshold (repeats >= 2).
	//    Score = freq*len(sub) - overhead.
	//    Overhead can represent the token map in JSON, etc.
	var candidates []SubstringPattern
	for sub, count := range freqMap {
		if count < 2 {
			continue
		}
		length := len(sub)
		// Example overhead: 8 + substring length (just a rough guess).
		overhead := 8 + length
		score := count*length - overhead
		if score > 0 {
			candidates = append(candidates, SubstringPattern{
				Text:  sub,
				Count: count,
				Score: score,
			})
		}
	}

	// 4. Sort candidates by descending score, tie-break on longer substring first.
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return len(candidates[i].Text) > len(candidates[j].Text)
		}
		return candidates[i].Score > candidates[j].Score
	})

	// 5. Pick top patterns, skipping any entirely contained in a previously picked pattern.
	var final []SubstringPattern
	maxTokens := 25 // user preference for the number of tokens
	for _, cand := range candidates {
		if len(final) >= maxTokens {
			break
		}
		contained := false
		for _, chosen := range final {
			if strings.Contains(chosen.Text, cand.Text) {
				contained = true
				break
			}
		}
		if !contained {
			final = append(final, cand)
		}
	}
	return final
}

// ReplaceSubstrings performs boundary-agnostic replacements of each chosen substring
// in the file text. Each pattern is replaced with "@N" where N is the pattern index.
func ReplaceSubstrings(original string, patterns []SubstringPattern) (string, map[string]string) {
	compressed := original
	tokenMap := make(map[string]string)

	// patterns are expected to be sorted biggest-first, so they don't overshadow each other
	for i, pat := range patterns {
		tokenStr := fmt.Sprintf("@%d", i)
		tokenMap[tokenStr] = pat.Text
		compressed = strings.ReplaceAll(compressed, pat.Text, tokenStr)
	}
	return compressed, tokenMap
}

// DoRLE compresses runs of the same character. Repeated characters above minRLECount
// become "5x" style notations.
func DoRLE(input string, minRLECount int) string {
	var result strings.Builder
	var currentChar rune
	var count int

	for _, r := range input {
		if r == currentChar {
			count++
		} else {
			if count > 0 {
				flushRepeats(&result, currentChar, count, minRLECount)
			}
			currentChar = r
			count = 1
		}
	}
	if count > 0 {
		flushRepeats(&result, currentChar, count, minRLECount)
	}
	return result.String()
}

// flushRepeats writes either "Nchar" or raw repeated chars depending on the threshold.
func flushRepeats(out *strings.Builder, char rune, count, minRLECount int) {
	if count > minRLECount {
		out.WriteString(fmt.Sprintf("%d%c", count, char))
	} else {
		for i := 0; i < count; i++ {
			out.WriteRune(char)
		}
	}
}

// NewBatchOutput packages final RLE results and tokens for multi-file usage in JSON.
func NewBatchOutput(files []FileOutput, tokens map[string]string) BatchOutput {
	return BatchOutput{
		Format:  "smart-code-rle",
		Example: "Common patterns are tokenized (@N), runs are compressed (3x)",
		Description: `This content is compressed using smart code RLE with global pattern analysis.
To decode:
1. Replace all tokens (@N) with their definitions from the tokens map
2. Expand number+character sequences (e.g., "3x" -> "xxx")
3. Process remaining content as-is`,
		Tokens: tokens,
		Files:  files,
	}
}
