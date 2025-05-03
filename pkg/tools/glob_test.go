package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGlobTool_Execute(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "glob-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with different extensions
	testFiles := []string{
		"file1.txt",
		"file2.txt",
		"config.json",
		"main.go",
		"util.go",
		"README.md",
	}

	// Create a subdirectory with files
	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subDirFiles := []string{
		"nested1.txt",
		"nested2.go",
	}

	// Create all files
	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
		// Space out creation times for sorting test
		time.Sleep(10 * time.Millisecond)
	}

	for _, file := range subDirFiles {
		filePath := filepath.Join(subDir, file)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	globTool := NewGlobTool()

	// Test cases
	testCases := []struct {
		name             string
		params           GlobParams
		wantMatchLen     int
		wantError        bool
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "Match all files",
			params: GlobParams{
				Pattern: "*",
				Path:    tempDir,
			},
			wantMatchLen:  len(testFiles) + 1, // +1 for the subdirectory
			shouldContain: []string{"file1.txt", "config.json", "subdir"},
		},
		{
			name: "Match by extension",
			params: GlobParams{
				Pattern: "*.go",
				Path:    tempDir,
			},
			wantMatchLen:     2,
			shouldContain:    []string{"main.go", "util.go"},
			shouldNotContain: []string{"file1.txt", "config.json"},
		},
		{
			name: "Match with wildcard prefix",
			params: GlobParams{
				Pattern: "file*.txt",
				Path:    tempDir,
			},
			wantMatchLen:     2,
			shouldContain:    []string{"file1.txt", "file2.txt"},
			shouldNotContain: []string{"config.json", "main.go"},
		},
		{
			name: "Recursive pattern",
			params: GlobParams{
				Pattern: "subdir/*.go", // Find .go files in subdirectories
				Path:    tempDir,
			},
			wantMatchLen:     1,
			shouldContain:    []string{"nested2.go"},
			shouldNotContain: []string{"main.go"}, // Not in subdirectory
		},
		{
			name: "No matches",
			params: GlobParams{
				Pattern: "nonexistent*.xyz",
				Path:    tempDir,
			},
			wantMatchLen: 0,
		},
		{
			name: "Invalid path",
			params: GlobParams{
				Pattern: "*",
				Path:    filepath.Join(tempDir, "nonexistent"),
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

			result, err := globTool.Execute(context.Background(), paramsJSON)

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
			matches, ok := result.([]GlobMatch)
			if !ok {
				t.Fatalf("Expected result type []GlobMatch, got %T", result)
			}

			// Check number of matches
			if len(matches) != tc.wantMatchLen {
				t.Errorf("Expected %d matches, got %d", tc.wantMatchLen, len(matches))
			}

			// Check that expected files are included
			matchPaths := make(map[string]bool)
			for _, match := range matches {
				basename := filepath.Base(match.Path)
				matchPaths[basename] = true
			}

			for _, expected := range tc.shouldContain {
				if len(tc.shouldContain) > 0 && !matchPaths[expected] {
					t.Errorf("Expected %s to be matched but it wasn't", expected)
				}
			}

			for _, unexpected := range tc.shouldNotContain {
				if len(tc.shouldNotContain) > 0 && matchPaths[unexpected] {
					t.Errorf("File %s should not have been matched", unexpected)
				}
			}

			// Check sorting by modification time (if we have multiple matches)
			if len(matches) > 1 {
				isSorted := true
				for i := 0; i < len(matches)-1; i++ {
					if matches[i].LastModified.Before(matches[i+1].LastModified) {
						isSorted = false
						break
					}
				}
				if !isSorted {
					t.Errorf("Results are not sorted by modification time (newest first)")
				}
			}
		})
	}
}
