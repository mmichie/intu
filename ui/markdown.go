package ui

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// NewMarkdownFormatter creates a configured Markdown formatter
var markdownParser goldmark.Markdown
var glamourRenderer *glamour.TermRenderer

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

	// Initialize Glamour renderer with the default dark style
	var err error

	// Create a custom style with explicit styling options
	customStyleJSON := `{
		"document": {},
		"block_quote": {
			"indent": 2,
			"margin": 1
		},
		"paragraph": {},
		"list": {
			"level_indent": 2
		},
		"heading": {
			"block_suffix": "\n",
			"level_1": {
				"prefix": "# ",
				"bold": true,
				"underline": true
			},
			"level_2": {
				"prefix": "## ",
				"bold": true
			},
			"level_3": {
				"prefix": "### ",
				"bold": true
			},
			"level_4": {
				"prefix": "#### ",
				"bold": true
			},
			"level_5": {
				"prefix": "##### ",
				"bold": true
			},
			"level_6": {
				"prefix": "###### ",
				"bold": true
			}
		},
		"code_block": {
			"margin": 0,
			"background_color": "#2f2f2f",
			"border_style": "rounded",
			"indent": 0,
			"wrap": false,
			"block_prefix": "\n",
			"block_suffix": "\n"
		},
		"horizontal_rule": {},
		"table": {},
		"link": {}
	}`

	// Use a combination of custom styles and standard options
	glamourRenderer, err = glamour.NewTermRenderer(
		glamour.WithWordWrap(120),
		glamour.WithEmoji(),
		glamour.WithPreservedNewLines(),
		glamour.WithStylesFromJSONBytes([]byte(customStyleJSON)),
		glamour.WithChromaFormatter("terminal256"),
	)
	if err != nil {
		// Fall back to defaults if glamour initialization fails
		fmt.Printf("Warning: Failed to initialize Glamour renderer: %v\n", err)
	}
}

// FormatMarkdown formats the text as Markdown
// It uses Glamour for beautiful terminal rendering if available
// Falls back to basic formatting if Glamour is unavailable
func FormatMarkdown(text string) string {
	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// Fix potential code block formatting issues from LLMs before rendering
	text = fixCodeBlockSpacing(text)

	// Use Glamour for rendering if available
	if glamourRenderer != nil {
		// Attempt to render with Glamour
		rendered, err := glamourRenderer.Render(text)
		if err == nil {
			// Glamour successfully rendered the markdown
			return rendered
		}
		// Log the error but continue with the fallback
		fmt.Printf("Warning: Glamour rendering failed: %v\n", err)
	}

	// Fallback to our manual formatting

	// Special handling for code blocks to preserve formatting
	text = FormatCodeBlocks(text)

	// Special handling for line breaks in lists
	text = FormatListItems(text)

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
			// Get the lines in this code block
			codeLines := strings.Split(parts[i], "\n")

			// Check if this is an empty code block
			if len(codeLines) == 0 || (len(codeLines) == 1 && codeLines[0] == "") {
				// Empty code block - just leave it blank
				parts[i] = ""
				continue
			}

			// Handle the language identifier and code content
			var formattedCode strings.Builder

			// First line might be a language identifier or code
			firstLine := strings.TrimSpace(codeLines[0])

			// Determine if the first line is a language identifier
			isLangIdentifier := false
			if firstLine != "" && !strings.Contains(firstLine, " ") && len(firstLine) < 20 {
				// Simple heuristic: language identifiers are usually single words less than 20 chars
				// Common languages: python, javascript, go, java, etc.
				isLangIdentifier = true
			}

			// Add the language identifier or first line
			formattedCode.WriteString(codeLines[0])
			formattedCode.WriteString("\n")

			// Process the rest of the code
			startIdx := 1
			if !isLangIdentifier && len(codeLines) > 1 && codeLines[1] == "" {
				// Skip the empty line after first line if it's not a language identifier
				startIdx = 2
			}

			// Process remaining lines
			for j := startIdx; j < len(codeLines); j++ {
				line := codeLines[j]

				// Add the line with proper formatting
				formattedCode.WriteString(line)

				// Add newline if not the last line
				if j < len(codeLines)-1 {
					formattedCode.WriteString("\n")
				} else if len(strings.TrimSpace(line)) > 0 {
					// Add final newline if the last line isn't empty
					formattedCode.WriteString("\n")
				}
			}

			// Update the part with formatted code
			parts[i] = formattedCode.String()
		}
	}

	// Reconstruct the text
	var result strings.Builder
	result.WriteString(parts[0])

	for i := 1; i < len(parts); i++ {
		if i%2 == 1 {
			// Code block starts
			result.WriteString("```")
			result.WriteString(parts[i])

			// If the code block doesn't end with a newline, add one
			if len(parts[i]) > 0 && !strings.HasSuffix(parts[i], "\n") {
				result.WriteString("\n")
			}
		} else {
			// Code block ends - ensure proper spacing after
			result.WriteString("```")

			// Add double newline after code blocks for separation
			// unless it's the last part and it's empty
			if i < len(parts)-1 || len(strings.TrimSpace(parts[i])) > 0 {
				if !strings.HasPrefix(parts[i], "\n") {
					result.WriteString("\n")
				}
				if !strings.HasPrefix(parts[i], "\n\n") && !strings.HasPrefix(parts[i], "\n \n") {
					result.WriteString("\n")
				}
				result.WriteString(parts[i])
			} else {
				result.WriteString(parts[i])
			}
		}
	}

	return result.String()
}

// This function is no longer used, as we preserve original formatting in code blocks
// Left here for reference in case we want to re-enable automatic code formatting
func ensureCodeLineBreaks(code string) string {
	// We currently preserve original code formatting
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

// cleanResponseArtifacts cleans up common response artifacts from API responses
func cleanResponseArtifacts(text string) string {
	// Remove common LLM response markers
	text = strings.TrimPrefix(text, "```response")
	text = strings.TrimPrefix(text, "```answer")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")

	// Clean up any "Answer: " prefixes that some models add
	text = strings.TrimPrefix(text, "Answer: ")

	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")

	return text
}

// fixCodeBlockSpacing fixes spacing around code blocks and headers for better rendering
// This is specifically designed to handle common LLM output issues
func fixCodeBlockSpacing(text string) string {
	// If there are no code blocks or headers, return the original text
	if !strings.Contains(text, "```") && !strings.Contains(text, "#") {
		return text
	}

	lines := strings.Split(text, "\n")
	var result strings.Builder
	inCodeBlock := false

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Handle code block boundaries
		if strings.HasPrefix(trimmedLine, "```") {
			// If starting a code block
			if !inCodeBlock {
				// Add a newline before code block if not already there
				if i > 0 && len(strings.TrimSpace(lines[i-1])) > 0 {
					result.WriteString("\n")
				}

				// Fix various code block issues that LLMs often introduce
				if len(trimmedLine) > 3 && !strings.HasPrefix(trimmedLine, "``` ") {
					langPart := trimmedLine[3:]

					// Handle common language variants like 'lisp(defun' or 'python def'
					// where LLM merges language with first line of code
					if strings.Contains(langPart, "(") && !strings.Contains(langPart, " ") {
						parts := strings.SplitN(langPart, "(", 2)
						if len(parts) == 2 {
							result.WriteString("```")
							result.WriteString(parts[0]) // language part
							result.WriteString("\n(")
							result.WriteString(parts[1]) // code part
							// Skip the normal newline since we've already added it
							continue
						}
					}

					// Fix cases like ```pythonx = 5 by extracting common language names
					for _, lang := range []string{"python", "javascript", "go", "java", "lisp", "c", "cpp", "rust", "typescript", "ruby", "php"} {
						if strings.HasPrefix(strings.ToLower(langPart), lang) && len(langPart) > len(lang) {
							// Split into language and code
							codePart := langPart[len(lang):]
							result.WriteString("```")
							result.WriteString(lang)
							result.WriteString("\n")
							result.WriteString(codePart)
							result.WriteString("\n")
							inCodeBlock = true
							continue
						}
					}

					// Default case: just preserve the language identifier
					result.WriteString("```")
					result.WriteString(langPart)
				} else {
					result.WriteString(line)
				}

				result.WriteString("\n")
				inCodeBlock = true
			} else {
				// Ending a code block
				result.WriteString(line)
				result.WriteString("\n")
				// Add a newline after code block if next line isn't empty
				if i < len(lines)-1 && len(strings.TrimSpace(lines[i+1])) > 0 {
					result.WriteString("\n")
				}
				inCodeBlock = false
			}
			continue
		}

		// Check for header lines that might be malformed (no space after #)
		if !inCodeBlock && strings.HasPrefix(trimmedLine, "#") {
			// Fix headers like ###Header to ### Header
			headerPrefix := ""
			headerContent := trimmedLine

			// Count the # symbols
			hashCount := 0
			for _, char := range trimmedLine {
				if char == '#' {
					hashCount++
				} else {
					break
				}
			}

			// If we have a header with no space after the #s
			if hashCount > 0 && hashCount <= 6 && len(trimmedLine) > hashCount && trimmedLine[hashCount] != ' ' {
				headerPrefix = trimmedLine[:hashCount] + " "
				headerContent = trimmedLine[hashCount:]

				// Add newline before header if needed
				if i > 0 && len(strings.TrimSpace(lines[i-1])) > 0 {
					result.WriteString("\n")
				}

				result.WriteString(headerPrefix)
				result.WriteString(headerContent)
				result.WriteString("\n")

				// Add newline after header if needed
				if i < len(lines)-1 && len(strings.TrimSpace(lines[i+1])) > 0 {
					result.WriteString("\n")
				}

				continue
			}
		}

		// Inside or outside code block, just add the line
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// preprocessMarkdown does extensive preprocessing on raw LLM outputs
// to ensure proper formatting in the terminal
func preprocessMarkdown(text string) string {
	// First pass: identify sections and ensure proper spacing
	lines := strings.Split(text, "\n")
	var result strings.Builder
	inCodeBlock := false
	prevLineIsHeader := false
	prevLineIsEmpty := false

	// Try to detect paragraphs joined with no spaces
	text = strings.ReplaceAll(text, ".###", ".\n\n###")
	text = strings.ReplaceAll(text, "?###", "?\n\n###")
	text = strings.ReplaceAll(text, "!###", "!\n\n###")
	text = strings.ReplaceAll(text, ".##", ".\n\n##")
	text = strings.ReplaceAll(text, "?##", "?\n\n##")
	text = strings.ReplaceAll(text, "!##", "!\n\n##")
	text = strings.ReplaceAll(text, ".#", ".\n\n#")
	text = strings.ReplaceAll(text, "?#", "?\n\n#")
	text = strings.ReplaceAll(text, "!#", "!\n\n#")

	// Add missing backticks for language identifiers
	// This handles cases where the language identifier appears without code block markers
	for _, lang := range []string{"haskell", "python", "javascript", "go", "java", "lisp", "c", "cpp", "rust", "typescript", "ruby", "php"} {
		// Case 1: Language name on a line by itself
		pattern1 := "\\n" + lang + "\\n"
		replacement1 := "\n```" + lang + "\n"
		regex1 := regexp.MustCompile(pattern1)
		text = regex1.ReplaceAllString(text, replacement1)

		// Case 2: Language immediately followed by code (like "haskell--")
		// Uses a positive lookahead to ensure there's actual code after the language name
		pattern2 := "\\n" + lang + "([^a-zA-Z0-9\\s][^\\n]+)"
		replacement2 := "\n```" + lang + "\n$1"
		regex2 := regexp.MustCompile(pattern2)
		text = regex2.ReplaceAllString(text, replacement2)
	}

	// Make sure all headers have a space after the # symbols
	for i := 6; i >= 1; i-- {
		hashes := strings.Repeat("#", i)
		pattern := "\\n" + hashes + "([^#\\s])"
		replacement := "\n" + hashes + " $1"
		regex := regexp.MustCompile(pattern)
		text = regex.ReplaceAllString(text, replacement)
	}

	// Ensure proper code block spacing
	text = fixCodeBlockDetection(text)

	// Reset our line handling with the fixed text
	lines = strings.Split(text, "\n")

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		isHeader := false
		isCodeBlockMarker := false

		// Detect if this line is a header
		if strings.HasPrefix(trimmedLine, "#") {
			isHeader = true
		}

		// Detect if this line is a code block marker
		if strings.HasPrefix(trimmedLine, "```") {
			isCodeBlockMarker = true
			inCodeBlock = !inCodeBlock
		}

		// Add spacing before headers if needed
		if isHeader && !prevLineIsEmpty && !prevLineIsHeader && i > 0 {
			result.WriteString("\n")
		}

		// Add spacing before code blocks if needed
		if isCodeBlockMarker && !inCodeBlock && !prevLineIsEmpty && i > 0 {
			result.WriteString("\n")
		}

		// Add the line content
		result.WriteString(line)

		// Add a newline after the line (unless it's the last line)
		if i < len(lines)-1 {
			result.WriteString("\n")

			// Add extra newline after headers
			if isHeader {
				result.WriteString("\n")
			}

			// Add extra newline after closing code blocks
			if isCodeBlockMarker && !inCodeBlock {
				result.WriteString("\n")
			}
		}

		// Update state for next iteration
		prevLineIsHeader = isHeader
		prevLineIsEmpty = trimmedLine == ""
	}

	return result.String()
}

// fixCodeBlockDetection tries to fix missing code block markers in LLM responses
func fixCodeBlockDetection(text string) string {
	// Count the number of code block markers
	count := strings.Count(text, "```")

	// If there's an odd number, try to detect where to add a closing marker
	if count%2 == 1 {
		// Find the last opening marker
		lastOpenIdx := strings.LastIndex(text, "```")

		// Look for a likely end of the code block after this
		lines := strings.Split(text[lastOpenIdx:], "\n")
		if len(lines) > 2 {
			// Add a closing marker after a reasonable number of lines
			// or before the next paragraph or heading
			for i := 2; i < len(lines); i++ {
				trimmed := strings.TrimSpace(lines[i])
				if trimmed == "" || strings.HasPrefix(trimmed, "#") {
					// Insert closing marker before this line
					before := text[:lastOpenIdx+len(lines[0])]
					for j := 1; j < i; j++ {
						before += "\n" + lines[j]
					}
					after := "\n```"
					for j := i; j < len(lines); j++ {
						after += "\n" + lines[j]
					}
					text = before + after
					break
				}
			}
		}
	}

	return text
}

// fixMarkdownSpacing ensures proper spacing in markdown responses,
// particularly around code blocks and headings
func fixMarkdownSpacing(text string) string {
	var result strings.Builder
	lines := strings.Split(text, "\n")
	inCodeBlock := false

	for i, line := range lines {
		// Handle code block boundaries
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock

			// If starting a code block, ensure it has proper spacing before it
			if inCodeBlock && i > 0 && len(strings.TrimSpace(lines[i-1])) > 0 && !strings.HasPrefix(strings.TrimSpace(lines[i-1]), "#") {
				result.WriteString("\n")
			}

			result.WriteString(line)
			result.WriteString("\n")

			// If ending a code block, ensure it has proper spacing after it
			if !inCodeBlock && i < len(lines)-1 && len(strings.TrimSpace(lines[i+1])) > 0 {
				result.WriteString("\n")
			}
			continue
		}

		// Inside code blocks, preserve exact formatting
		if inCodeBlock {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Handle headers (ensure space before and after)
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			if i > 0 && len(strings.TrimSpace(lines[i-1])) > 0 && !strings.HasPrefix(strings.TrimSpace(lines[i-1]), "#") {
				result.WriteString("\n")
			}
			result.WriteString(line)
			result.WriteString("\n")
			if i < len(lines)-1 && len(strings.TrimSpace(lines[i+1])) > 0 {
				result.WriteString("\n")
			}
			continue
		}

		// Regular line
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	// Normalize line endings and clean up excessive spacing
	final := result.String()
	final = strings.ReplaceAll(final, "\r\n", "\n")

	// Replace triple newlines with double newlines to avoid excessive spacing
	for strings.Contains(final, "\n\n\n") {
		final = strings.ReplaceAll(final, "\n\n\n", "\n\n")
	}

	return final
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
