// Package fileops provides functionality for file operations
package fileops

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Options struct defines the configuration for file operations
type Options struct {
	Recursive bool
	Extended  bool
	Ignore    []string
}

// FileInfo struct contains information about a file
type FileInfo struct {
	Filename     string    `json:"filename"`
	RelativePath string    `json:"relative_path"`
	FileType     string    `json:"file_type"`
	Content      string    `json:"content"`
	FileSize     int64     `json:"file_size,omitempty"`
	LastModified time.Time `json:"last_modified,omitempty"`
	LineCount    int       `json:"line_count,omitempty"`
	MD5Checksum  string    `json:"md5_checksum,omitempty"`
}

// FileOperator interface defines methods for file operations
type FileOperator interface {
	FindFiles(ctx context.Context, pattern string, options Options) ([]string, error)
	ReadFile(ctx context.Context, path string) (string, error)
	GetBasicFileInfo(ctx context.Context, path string, content string) (FileInfo, error)
	GetExtendedFileInfo(ctx context.Context, path string, content string) (FileInfo, error)
}

// LocalFileOperator implements FileOperator for local file system operations
type LocalFileOperator struct{}

// NewFileOperator creates a new LocalFileOperator
func NewFileOperator() FileOperator {
	return &LocalFileOperator{}
}

// FindFiles searches for files matching the given pattern with the specified options
func (lfo *LocalFileOperator) FindFiles(ctx context.Context, pattern string, options Options) ([]string, error) {
	var files []string
	var mu sync.Mutex
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if info.IsDir() {
			if !options.Recursive && path != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if the file should be ignored
		for _, ignore := range options.Ignore {
			if matched, _ := filepath.Match(ignore, filepath.Base(path)); matched {
				return nil
			}
		}

		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			mu.Lock()
			files = append(files, path)
			mu.Unlock()
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error finding files: %w", err)
	}

	return files, nil
}

// ReadFile reads the content of a file
func (lfo *LocalFileOperator) ReadFile(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	var content strings.Builder
	buffer := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("error reading file %s: %w", path, err)
		}
		if n == 0 {
			break
		}
		content.Write(buffer[:n])
	}
	return content.String(), nil
}

// GetBasicFileInfo retrieves basic information about a file
func (lfo *LocalFileOperator) GetBasicFileInfo(ctx context.Context, path string, content string) (FileInfo, error) {
	select {
	case <-ctx.Done():
		return FileInfo{}, ctx.Err()
	default:
	}

	return FileInfo{
		Filename:     filepath.Base(path),
		RelativePath: path,
		FileType:     getFileType(path),
		Content:      content,
	}, nil
}

// GetExtendedFileInfo retrieves extended information about a file
func (lfo *LocalFileOperator) GetExtendedFileInfo(ctx context.Context, path string, content string) (FileInfo, error) {
	select {
	case <-ctx.Done():
		return FileInfo{}, ctx.Err()
	default:
	}

	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to get file info for %s: %w", path, err)
	}

	md5sum := md5.Sum([]byte(content))
	checksum := hex.EncodeToString(md5sum[:])

	return FileInfo{
		Filename:     filepath.Base(path),
		RelativePath: path,
		FileType:     getFileType(path),
		Content:      content,
		FileSize:     info.Size(),
		LastModified: info.ModTime(),
		LineCount:    countLines(content),
		MD5Checksum:  checksum,
	}, nil
}

// getFileType returns the file extension
func getFileType(filename string) string {
	return filepath.Ext(filename)
}

// countLines counts the number of lines in a string
func countLines(s string) int {
	return len(strings.Split(s, "\n"))
}
