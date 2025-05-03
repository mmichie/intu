package ui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

// Simple fix for the Claude chunk handling issues
func cleanupChunk(chunk string) string {
	// Remove common Claude streaming artifacts
	chunk = strings.TrimPrefix(chunk, "data: ")

	// Try to extract pure text from JSON if it seems to be JSON
	if strings.HasPrefix(chunk, "{") && strings.HasSuffix(chunk, "}") {
		// This looks like a JSON object

		// Check for content_block_delta format (Claude API)
		if strings.Contains(chunk, "\"delta\"") && strings.Contains(chunk, "\"text\"") {
			// Extract text from the delta object
			textStart := strings.Index(chunk, "\"text\"")
			if textStart > 0 {
				// Find opening quote after "text":
				quotePos := strings.Index(chunk[textStart+6:], "\"")
				if quotePos > 0 {
					// Find closing quote
					textStart = textStart + 6 + quotePos + 1 // Position after opening quote
					textEnd := strings.Index(chunk[textStart:], "\"")
					if textEnd > 0 {
						chunk = chunk[textStart : textStart+textEnd]
					}
				}
			}
		} else if strings.Contains(chunk, "\"content_block\"") && strings.Contains(chunk, "\"text\"") {
			// Extract text from content_block object (initial block)
			textStart := strings.Index(chunk, "\"text\"")
			if textStart > 0 {
				// Find opening quote after "text":
				quotePos := strings.Index(chunk[textStart+6:], "\"")
				if quotePos > 0 {
					// Find closing quote
					textStart = textStart + 6 + quotePos + 1 // Position after opening quote
					textEnd := strings.Index(chunk[textStart:], "\"")
					if textEnd > 0 {
						chunk = chunk[textStart : textStart+textEnd]
					}
				}
			}
		}
	}

	// Handle any JSON escape sequences
	chunk = strings.ReplaceAll(chunk, "\\n", "\n")
	chunk = strings.ReplaceAll(chunk, "\\\"", "\"")
	chunk = strings.ReplaceAll(chunk, "\\\\", "\\")

	// Clean up any UI text and duplications
	chunk = cleanResponse(chunk)

	return chunk
}

// streamingResponseCmdFixed creates a command to handle streaming responses properly
func streamingResponseCmdFixed(agent Agent, ctx context.Context, prompt string) tea.Cmd {
	return func() tea.Msg {
		// Create a buffered channel for chunks
		chunkChan := make(chan string, 100)
		doneChan := make(chan error, 1)

		// Create a handler for streaming chunks
		handleChunk := func(chunk string) error {
			// Clean up the chunk
			chunk = cleanupChunk(chunk)

			if chunk == "" {
				return nil // Skip empty chunks
			}

			// Send the chunk via channel (non-blocking)
			select {
			case chunkChan <- chunk:
				// Sent successfully
			default:
				// Channel full, dropping chunk
			}

			// Send message to update UI immediately
			var p *tea.Program
			activeProgramMu.Lock()
			p = activeProgram
			activeProgramMu.Unlock()

			if p != nil {
				// Use goroutine to avoid blocking
				go func(msg tea.Msg) {
					p.Send(msg)
				}(aiStreamChunkMsg{chunk: chunk, done: false})
			}

			return nil
		}

		// Start streaming in a goroutine
		go func() {
			err := agent.ProcessStreaming(ctx, prompt, "", handleChunk)
			close(chunkChan)
			doneChan <- err
		}()

		// Wait for first chunk or completion
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				// Channel closed before any chunks
				err := <-doneChan
				return aiResponseMsg{
					response: "",
					err:      err,
				}
			}

			// Return first chunk
			return aiStreamChunkMsg{
				chunk: chunk,
				done:  false,
				err:   nil,
			}

		case err := <-doneChan:
			// Streaming completed without any chunks
			if err != nil {
				return aiStreamChunkMsg{
					chunk: "",
					done:  true,
					err:   err,
				}
			}

			// No error but no chunks either
			return aiResponseMsg{
				response: "",
				err:      nil,
			}
		}
	}
}
