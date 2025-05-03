package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BashParams defines the parameters for the Bash tool
type BashParams struct {
	Command     string `json:"command"`
	Timeout     int    `json:"timeout,omitempty"`
	Description string `json:"description,omitempty"`
}

// BashResult represents the result of executing a bash command
type BashResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error,omitempty"`
}

// BashTool implements the Bash command
type BashTool struct {
	BaseTool
}

// NewBashTool creates a new Bash tool
func NewBashTool() *BashTool {
	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Optional timeout in milliseconds (max 600000)",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Clear, concise description of what this command does in 5-10 words",
			},
		},
		"required": []string{"command"},
	}

	return &BashTool{
		BaseTool: BaseTool{
			ToolName:        "Bash",
			ToolDescription: "Executes a given bash command in a persistent shell session with optional timeout",
			ToolParams:      paramSchema,
			PermLevel:       PermissionShellExec,
		},
	}
}

// isDangerousCommand checks if a command might be dangerous
func isDangerousCommand(cmd string) bool {
	// List of potentially dangerous commands to warn about
	dangerousPatterns := []string{
		"rm -rf /", "rm -r /", "rm -r /*", "rm -r ~", "rm -r ~/",
		"mkfs", "dd if=", "dd of=", "> /dev/", "> /etc/", "> /sys/",
		"chmod -R 777 /", "chmod -R 000 /", "chmod -R o+w /",
		"wget -O", "curl -O", "&& rm -rf", "|| rm -rf",
	}

	// Don't flag pipes as dangerous
	// if strings.Contains(cmd, "|") {
	//    return true
	// }

	cmd = strings.ToLower(cmd)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmd, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// setupSandbox creates a restricted environment for command execution
func setupSandbox() (string, func(), error) {
	// Create a temporary directory for the sandbox
	tempDir, err := os.MkdirTemp("", "intu-sandbox-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create sandbox directory: %w", err)
	}

	// Create cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	// Create some safe directories inside the sandbox
	dirs := []string{"data", "tmp", "logs"}
	for _, dir := range dirs {
		dirPath := filepath.Join(tempDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	return tempDir, cleanup, nil
}

// Execute runs the Bash tool
func (t *BashTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p BashParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Ensure command is provided
	if p.Command == "" {
		return nil, fmt.Errorf("command parameter is required")
	}

	// Check for dangerous commands
	if isDangerousCommand(p.Command) {
		return nil, fmt.Errorf("potentially dangerous command detected: %s", p.Command)
	}

	// Set default timeout if not provided or cap it if too large
	if p.Timeout <= 0 {
		p.Timeout = 120000 // 2 minutes default
	} else if p.Timeout > 600000 {
		p.Timeout = 600000 // 10 minutes max
	}

	// Set up sandbox environment
	sandboxDir, cleanup, err := setupSandbox()
	if err != nil {
		return nil, fmt.Errorf("failed to set up sandbox: %w", err)
	}
	defer cleanup()

	// Create a context with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(p.Timeout)*time.Millisecond)
	defer cancel()

	// Execute command
	var shell string
	var args []string
	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
		args = []string{"/C", p.Command}
	} else {
		shell = "/bin/sh"
		args = []string{"-c", p.Command}
	}

	cmd := exec.CommandContext(execCtx, shell, args...)

	// Set up working directory to sandbox
	cmd.Dir = sandboxDir

	// Set environment variables with sandbox paths
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SANDBOX_DIR=%s", sandboxDir),
		fmt.Sprintf("SANDBOX_DATA=%s", filepath.Join(sandboxDir, "data")),
		fmt.Sprintf("SANDBOX_TMP=%s", filepath.Join(sandboxDir, "tmp")),
		fmt.Sprintf("SANDBOX_LOGS=%s", filepath.Join(sandboxDir, "logs")),
	)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

	// Prepare result
	result := BashResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	// Handle errors and exit code
	if err != nil {
		result.Error = err.Error()

		// Try to get exit code if command started but failed
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	} else if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	// Limit output size if too large (30k chars)
	const maxOutputSize = 30000
	if len(result.Stdout) > maxOutputSize {
		result.Stdout = result.Stdout[:maxOutputSize] + "\n... [output truncated, too large]"
	}
	if len(result.Stderr) > maxOutputSize {
		result.Stderr = result.Stderr[:maxOutputSize] + "\n... [output truncated, too large]"
	}

	return result, nil
}
