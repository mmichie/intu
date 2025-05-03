package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mmichie/intu/pkg/aikit"
)

// MockTool implements the Tool interface for testing
type MockTool struct {
	BaseTool
	ExecuteFunc func(ctx context.Context, params json.RawMessage) (interface{}, error)
}

// Execute calls the mock function
func (m *MockTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return m.ExecuteFunc(ctx, params)
}

func TestPermissionLevel_String(t *testing.T) {
	tests := []struct {
		level PermissionLevel
		want  string
	}{
		{PermissionReadOnly, "read-only"},
		{PermissionShellExec, "shell-execution"},
		{PermissionFileWrite, "file-write"},
		{PermissionNetwork, "network"},
		{PermissionLevel(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("PermissionLevel.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseTool_Methods(t *testing.T) {
	baseTool := BaseTool{
		ToolName:        "test_tool",
		ToolDescription: "A test tool",
		ToolParams: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "A test parameter",
				},
			},
		},
		PermLevel: PermissionReadOnly,
	}

	t.Run("Name", func(t *testing.T) {
		if got := baseTool.Name(); got != "test_tool" {
			t.Errorf("BaseTool.Name() = %v, want %v", got, "test_tool")
		}
	})

	t.Run("Description", func(t *testing.T) {
		if got := baseTool.Description(); got != "A test tool" {
			t.Errorf("BaseTool.Description() = %v, want %v", got, "A test tool")
		}
	})

	t.Run("ParameterSchema", func(t *testing.T) {
		got := baseTool.ParameterSchema()
		if !reflect.DeepEqual(got, baseTool.ToolParams) {
			t.Errorf("BaseTool.ParameterSchema() = %v, want %v", got, baseTool.ToolParams)
		}
	})

	t.Run("GetPermissionLevel", func(t *testing.T) {
		if got := baseTool.GetPermissionLevel(); got != PermissionReadOnly {
			t.Errorf("BaseTool.GetPermissionLevel() = %v, want %v", got, PermissionReadOnly)
		}
	})

	t.Run("ToFunctionDefinition", func(t *testing.T) {
		got := baseTool.ToFunctionDefinition()
		want := aikit.FunctionDefinition{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters:  baseTool.ToolParams,
		}

		if got.Name != want.Name {
			t.Errorf("ToFunctionDefinition().Name = %v, want %v", got.Name, want.Name)
		}

		if got.Description != want.Description {
			t.Errorf("ToFunctionDefinition().Description = %v, want %v", got.Description, want.Description)
		}

		if !reflect.DeepEqual(got.Parameters, want.Parameters) {
			t.Errorf("ToFunctionDefinition().Parameters = %v, want %v", got.Parameters, want.Parameters)
		}
	})
}

func TestMockTool(t *testing.T) {
	mockExecute := func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "mock result", nil
	}

	tool := &MockTool{
		BaseTool: BaseTool{
			ToolName:        "mock_tool",
			ToolDescription: "A mock tool",
			ToolParams:      map[string]interface{}{},
			PermLevel:       PermissionReadOnly,
		},
		ExecuteFunc: mockExecute,
	}

	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Errorf("MockTool.Execute() error = %v", err)
		return
	}

	if result != "mock result" {
		t.Errorf("MockTool.Execute() = %v, want %v", result, "mock result")
	}
}
