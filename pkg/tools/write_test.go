package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteTool_Execute(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "write-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an existing file for overwrite tests
	existingFilePath := filepath.Join(tempDir, "existing.txt")
	existingContent := "This is an existing file.\n"
	err = os.WriteFile(existingFilePath, []byte(existingContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Paths for new files
	newFilePath := filepath.Join(tempDir, "newfile.txt")
	newFileInDirPath := filepath.Join(tempDir, "newdir", "newfile.txt")

	writeTool := NewWriteTool()

	// Test cases
	testCases := []struct {
		name            string
		params          WriteParams
		wantError       bool
		expectedFile    string
		checkContent    string
		shouldOverwrite bool
	}{
		{
			name: "Create new file",
			params: WriteParams{
				FilePath: newFilePath,
				Content:  "This is new file content.\n",
			},
			wantError:       false,
			expectedFile:    newFilePath,
			checkContent:    "This is new file content.\n",
			shouldOverwrite: false,
		},
		{
			name: "Create new file in new directory",
			params: WriteParams{
				FilePath: newFileInDirPath,
				Content:  "This is new file in a new directory.\n",
			},
			wantError:       false,
			expectedFile:    newFileInDirPath,
			checkContent:    "This is new file in a new directory.\n",
			shouldOverwrite: false,
		},
		{
			name: "Overwrite existing file",
			params: WriteParams{
				FilePath: existingFilePath,
				Content:  "This content should replace the existing content.\n",
			},
			wantError:       false,
			expectedFile:    existingFilePath,
			checkContent:    "This content should replace the existing content.\n",
			shouldOverwrite: true,
		},
		{
			name: "Write empty content",
			params: WriteParams{
				FilePath: filepath.Join(tempDir, "empty.txt"),
				Content:  "",
			},
			wantError:       false,
			expectedFile:    filepath.Join(tempDir, "empty.txt"),
			checkContent:    "",
			shouldOverwrite: false,
		},
		{
			name: "Missing file path",
			params: WriteParams{
				FilePath: "",
				Content:  "Some content",
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsJSON, err := json.Marshal(tc.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			result, err := writeTool.Execute(context.Background(), paramsJSON)

			// Check error
			if tc.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check result type
			writeResult, ok := result.(WriteResult)
			if !ok {
				t.Fatalf("Expected result type WriteResult, got %T", result)
			}

			// Check overwritten flag
			if writeResult.Overwritten != tc.shouldOverwrite {
				t.Errorf("Expected Overwritten=%v, got %v", tc.shouldOverwrite, writeResult.Overwritten)
			}

			// Verify the file exists
			_, err = os.Stat(tc.expectedFile)
			if err != nil {
				t.Errorf("Expected file %s to exist, but got error: %v", tc.expectedFile, err)
			}

			// Check file content
			content, err := os.ReadFile(tc.expectedFile)
			if err != nil {
				t.Fatalf("Failed to read result file: %v", err)
			}

			if string(content) != tc.checkContent {
				t.Errorf("File content mismatch.\nExpected:\n%s\nGot:\n%s", tc.checkContent, string(content))
			}

			// Check file size matches result
			if len(tc.checkContent) != writeResult.Size {
				t.Errorf("Expected size %d, got %d", len(tc.checkContent), writeResult.Size)
			}

			// Verify backup file if it's an overwrite operation
			if tc.shouldOverwrite {
				backupPath := tc.expectedFile + ".bak"
				_, err = os.Stat(backupPath)
				if err != nil {
					t.Errorf("Expected backup file %s to exist, but got error: %v", backupPath, err)
				}

				// Check backup content matches original content
				backupContent, err := os.ReadFile(backupPath)
				if err != nil {
					t.Fatalf("Failed to read backup file: %v", err)
				}

				if string(backupContent) != existingContent {
					t.Errorf("Backup content mismatch.\nExpected:\n%s\nGot:\n%s", existingContent, string(backupContent))
				}
			}
		})
	}
}
