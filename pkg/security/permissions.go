// Package security implements the permission system for Intu tools
package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PermissionLevel defines the security level of a tool
type PermissionLevel int

const (
	// PermissionReadOnly allows reading files and metadata but no modifications
	PermissionReadOnly PermissionLevel = iota

	// PermissionShellExec allows executing shell commands
	PermissionShellExec

	// PermissionFileWrite allows modifying and creating files
	PermissionFileWrite

	// PermissionNetwork allows network access
	PermissionNetwork
)

// String returns the string representation of the permission level
func (p PermissionLevel) String() string {
	names := map[PermissionLevel]string{
		PermissionReadOnly:  "read-only",
		PermissionShellExec: "shell-execution",
		PermissionFileWrite: "file-write",
		PermissionNetwork:   "network",
	}

	if name, ok := names[p]; ok {
		return name
	}
	return "unknown"
}

// PermissionRequest represents a request to use a tool with a certain permission level
type PermissionRequest struct {
	ToolName string
	ToolDesc string
	Level    PermissionLevel
	Command  string // Optional: For bash commands
	FilePath string // Optional: For file operations
	URL      string // Optional: For network operations
}

// PermissionResponse represents the user's response to a permission request
type PermissionResponse int

const (
	// PermissionDenied indicates the user denied the permission request
	PermissionDenied PermissionResponse = iota

	// PermissionGrantedOnce indicates the user granted permission for this request only
	PermissionGrantedOnce

	// PermissionGrantedAlways indicates the user granted permission for all future similar requests
	PermissionGrantedAlways
)

// PromptFunc defines a function that prompts the user for permission
type PromptFunc func(req PermissionRequest) (PermissionResponse, error)

// PermissionManager handles tool permissions
type PermissionManager struct {
	mu              sync.RWMutex
	prompt          PromptFunc
	allowedTools    map[string]bool
	grantedCommands map[string]bool
	grantedPaths    map[string]bool
	grantedURLs     map[string]bool
	projectRoot     string
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager(prompt PromptFunc) (*PermissionManager, error) {
	// Determine project root
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return &PermissionManager{
		prompt:          prompt,
		allowedTools:    make(map[string]bool),
		grantedCommands: make(map[string]bool),
		grantedPaths:    make(map[string]bool),
		grantedURLs:     make(map[string]bool),
		projectRoot:     wd,
	}, nil
}

// AllowTool explicitly allows a tool without prompting
func (pm *PermissionManager) AllowTool(toolName string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.allowedTools[toolName] = true
}

// DisallowTool explicitly disallows a tool
func (pm *PermissionManager) DisallowTool(toolName string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.allowedTools[toolName] = false
}

// SetAllowedTools sets which tools are allowed
func (pm *PermissionManager) SetAllowedTools(tools []string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Reset the allowed tools
	pm.allowedTools = make(map[string]bool)

	// Add each tool
	for _, tool := range tools {
		pm.allowedTools[tool] = true
	}
}

// CheckPermission verifies if the tool is allowed to execute
func (pm *PermissionManager) CheckPermission(req PermissionRequest) error {
	pm.mu.RLock()

	// Read-only tools are always allowed without prompting
	if req.Level == PermissionReadOnly {
		pm.mu.RUnlock()
		return nil
	}

	// Check if the tool is explicitly allowed or denied
	if allowed, exists := pm.allowedTools[req.ToolName]; exists {
		if !allowed {
			pm.mu.RUnlock()
			return fmt.Errorf("tool '%s' is not allowed", req.ToolName)
		}

		// For shell commands, check if the command is allowed
		if req.Level == PermissionShellExec && req.Command != "" {
			if _, ok := pm.grantedCommands[req.Command]; ok {
				pm.mu.RUnlock()
				return nil
			}
		}

		// For file operations, check if the path is allowed
		if req.Level == PermissionFileWrite && req.FilePath != "" {
			absPath, err := filepath.Abs(req.FilePath)
			if err == nil {
				// Check for explicit path grants
				if _, ok := pm.grantedPaths[absPath]; ok {
					pm.mu.RUnlock()
					return nil
				}

				// Check for parent directory grants
				for path := range pm.grantedPaths {
					if strings.HasPrefix(absPath, path+"/") {
						pm.mu.RUnlock()
						return nil
					}
				}
			}
		}

		// For network operations, check if the URL is allowed
		if req.Level == PermissionNetwork && req.URL != "" {
			if _, ok := pm.grantedURLs[req.URL]; ok {
				pm.mu.RUnlock()
				return nil
			}

			// Check for domain grants
			urlDomain := extractDomain(req.URL)
			for url := range pm.grantedURLs {
				if urlDomain == extractDomain(url) {
					pm.mu.RUnlock()
					return nil
				}
			}
		}
	}

	pm.mu.RUnlock()

	// If we get here, we need to prompt for permission
	if pm.prompt == nil {
		return errors.New("permission denied (no prompt handler)")
	}

	response, err := pm.prompt(req)
	if err != nil {
		return err
	}

	if response == PermissionDenied {
		return errors.New("permission denied by user")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// For always grants, store the permission
	if response == PermissionGrantedAlways {
		// Store tool permission
		pm.allowedTools[req.ToolName] = true

		// Store command permission
		if req.Level == PermissionShellExec && req.Command != "" {
			pm.grantedCommands[req.Command] = true
		}

		// Store path permission
		if req.Level == PermissionFileWrite && req.FilePath != "" {
			// Canonicalize the path
			absPath, err := filepath.Abs(req.FilePath)
			if err == nil {
				pm.grantedPaths[absPath] = true
			}
		}

		// Store URL permission
		if req.Level == PermissionNetwork && req.URL != "" {
			pm.grantedURLs[req.URL] = true
		}
	}

	return nil
}

// IsSafePath checks if a file path is safe to access
func (pm *PermissionManager) IsSafePath(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Ensure path is within project root
	return strings.HasPrefix(absPath, pm.projectRoot)
}

// GetProjectRoot returns the project root path
func (pm *PermissionManager) GetProjectRoot() string {
	return pm.projectRoot
}

// Helper functions
func extractDomain(url string) string {
	// Simple domain extraction, could be improved
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "www.")

	// Get domain part (everything before first /)
	if idx := strings.Index(url, "/"); idx > 0 {
		url = url[:idx]
	}

	return url
}
