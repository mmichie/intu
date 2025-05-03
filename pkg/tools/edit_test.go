package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEditTool_Execute(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "edit-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file with content for editing
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "This is line 1.\nThis is line 2.\nThis is line 3.\nThis is line 2 again.\n"
	err = os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// New file path for creation test
	newFilePath := filepath.Join(tempDir, "newfile.txt")

	// New file in a non-existent directory
	newDirPath := filepath.Join(tempDir, "newdir")
	newDirFilePath := filepath.Join(newDirPath, "newfile.txt")

	editTool := NewEditTool()

	// Test cases
	testCases := []struct {
		name         string
		params       EditParams
		wantError    bool
		expectedFile string
		checkContent string
	}{
		{
			name: "Replace single occurrence",
			params: EditParams{
				FilePath:             testFilePath,
				OldString:            "This is line 2.",
				NewString:            "This is modified line 2.",
				ExpectedReplacements: 1,
			},
			wantError:    false,
			expectedFile: testFilePath,
			checkContent: "This is line 1.\nThis is modified line 2.\nThis is line 3.\nThis is line 2 again.\n",
		},
		{
			name: "Replace multiple occurrences",
			params: EditParams{
				FilePath:             testFilePath,
				OldString:            "This is line",
				NewString:            "Here is line",
				ExpectedReplacements: 4, // Counting all occurrences in the file
			},
			wantError:    false,
			expectedFile: testFilePath,
			checkContent: "Here is line 1.\nHere is line 2.\nHere is line 3.\nHere is line 2 again.\n",
		},
		{
			name: "Wrong expected replacements",
			params: EditParams{
				FilePath:             testFilePath,
				OldString:            "nonexistent text",
				NewString:            "replacement",
				ExpectedReplacements: 1,
			},
			wantError: true,
		},
		{
			name: "Create new file",
			params: EditParams{
				FilePath:  newFilePath,
				OldString: "",
				NewString: "This is a new file content.\n",
			},
			wantError:    false,
			expectedFile: newFilePath,
			checkContent: "This is a new file content.\n",
		},
		{
			name: "Create new file in new directory",
			params: EditParams{
				FilePath:  newDirFilePath,
				OldString: "",
				NewString: "New file in new directory.\n",
			},
			wantError:    false,
			expectedFile: newDirFilePath,
			checkContent: "New file in new directory.\n",
		},
		{
			name: "Invalid file path",
			params: EditParams{
				FilePath:  "/nonexistent/path/test.txt",
				OldString: "old",
				NewString: "new",
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset test file for each test
			if tc.params.FilePath == testFilePath {
				err = os.WriteFile(testFilePath, []byte(testContent), 0644)
				if err != nil {
					t.Fatalf("Failed to reset test file: %v", err)
				}
			}

			paramsJSON, err := json.Marshal(tc.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			result, err := editTool.Execute(context.Background(), paramsJSON)

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
			editResult, ok := result.(EditResult)
			if !ok {
				t.Fatalf("Expected result type EditResult, got %T", result)
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

			// If creating a new file, replacements should be 0
			if tc.params.OldString == "" {
				if editResult.Replacements != 0 {
					t.Errorf("Expected 0 replacements for new file, got %d", editResult.Replacements)
				}
			} else {
				// Otherwise, check that replacements match expected
				if editResult.Replacements != tc.params.ExpectedReplacements {
					t.Errorf("Expected %d replacements, got %d", tc.params.ExpectedReplacements, editResult.Replacements)
				}
			}

			// Verify backup file if it's an edit operation (not a create operation)
			if tc.params.OldString != "" {
				backupPath := tc.expectedFile + ".bak"
				_, err = os.Stat(backupPath)
				if err != nil {
					t.Errorf("Expected backup file %s to exist, but got error: %v", backupPath, err)
				}

				// Check that the backup matches the original content
				backupContent, err := os.ReadFile(backupPath)
				if err != nil {
					t.Fatalf("Failed to read backup file: %v", err)
				}

				// For the "Replace multiple occurrences" test, the backup should match the output of the previous test
				if tc.name == "Replace multiple occurrences" {
					// The test file is reset for each test except this one,
					// so the backup should have the original test content
					if string(backupContent) != testContent {
						t.Errorf("Backup content mismatch for multiple replacements.\nExpected:\n%s\nGot:\n%s",
							testContent, string(backupContent))
					}
				} else if string(backupContent) != testContent {
					t.Errorf("Backup content mismatch.\nExpected:\n%s\nGot:\n%s", testContent, string(backupContent))
				}
			}
		})
	}
}
