package intu

import (
	"context"
	"fmt"
	"sync"

	"github.com/mmichie/intu/internal/ai"
	"github.com/mmichie/intu/internal/fileutils"
	"github.com/mmichie/intu/internal/filters"
)

// Client is the main client for interacting with AI providers
type Client struct {
	Provider ai.Provider
	Filters  []filters.Filter
}

// NewClient creates a new Client with the specified provider
func NewClient(provider ai.Provider) *Client {
	return &Client{
		Provider: provider,
	}
}

// AddFilter adds a filter to the client
func (c *Client) AddFilter(filter filters.Filter) {
	c.Filters = append(c.Filters, filter)
}

// GenerateCommitMessage generates a commit message based on the provided diff
func (c *Client) GenerateCommitMessage(ctx context.Context, diffOutput string) (string, error) {
	prompt := generateCommitPrompt(diffOutput)
	return c.Provider.GenerateResponse(ctx, prompt)
}

// CatFiles processes files matching the given pattern
func (c *Client) CatFiles(ctx context.Context, pattern string, options fileutils.Options) ([]fileutils.FileInfo, error) {
	files, err := fileutils.FindFiles(pattern, options)
	if err != nil {
		return nil, fmt.Errorf("error finding files: %w", err)
	}

	var wg sync.WaitGroup
	results := make([]fileutils.FileInfo, len(files))
	errors := make(chan error, len(files))

	for i, file := range files {
		wg.Add(1)
		go func(i int, file string) {
			defer wg.Done()
			info, err := c.processFile(file, options.Extended)
			if err != nil {
				errors <- fmt.Errorf("error processing %s: %w", file, err)
				return
			}
			results[i] = info
		}(i, file)
	}

	wg.Wait()
	close(errors)

	if len(errors) > 0 {
		return results, <-errors
	}

	return results, nil
}

func (c *Client) processFile(file string, extended bool) (fileutils.FileInfo, error) {
	content, err := fileutils.ReadFile(file)
	if err != nil {
		return fileutils.FileInfo{}, err
	}

	for _, filter := range c.Filters {
		content = filter.Process(content)
	}

	if extended {
		return fileutils.GetExtendedFileInfo(file, content)
	}
	return fileutils.GetBasicFileInfo(file, content)
}

func generateCommitPrompt(diffOutput string) string {
	return fmt.Sprintf(`Generate a concise git commit message in conventional style for the following diff:

%s

Provide a short summary in the first line, followed by a blank line and a more detailed description using bullet points.
Optimize for a FAANG engineer experienced with the code. Keep line width to about 79 characters.`, diffOutput)
}
