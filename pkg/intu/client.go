package intu

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mmichie/intu/pkg/filters"
)

// IntuClient is the main client for interacting with AI providers
type IntuClient struct {
	Provider      Provider
	ActiveFilters []filters.Filter
}

// FileInfo represents the metadata and content of a file
type FileInfo struct {
	Filename      string    `json:"filename"`
	RelativePath  string    `json:"relative_path"`
	FileSize      int64     `json:"file_size"`
	ContentSize   int64     `json:"content_size"`
	LastModified  time.Time `json:"last_modified"`
	FileType      string    `json:"file_type"`
	LineCount     int       `json:"line_count"`
	FileExtension string    `json:"file_extension"`
	MD5Checksum   string    `json:"md5_checksum"`
	Content       string    `json:"content"`
}

func NewIntuClient(provider Provider) *IntuClient {
	return &IntuClient{Provider: provider}
}

func (c *IntuClient) CatFiles(pattern string, recursive bool) (map[string]FileInfo, error) {
	var files []string
	var err error

	walkFunc := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Warning: Error accessing %s: %v\n", path, err)
			return nil
		}
		if info.IsDir() {
			if !recursive && path != "." {
				return filepath.SkipDir
			}
			return nil
		}
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if matched {
			files = append(files, path)
		}
		return nil
	}

	if recursive {
		err = filepath.Walk(".", walkFunc)
	} else {
		// For non-recursive, we'll use Glob and then filter out directories
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				fmt.Printf("Warning: Error accessing %s: %v\n", match, err)
				continue
			}
			if !info.IsDir() {
				files = append(files, match)
			}
		}
	}

	if err != nil {
		return nil, err
	}

	result := make(map[string]FileInfo)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Warning: Error reading %s: %v\n", file, err)
			continue
		}

		// Apply filters to content if any
		for _, filter := range c.ActiveFilters {
			content = []byte(filter.Process(string(content)))
		}

		fileInfo, err := os.Stat(file)
		if err != nil {
			fmt.Printf("Warning: Error getting info for %s: %v\n", file, err)
			continue
		}

		md5sum := md5.Sum(content)
		checksum := hex.EncodeToString(md5sum[:])

		info := FileInfo{
			Filename:      filepath.Base(file),
			RelativePath:  file,
			FileSize:      fileInfo.Size(),
			ContentSize:   int64(len(content)),
			LastModified:  fileInfo.ModTime(),
			FileType:      getFileType(file),
			LineCount:     bytes.Count(content, []byte{'\n'}) + 1,
			FileExtension: filepath.Ext(file),
			MD5Checksum:   checksum,
			Content:       string(content),
		}

		result[file] = info
	}

	return result, nil
}

func getFileType(filename string) string {
	cmd := exec.Command("file", "-b", filename)
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func (c *IntuClient) GenerateCommitMessage() (string, error) {
	cmd := exec.Command("git", "diff", "--staged")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	prompt := "Create a git commit message using conventional style, be concise:\n\n" + out.String()
	return c.Provider.GenerateResponse(prompt)
}
