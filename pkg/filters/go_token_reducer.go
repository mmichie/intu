package filters

import (
	"strings"
	"unicode"
)

type GoCodeCompressFilter struct{}

func (f *GoCodeCompressFilter) Process(content string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip comments and empty lines to reduce tokens
		if strings.HasPrefix(trimmed, "//") || trimmed == "" {
			continue
		}

		// Specific transformations for Go syntax
		trimmed = transformGoSyntax(trimmed)

		// Write the processed line with a newline
		result.WriteString(trimmed + "\n")
	}

	return result.String()
}

func (f *GoCodeCompressFilter) Name() string {
	return "goCodeCompress"
}

// transformGoSyntax applies Go-specific syntax transformations
func transformGoSyntax(code string) string {
	code = strings.ReplaceAll(code, " := ", ":=")
	code = strings.ReplaceAll(code, " = ", "=")
	code = strings.ReplaceAll(code, " + ", "+")
	code = strings.ReplaceAll(code, " - ", "-")
	code = strings.ReplaceAll(code, "{ ", "{")
	code = strings.ReplaceAll(code, " }", "}")

	return compressSpaces(code)
}

// compressSpaces removes unnecessary spaces next to operators and commas
func compressSpaces(code string) string {
	var result strings.Builder
	var last rune

	for _, c := range code {
		if unicode.IsSpace(c) && (last == ',' || last == '+' || last == '-' || last == '*' || last == '/' || last == '=' || last == '{' || last == '}') {
			continue
		}
		if !unicode.IsSpace(c) {
			result.WriteRune(c)
		}
		last = c
	}

	return result.String()
}

func init() {
	Register(&GoCodeCompressFilter{})
}
