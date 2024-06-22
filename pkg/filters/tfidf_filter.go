package filters

import (
	"math"
	"sort"
	"strings"
)

type TFIDFFilter struct{}

func (f *TFIDFFilter) Process(content string) string {
	words := strings.Fields(strings.ToLower(content))
	if len(words) == 0 {
		return content
	}

	totalTerms := len(words)
	termFrequency := calculateFrequency(words)
	tfidfScores := make(map[string]float64)
	lowestScore := math.MaxFloat64

	for word, count := range termFrequency {
		tf := float64(count) / float64(totalTerms)
		idf := math.Log10(1.0 + float64(totalTerms)/float64(count))
		tfidfScores[word] = tf * idf
		if tfidfScores[word] < lowestScore {
			lowestScore = tfidfScores[word]
		}
	}

	threshold := lowestScore + (lowestScore * 0.5)
	compressedWords := filterAndSortByScore(tfidfScores, threshold)

	return strings.Join(compressedWords, " ")
}

func (f *TFIDFFilter) Name() string {
	return "tfidf"
}

func calculateFrequency(words []string) map[string]int {
	freq := make(map[string]int)
	for _, word := range words {
		freq[word]++
	}
	return freq
}

func filterAndSortByScore(scores map[string]float64, threshold float64) []string {
	words := []string{}
	for word, score := range scores {
		if score >= threshold {
			words = append(words, word)
		}
	}
	sort.Strings(words)
	return words
}

func init() {
	Register(&TFIDFFilter{})
}
