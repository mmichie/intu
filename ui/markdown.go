package ui

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// NewMarkdownFormatter creates a configured Markdown formatter
var markdownParser goldmark.Markdown

func init() {
	// Initialize the markdown parser with desired extensions
	markdownParser = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,            // GitHub flavored markdown
			extension.DefinitionList, // Definition lists
			extension.Footnote,       // Footnotes
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow raw HTML
		),
	)
}

// FormatMarkdown formats the text as Markdown
// It processes the Markdown to ensure proper handling of line breaks
// This does not convert to HTML, but ensures proper line breaks and formatting
func FormatMarkdown(text string) string {
	// Special handling for code blocks to preserve formatting
	text = FormatCodeBlocks(text)

	// Special handling for line breaks in lists
	text = FormatListItems(text)

	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// Replace triple newlines with double newlines to avoid excessive spacing
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return text
}

// FormatCodeBlocks ensures code blocks have proper formatting and line breaks
func FormatCodeBlocks(text string) string {
	// If there are no code blocks, return the original text
	if !strings.Contains(text, "```") {
		return text
	}

	// Split the text by code block markers
	parts := strings.Split(text, "```")

	// Process each part
	for i := 1; i < len(parts); i += 2 {
		if i < len(parts) { // Inside a code block
			// Ensure the code block has proper line breaks
			codeLines := strings.Split(parts[i], "\n")

			// Language indicator is on the first line, preserve it
			if len(codeLines) > 0 {
				language := strings.TrimSpace(codeLines[0])

				// Ensure proper line breaks in the code content
				if len(codeLines) > 1 {
					codeContent := strings.Join(codeLines[1:], "\n")

					// Format the code content
					formattedCode := ensureCodeLineBreaks(codeContent)

					// Reconstruct the code block
					parts[i] = language + "\n" + formattedCode
				}
			}
		}
	}

	// Reconstruct the text
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if i%2 == 1 {
			// Code block starts
			result += "```" + parts[i]
		} else {
			// Code block ends
			if i < len(parts)-1 {
				result += "```\n\n" + parts[i]
			} else {
				result += "```" + parts[i]
			}
		}
	}

	return result
}

// ensureCodeLineBreaks ensures code blocks have appropriate line breaks
func ensureCodeLineBreaks(code string) string {
	// Look for Python-style line continuations
	code = strings.ReplaceAll(code, ":", ":\n")

	// Ensure line breaks after common Python statement starters
	statements := []string{"def ", "class ", "if ", "elif ", "else:", "for ", "while ", "try:", "except ", "finally:", "with "}
	for _, stmt := range statements {
		code = strings.ReplaceAll(code, stmt, "\n"+stmt)
	}

	// Fix multiple consecutive newlines
	for strings.Contains(code, "\n\n\n") {
		code = strings.ReplaceAll(code, "\n\n\n", "\n\n")
	}

	return code
}

// FormatListItems ensures list items have proper line breaks
func FormatListItems(text string) string {
	// Handle bullet lists (-, *, +)
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		// Check if this line is a list item
		if i > 0 && (strings.HasPrefix(strings.TrimSpace(lines[i]), "- ") ||
			strings.HasPrefix(strings.TrimSpace(lines[i]), "* ") ||
			strings.HasPrefix(strings.TrimSpace(lines[i]), "+ ")) {

			// If previous line isn't empty and isn't a list item, add a newline
			prevLine := strings.TrimSpace(lines[i-1])
			if prevLine != "" &&
				!strings.HasPrefix(prevLine, "- ") &&
				!strings.HasPrefix(prevLine, "* ") &&
				!strings.HasPrefix(prevLine, "+ ") {
				lines[i-1] = lines[i-1] + "\n"
			}
		}
	}

	return strings.Join(lines, "\n")
}

// MarkdownToTerminal converts Markdown to a terminal-friendly format
// This is a more advanced function that could be used for pretty-printing
// Markdown in a terminal, but is not used in the current implementation
func MarkdownToTerminal(source string) string {
	var buf bytes.Buffer
	if err := markdownParser.Convert([]byte(source), &buf); err != nil {
		// If there's an error, return the source text unchanged
		fmt.Printf("Error converting Markdown: %v\n", err)
		return source
	}

	// Convert the HTML output to terminal-friendly text
	// This is a simplified conversion and may need improvement
	result := buf.String()

	// Remove HTML tags
	result = strings.ReplaceAll(result, "<p>", "")
	result = strings.ReplaceAll(result, "</p>", "\n\n")
	result = strings.ReplaceAll(result, "<h1>", "\n")
	result = strings.ReplaceAll(result, "</h1>", "\n\n")
	result = strings.ReplaceAll(result, "<h2>", "\n")
	result = strings.ReplaceAll(result, "</h2>", "\n\n")
	result = strings.ReplaceAll(result, "<h3>", "\n")
	result = strings.ReplaceAll(result, "</h3>", "\n\n")
	result = strings.ReplaceAll(result, "<pre><code>", "\n")
	result = strings.ReplaceAll(result, "</code></pre>", "\n")
	result = strings.ReplaceAll(result, "<code>", "")
	result = strings.ReplaceAll(result, "</code>", "")
	result = strings.ReplaceAll(result, "<strong>", "")
	result = strings.ReplaceAll(result, "</strong>", "")
	result = strings.ReplaceAll(result, "<em>", "")
	result = strings.ReplaceAll(result, "</em>", "")

	// Clean up extra newlines
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(result)
}
