package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTool_Execute(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "read-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file with multiple lines
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10\n"
	err = os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create an empty file
	emptyFilePath := filepath.Join(tempDir, "empty.txt")
	err = os.WriteFile(emptyFilePath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// Create a large file
	largeFilePath := filepath.Join(tempDir, "large.txt")
	largeContent := strings.Repeat("This is a long line of text that will be repeated many times.\n", 100)
	err = os.WriteFile(largeFilePath, []byte(largeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Create a file with a very long line
	longLinePath := filepath.Join(tempDir, "longline.txt")
	longLineContent := strings.Repeat("x", 3000) + "\nShort line\n"
	err = os.WriteFile(longLinePath, []byte(longLineContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create long line file: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	readTool := NewReadTool()

	// Test cases
	testCases := []struct {
		name           string
		params         ReadParams
		wantError      bool
		expectedLines  int
		expectedPrefix string
	}{
		{
			name: "Read entire file",
			params: ReadParams{
				FilePath: testFilePath,
			},
			wantError:      false,
			expectedLines:  10,
			expectedPrefix: "     1\tLine 1",
		},
		{
			name: "Read with offset",
			params: ReadParams{
				FilePath: testFilePath,
				Offset:   5,
			},
			wantError:      false,
			expectedLines:  4,
			expectedPrefix: "     6\tLine 7",
		},
		{
			name: "Read with limit",
			params: ReadParams{
				FilePath: testFilePath,
				Limit:    3,
			},
			wantError:      false,
			expectedLines:  3,
			expectedPrefix: "     1\tLine 1",
		},
		{
			name: "Read with offset and limit",
			params: ReadParams{
				FilePath: testFilePath,
				Offset:   3,
				Limit:    2,
			},
			wantError:      false,
			expectedLines:  2,
			expectedPrefix: "     4\tLine 5",
		},
		{
			name: "Read empty file",
			params: ReadParams{
				FilePath: emptyFilePath,
			},
			wantError:     false,
			expectedLines: 0,
		},
		{
			name: "Read non-existent file",
			params: ReadParams{
				FilePath: filepath.Join(tempDir, "nonexistent.txt"),
			},
			wantError: true,
		},
		{
			name: "Read directory",
			params: ReadParams{
				FilePath: subDir,
			},
			wantError: true,
		},
		{
			name: "Read file with very long line",
			params: ReadParams{
				FilePath: longLinePath,
			},
			wantError:     false,
			expectedLines: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsJSON, err := json.Marshal(tc.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			result, err := readTool.Execute(context.Background(), paramsJSON)

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
			readResult, ok := result.(ReadResult)
			if !ok {
				t.Fatalf("Expected result type ReadResult, got %T", result)
			}

			// Check line count
			if readResult.LineCount != tc.expectedLines {
				t.Errorf("Expected %d lines, got %d", tc.expectedLines, readResult.LineCount)
			}

			// Check content format
			lines := strings.Split(readResult.Content, "\n")
			if tc.expectedLines > 0 {
				if len(lines) != tc.expectedLines {
					t.Errorf("Expected %d lines in content, got %d", tc.expectedLines, len(lines))
				}

				if !strings.HasPrefix(lines[0], tc.expectedPrefix) {
					t.Errorf("Expected content to start with '%s', got '%s'", tc.expectedPrefix, lines[0])
				}
			}

			// For long line case, check if line was truncated
			if tc.params.FilePath == longLinePath {
				if len(lines) > 0 && !strings.Contains(lines[0], "[truncated]") {
					t.Errorf("Expected long line to be truncated")
				}
			}
		})
	}
}
