package provider

import (
	"context"
	"time"
)

// SimulateStreaming breaks a response into chunks for testing/fallback
func SimulateStreaming(ctx context.Context, fullResponse string, handler StreamHandler) error {
	// Split into chunks
	chunks := splitTextChunks(fullResponse, 15)

	// Create a timer for the simulation delay
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for i, chunk := range chunks {
		// Check for context cancellation before each chunk
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}

		// Send chunk
		isFinal := i == len(chunks)-1
		if err := handler(ResponseChunk{
			Content: chunk,
			IsFinal: isFinal,
		}); err != nil {
			return err
		}

		// Wait for ticker or context cancellation if not the last chunk
		if !isFinal {
			select {
			case <-ticker.C:
				// Time for next chunk
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// Use splitTextChunks in claude.go instead
