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

	// Special case: exact doubled string (e.g., "Hello! Hello!")
	halfLen := len(s) / 2
	if halfLen > 3 && s[:halfLen] == s[halfLen:] {
		return s[:halfLen]
	}

	// Check for repeated words/phrases
	words := strings.Fields(s)
	if len(words) > 3 {
		var result []string
		for i := 0; i < len(words); i++ {
			// Skip if this word repeats the previous one
			if i > 0 && words[i] == words[i-1] {
				continue
			}

			// Check for repeated phrases (2-4 words)
			skip := false
			for phraseLen := 2; phraseLen <= 4 && i+phraseLen <= len(words); phraseLen++ {
				if i >= phraseLen {
					prevPhrase := strings.Join(words[i-phraseLen:i], " ")
					currPhrase := strings.Join(words[i:i+phraseLen], " ")
					if prevPhrase == currPhrase {
						skip = true
						break
					}
				}
			}

			if !skip {
				result = append(result, words[i])
			}
		}

		// If we removed duplicates, return the cleaned version
		if len(result) < len(words) {
			return strings.Join(result, " ")
		}
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

				// If they're adjacent or very close, remove the duplication
				if second <= first+offset+5 {
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

	// Fix word spacing issues - when words are run together without spaces
	// This often happens in streaming responses
	response = fixWordSpacing(response)

	// Remove duplicate words that appear consecutively
	response = removeDuplicates(response)

	// Trim extra whitespace
	response = strings.TrimSpace(response)

	return response
}

// fixWordSpacing adds spaces between words that are incorrectly joined together
// This is common in streaming responses where words get concatenated
func fixWordSpacing(text string) string {
	// Don't fix code blocks
	if strings.Contains(text, "```") {
		parts := strings.Split(text, "```")
		for i := 0; i < len(parts); i++ {
			// Only fix the non-code parts (even indices)
			if i%2 == 0 {
				parts[i] = fixWordSpacingInText(parts[i])
			}
		}
		return strings.Join(parts, "```")
	}

	return fixWordSpacingInText(text)
}

// fixWordSpacingInText handles the actual fixing of word spacing in text (not code)
func fixWordSpacingInText(text string) string {
	// Pattern to detect: lowercase followed by uppercase letter (indicates missing space)
	// Example: "HelloWorld" -> "Hello World"
	var result strings.Builder
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		result.WriteRune(runes[i])

		// Check if we need to add a space
		if i < len(runes)-1 &&
			unicode.IsLower(runes[i]) &&
			unicode.IsUpper(runes[i+1]) {
			// Don't add spaces in known patterns that shouldn't be split:
			// Common acronyms, product names, etc.
			skipPatterns := []string{"AI", "UI", "API", "JSON", "HTML", "CSS", "IoT"}
			shouldSkip := false

			// Look ahead to see if we're in a skip pattern
			for _, pattern := range skipPatterns {
				if i+len(pattern) <= len(runes) {
					potentialMatch := string(runes[i+1 : i+len(pattern)+1])
					if potentialMatch == pattern {
						shouldSkip = true
						break
					}
				}
			}

			if !shouldSkip {
				result.WriteRune(' ')
			}
		}
	}

	return result.String()
}
