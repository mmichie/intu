package security

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// TerminalPrompt creates a permission prompt that uses the terminal
func TerminalPrompt(w io.Writer, r io.Reader) PromptFunc {
	return func(req PermissionRequest) (PermissionResponse, error) {
		var message string

		switch req.Level {
		case PermissionReadOnly:
			message = fmt.Sprintf("The tool '%s' is requesting read-only access.\n%s", req.ToolName, req.ToolDesc)
		case PermissionShellExec:
			message = fmt.Sprintf("The tool '%s' is requesting permission to execute a shell command:\n%s\n%s", req.ToolName, req.Command, req.ToolDesc)
		case PermissionFileWrite:
			message = fmt.Sprintf("The tool '%s' is requesting permission to write to a file:\n%s\n%s", req.ToolName, req.FilePath, req.ToolDesc)
		case PermissionNetwork:
			message = fmt.Sprintf("The tool '%s' is requesting network access to:\n%s\n%s", req.ToolName, req.URL, req.ToolDesc)
		default:
			message = fmt.Sprintf("The tool '%s' is requesting permission (unknown level).\n%s", req.ToolName, req.ToolDesc)
		}

		fmt.Fprintf(w, "\n%s\n\n", message)
		fmt.Fprintf(w, "Allow? [y]es/[n]o/[a]lways: ")

		var response string
		fmt.Fscanln(r, &response)
		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "y", "yes":
			return PermissionGrantedOnce, nil
		case "a", "always":
			return PermissionGrantedAlways, nil
		default:
			return PermissionDenied, nil
		}
	}
}

// DefaultPrompt returns the default permission prompt function
func DefaultPrompt() PromptFunc {
	return TerminalPrompt(os.Stdout, os.Stdin)
}

// NoPrompt returns a permission function that always grants permission
// Useful for testing or non-interactive environments
func NoPrompt() PromptFunc {
	return func(req PermissionRequest) (PermissionResponse, error) {
		return PermissionGrantedOnce, nil
	}
}

// DenyPrompt returns a permission function that always denies permission
// Useful for secure environments or testing
func DenyPrompt() PromptFunc {
	return func(req PermissionRequest) (PermissionResponse, error) {
		return PermissionDenied, nil
	}
}

// AllowListPrompt returns a permission function that grants permission based on an allow list
func AllowListPrompt(allowedTools map[string]bool) PromptFunc {
	return func(req PermissionRequest) (PermissionResponse, error) {
		if allowed, exists := allowedTools[req.ToolName]; exists && allowed {
			return PermissionGrantedAlways, nil
		}
		return PermissionDenied, nil
	}
}
