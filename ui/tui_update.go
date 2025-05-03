package ui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
)

// simpleStreamingCmd creates a simpler non-streaming fallback command
// This is a simple but reliable alternative to the streaming implementation
func simpleStreamingCmd(agent Agent, ctx context.Context, prompt string) tea.Cmd {
	return func() tea.Msg {
		// Just use the normal non-streaming process
		response, err := agent.Process(ctx, prompt, "")

		// If successful, clean up the response and format Markdown
		if err == nil {
			response = cleanResponse(response)
			// Format as Markdown to ensure proper line breaks and code formatting
			response = FormatMarkdown(response)
		}

		// Return as a regular message (no streaming)
		return aiResponseMsg{
			response: response,
			err:      err,
		}
	}
}
