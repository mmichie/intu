package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

// mockTool is a test tool for batch testing
type mockTool struct {
	BaseTool
	sleep      time.Duration
	response   interface{}
	shouldFail bool
}

func newMockTool(name string, sleep time.Duration, response interface{}, shouldFail bool) *mockTool {
	return &mockTool{
		BaseTool: BaseTool{
			ToolName:        name,
			ToolDescription: "Mock tool for testing",
			ToolParams:      map[string]interface{}{},
			PermLevel:       PermissionReadOnly,
		},
		sleep:      sleep,
		response:   response,
		shouldFail: shouldFail,
	}
}

func (t *mockTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	// Simulate processing time
	time.Sleep(t.sleep)

	if t.shouldFail {
		return nil, fmt.Errorf("simulated error from tool %s", t.Name())
	}

	return t.response, nil
}

// mockRegistry implements the ToolRegistry interface for testing
type mockRegistry struct {
	tools map[string]Tool
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *mockRegistry) Register(tool Tool) error {
	r.tools[tool.Name()] = tool
	return nil
}

func (r *mockRegistry) ExecuteTool(ctx context.Context, name string, params json.RawMessage) (interface{}, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return tool.Execute(ctx, params)
}

func TestBatchTool_Execute(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "batch-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a registry and add some mock tools
	registry := newMockRegistry()

	// Register mock tools with different behaviors
	registry.Register(newMockTool("FastTool", 10*time.Millisecond, "fast result", false))
	registry.Register(newMockTool("SlowTool", 100*time.Millisecond, "slow result", false))
	registry.Register(newMockTool("FailingTool", 10*time.Millisecond, nil, true))

	// Create batch tool
	batchTool := NewBatchTool(registry)

	// Test cases
	testCases := []struct {
		name           string
		params         BatchParams
		expectError    bool
		expectedTools  []string
		expectedErrors int
	}{
		{
			name: "Single tool execution",
			params: BatchParams{
				Description: "Test single tool",
				Invocations: []BatchInvocation{
					{
						ToolName: "FastTool",
						Input:    json.RawMessage(`{}`),
					},
				},
			},
			expectError:    false,
			expectedTools:  []string{"FastTool"},
			expectedErrors: 0,
		},
		{
			name: "Multiple successful tools",
			params: BatchParams{
				Description: "Test multiple tools",
				Invocations: []BatchInvocation{
					{
						ToolName: "FastTool",
						Input:    json.RawMessage(`{}`),
					},
					{
						ToolName: "SlowTool",
						Input:    json.RawMessage(`{}`),
					},
				},
			},
			expectError:    false,
			expectedTools:  []string{"FastTool", "SlowTool"},
			expectedErrors: 0,
		},
		{
			name: "Mixed success and failure",
			params: BatchParams{
				Description: "Test mixed results",
				Invocations: []BatchInvocation{
					{
						ToolName: "FastTool",
						Input:    json.RawMessage(`{}`),
					},
					{
						ToolName: "FailingTool",
						Input:    json.RawMessage(`{}`),
					},
				},
			},
			expectError:    false, // The batch itself succeeds even if individual tools fail
			expectedTools:  []string{"FastTool", "FailingTool"},
			expectedErrors: 1,
		},
		{
			name: "Non-existent tool",
			params: BatchParams{
				Description: "Test non-existent tool",
				Invocations: []BatchInvocation{
					{
						ToolName: "NonExistentTool",
						Input:    json.RawMessage(`{}`),
					},
				},
			},
			expectError:    false, // The batch succeeds but the tool execution fails
			expectedTools:  []string{"NonExistentTool"},
			expectedErrors: 1,
		},
		{
			name: "Empty invocations",
			params: BatchParams{
				Description: "Test empty invocations",
				Invocations: []BatchInvocation{},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsJSON, err := json.Marshal(tc.params)
			if err != nil {
				t.Fatalf("Failed to marshal params: %v", err)
			}

			startTime := time.Now()
			result, err := batchTool.Execute(context.Background(), paramsJSON)
			duration := time.Since(startTime)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check result type
			batchResult, ok := result.(BatchResult)
			if !ok {
				t.Fatalf("Expected result type BatchResult, got %T", result)
			}

			// Check number of results
			if len(batchResult.Results) != len(tc.expectedTools) {
				t.Errorf("Expected %d results, got %d", len(tc.expectedTools), len(batchResult.Results))
			}

			// Count errors
			errorCount := 0
			for i, resultMap := range batchResult.Results {
				// Check tool name
				toolName, _ := resultMap["tool_name"].(string)
				if toolName != tc.expectedTools[i] {
					t.Errorf("Expected tool %s at position %d, got %s", tc.expectedTools[i], i, toolName)
				}

				// Count errors
				if _, hasError := resultMap["error"]; hasError {
					errorCount++
				}
			}

			if errorCount != tc.expectedErrors {
				t.Errorf("Expected %d errors, got %d", tc.expectedErrors, errorCount)
			}

			// For parallel tests with both fast and slow tools, verify that it didn't take as long as sequential execution
			if tc.name == "Multiple successful tools" {
				expectedSequentialTime := 110 * time.Millisecond // 10ms + 100ms
				if duration >= expectedSequentialTime {
					t.Errorf("Expected parallel execution to be faster than sequential. Got %v", duration)
				}
			}
		})
	}
}
