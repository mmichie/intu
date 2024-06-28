package fileops

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Recursive bool
	Extended  bool
	Ignore    []string
}

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

type FileOperator interface {
	FindFiles(pattern string, options Options) ([]string, error)
	ReadFile(path string) (string, error)
	GetBasicFileInfo(path string, content string) (FileInfo, error)
	GetExtendedFileInfo(path string, content string) (FileInfo, error)
}

type LocalFileOperator struct{}

func NewFileOperator() FileOperator {
	return &LocalFileOperator{}
}

func (lfo *LocalFileOperator) FindFiles(pattern string, options Options) ([]string, error) {
	var files []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func (lfo *LocalFileOperator) ReadFile(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (lfo *LocalFileOperator) GetBasicFileInfo(path string, content string) (FileInfo, error) {
	return FileInfo{
		Filename:     filepath.Base(path),
		RelativePath: path,
		FileType:     getFileType(path),
		Content:      content,
	}, nil
}

func (lfo *LocalFileOperator) GetExtendedFileInfo(path string, content string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
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

func getFileType(filename string) string {
	return filepath.Ext(filename)
}

func countLines(s string) int {
	return len(strings.Split(s, "\n"))
}
