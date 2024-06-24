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

func NewIntuClient(providerName string) (*IntuClient, error) {
	provider, err := selectProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %v", err)
	}
	return &IntuClient{Provider: provider}, nil
}

func selectProvider(providerName string) (Provider, error) {
	if providerName == "" {
		providerName = os.Getenv("INTU_PROVIDER")
	}

	switch strings.ToLower(providerName) {
	case "openai":
		return NewOpenAIProvider()
	case "claude":
		return NewClaudeAIProvider()
	case "":
		// Default to OpenAI if no provider is specified
		return NewOpenAIProvider()
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}

func (c *IntuClient) GenerateCommitMessage(diffOutput string) (string, error) {
	basePrompt := c.generateBasePrompt(diffOutput)

	// Format the prompt based on the provider type
	var formattedPrompt string
	switch c.Provider.(type) {
	case *ClaudeAIProvider:
		formattedPrompt = fmt.Sprintf("\n\nHuman: %s\n\nAssistant: Certainly! Here's a concise git commit message for the changes you've described:", basePrompt)
	default: // OpenAI and any other providers
		formattedPrompt = basePrompt
	}

	return c.Provider.GenerateResponse(formattedPrompt)
}

func (c *IntuClient) generateBasePrompt(diffOutput string) string {
	return fmt.Sprintf(`You are a helpful assistant that generates concise git commit messages in conventional style.

%s

Please generate a concise git commit message using conventional style for the
above diff output.

Provide the message in multiple lines if necessary, with a short summary in the first
line followed by a blank line and then a more detailed description, using bullet points.
Optimize the output for Github and assume the engineer reading it is a FAANG engineer
experienced in the code and only needs the most salient points in the git history.
The width of text should be about 79 characters to avoid long lines.`, diffOutput)
}

func (c *IntuClient) CatFiles(pattern string, recursive bool, ignorePatterns []string) (map[string]FileInfo, error) {
	var files []string
	var err error

	shouldIgnore := func(path string) bool {
		for _, ignorePattern := range ignorePatterns {
			trimmedPattern := strings.Trim(ignorePattern, "*")
			if strings.Contains(path, trimmedPattern) {
				return true
			}
		}
		return false
	}

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
		if shouldIgnore(path) {
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
			if !info.IsDir() && !shouldIgnore(match) {
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
