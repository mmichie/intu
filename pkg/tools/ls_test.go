package tools

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestLSTool_Execute(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := ioutil.TempDir("", "ls-tool-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some files and subdirectories
	testFiles := []string{"file1.txt", "file2.go", "file3.md"}
	testDirs := []string{"dir1", "dir2"}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		err := ioutil.WriteFile(filePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	for _, dir := range testDirs {
		dirPath := filepath.Join(tempDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}

	lsTool := NewLSTool()

	t.Run("List directory", func(t *testing.T) {
		params := LSParams{
			Path: tempDir,
		}

		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal parameters: %v", err)
		}

		result, err := lsTool.Execute(context.Background(), paramsJSON)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		fileInfos, ok := result.([]FileInfo)
		if !ok {
			t.Fatalf("Result is not of type []FileInfo")
		}

		// Check if all expected files and directories are in the result
		if len(fileInfos) != len(testFiles)+len(testDirs) {
			t.Errorf("Expected %d entries, got %d", len(testFiles)+len(testDirs), len(fileInfos))
		}

		foundNames := make(map[string]bool)
		for _, info := range fileInfos {
			foundNames[info.Name] = true
		}

		for _, file := range testFiles {
			if !foundNames[file] {
				t.Errorf("File %s not found in result", file)
			}
		}

		for _, dir := range testDirs {
			if !foundNames[dir] {
				t.Errorf("Directory %s not found in result", dir)
			}
		}
	})

	t.Run("List with ignore pattern", func(t *testing.T) {
		params := LSParams{
			Path:   tempDir,
			Ignore: []string{"*.go", "dir1"},
		}

		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal parameters: %v", err)
		}

		result, err := lsTool.Execute(context.Background(), paramsJSON)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		fileInfos, ok := result.([]FileInfo)
		if !ok {
			t.Fatalf("Result is not of type []FileInfo")
		}

		// We should have filtered out file2.go and dir1
		expectedCount := len(testFiles) + len(testDirs) - 2
		if len(fileInfos) != expectedCount {
			t.Errorf("Expected %d entries after filtering, got %d", expectedCount, len(fileInfos))
		}

		for _, info := range fileInfos {
			if info.Name == "file2.go" || info.Name == "dir1" {
				t.Errorf("Found %s which should have been ignored", info.Name)
			}
		}
	})

	t.Run("Invalid path", func(t *testing.T) {
		params := LSParams{
			Path: filepath.Join(tempDir, "nonexistent"),
		}

		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal parameters: %v", err)
		}

		_, err = lsTool.Execute(context.Background(), paramsJSON)
		if err == nil {
			t.Errorf("Expected error for nonexistent path, got nil")
		}
	})

	t.Run("Invalid parameters", func(t *testing.T) {
		_, err := lsTool.Execute(context.Background(), []byte(`{invalid json}`))
		if err == nil {
			t.Errorf("Expected error for invalid JSON, got nil")
		}
	})

	t.Run("Missing path", func(t *testing.T) {
		_, err := lsTool.Execute(context.Background(), []byte(`{}`))
		if err == nil {
			t.Errorf("Expected error for missing path, got nil")
		}
	})
}
