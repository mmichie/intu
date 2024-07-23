package intu

import (
	"fmt"
	"regexp"
	"strings"
)

// ParseTaggedContent extracts content between specified XML-like tags
func ParseTaggedContent(input, tag string) (string, error) {
	// Look for the tags within the input, ignoring any text before or after
	pattern := fmt.Sprintf(`(?s)<\s*%s\s*>\s*(.*?)\s*<\s*/%s\s*>`, tag, tag)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1]), nil
	}

	// If we haven't found anything, return an error
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

// ParseSecurityReview extracts the security review from the AI response
func ParseSecurityReview(input string) (string, error) {
	return ParseTaggedContent(input, "security_review")
}
