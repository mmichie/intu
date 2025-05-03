package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// EditParams defines the parameters for the Edit tool
type EditParams struct {
	FilePath             string `json:"file_path"`
	OldString            string `json:"old_string"`
	NewString            string `json:"new_string"`
	ExpectedReplacements int    `json:"expected_replacements,omitempty"`
}

// EditResult represents the result of editing a file
type EditResult struct {
	FilePath       string `json:"file_path"`
	Replacements   int    `json:"replacements"`
	OriginalSize   int    `json:"original_size"`
	NewSize        int    `json:"new_size"`
	BackupCreated  bool   `json:"backup_created,omitempty"`
	BackupLocation string `json:"backup_location,omitempty"`
}

// EditTool implements the Edit command
type EditTool struct {
	BaseTool
}

// NewEditTool creates a new Edit tool
func NewEditTool() *EditTool {
	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace it with",
			},
			"expected_replacements": map[string]interface{}{
				"type":        "integer",
				"description": "The expected number of replacements to perform. Defaults to 1 if not specified.",
			},
		},
		"required": []string{"file_path", "old_string", "new_string"},
	}

	return &EditTool{
		BaseTool: BaseTool{
			ToolName:        "Edit",
			ToolDescription: "Edits a file by replacing occurrences of a specified text",
			ToolParams:      paramSchema,
			PermLevel:       PermissionFileWrite,
		},
	}
}

// Execute runs the Edit tool
func (t *EditTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p EditParams
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

	// Default to 1 expected replacement if not specified
	if p.ExpectedReplacements <= 0 {
		p.ExpectedReplacements = 1
	}

	// Set up the result
	result := EditResult{
		FilePath:     absPath,
		Replacements: 0,
	}

	// Check if the path exists
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		// If the file doesn't exist, but both old_string is empty, create the file
		if os.IsNotExist(err) && p.OldString == "" {
			return t.createNewFile(absPath, p.NewString)
		}
		return nil, fmt.Errorf("failed to access file: %w", err)
	}

	// Check if it's a directory
	if fileInfo.IsDir() {
		return nil, fmt.Errorf("cannot edit a directory: %s", absPath)
	}

	// Read the file
	content, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Store original size
	result.OriginalSize = len(content)

	// Count occurrences of the old string
	occurrences := strings.Count(string(content), p.OldString)

	// Verify expected replacements
	if occurrences != p.ExpectedReplacements {
		return nil, fmt.Errorf("expected %d replacements but found %d", p.ExpectedReplacements, occurrences)
	}

	// Create backup
	backupPath := absPath + ".bak"
	err = ioutil.WriteFile(backupPath, content, fileInfo.Mode())
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}
	result.BackupCreated = true
	result.BackupLocation = backupPath

	// Replace all occurrences
	newContent := strings.Replace(string(content), p.OldString, p.NewString, p.ExpectedReplacements)
	result.Replacements = p.ExpectedReplacements
	result.NewSize = len(newContent)

	// Write to a temporary file first
	tempFile, err := ioutil.TempFile(filepath.Dir(absPath), "edit-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Clean up in case of error

	// Write the new content to the temporary file
	if _, err := tempFile.WriteString(newContent); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tempFile.Close()

	// Set the original file's permissions on the temporary file
	if err := os.Chmod(tempPath, fileInfo.Mode()); err != nil {
		return nil, fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Rename the temporary file to the original (atomic operation)
	if err := os.Rename(tempPath, absPath); err != nil {
		return nil, fmt.Errorf("failed to replace original file: %w", err)
	}

	return result, nil
}

// createNewFile creates a new file with the given content
func (t *EditTool) createNewFile(path string, content string) (interface{}, error) {
	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create parent directories: %w", err)
	}

	// Create a temporary file first
	tempFile, err := ioutil.TempFile(dir, "edit-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Clean up in case of error

	// Write the content to the temporary file
	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tempFile.Close()

	// Set default permissions
	if err := os.Chmod(tempPath, 0644); err != nil {
		return nil, fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Rename the temporary file to the target path (atomic operation)
	if err := os.Rename(tempPath, path); err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	result := EditResult{
		FilePath:     path,
		Replacements: 0,
		OriginalSize: 0,
		NewSize:      len(content),
	}

	return result, nil
}
