package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// WriteParams defines the parameters for the Write tool
type WriteParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// WriteResult represents the result of writing a file
type WriteResult struct {
	FilePath    string `json:"file_path"`
	Size        int    `json:"size"`
	Overwritten bool   `json:"overwritten"`
}

// WriteTool implements the Write command
type WriteTool struct {
	BaseTool
}

// NewWriteTool creates a new Write tool
func NewWriteTool() *WriteTool {
	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to write (must be absolute, not relative)",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"file_path", "content"},
	}

	return &WriteTool{
		BaseTool: BaseTool{
			ToolName:        "Write",
			ToolDescription: "Write a file to the local filesystem. Overwrites the existing file if there is one.",
			ToolParams:      paramSchema,
			PermLevel:       PermissionFileWrite,
		},
	}
}

// Execute runs the Write tool
func (t *WriteTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p WriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Validate required parameters
	if p.FilePath == "" {
		return nil, fmt.Errorf("file_path parameter is required")
	}

	// Get absolute path
	absPath, err := filepath.Abs(p.FilePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check if file exists (to determine if we're overwriting)
	overwriting := false
	_, err = os.Stat(absPath)
	if err == nil {
		overwriting = true
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("error checking file: %w", err)
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Create a temporary file first
	tempFile, err := ioutil.TempFile(dir, "write-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Clean up in case of error

	// Write the content to the temporary file
	if _, err := tempFile.WriteString(p.Content); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tempFile.Close()

	// Set default permissions
	if err := os.Chmod(tempPath, 0644); err != nil {
		return nil, fmt.Errorf("failed to set file permissions: %w", err)
	}

	// If overwriting, create a backup if we can
	if overwriting {
		backupPath := absPath + ".bak"
		content, err := ioutil.ReadFile(absPath)
		if err == nil {
			err = ioutil.WriteFile(backupPath, content, 0644)
			if err != nil {
				// Log but continue - backup is optional
				fmt.Printf("Failed to create backup: %v\n", err)
			}
		}
	}

	// Rename the temporary file to the target path (atomic operation)
	if err := os.Rename(tempPath, absPath); err != nil {
		return nil, fmt.Errorf("failed to move file to target location: %w", err)
	}

	result := WriteResult{
		FilePath:    absPath,
		Size:        len(p.Content),
		Overwritten: overwriting,
	}

	return result, nil
}
