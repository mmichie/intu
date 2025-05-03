package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mmichie/intu/pkg/taskmanager"
)

// TodoReadParams defines the parameters for the TodoRead tool
// This tool doesn't need parameters, but we define an empty struct
// to maintain consistency with the tool interface
type TodoReadParams struct{}

// TodoReadResult represents the result of reading todo items
type TodoReadResult struct {
	Todos []*taskmanager.Task `json:"todos"`
}

// TodoReadTool implements the TodoRead tool
type TodoReadTool struct {
	BaseTool
	taskManager *taskmanager.TaskManager
}

// NewTodoReadTool creates a new TodoRead tool
func NewTodoReadTool(taskManager *taskmanager.TaskManager) *TodoReadTool {
	// Create an empty parameter schema since this tool doesn't need parameters
	paramSchema := map[string]interface{}{
		"type":                 "object",
		"properties":           map[string]interface{}{},
		"additionalProperties": false,
		"description":          "No input is required, leave this field blank.",
	}

	return &TodoReadTool{
		BaseTool: BaseTool{
			ToolName: "TodoRead",
			ToolDescription: `Use this tool to read the current to-do list for the session. This tool should be used proactively and frequently to ensure that you are aware of
the status of the current task list. You should make use of this tool as often as possible, especially in the following situations:
- At the beginning of conversations to see what's pending
- Before starting new tasks to prioritize work
- When the user asks about previous tasks or plans
- Whenever you're uncertain about what to do next
- After completing tasks to update your understanding of remaining work
- After every few messages to ensure you're on track

Usage:
- This tool takes in no parameters. So leave the input blank or empty. DO NOT include a dummy object, placeholder string or a key like "input" or "empty". LEAVE IT BLANK.
- Returns a list of todo items with their status, priority, and content
- Use this information to track progress and plan next steps
- If no todos exist yet, an empty list will be returned`,
			ToolParams: paramSchema,
			PermLevel:  PermissionReadOnly,
		},
		taskManager: taskManager,
	}
}

// Execute runs the TodoRead tool
func (t *TodoReadTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	// Load tasks from context first to ensure we have the latest data
	if err := t.taskManager.LoadFromContext(); err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	// Get all tasks
	tasks, err := t.taskManager.ListTasks("")
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Return tasks as the result
	return TodoReadResult{
		Todos: tasks,
	}, nil
}
