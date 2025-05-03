package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mmichie/intu/pkg/taskmanager"
)

// TodoItemParams defines the structure for individual todo items
type TodoItemParams struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

// TodoWriteParams defines the parameters for the TodoWrite tool
type TodoWriteParams struct {
	Todos []TodoItemParams `json:"todos"`
}

// TodoWriteResult represents the result of writing todo items
type TodoWriteResult struct {
	Success  bool                `json:"success"`
	Message  string              `json:"message"`
	NewTodos []*taskmanager.Task `json:"newTodos"`
}

// TodoWriteTool implements the TodoWrite tool
type TodoWriteTool struct {
	BaseTool
	taskManager *taskmanager.TaskManager
}

// NewTodoWriteTool creates a new TodoWrite tool
func NewTodoWriteTool(taskManager *taskmanager.TaskManager) *TodoWriteTool {
	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"todos": map[string]interface{}{
				"type":        "array",
				"description": "The updated todo list",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "string",
						},
						"content": map[string]interface{}{
							"type":      "string",
							"minLength": 1,
						},
						"status": map[string]interface{}{
							"type": "string",
							"enum": []string{
								taskmanager.StatusPending,
								taskmanager.StatusInProgress,
								taskmanager.StatusCompleted,
							},
						},
						"priority": map[string]interface{}{
							"type": "string",
							"enum": []string{
								taskmanager.PriorityHigh,
								taskmanager.PriorityMedium,
								taskmanager.PriorityLow,
							},
						},
					},
					"required": []string{"content", "status", "priority", "id"},
				},
			},
		},
		"required": []string{"todos"},
	}

	return &TodoWriteTool{
		BaseTool: BaseTool{
			ToolName: "TodoWrite",
			ToolDescription: `Use this tool to create and manage a structured task list for your current coding session. This helps you track progress, organize complex tasks, and demonstrate thoroughness to the user.
It also helps the user understand the progress of the task and overall progress of their requests.

## When to Use This Tool
Use this tool proactively in these scenarios:

1. Complex multi-step tasks - When a task requires 3 or more distinct steps or actions
2. Non-trivial and complex tasks - Tasks that require careful planning or multiple operations
3. User explicitly requests todo list - When the user directly asks you to use the todo list
4. User provides multiple tasks - When users provide a list of things to be done (numbered or comma-separated)
5. After receiving new instructions - Immediately capture user requirements as todos
6. After completing a task - Mark it complete and add any new follow-up tasks
7. When you start working on a new task, mark the todo as in_progress. Ideally you should only have one todo as in_progress at a time. Complete existing tasks before starting new ones.

## When NOT to Use This Tool

Skip using this tool when:
1. There is only a single, straightforward task
2. The task is trivial and tracking it provides no organizational benefit
3. The task can be completed in less than 3 trivial steps
4. The task is purely conversational or informational

NOTE that you should use should not use this tool if there is only one trivial task to do. In this case you are better off just doing the task directly.

## Task States and Management

1. **Task States**: Use these states to track progress:
   - pending: Task not yet started
   - in_progress: Currently working on (limit to ONE task at a time)
   - completed: Task finished successfully
   - cancelled: Task no longer needed

2. **Task Management**:
   - Update task status in real-time as you work
   - Mark tasks complete IMMEDIATELY after finishing (don't batch completions)
   - Only have ONE task in_progress at any time
   - Complete current tasks before starting new ones
   - Cancel tasks that become irrelevant

3. **Task Breakdown**:
   - Create specific, actionable items
   - Break complex tasks into smaller, manageable steps
   - Use clear, descriptive task names

When in doubt, use this tool. Being proactive with task management demonstrates attentiveness and ensures you complete all requirements successfully.`,
			ToolParams: paramSchema,
			PermLevel:  PermissionReadOnly,
		},
		taskManager: taskManager,
	}
}

// Execute runs the TodoWrite tool
func (t *TodoWriteTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p TodoWriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Validate parameters
	if len(p.Todos) == 0 {
		return nil, fmt.Errorf("at least one todo item is required")
	}

	// Load current tasks
	if err := t.taskManager.LoadFromContext(); err != nil {
		// Load error is not critical if no tasks exist yet
		if !strings.Contains(err.Error(), "context not found") {
			return nil, fmt.Errorf("failed to load tasks: %w", err)
		}
	}

	// Get existing tasks first
	currentTasks, _ := t.taskManager.ListTasks("")
	currentTaskMap := make(map[string]*taskmanager.Task)
	for _, task := range currentTasks {
		currentTaskMap[task.ID] = task
	}

	// Process each todo item
	var messages []string
	var newTasks []*taskmanager.Task

	for _, todoItem := range p.Todos {
		// Check if this is an existing task
		if _, exists := currentTaskMap[todoItem.ID]; exists {
			// Update existing task
			task, err := t.taskManager.UpdateTask(
				todoItem.ID,
				todoItem.Content,
				todoItem.Status,
				todoItem.Priority,
			)
			if err != nil {
				messages = append(messages, fmt.Sprintf("Failed to update task %s: %v", todoItem.ID, err))
				continue
			}
			newTasks = append(newTasks, task)
			messages = append(messages, fmt.Sprintf("Updated task: %s", task.Content))
		} else {
			// Create new task
			task, err := t.taskManager.CreateTask(todoItem.Content, todoItem.Priority)
			if err != nil {
				messages = append(messages, fmt.Sprintf("Failed to create task: %v", err))
				continue
			}

			// If status is not pending, update it
			if todoItem.Status != taskmanager.StatusPending {
				task, err = t.taskManager.UpdateTask(task.ID, "", todoItem.Status, "")
				if err != nil {
					messages = append(messages, fmt.Sprintf("Failed to set task status: %v", err))
					continue
				}
			}

			newTasks = append(newTasks, task)
			messages = append(messages, fmt.Sprintf("Created task: %s", task.Content))
		}
	}

	// Get updated task list for the response
	updatedTasks, err := t.taskManager.ListTasks("")
	if err != nil {
		return nil, fmt.Errorf("failed to get updated task list: %w", err)
	}

	// Return results
	return TodoWriteResult{
		Success:  true,
		Message:  "Todos have been modified successfully. Ensure that you continue to read and update the todo list as you work on tasks.",
		NewTodos: updatedTasks,
	}, nil
}
