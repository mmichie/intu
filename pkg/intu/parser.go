package intu

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

// ParseTaggedContent extracts content between specified XML-like tags
func ParseTaggedContent(input, tag string) (string, error) {
	log.Printf("Debug: Parsing content for tag: %s", tag)
	log.Printf("Debug: Input: %s", input)

	// Look for the tags within the input, ignoring any text before or after
	pattern := fmt.Sprintf(`(?s)<%s>\n?(.*?)\n?</%s>`, tag, tag)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) == 2 {
		log.Printf("Debug: Match found")
		return strings.TrimSpace(matches[1]), nil
	}

	// If we haven't found anything, return an error
	log.Printf("Debug: No match found")
	return "", fmt.Errorf("no content found between <%s> tags", tag)
}

// ParseCommitMessage extracts the commit message from the AI response
func ParseCommitMessage(input string) (string, error) {
	return ParseTaggedContent(input, "commit_message")
}

// ParseReviewComments extracts the review comments from the AI response
func ParseReviewComments(input string) (string, error) {
	return ParseTaggedContent(input, "review_comments")
}
