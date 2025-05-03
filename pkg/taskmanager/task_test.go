package taskmanager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	contextPkg "github.com/mmichie/intu/pkg/context"
)

// TestTaskCreation tests creating a new task
func TestTaskCreation(t *testing.T) {
	// Create a memory context store for testing
	store := contextPkg.NewMemoryContextStore()
	tm := NewTaskManager(store, "test-tasks")

	// Create a task
	task, err := tm.CreateTask("Test task", PriorityHigh)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Verify task properties
	if task.ID == "" {
		t.Error("Task ID should not be empty")
	}

	if task.Content != "Test task" {
		t.Errorf("Task content mismatch, got %s, want %s", task.Content, "Test task")
	}

	if task.Status != StatusPending {
		t.Errorf("New task should have status pending, got %s", task.Status)
	}

	if task.Priority != PriorityHigh {
		t.Errorf("Task priority mismatch, got %s, want %s", task.Priority, PriorityHigh)
	}

	// Verify task is saved to context
	ctx, err := store.Get("test-tasks")
	if err != nil {
		t.Fatalf("Failed to get context: %v", err)
	}

	tasksData, ok := ctx.Data["tasks"]
	if !ok {
		t.Fatal("Tasks not saved to context")
	}

	// Convert to JSON and back to handle interface{} conversion
	tasksJSON, err := json.Marshal(tasksData)
	if err != nil {
		t.Fatalf("Failed to marshal tasks data: %v", err)
	}

	var tasks []*Task
	if err := json.Unmarshal(tasksJSON, &tasks); err != nil {
		t.Fatalf("Failed to unmarshal tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}
}

// TestTaskUpdate tests updating an existing task
func TestTaskUpdate(t *testing.T) {
	// Create a memory context store for testing
	store := contextPkg.NewMemoryContextStore()
	tm := NewTaskManager(store, "test-tasks")

	// Create a task
	task, err := tm.CreateTask("Original task", PriorityMedium)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Update the task
	updatedTask, err := tm.UpdateTask(task.ID, "Updated task", StatusInProgress, PriorityHigh)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	// Verify updates
	if updatedTask.Content != "Updated task" {
		t.Errorf("Content not updated, got %s, want %s", updatedTask.Content, "Updated task")
	}

	if updatedTask.Status != StatusInProgress {
		t.Errorf("Status not updated, got %s, want %s", updatedTask.Status, StatusInProgress)
	}

	if updatedTask.Priority != PriorityHigh {
		t.Errorf("Priority not updated, got %s, want %s", updatedTask.Priority, PriorityHigh)
	}

	// Check completion time handling
	if updatedTask.CompletedAt != nil {
		t.Errorf("Non-completed task should have nil CompletedAt, got %v", updatedTask.CompletedAt)
	}

	// Update to completed
	completedTask, err := tm.UpdateTask(task.ID, "", StatusCompleted, "")
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}

	if completedTask.Status != StatusCompleted {
		t.Errorf("Status not updated to completed, got %s", completedTask.Status)
	}

	if completedTask.CompletedAt == nil {
		t.Error("Completed task should have CompletedAt timestamp")
	}
}

// TestTaskDeletion tests deleting a task
func TestTaskDeletion(t *testing.T) {
	// Create a memory context store for testing
	store := contextPkg.NewMemoryContextStore()
	tm := NewTaskManager(store, "test-tasks")

	// Create two tasks
	task1, err := tm.CreateTask("Task 1", PriorityHigh)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	_, err = tm.CreateTask("Task 2", PriorityMedium)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Delete the first task
	err = tm.DeleteTask(task1.ID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify task is deleted
	_, err = tm.GetTask(task1.ID)
	if err == nil {
		t.Error("Expected task not found error, got nil")
	}

	// Verify only one task remains
	tasks, err := tm.ListTasks("")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task after deletion, got %d", len(tasks))
	}

	if tasks[0].Content != "Task 2" {
		t.Errorf("Expected remaining task to be 'Task 2', got '%s'", tasks[0].Content)
	}
}

// TestTaskFiltering tests filtering tasks by status
func TestTaskFiltering(t *testing.T) {
	// Create a memory context store for testing
	store := contextPkg.NewMemoryContextStore()
	tm := NewTaskManager(store, "test-tasks")

	// Create tasks with different statuses
	_, err := tm.CreateTask("Pending task", PriorityHigh)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	inProgressTask, err := tm.CreateTask("In progress task", PriorityMedium)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	_, err = tm.UpdateTask(inProgressTask.ID, "", StatusInProgress, "")
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	completedTask, err := tm.CreateTask("Completed task", PriorityLow)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	_, err = tm.UpdateTask(completedTask.ID, "", StatusCompleted, "")
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	// Test filtering by status
	pendingTasks, err := tm.ListTasks(StatusPending)
	if err != nil {
		t.Fatalf("Failed to list pending tasks: %v", err)
	}
	if len(pendingTasks) != 1 {
		t.Errorf("Expected 1 pending task, got %d", len(pendingTasks))
	}

	inProgressTasks, err := tm.ListTasks(StatusInProgress)
	if err != nil {
		t.Fatalf("Failed to list in-progress tasks: %v", err)
	}
	if len(inProgressTasks) != 1 {
		t.Errorf("Expected 1 in-progress task, got %d", len(inProgressTasks))
	}

	completedTasks, err := tm.ListTasks(StatusCompleted)
	if err != nil {
		t.Fatalf("Failed to list completed tasks: %v", err)
	}
	if len(completedTasks) != 1 {
		t.Errorf("Expected 1 completed task, got %d", len(completedTasks))
	}

	// Test getting all tasks
	allTasks, err := tm.ListTasks("")
	if err != nil {
		t.Fatalf("Failed to list all tasks: %v", err)
	}
	if len(allTasks) != 3 {
		t.Errorf("Expected 3 tasks in total, got %d", len(allTasks))
	}
}

// TestFilePersistence tests saving and loading tasks from a file
func TestFilePersistence(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "taskmanager-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tasksFile := filepath.Join(tempDir, "tasks.json")

	// Create task manager and some tasks
	store := contextPkg.NewMemoryContextStore()
	tm := NewTaskManager(store, "test-tasks")

	_, err = tm.CreateTask("Task 1", PriorityHigh)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	_, err = tm.CreateTask("Task 2", PriorityMedium)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Save tasks to file
	err = tm.SaveToFile(tasksFile)
	if err != nil {
		t.Fatalf("Failed to save tasks to file: %v", err)
	}

	// Create a new task manager and load from file
	newTM := NewTaskManager(store, "new-test-tasks")
	err = newTM.LoadFromFile(tasksFile)
	if err != nil {
		t.Fatalf("Failed to load tasks from file: %v", err)
	}

	// Verify tasks were loaded correctly
	tasks, err := newTM.ListTasks("")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	if tasks[0].Content != "Task 1" || tasks[1].Content != "Task 2" {
		t.Errorf("Tasks not loaded correctly")
	}
}

// TestContextPersistence tests saving and loading tasks from context
func TestContextPersistence(t *testing.T) {
	// Create a memory context store for testing
	store := contextPkg.NewMemoryContextStore()

	// Create a task manager and add tasks
	tm1 := NewTaskManager(store, "test-tasks")
	_, err := tm1.CreateTask("Context Task 1", PriorityHigh)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	_, err = tm1.CreateTask("Context Task 2", PriorityMedium)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Create a new task manager with the same context ID
	tm2 := NewTaskManager(store, "test-tasks")

	// Load tasks from context
	err = tm2.LoadFromContext()
	if err != nil {
		t.Fatalf("Failed to load tasks from context: %v", err)
	}

	// Verify tasks were loaded correctly
	tasks, err := tm2.ListTasks("")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// Verify task content
	var foundTask1, foundTask2 bool
	for _, task := range tasks {
		if task.Content == "Context Task 1" && task.Priority == PriorityHigh {
			foundTask1 = true
		}
		if task.Content == "Context Task 2" && task.Priority == PriorityMedium {
			foundTask2 = true
		}
	}

	if !foundTask1 || !foundTask2 {
		t.Errorf("Tasks not loaded correctly from context")
	}
}
