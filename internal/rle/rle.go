package rle

import (
	"fmt"
	"strconv"
	"strings"
)

// FileOutput represents a single compressed file
type FileOutput struct {
	RelativePath string `json:"relative_path"`
	RLEContent   string `json:"rle_content"`
}

// BatchOutput represents a collection of compressed files with shared metadata
type BatchOutput struct {
	Format      string       `json:"format"`
	Example     string       `json:"example"`
	Description string       `json:"description"`
	Files       []FileOutput `json:"files"`
}

// NewBatchOutput creates a new batch of RLE outputs with shared metadata
func NewBatchOutput(files []FileOutput) BatchOutput {
	return BatchOutput{
		Format:  "run-length-encoding",
		Example: "3ab -> aaab",
		Description: `This content is compressed using run-length encoding (RLE).
To decode: Each token is either a single character (representing one occurrence)
or a number followed by a character (representing N occurrences).
For example, "3ab" decodes to "aaab" - "3a" means three "a"s, and "b" means one "b".`,
		Files: files,
	}
}

// CompressFile creates a new RLE output for a single file
func CompressFile(path, content string) FileOutput {
	return FileOutput{
		RelativePath: path,
		RLEContent:   Compress(content),
	}
}

// Compress performs optimized ASCII run-length encoding on the input string
func Compress(input string) string {
	if len(input) == 0 {
		return ""
	}

	var compressed strings.Builder
	currentChar := input[0]
	count := 1

	for i := 1; i < len(input); i++ {
		if input[i] == currentChar {
			count++
		} else {
			writeChunk(&compressed, currentChar, count)
			currentChar = input[i]
			count = 1
		}
	}
	// Write the last chunk
	writeChunk(&compressed, currentChar, count)

	return compressed.String()
}

// Decompress decompresses an RLE-encoded string
func Decompress(compressed string) (string, error) {
	if len(compressed) == 0 {
		return "", nil
	}

	var decompressed strings.Builder
	var numStr strings.Builder

	for i := 0; i < len(compressed); i++ {
		if compressed[i] >= '0' && compressed[i] <= '9' {
			numStr.WriteByte(compressed[i])
		} else {
			if numStr.Len() == 0 {
				// Single character case
				decompressed.WriteByte(compressed[i])
			} else {
				// Number + character case
				count, err := strconv.Atoi(numStr.String())
				if err != nil {
					return "", fmt.Errorf("invalid RLE format: invalid count")
				}

				for j := 0; j < count; j++ {
					decompressed.WriteByte(compressed[i])
				}
				numStr.Reset()
			}
		}
	}

	return decompressed.String(), nil
}

// writeChunk writes a single chunk of RLE-encoded data
func writeChunk(builder *strings.Builder, char byte, count int) {
	if count == 1 {
		builder.WriteByte(char)
	} else {
		builder.WriteString(strconv.Itoa(count))
		builder.WriteByte(char)
	}
}
