package intu

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// IntuClient is the main client for interacting with AI providers
type IntuClient struct {
	Provider Provider
}

func NewIntuClient(provider Provider) *IntuClient {
	return &IntuClient{Provider: provider}
}

// FileInfo represents the metadata and content of a file
type FileInfo struct {
	Filename      string    `json:"filename"`
	RelativePath  string    `json:"relative_path"`
	FileSize      int64     `json:"file_size"`
	LastModified  time.Time `json:"last_modified"`
	FileType      string    `json:"file_type"`
	LineCount     int       `json:"line_count"`
	FileExtension string    `json:"file_extension"`
	MD5Checksum   string    `json:"md5_checksum"`
	Content       string    `json:"content"`
}

func (c *IntuClient) CatFiles(pattern string, recursive bool) (map[string]FileInfo, error) {
	var files []string
	var err error

	if recursive {
		err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				matched, err := filepath.Match(pattern, filepath.Base(path))
				if err != nil {
					return err
				}
				if matched {
					files = append(files, path)
				}
			}
			return nil
		})
	} else {
		files, err = filepath.Glob(pattern)
	}

	if err != nil {
		return nil, err
	}

	result := make(map[string]FileInfo)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		fileInfo, err := os.Stat(file)
		if err != nil {
			return nil, err
		}

		md5sum := md5.Sum(content)
		checksum := hex.EncodeToString(md5sum[:])

		info := FileInfo{
			Filename:      filepath.Base(file),
			RelativePath:  file,
			FileSize:      fileInfo.Size(),
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
