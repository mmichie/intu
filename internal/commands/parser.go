package commands

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func ParseTaggedContent(input, tag string) (string, error) {
	pattern := fmt.Sprintf(`(?s)<\s*%s\s*>\s*(.*?)\s*<\s*/%s\s*>`, tag, tag)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1]), nil
	}
	return "", fmt.Errorf("no content found between <%s> tags", tag)
}

func ParseCommitMessage(input string) (string, error) {
	return ParseTaggedContent(input, "commit_message")
}

func ParseReviewComments(input string) (string, error) {
	return ParseTaggedContent(input, "review_comments")
}

func ParseSecurityReview(input string) (string, error) {
	return ParseTaggedContent(input, "security_review")
}

func readFileContent(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
