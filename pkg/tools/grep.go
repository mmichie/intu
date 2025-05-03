package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepParams defines the parameters for the Grep tool
type GrepParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
	Include string `json:"include,omitempty"`
}

// GrepMatch represents a single match from the Grep tool
type GrepMatch struct {
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	MatchedLine string `json:"matched_line"`
	Context     string `json:"context,omitempty"`
}

// GrepTool implements the Grep command
type GrepTool struct {
	BaseTool
}

// NewGrepTool creates a new Grep tool
func NewGrepTool() *GrepTool {
	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The regular expression pattern to search for in file contents",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The directory to search in. Defaults to the current working directory.",
			},
			"include": map[string]interface{}{
				"type":        "string",
				"description": "File pattern to include in the search (e.g. \"*.js\", \"*.{ts,tsx}\")",
			},
		},
		"required": []string{"pattern"},
	}

	return &GrepTool{
		BaseTool: BaseTool{
			ToolName:        "Grep",
			ToolDescription: "Searches file contents using regular expressions",
			ToolParams:      paramSchema,
			PermLevel:       PermissionReadOnly,
		},
	}
}

// Execute runs the Grep tool
func (t *GrepTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p GrepParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Check required parameters
	if p.Pattern == "" {
		return nil, fmt.Errorf("pattern parameter is required")
	}

	// Compile regex
	pattern, err := regexp.Compile(p.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Determine search path
	searchPath := "."
	if p.Path != "" {
		searchPath = p.Path
	}

	// Get absolute path
	absPath, err := filepath.Abs(searchPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Determine file pattern to include
	filePatterns := []string{"*"}
	if p.Include != "" {
		// Handle patterns like *.{js,ts}
		if strings.Contains(p.Include, "{") && strings.Contains(p.Include, "}") {
			// Extract options from braces
			start := strings.Index(p.Include, "{")
			end := strings.Index(p.Include, "}")
			if start >= 0 && end > start {
				prefix := p.Include[:start]
				suffix := p.Include[end+1:]
				options := strings.Split(p.Include[start+1:end], ",")
				for _, opt := range options {
					filePatterns = append(filePatterns, prefix+opt+suffix)
				}
			}
		} else {
			filePatterns = []string{p.Include}
		}
	}

	matches := []GrepMatch{}

	// Walk the directory and process files
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file matches any pattern
		fileMatched := false
		for _, fp := range filePatterns {
			matched, err := filepath.Match(fp, filepath.Base(path))
			if err == nil && matched {
				fileMatched = true
				break
			}
		}

		if !fileMatched {
			return nil
		}

		// Skip binary files
		if isBinaryFile(path) {
			return nil
		}

		// Read file contents
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if pattern.MatchString(line) {
				// Create match with context (3 lines before and after)
				contextLines := []string{}

				// Add lines before
				start := max(0, i-3)
				for j := start; j < i; j++ {
					contextLines = append(contextLines, fmt.Sprintf("%d: %s", j+1, lines[j]))
				}

				// Add the matched line
				contextLines = append(contextLines, fmt.Sprintf("%d: %s", i+1, line))

				// Add lines after
				end := min(len(lines), i+4)
				for j := i + 1; j < end; j++ {
					contextLines = append(contextLines, fmt.Sprintf("%d: %s", j+1, lines[j]))
				}

				matches = append(matches, GrepMatch{
					FilePath:    path,
					LineNumber:  i + 1,
					MatchedLine: line,
					Context:     strings.Join(contextLines, "\n"),
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error searching files: %w", err)
	}

	return matches, nil
}

// Helper functions
func isBinaryFile(filePath string) bool {
	// Check file extension first
	ext := strings.ToLower(filepath.Ext(filePath))
	binaryExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
		".pdf": true, ".doc": true, ".docx": true, ".ppt": true, ".pptx": true,
		".xls": true, ".xlsx": true, ".zip": true, ".tar": true, ".gz": true,
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
	}

	if binaryExts[ext] {
		return true
	}

	// Peek at the content
	file, err := os.Open(filePath)
	if err != nil {
		return false // If we can't open it, assume it's not binary
	}
	defer file.Close()

	// Read first 512 bytes to check for binary content
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return false
	}

	// Simple heuristic: if more than 10% of the first 512 bytes are null or control chars
	binaryCount := 0
	for i := 0; i < n; i++ {
		if buffer[i] == 0 || (buffer[i] < 9 && buffer[i] != '\t' && buffer[i] != '\n' && buffer[i] != '\r') {
			binaryCount++
		}
	}

	return float64(binaryCount)/float64(n) > 0.1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
