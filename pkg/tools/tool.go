// Package tools provides the tool system for AI-assisted code editing
package tools

import (
	"context"
	"encoding/json"

	"github.com/mmichie/intu/pkg/aikit"
	securityPkg "github.com/mmichie/intu/pkg/security"
)

// PermissionLevel defines the security level of a tool
type PermissionLevel = securityPkg.PermissionLevel

// Permission levels
const (
	PermissionReadOnly  = securityPkg.PermissionReadOnly
	PermissionShellExec = securityPkg.PermissionShellExec
	PermissionFileWrite = securityPkg.PermissionFileWrite
	PermissionNetwork   = securityPkg.PermissionNetwork
)

// Tool defines the interface for all tools
type Tool interface {
	// Name returns the unique identifier for this tool
	Name() string

	// Description returns the tool's description
	Description() string

	// ParameterSchema returns the JSON schema for the tool's parameters
	ParameterSchema() map[string]interface{}

	// GetPermissionLevel returns the permission level required to use this tool
	GetPermissionLevel() PermissionLevel

	// Execute runs the tool with the given parameters
	Execute(ctx context.Context, params json.RawMessage) (interface{}, error)

	// ToFunctionDefinition converts the tool to a function definition
	ToFunctionDefinition() aikit.FunctionDefinition
}

// BaseTool implements common Tool functionality
type BaseTool struct {
	ToolName        string
	ToolDescription string
	ToolParams      map[string]interface{}
	PermLevel       PermissionLevel
}

// Name returns the tool's name
func (b *BaseTool) Name() string {
	return b.ToolName
}

// Description returns the tool's description
func (b *BaseTool) Description() string {
	return b.ToolDescription
}

// ParameterSchema returns the JSON schema for the tool's parameters
func (b *BaseTool) ParameterSchema() map[string]interface{} {
	return b.ToolParams
}

// GetPermissionLevel returns the permission level required to use this tool
func (b *BaseTool) GetPermissionLevel() PermissionLevel {
	return b.PermLevel
}

// ToFunctionDefinition converts the tool to a function definition
func (b *BaseTool) ToFunctionDefinition() aikit.FunctionDefinition {
	return aikit.FunctionDefinition{
		Name:        b.ToolName,
		Description: b.ToolDescription,
		Parameters:  b.ToolParams,
	}
}
