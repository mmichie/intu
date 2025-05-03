package ui

// This file is kept for compatibility but is no longer actively used.
// Direct non-streaming is used in the TUI to improve stability.

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

// simpleStreamingCmd creates a command to handle streaming responses
// This function is kept for backward compatibility but is no longer used by default
func simpleStreamingCmd(agent Agent, ctx context.Context, prompt string) tea.Cmd {
	return func() tea.Msg {
		// Use non-streaming with timeout protection
		timeoutCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()

		// Use a channel for the response
		responseChan := make(chan aiResponseMsg, 1)

		// Process in a goroutine
		go func() {
			response, err := agent.Process(timeoutCtx, prompt, "")

			// Format the response (non-streaming doesn't need special handling)
			if err == nil {
				// Apply the same formatting as streaming responses for consistency
				response = FormatMarkdown(cleanResponse(response))
			}

			// Send response
			responseChan <- aiResponseMsg{
				response: response,
				err:      err,
			}
		}()

		// Setup backup timeout
		backupTimer := time.NewTimer(95 * time.Second)
		defer backupTimer.Stop()

		// Wait for response or timeout
		select {
		case resp := <-responseChan:
			return resp
		case <-backupTimer.C:
			cancel() // Cancel context to stop any ongoing work
			return aiResponseMsg{
				response: "",
				err:      fmt.Errorf("request timed out after 95 seconds"),
			}
		}
	}
}
