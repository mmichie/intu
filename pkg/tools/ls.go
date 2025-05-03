package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LSParams defines the parameters for the LS tool
type LSParams struct {
	Path   string   `json:"path"`
	Ignore []string `json:"ignore,omitempty"`
}

// FileInfo represents information about a file or directory
type FileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsDir        bool      `json:"is_dir"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

// LSTool implements the LS command
type LSTool struct {
	BaseTool
}

// NewLSTool creates a new LS tool
func NewLSTool() *LSTool {
	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the directory to list (must be absolute, not relative)",
			},
			"ignore": map[string]interface{}{
				"type":        "array",
				"description": "List of glob patterns to ignore",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
		},
		"required": []string{"path"},
	}

	return &LSTool{
		BaseTool: BaseTool{
			ToolName:        "LS",
			ToolDescription: "Lists files and directories in a given path",
			ToolParams:      paramSchema,
			PermLevel:       PermissionReadOnly,
		},
	}
}

// Execute runs the LS tool
func (t *LSTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p LSParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Ensure path is provided
	if p.Path == "" {
		return nil, fmt.Errorf("path parameter is required")
	}

	// Get absolute path
	absPath, err := filepath.Abs(p.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access path: %w", err)
	}

	// If it's a file, return the file info
	if !info.IsDir() {
		return []FileInfo{
			{
				Name:         info.Name(),
				Path:         absPath,
				IsDir:        false,
				Size:         info.Size(),
				LastModified: info.ModTime(),
			},
		}, nil
	}

	// Read directory contents
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Create ignore matcher
	ignoreMatcher := func(name string) bool {
		for _, pattern := range p.Ignore {
			matched, err := filepath.Match(pattern, name)
			if err == nil && matched {
				return true
			}
		}
		return false
	}

	// Build result
	result := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()

		// Skip ignored files
		if ignoreMatcher(name) {
			continue
		}

		entryPath := filepath.Join(absPath, name)
		info, err := entry.Info()
		if err != nil {
			// Skip entries we can't get info for
			continue
		}

		result = append(result, FileInfo{
			Name:         name,
			Path:         entryPath,
			IsDir:        entry.IsDir(),
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})
	}

	return result, nil
}
