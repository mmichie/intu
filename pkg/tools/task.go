package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/providers"
)

// TaskParams defines the parameters for the Task tool
type TaskParams struct {
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
}

// TaskResult represents the result of executing a task
type TaskResult struct {
	Result string `json:"result"`
}

// TaskTool implements the Task command for spawning a sub-agent
type TaskTool struct {
	BaseTool
	registry       ToolRegistry
	providerClient aikit.Provider
}

// NewTaskTool creates a new Task tool with a reference to the registry
func NewTaskTool(registry ToolRegistry, providerClient aikit.Provider) *TaskTool {
	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"description": map[string]interface{}{
				"type":        "string",
				"description": "A short (3-5 word) description of the task",
			},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "The task for the agent to perform",
			},
		},
		"required": []string{"description", "prompt"},
	}

	return &TaskTool{
		BaseTool: BaseTool{
			ToolName:        "Task",
			ToolDescription: "Launch a new agent that has access to tools for more complex operations",
			ToolParams:      paramSchema,
			// Use the lowest permission level - individual tools will request permissions as needed
			PermLevel: PermissionReadOnly,
		},
		registry:       registry,
		providerClient: providerClient,
	}
}

// Execute runs the Task tool
func (t *TaskTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p TaskParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Validate parameters
	if p.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	if p.Description == "" {
		return nil, fmt.Errorf("description is required")
	}

	// Get function definitions from the registry
	functionDefs := getFunctionDefinitionsFromRegistry(t.registry)

	// Create the prompt for the agent
	prompt := formatAgentPrompt(p.Prompt, functionDefs)

	// Set a timeout for the agent execution
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Execute the agent using the provider with function calling
	result, err := executeAgentWithFunctions(ctx, t.providerClient, prompt, t.registry)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	return TaskResult{
		Result: result,
	}, nil
}

// Helper function to get function definitions from the registry
func getFunctionDefinitionsFromRegistry(registry ToolRegistry) []providers.FunctionDefinition {
	// This is just a placeholder until we define how to get the function definitions
	// In a real implementation, this would query the registry for all available tools
	// and convert them to function definitions

	// For now, we're assuming registry is a *Registry that implements ToolRegistry
	// In a production system, we would properly define this interface
	if reg, ok := registry.(*Registry); ok {
		// Convert from aikit.FunctionDefinition to providers.FunctionDefinition
		aiDefs := reg.GetFunctionDefinitions()
		result := make([]providers.FunctionDefinition, 0, len(aiDefs))

		for _, def := range aiDefs {
			result = append(result, providers.FunctionDefinition{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  def.Parameters,
			})
		}

		return result
	}

	// Fallback to empty list if registry doesn't support getting function definitions
	return []providers.FunctionDefinition{}
}

// Helper function to format the agent prompt
func formatAgentPrompt(userPrompt string, functions []providers.FunctionDefinition) string {
	// Build a description of available tools
	var toolDescriptions strings.Builder
	for _, fn := range functions {
		toolDescriptions.WriteString(fmt.Sprintf("- %s: %s\n", fn.Name, fn.Description))
	}

	// Format the complete prompt
	prompt := fmt.Sprintf(`You are a helpful assistant that can use tools to solve tasks. 
You have access to the following tools:

%s

Your task is:
%s

Think step by step about how to solve this task using the available tools.
When using tools, make sure to format your tool invocations correctly.
Focus on completing the task accurately and efficiently.

Respond with your approach and the results of your tool executions.
`, toolDescriptions.String(), userPrompt)

	return prompt
}

// Helper function to execute an agent with function calling
func executeAgentWithFunctions(ctx context.Context, provider aikit.Provider, prompt string, registry ToolRegistry) (string, error) {
	// This function will handle the iterative process of the agent:
	// 1. Send the prompt to the provider
	// 2. If the provider returns a function call, execute it using the registry
	// 3. Send the function result back to the provider
	// 4. Repeat until the provider returns a final text response

	// Create a function executor to process function calls
	functionExecutor := func(call providers.FunctionCall) (providers.FunctionResponse, error) {
		// Convert function call to registry tool execution
		result, err := registry.ExecuteTool(ctx, call.Name, call.Parameters)

		response := providers.FunctionResponse{
			Name:    call.Name,
			Content: result,
		}

		if err != nil {
			response.Error = err.Error()
		}

		return response, nil
	}

	response, err := provider.GenerateResponseWithFunctions(ctx, prompt, functionExecutor)

	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	return response, nil
}
