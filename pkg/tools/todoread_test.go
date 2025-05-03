package tools

import (
	"context"
	"encoding/json"
	"testing"

	contextPkg "github.com/mmichie/intu/pkg/context"
	"github.com/mmichie/intu/pkg/taskmanager"
)

func TestTodoReadTool(t *testing.T) {
	// Create a context store for the task manager
	store := contextPkg.NewMemoryContextStore()

	// Create a task manager
	tm := taskmanager.NewTaskManager(store, "test-tasks")

	// Create a TodoRead tool
	todoReadTool := NewTodoReadTool(tm)

	// Verify tool properties
	if todoReadTool.Name() != "TodoRead" {
		t.Errorf("Expected tool name to be 'TodoRead', got '%s'", todoReadTool.Name())
	}

	if todoReadTool.GetPermissionLevel() != PermissionReadOnly {
		t.Errorf("Expected permission level to be ReadOnly, got %v", todoReadTool.GetPermissionLevel())
	}

	// Test with empty task list
	result, err := todoReadTool.Execute(context.Background(), json.RawMessage("{}"))
	if err != nil {
		t.Fatalf("Failed to execute TodoRead tool: %v", err)
	}

	todosResult, ok := result.(TodoReadResult)
	if !ok {
		t.Fatalf("Expected result to be TodoReadResult, got %T", result)
	}

	if len(todosResult.Todos) != 0 {
		t.Errorf("Expected empty todo list, got %d todos", len(todosResult.Todos))
	}

	// Add some tasks
	task1, err := tm.CreateTask("Test task 1", taskmanager.PriorityHigh)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	task2, err := tm.CreateTask("Test task 2", taskmanager.PriorityMedium)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Update task status
	_, err = tm.UpdateTask(task2.ID, "", taskmanager.StatusInProgress, "")
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	// Test reading tasks
	result, err = todoReadTool.Execute(context.Background(), json.RawMessage("{}"))
	if err != nil {
		t.Fatalf("Failed to execute TodoRead tool: %v", err)
	}

	todosResult, ok = result.(TodoReadResult)
	if !ok {
		t.Fatalf("Expected result to be TodoReadResult, got %T", result)
	}

	if len(todosResult.Todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todosResult.Todos))
	}

	// Verify task data
	var foundTask1, foundTask2 bool
	for _, task := range todosResult.Todos {
		if task.ID == task1.ID {
			if task.Content != "Test task 1" {
				t.Errorf("Expected task content to be 'Test task 1', got '%s'", task.Content)
			}
			if task.Status != taskmanager.StatusPending {
				t.Errorf("Expected task status to be 'pending', got '%s'", task.Status)
			}
			if task.Priority != taskmanager.PriorityHigh {
				t.Errorf("Expected task priority to be 'high', got '%s'", task.Priority)
			}
			foundTask1 = true
		}

		if task.ID == task2.ID {
			if task.Content != "Test task 2" {
				t.Errorf("Expected task content to be 'Test task 2', got '%s'", task.Content)
			}
			if task.Status != taskmanager.StatusInProgress {
				t.Errorf("Expected task status to be 'in_progress', got '%s'", task.Status)
			}
			if task.Priority != taskmanager.PriorityMedium {
				t.Errorf("Expected task priority to be 'medium', got '%s'", task.Priority)
			}
			foundTask2 = true
		}
	}

	if !foundTask1 || !foundTask2 {
		t.Errorf("Not all expected tasks were found in the result")
	}
}
