package ui

import (
	"strings"
	"unicode"
)

// removeDuplicates detects and removes duplicated text in a string
func removeDuplicates(s string) string {
	// If the string is too short, no duplicates possible
	if len(s) < 4 {
		return s
	}

	// Try to find duplicated substrings (at least 4 chars long)
	// by comparing the string with itself offset by different amounts
	for offset := 4; offset < len(s)/2; offset++ {
		for i := 0; i < len(s)-offset; i++ {
			// Extract substring of length offset
			sub := s[i : i+offset]

			// See if this substring appears multiple times
			if strings.Count(s, sub) > 1 {
				// Find first and second occurrences
				first := strings.Index(s, sub)
				second := strings.Index(s[first+1:], sub) + first + 1

				// If they're adjacent, remove the duplication
				if second == first+offset {
					return s[:first+offset] + s[second+offset:]
				}
			}
		}
	}

	return s
}

// cleanResponse removes UI text and known corrupted patterns
func cleanResponse(response string) string {
	// List of strings that should be removed from responses
	removeStrings := []string{
		"(ctrl+c/ctrl+d to quit, ctrl+l to clear history)",
		"Loading response",
		"Streaming response",
		"AI:",
	}

	for _, s := range removeStrings {
		response = strings.ReplaceAll(response, s, "")
	}

	// Remove spinner characters
	spinnerChars := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	for _, char := range spinnerChars {
		response = strings.ReplaceAll(response, string(char), "")
	}

	// Remove non-printable characters
	var b strings.Builder
	for _, r := range response {
		if unicode.IsPrint(r) {
			b.WriteRune(r)
		}
	}
	response = b.String()

	// Remove duplicate words that appear consecutively
	response = removeDuplicates(response)

	// Fix code blocks and preserve line breaks
	// This step is now handled by FormatMarkdown

	// Trim extra whitespace
	response = strings.TrimSpace(response)

	return response
}
