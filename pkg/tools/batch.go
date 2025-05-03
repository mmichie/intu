package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// BatchInvocation represents a single tool invocation within a batch
type BatchInvocation struct {
	ToolName string          `json:"tool_name"`
	Input    json.RawMessage `json:"input"`
}

// BatchParams defines the parameters for the Batch tool
type BatchParams struct {
	Description string            `json:"description"`
	Invocations []BatchInvocation `json:"invocations"`
}

// BatchResult represents the result of executing a batch of tools
type BatchResult struct {
	Results []map[string]interface{} `json:"results"`
}

// BatchTool implements the Batch command for parallel tool execution
type BatchTool struct {
	BaseTool
	registry ToolRegistry // Reference to the tool registry interface
}

// NewBatchTool creates a new Batch tool with a reference to the registry
func NewBatchTool(registry ToolRegistry) *BatchTool {
	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"description": map[string]interface{}{
				"type":        "string",
				"description": "A short (3-5 word) description of the batch operation",
			},
			"invocations": map[string]interface{}{
				"type":        "array",
				"description": "The list of tool invocations to execute (required -- you MUST provide at least one tool invocation)",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"tool_name": map[string]interface{}{
							"type":        "string",
							"description": "The name of the tool to invoke",
						},
						"input": map[string]interface{}{
							"type":        "object",
							"description": "The input to pass to the tool",
						},
					},
					"required": []string{"tool_name", "input"},
				},
			},
		},
		"required": []string{"description", "invocations"},
	}

	return &BatchTool{
		BaseTool: BaseTool{
			ToolName:        "Batch",
			ToolDescription: "Batch execution tool that runs multiple tool invocations in a single request",
			ToolParams:      paramSchema,
			// Use the lowest permission level - individual tools will request permissions as needed
			PermLevel: PermissionReadOnly,
		},
		registry: registry,
	}
}

// Execute runs the Batch tool
func (t *BatchTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p BatchParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Validate parameters
	if len(p.Invocations) == 0 {
		return nil, fmt.Errorf("at least one tool invocation is required")
	}

	// Create channel for results to preserve order
	results := make([]map[string]interface{}, len(p.Invocations))

	// Use WaitGroup for parallel execution
	var wg sync.WaitGroup
	var mu sync.Mutex // For protecting error reporting
	var firstErr error

	for i, invocation := range p.Invocations {
		wg.Add(1)
		go func(idx int, inv BatchInvocation) {
			defer wg.Done()

			// Prepare result map for this invocation
			resultMap := map[string]interface{}{
				"tool_name": inv.ToolName,
			}

			// Execute the tool
			result, err := t.registry.ExecuteTool(ctx, inv.ToolName, inv.Input)

			if err != nil {
				// Store the error in the result map
				resultMap["error"] = err.Error()

				// Also store the first error encountered for potential reporting
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("tool %s failed: %w", inv.ToolName, err)
				}
				mu.Unlock()
			} else {
				// Store the successful result
				resultMap["result"] = result
			}

			// Store the result in its position in the results slice
			results[idx] = resultMap
		}(i, invocation)
	}

	// Wait for all executions to complete
	wg.Wait()

	return BatchResult{Results: results}, nil
}
