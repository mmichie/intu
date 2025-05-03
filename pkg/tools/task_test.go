package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/mmichie/intu/pkg/aikit"
)

var ErrGenerationFailed = errors.New("generation failed")

// mockProvider is a mock implementation of the Provider interface
type mockProvider struct {
	responseText  string
	shouldFail    bool
	functionCalls []aikit.FunctionCall
}

func newMockProvider(responseText string, shouldFail bool, functionCalls []aikit.FunctionCall) *mockProvider {
	return &mockProvider{
		responseText:  responseText,
		shouldFail:    shouldFail,
		functionCalls: functionCalls,
	}
}

func (p *mockProvider) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	return p.responseText, nil
}

func (p *mockProvider) GenerateResponseWithFunctions(
	ctx context.Context,
	prompt string,
	functionExecutor aikit.FunctionExecutorFunc,
) (string, error) {
	// If configured to fail, return error
	if p.shouldFail {
		return "", ErrGenerationFailed
	}

	// If we have function calls to make, execute them
	for _, call := range p.functionCalls {
		resp, _ := functionExecutor(call)
		// In a real implementation, we might use the response
		// but for testing we just execute the function
		_ = resp
	}

	// Return the final response
	return p.responseText, nil
}

func (p *mockProvider) RegisterFunctions(functions []aikit.FunctionDefinition) {
	// Do nothing for the mock
}

// mockToolForTask is a simple implementation for testing the task tool
type mockToolForTask struct {
	BaseTool
	response interface{}
}

func newMockToolForTask(name string, response interface{}) *mockToolForTask {
	return &mockToolForTask{
		BaseTool: BaseTool{
			ToolName:        name,
			ToolDescription: "Mock tool for testing",
			ToolParams:      map[string]interface{}{},
			PermLevel:       PermissionReadOnly,
		},
		response: response,
	}
}

func (t *mockToolForTask) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return t.response, nil
}

// mockRegistryForTask is a simple implementation for testing the task tool
type mockRegistryForTask struct {
	tools map[string]Tool
}

func newMockRegistryForTask() *mockRegistryForTask {
	return &mockRegistryForTask{
		tools: make(map[string]Tool),
	}
}

func (r *mockRegistryForTask) Register(tool Tool) error {
	r.tools[tool.Name()] = tool
	return nil
}

func (r *mockRegistryForTask) ExecuteTool(ctx context.Context, name string, params json.RawMessage) (interface{}, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return tool.Execute(ctx, params)
}

func TestTaskTool_Execute(t *testing.T) {
	// Create registry with tools
	registry := newMockRegistryForTask()
	registry.Register(newMockToolForTask("TestTool", "test result"))

	// Test cases
	testCases := []struct {
		name           string
		description    string
		prompt         string
		provider       *mockProvider
		expectError    bool
		expectedResult string
	}{
		{
			name:           "Simple task execution",
			description:    "Test task",
			prompt:         "Run a simple test",
			provider:       newMockProvider("Task completed successfully", false, nil),
			expectError:    false,
			expectedResult: "Task completed successfully",
		},
		{
			name:        "Task with function calls",
			description: "Test task with function calls",
			prompt:      "Run a test tool",
			provider: newMockProvider(
				"Task completed with test tool",
				false,
				[]aikit.FunctionCall{
					{
						Name:       "TestTool",
						Parameters: json.RawMessage(`{}`),
					},
				},
			),
			expectError:    false,
			expectedResult: "Task completed with test tool",
		},
		{
			name:        "Failed task execution",
			description: "Failing task",
			prompt:      "This will fail",
			provider:    newMockProvider("", true, nil),
			expectError: true,
		},
		{
			name:        "Missing prompt",
			description: "Test task",
			prompt:      "", // Empty prompt
			provider:    newMockProvider("", false, nil),
			expectError: true,
		},
		{
			name:        "Missing description",
			description: "", // Empty description
			prompt:      "Test task",
			provider:    newMockProvider("", false, nil),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create task tool with mock provider
			taskTool := NewTaskTool(registry, tc.provider)

			// Create task parameters
			params := TaskParams{
				Description: tc.description,
				Prompt:      tc.prompt,
			}

			// Marshal parameters
			paramsJSON, err := json.Marshal(params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			// Execute task
			result, err := taskTool.Execute(context.Background(), paramsJSON)

			// Check error
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check result
			taskResult, ok := result.(TaskResult)
			if !ok {
				t.Fatalf("Expected result type TaskResult, got %T", result)
			}

			if taskResult.Result != tc.expectedResult {
				t.Errorf("Expected result %q, got %q", tc.expectedResult, taskResult.Result)
			}
		})
	}
}
