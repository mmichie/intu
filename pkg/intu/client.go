package intu

import (
	"context"
	"fmt"
	"sync"

	"github.com/mmichie/intu/internal/ai"
	"github.com/mmichie/intu/internal/fileops"
	"github.com/mmichie/intu/internal/filters"
)

// Client is the main client
type Client struct {
	Provider ai.Provider
	FileOps  *fileops.FileOperator
	Filters  []filters.Filter
}

// NewClient creates a new Client
func NewClient(provider ai.Provider) *Client {
	return &Client{
		Provider: provider,
		FileOps:  fileops.NewFileOperator(),
	}
}

// AddFilter adds a filter to the client
func (c *Client) AddFilter(filter filters.Filter) {
	c.Filters = append(c.Filters, filter)
}

// GenerateCommitMessage generates a commit message based on the provided diff
func (c *Client) GenerateCommitMessage(ctx context.Context, diffOutput string) (string, error) {
	prompt := generateCommitPrompt(diffOutput)
	message, err := c.Provider.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate commit message: %w", err)
	}
	return message, nil
}

// CatFiles processes files matching the given pattern
func (c *Client) CatFiles(ctx context.Context, pattern string, options fileops.Options) ([]fileops.FileInfo, error) {
	files, err := c.FileOps.FindFiles(pattern, options)
	if err != nil {
		return nil, fmt.Errorf("error finding files: %w", err)
	}

	var wg sync.WaitGroup
	results := make([]fileops.FileInfo, len(files))
	errs := make([]error, len(files))

	for i, file := range files {
		wg.Add(1)
		go func(i int, file string) {
			defer wg.Done()
			info, err := c.processFile(ctx, file, options.Extended)
			if err != nil {
				errs[i] = fmt.Errorf("error processing %s: %w", file, err)
				return
			}
			results[i] = info
		}(i, file)
	}

	wg.Wait()

	// Collect all non-nil errors
	var processErrors []error
	for _, err := range errs {
		if err != nil {
			processErrors = append(processErrors, err)
		}
	}

	if len(processErrors) > 0 {
		return results, fmt.Errorf("errors occurred while processing files: %v", processErrors)
	}

	return results, nil
}

func (c *Client) processFile(ctx context.Context, file string, extended bool) (fileops.FileInfo, error) {
	content, err := c.FileOps.ReadFile(file)
	if err != nil {
		return fileops.FileInfo{}, fmt.Errorf("failed to read file: %w", err)
	}

	for _, filter := range c.Filters {
		select {
		case <-ctx.Done():
			return fileops.FileInfo{}, ctx.Err()
		default:
			content = filter.Process(content)
		}
	}

	var info fileops.FileInfo
	var infoErr error
	if extended {
		info, infoErr = c.FileOps.GetExtendedFileInfo(file, content)
	} else {
		info, infoErr = c.FileOps.GetBasicFileInfo(file, content)
	}

	if infoErr != nil {
		return fileops.FileInfo{}, fmt.Errorf("failed to get file info: %w", infoErr)
	}

	return info, nil
}

func generateCommitPrompt(diffOutput string) string {
	return fmt.Sprintf(`Generate a concise git commit message in conventional style for the following diff:

%s

Provide a short summary in the first line, followed by a blank line and a more detailed description using bullet points.
Optimize for a FAANG engineer experienced with the code. Keep line width to about 79 characters.`, diffOutput)
}
