package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadParams defines the parameters for the Read tool
type ReadParams struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// ReadResult represents the result of reading a file
type ReadResult struct {
	Content   string `json:"content"`
	LineCount int    `json:"line_count"`
}

// ReadTool implements the Read command
type ReadTool struct {
	BaseTool
}

// NewReadTool creates a new Read tool
func NewReadTool() *ReadTool {
	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to read",
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "The line number to start reading from (0-based). Only provide if the file is too large to read at once",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "The number of lines to read. Only provide if the file is too large to read at once.",
			},
		},
		"required": []string{"file_path"},
	}

	return &ReadTool{
		BaseTool: BaseTool{
			ToolName:        "Read",
			ToolDescription: "Reads a file from the local filesystem",
			ToolParams:      paramSchema,
			PermLevel:       PermissionReadOnly,
		},
	}
}

// Execute runs the Read tool
func (t *ReadTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p ReadParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Ensure file_path is provided
	if p.FilePath == "" {
		return nil, fmt.Errorf("file_path parameter is required")
	}

	// Get absolute path
	absPath, err := filepath.Abs(p.FilePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check if file exists
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access file: %w", err)
	}

	// Check if it's a directory
	if info.IsDir() {
		return nil, fmt.Errorf("cannot read a directory: %s", absPath)
	}

	// Default values for limit if not provided
	if p.Limit <= 0 {
		p.Limit = 2000 // Default limit of 2000 lines
	}

	// Read the file with offset and limit
	content, lineCount, err := readFileWithLimits(absPath, p.Offset, p.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ReadResult{
		Content:   content,
		LineCount: lineCount,
	}, nil
}

// readFileWithLimits reads a file with specified line offset and limit
func readFileWithLimits(filePath string, offset, limit int) (string, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip lines until we reach the offset
	lineNumber := 1
	if offset > 0 {
		for scanner.Scan() && lineNumber <= offset {
			lineNumber++
		}

		if err := scanner.Err(); err != nil {
			return "", 0, err
		}
	}

	// Read lines until we hit the limit
	var lines []string
	for scanner.Scan() && len(lines) < limit {
		// Format line with number (cat -n format)
		lineText := scanner.Text()

		// Truncate very long lines
		if len(lineText) > 2000 {
			lineText = lineText[:2000] + "... [truncated]"
		}

		formattedLine := fmt.Sprintf("%6d\t%s", lineNumber, lineText)
		lines = append(lines, formattedLine)
		lineNumber++
	}

	if err := scanner.Err(); err != nil {
		return "", 0, err
	}

	return strings.Join(lines, "\n"), len(lines), nil
}
