package tools

import (
	"context"
	"encoding/json"
	"testing"

	contextPkg "github.com/mmichie/intu/pkg/context"
	"github.com/mmichie/intu/pkg/taskmanager"
)

func TestTodoWriteTool(t *testing.T) {
	// Create a context store for the task manager
	store := contextPkg.NewMemoryContextStore()

	// Create a task manager
	tm := taskmanager.NewTaskManager(store, "test-tasks")

	// Create a TodoWrite tool
	todoWriteTool := NewTodoWriteTool(tm)

	// Verify tool properties
	if todoWriteTool.Name() != "TodoWrite" {
		t.Errorf("Expected tool name to be 'TodoWrite', got '%s'", todoWriteTool.Name())
	}

	if todoWriteTool.GetPermissionLevel() != PermissionReadOnly {
		t.Errorf("Expected permission level to be ReadOnly, got %v", todoWriteTool.GetPermissionLevel())
	}

	// Create test parameters with one new task
	params := TodoWriteParams{
		Todos: []TodoItemParams{
			{
				ID:       "1",
				Content:  "Test new task",
				Status:   taskmanager.StatusPending,
				Priority: taskmanager.PriorityHigh,
			},
		},
	}

	// Convert parameters to JSON
	paramsJson, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal parameters: %v", err)
	}

	// Execute the TodoWrite tool
	result, err := todoWriteTool.Execute(context.Background(), paramsJson)
	if err != nil {
		t.Fatalf("Failed to execute TodoWrite tool: %v", err)
	}

	writeResult, ok := result.(TodoWriteResult)
	if !ok {
		t.Fatalf("Expected result to be TodoWriteResult, got %T", result)
	}

	if !writeResult.Success {
		t.Errorf("Expected success to be true, got false")
	}

	if len(writeResult.NewTodos) != 1 {
		t.Errorf("Expected 1 todo in the result, got %d", len(writeResult.NewTodos))
	}

	// Verify task was created
	tasks, err := tm.ListTasks("")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	if tasks[0].Content != "Test new task" {
		t.Errorf("Expected task content to be 'Test new task', got '%s'", tasks[0].Content)
	}

	// Test updating an existing task
	// First, we need to get the real ID
	existingTaskID := tasks[0].ID

	// Create update parameters
	updateParams := TodoWriteParams{
		Todos: []TodoItemParams{
			{
				ID:       existingTaskID,
				Content:  "Updated task",
				Status:   taskmanager.StatusInProgress,
				Priority: taskmanager.PriorityMedium,
			},
		},
	}

	// Convert parameters to JSON
	updateParamsJson, err := json.Marshal(updateParams)
	if err != nil {
		t.Fatalf("Failed to marshal update parameters: %v", err)
	}

	// Execute the TodoWrite tool to update the task
	updateResult, err := todoWriteTool.Execute(context.Background(), updateParamsJson)
	if err != nil {
		t.Fatalf("Failed to execute TodoWrite tool for update: %v", err)
	}

	updateWriteResult, ok := updateResult.(TodoWriteResult)
	if !ok {
		t.Fatalf("Expected update result to be TodoWriteResult, got %T", updateResult)
	}

	if !updateWriteResult.Success {
		t.Errorf("Expected update success to be true, got false")
	}

	// Verify task was updated
	updatedTasks, err := tm.ListTasks("")
	if err != nil {
		t.Fatalf("Failed to list updated tasks: %v", err)
	}

	if len(updatedTasks) != 1 {
		t.Errorf("Expected 1 updated task, got %d", len(updatedTasks))
	}

	if updatedTasks[0].Content != "Updated task" {
		t.Errorf("Expected content to be 'Updated task', got '%s'", updatedTasks[0].Content)
	}

	if updatedTasks[0].Status != taskmanager.StatusInProgress {
		t.Errorf("Expected status to be 'in_progress', got '%s'", updatedTasks[0].Status)
	}

	if updatedTasks[0].Priority != taskmanager.PriorityMedium {
		t.Errorf("Expected priority to be 'medium', got '%s'", updatedTasks[0].Priority)
	}

	// Test creating multiple tasks at once
	multiParams := TodoWriteParams{
		Todos: []TodoItemParams{
			{
				ID:       existingTaskID,
				Content:  "Updated task",
				Status:   taskmanager.StatusInProgress,
				Priority: taskmanager.PriorityMedium,
			},
			{
				ID:       "new1",
				Content:  "New task 1",
				Status:   taskmanager.StatusPending,
				Priority: taskmanager.PriorityHigh,
			},
			{
				ID:       "new2",
				Content:  "New task 2",
				Status:   taskmanager.StatusCompleted,
				Priority: taskmanager.PriorityLow,
			},
		},
	}

	// Convert parameters to JSON
	multiParamsJson, err := json.Marshal(multiParams)
	if err != nil {
		t.Fatalf("Failed to marshal multi parameters: %v", err)
	}

	// Execute the TodoWrite tool with multiple tasks
	multiResult, err := todoWriteTool.Execute(context.Background(), multiParamsJson)
	if err != nil {
		t.Fatalf("Failed to execute TodoWrite tool for multiple tasks: %v", err)
	}

	multiWriteResult, ok := multiResult.(TodoWriteResult)
	if !ok {
		t.Fatalf("Expected multi result to be TodoWriteResult, got %T", multiResult)
	}

	if !multiWriteResult.Success {
		t.Errorf("Expected multi success to be true, got false")
	}

	// Verify we now have 3 tasks
	finalTasks, err := tm.ListTasks("")
	if err != nil {
		t.Fatalf("Failed to list final tasks: %v", err)
	}

	if len(finalTasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(finalTasks))
	}

	// Check that all statuses are as expected
	completedCount := 0
	pendingCount := 0
	inProgressCount := 0

	for _, task := range finalTasks {
		switch task.Status {
		case taskmanager.StatusCompleted:
			completedCount++
		case taskmanager.StatusPending:
			pendingCount++
		case taskmanager.StatusInProgress:
			inProgressCount++
		}
	}

	if completedCount != 1 {
		t.Errorf("Expected 1 completed task, got %d", completedCount)
	}

	if pendingCount != 1 {
		t.Errorf("Expected 1 pending task, got %d", pendingCount)
	}

	if inProgressCount != 1 {
		t.Errorf("Expected 1 in-progress task, got %d", inProgressCount)
	}
}
