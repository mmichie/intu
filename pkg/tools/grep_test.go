package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepTool_Execute(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "grep-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"file1.txt": "This is line 1\nThis contains pattern123\nThis is line 3",
		"file2.js":  "// JavaScript file\nfunction test() {\n  console.log('pattern123');\n}",
		"file3.md":  "# Markdown file\n\nThis doesn't match\n\nBut this has pattern123 in it",
	}

	for name, content := range testFiles {
		filePath := filepath.Join(tempDir, name)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Create binary file
	binaryFilePath := filepath.Join(tempDir, "binary.bin")
	binaryContent := make([]byte, 100)
	for i := 0; i < 10; i++ {
		binaryContent[i] = 0 // Add some null bytes
	}
	binaryContent[50] = 'p'
	binaryContent[51] = 'a'
	binaryContent[52] = 't'
	binaryContent[53] = 't'
	binaryContent[54] = 'e'
	binaryContent[55] = 'r'
	binaryContent[56] = 'n'
	binaryContent[57] = '1'
	binaryContent[58] = '2'
	binaryContent[59] = '3'

	err = os.WriteFile(binaryFilePath, binaryContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create binary test file: %v", err)
	}

	grepTool := NewGrepTool()

	// Test cases
	testCases := []struct {
		name           string
		params         GrepParams
		wantMatchLen   int
		wantError      bool
		matchedFiles   []string
		unmatchedFiles []string
	}{
		{
			name: "Basic pattern match",
			params: GrepParams{
				Pattern: "pattern123",
				Path:    tempDir,
			},
			wantMatchLen: 3, // Matches in all 3 text files
			matchedFiles: []string{"file1.txt", "file2.js", "file3.md"},
		},
		{
			name: "Include filter",
			params: GrepParams{
				Pattern: "pattern123",
				Path:    tempDir,
				Include: "*.js",
			},
			wantMatchLen:   1, // Only matches in JS file
			matchedFiles:   []string{"file2.js"},
			unmatchedFiles: []string{"file1.txt", "file3.md"},
		},
		{
			name: "Regex pattern",
			params: GrepParams{
				Pattern: "pattern\\d+",
				Path:    tempDir,
			},
			wantMatchLen: 3, // Matches in all 3 text files
			matchedFiles: []string{"file1.txt", "file2.js", "file3.md"},
		},
		{
			name: "No matches",
			params: GrepParams{
				Pattern: "nonexistent",
				Path:    tempDir,
			},
			wantMatchLen: 0,
		},
		{
			name: "Invalid regex",
			params: GrepParams{
				Pattern: "pattern[",
				Path:    tempDir,
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

			result, err := grepTool.Execute(context.Background(), paramsJSON)

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
			matches, ok := result.([]GrepMatch)
			if !ok {
				t.Fatalf("Expected result type []GrepMatch, got %T", result)
			}

			// Check number of matches
			if len(matches) != tc.wantMatchLen {
				t.Errorf("Expected %d matches, got %d", tc.wantMatchLen, len(matches))
			}

			// Check that expected files are matched
			matchedFileMap := make(map[string]bool)
			for _, match := range matches {
				// Extract just the filename
				filename := filepath.Base(match.FilePath)
				matchedFileMap[filename] = true

				// Check that the matched line contains the pattern
				if !strings.Contains(match.MatchedLine, strings.TrimSuffix(tc.params.Pattern, "\\d+")) {
					t.Errorf("Matched line does not contain pattern: %s", match.MatchedLine)
				}

				// Check that context is not empty
				if match.Context == "" {
					t.Errorf("Context should not be empty")
				}
			}

			// Check that expected files were matched
			for _, file := range tc.matchedFiles {
				if !matchedFileMap[file] {
					t.Errorf("Expected file %s to be matched but it wasn't", file)
				}
			}

			// Check that unmatched files weren't matched
			for _, file := range tc.unmatchedFiles {
				if matchedFileMap[file] {
					t.Errorf("File %s should not have been matched", file)
				}
			}
		})
	}
}

func TestGrepTool_BinaryFileHandling(t *testing.T) {
	// This test confirms that binary files are skipped
	tempDir, err := os.MkdirTemp("", "grep-binary-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a text file
	textFilePath := filepath.Join(tempDir, "text.txt")
	err = os.WriteFile(textFilePath, []byte("This is a text file with pattern"), 0644)
	if err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	// Create a binary file
	binaryFilePath := filepath.Join(tempDir, "binary.bin")
	binaryContent := make([]byte, 100)
	for i := 0; i < 20; i++ {
		binaryContent[i] = 0 // Add null bytes
	}
	copy(binaryContent[50:], []byte("pattern")) // Add the search pattern

	err = os.WriteFile(binaryFilePath, binaryContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	grepTool := NewGrepTool()
	params := GrepParams{
		Pattern: "pattern",
		Path:    tempDir,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}

	result, err := grepTool.Execute(context.Background(), paramsJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	matches, ok := result.([]GrepMatch)
	if !ok {
		t.Fatalf("Expected result type []GrepMatch, got %T", result)
	}

	// Should only find matches in the text file, not the binary file
	if len(matches) != 1 {
		t.Errorf("Expected 1 match in text file only, got %d matches", len(matches))
	}

	if len(matches) > 0 && !strings.HasSuffix(matches[0].FilePath, "text.txt") {
		t.Errorf("Match should be in text.txt, not %s", matches[0].FilePath)
	}
}
