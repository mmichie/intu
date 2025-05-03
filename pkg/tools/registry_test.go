package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	
	"github.com/mmichie/intu/pkg/aikit"
)

func createMockTool(name string, level PermissionLevel) *MockTool {
	return &MockTool{
		BaseTool: BaseTool{
			ToolName:        name,
			ToolDescription: "Mock tool for testing",
			ToolParams:      map[string]interface{}{},
			PermLevel:       level,
		},
		ExecuteFunc: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			return name + " executed", nil
		},
	}
}

func createErrorMockTool(name string) *MockTool {
	return &MockTool{
		BaseTool: BaseTool{
			ToolName:        name,
			ToolDescription: "Mock tool that returns an error",
			ToolParams:      map[string]interface{}{},
			PermLevel:       PermissionReadOnly,
		},
		ExecuteFunc: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			return nil, errors.New("mock error")
		},
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	
	// Test successful registration
	tool1 := createMockTool("tool1", PermissionReadOnly)
	err := r.Register(tool1)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}
	
	// Test duplicate registration
	err = r.Register(tool1)
	if err == nil {
		t.Error("Register() expected error for duplicate tool, got nil")
	}
	
	// Test empty name
	emptyTool := createMockTool("", PermissionReadOnly)
	err = r.Register(emptyTool)
	if err == nil {
		t.Error("Register() expected error for empty tool name, got nil")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	tool1 := createMockTool("tool1", PermissionReadOnly)
	r.Register(tool1)
	
	// Test getting existing tool
	got, exists := r.Get("tool1")
	if !exists {
		t.Error("Get() exists = false, want true")
	}
	if got.Name() != "tool1" {
		t.Errorf("Get() = %v, want %v", got.Name(), "tool1")
	}
	
	// Test getting non-existent tool
	_, exists = r.Get("nonexistent")
	if exists {
		t.Error("Get() exists = true, want false")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	tool1 := createMockTool("tool1", PermissionReadOnly)
	tool2 := createMockTool("tool2", PermissionShellExec)
	r.Register(tool1)
	r.Register(tool2)
	
	// Test listing all tools
	list := r.List()
	if len(list) != 2 {
		t.Errorf("List() returned %d tools, want 2", len(list))
	}
	
	// Check that both tools are in the list
	names := make(map[string]bool)
	for _, tool := range list {
		names[tool.Name()] = true
	}
	
	if !names["tool1"] || !names["tool2"] {
		t.Errorf("List() returned tools %v, want to include tool1 and tool2", names)
	}
}

func TestRegistry_ListWithPermissionLevel(t *testing.T) {
	r := NewRegistry()
	tool1 := createMockTool("tool1", PermissionReadOnly)
	tool2 := createMockTool("tool2", PermissionReadOnly)
	tool3 := createMockTool("tool3", PermissionShellExec)
	r.Register(tool1)
	r.Register(tool2)
	r.Register(tool3)
	
	// Test listing read-only tools
	readOnlyTools := r.ListWithPermissionLevel(PermissionReadOnly)
	if len(readOnlyTools) != 2 {
		t.Errorf("ListWithPermissionLevel(PermissionReadOnly) returned %d tools, want 2", len(readOnlyTools))
	}
	
	// Test listing shell exec tools
	shellExecTools := r.ListWithPermissionLevel(PermissionShellExec)
	if len(shellExecTools) != 1 {
		t.Errorf("ListWithPermissionLevel(PermissionShellExec) returned %d tools, want 1", len(shellExecTools))
	}
	if shellExecTools[0].Name() != "tool3" {
		t.Errorf("ListWithPermissionLevel(PermissionShellExec)[0] = %v, want tool3", shellExecTools[0].Name())
	}
}

func TestRegistry_GetFunctionDefinitions(t *testing.T) {
	r := NewRegistry()
	tool1 := createMockTool("tool1", PermissionReadOnly)
	tool2 := createMockTool("tool2", PermissionShellExec)
	r.Register(tool1)
	r.Register(tool2)
	
	defs := r.GetFunctionDefinitions()
	if len(defs) != 2 {
		t.Errorf("GetFunctionDefinitions() returned %d definitions, want 2", len(defs))
	}
	
	// Check that the definitions match the tools
	for _, def := range defs {
		if def.Name != "tool1" && def.Name != "tool2" {
			t.Errorf("GetFunctionDefinitions() returned unexpected definition: %v", def.Name)
		}
		
		if def.Description != "Mock tool for testing" {
			t.Errorf("GetFunctionDefinitions() returned definition with wrong description: %v", def.Description)
		}
	}
}

func TestRegistry_ExecuteTool(t *testing.T) {
	r := NewRegistry()
	tool1 := createMockTool("tool1", PermissionReadOnly)
	errorTool := createErrorMockTool("errorTool")
	r.Register(tool1)
	r.Register(errorTool)
	
	// Test executing existing tool
	result, err := r.ExecuteTool(context.Background(), "tool1", nil)
	if err != nil {
		t.Errorf("ExecuteTool() error = %v", err)
	}
	if result != "tool1 executed" {
		t.Errorf("ExecuteTool() = %v, want %v", result, "tool1 executed")
	}
	
	// Test executing non-existent tool
	_, err = r.ExecuteTool(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("ExecuteTool() expected error for non-existent tool, got nil")
	}
	
	// Test executing tool that returns an error
	_, err = r.ExecuteTool(context.Background(), "errorTool", nil)
	if err == nil {
		t.Error("ExecuteTool() expected error from errorTool, got nil")
	}
}

func TestRegistry_ExecuteFunctionCall(t *testing.T) {
	r := NewRegistry()
	tool1 := createMockTool("tool1", PermissionReadOnly)
	errorTool := createErrorMockTool("errorTool")
	r.Register(tool1)
	r.Register(errorTool)
	
	// Test successful function call
	call := aikit.FunctionCall{
		Name:       "tool1",
		Parameters: json.RawMessage(`{}`),
	}
	
	response, err := r.ExecuteFunctionCall(context.Background(), call)
	if err != nil {
		t.Errorf("ExecuteFunctionCall() error = %v", err)
	}
	if response.Name != "tool1" {
		t.Errorf("ExecuteFunctionCall() response.Name = %v, want %v", response.Name, "tool1")
	}
	if response.Content != "tool1 executed" {
		t.Errorf("ExecuteFunctionCall() response.Content = %v, want %v", response.Content, "tool1 executed")
	}
	if response.Error != "" {
		t.Errorf("ExecuteFunctionCall() response.Error = %v, want empty", response.Error)
	}
	
	// Test function call that returns an error
	errorCall := aikit.FunctionCall{
		Name:       "errorTool",
		Parameters: json.RawMessage(`{}`),
	}
	
	errorResponse, _ := r.ExecuteFunctionCall(context.Background(), errorCall)
	if errorResponse.Error == "" {
		t.Error("ExecuteFunctionCall() expected error in response, got empty")
	}
	
	// Test non-existent function
	nonExistentCall := aikit.FunctionCall{
		Name:       "nonexistent",
		Parameters: json.RawMessage(`{}`),
	}
	
	nonExistentResponse, _ := r.ExecuteFunctionCall(context.Background(), nonExistentCall)
	if nonExistentResponse.Error == "" {
		t.Error("ExecuteFunctionCall() expected error in response for non-existent tool, got empty")
	}
}