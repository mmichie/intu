package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	
	"github.com/mmichie/intu/pkg/aikit"
)

// Registry manages the collection of available tools
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool with name %q already registered", name)
	}
	
	r.tools[name] = tool
	return nil
}

// Get returns a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	
	return result
}

// ListWithPermissionLevel returns tools with the specified permission level
func (r *Registry) ListWithPermissionLevel(level PermissionLevel) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var result []Tool
	for _, tool := range r.tools {
		if tool.GetPermissionLevel() == level {
			result = append(result, tool)
		}
	}
	
	return result
}

// GetFunctionDefinitions returns all tools as function definitions
func (r *Registry) GetFunctionDefinitions() []aikit.FunctionDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make([]aikit.FunctionDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool.ToFunctionDefinition())
	}
	
	return result
}

// ExecuteTool executes a tool by name
func (r *Registry) ExecuteTool(ctx context.Context, name string, params json.RawMessage) (interface{}, error) {
	tool, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	
	return tool.Execute(ctx, params)
}

// ExecuteFunctionCall executes a function call using the appropriate tool
func (r *Registry) ExecuteFunctionCall(ctx context.Context, call aikit.FunctionCall) (aikit.FunctionResponse, error) {
	result, err := r.ExecuteTool(ctx, call.Name, call.Parameters)
	
	response := aikit.FunctionResponse{
		Name:    call.Name,
		Content: result,
	}
	
	if err != nil {
		response.Error = err.Error()
	}
	
	return response, nil
}