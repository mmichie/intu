package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// GlobParams defines the parameters for the Glob tool
type GlobParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// GlobMatch represents a matched file from the Glob tool
type GlobMatch struct {
	Path         string    `json:"path"`
	LastModified time.Time `json:"last_modified"`
}

// GlobTool implements the Glob command
type GlobTool struct {
	BaseTool
}

// NewGlobTool creates a new Glob tool
func NewGlobTool() *GlobTool {
	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The glob pattern to match files against",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The directory to search in. If not specified, the current working directory will be used.",
			},
		},
		"required": []string{"pattern"},
	}

	return &GlobTool{
		BaseTool: BaseTool{
			ToolName:        "Glob",
			ToolDescription: "Fast file pattern matching tool that finds files by name patterns",
			ToolParams:      paramSchema,
			PermLevel:       PermissionReadOnly,
		},
	}
}

// Execute runs the Glob tool
func (t *GlobTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p GlobParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Check required parameters
	if p.Pattern == "" {
		return nil, fmt.Errorf("pattern parameter is required")
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

	// Check if path exists
	_, err = os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access path: %w", err)
	}

	// Use filepath.Glob to find matches
	matches, err := filepath.Glob(filepath.Join(absPath, p.Pattern))
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	// Create result with file info
	result := make([]GlobMatch, 0, len(matches))
	for _, match := range matches {
		// Get file info for modification time
		fileInfo, err := os.Stat(match)
		if err != nil {
			// Skip files we can't get info for
			continue
		}

		result = append(result, GlobMatch{
			Path:         match,
			LastModified: fileInfo.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastModified.After(result[j].LastModified)
	})

	return result, nil
}
