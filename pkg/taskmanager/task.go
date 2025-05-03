// Package taskmanager provides functionality for managing tasks and todo items
package taskmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	contextPkg "github.com/mmichie/intu/pkg/context"
)

// Task status values
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusCancelled  = "cancelled"
)

// Task priority values
const (
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// Task represents a single task item
type Task struct {
	// ID is a unique identifier for this task
	ID string `json:"id"`

	// Content contains the task description
	Content string `json:"content"`

	// Status of the task (pending, in_progress, completed, cancelled)
	Status string `json:"status"`

	// Priority of the task (high, medium, low)
	Priority string `json:"priority"`

	// Created is the time when this task was created
	Created time.Time `json:"created"`

	// Updated is the time when this task was last updated
	Updated time.Time `json:"updated"`

	// CompletedAt is the time when this task was completed (if applicable)
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// TaskList represents a list of tasks
type TaskList struct {
	Tasks []*Task `json:"tasks"`
}

// TaskManager handles operations on tasks
type TaskManager struct {
	mu           sync.RWMutex
	tasks        []*Task
	contextStore contextPkg.ContextStore
	contextID    string // The context ID where tasks are stored
}

// NewTaskManager creates a new task manager
func NewTaskManager(contextStore contextPkg.ContextStore, contextID string) *TaskManager {
	return &TaskManager{
		tasks:        make([]*Task, 0),
		contextStore: contextStore,
		contextID:    contextID,
	}
}

// CreateTask adds a new task
func (m *TaskManager) CreateTask(content string, priority string) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate priority
	if priority != PriorityHigh && priority != PriorityMedium && priority != PriorityLow {
		return nil, fmt.Errorf("invalid priority: %s", priority)
	}

	// Create the task
	now := time.Now()
	task := &Task{
		ID:       fmt.Sprintf("%d", len(m.tasks)+1), // Simple numeric ID
		Content:  content,
		Status:   StatusPending,
		Priority: priority,
		Created:  now,
		Updated:  now,
	}

	m.tasks = append(m.tasks, task)

	// Save to context
	if err := m.saveToContext(); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	return task, nil
}

// GetTask retrieves a task by ID
func (m *TaskManager) GetTask(id string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, task := range m.tasks {
		if task.ID == id {
			return task, nil
		}
	}

	return nil, errors.New("task not found")
}

// UpdateTask updates an existing task
func (m *TaskManager) UpdateTask(id string, content string, status string, priority string) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the task
	var task *Task
	for _, t := range m.tasks {
		if t.ID == id {
			task = t
			break
		}
	}

	if task == nil {
		return nil, errors.New("task not found")
	}

	// Update task properties
	if content != "" {
		task.Content = content
	}

	// Update status
	if status != "" {
		// Validate status
		if status != StatusPending && status != StatusInProgress &&
			status != StatusCompleted && status != StatusCancelled {
			return nil, fmt.Errorf("invalid status: %s", status)
		}

		// If we're completing the task, set the completion time
		if status == StatusCompleted && task.Status != StatusCompleted {
			now := time.Now()
			task.CompletedAt = &now
		} else if status != StatusCompleted {
			task.CompletedAt = nil
		}

		task.Status = status
	}

	// Update priority
	if priority != "" {
		// Validate priority
		if priority != PriorityHigh && priority != PriorityMedium && priority != PriorityLow {
			return nil, fmt.Errorf("invalid priority: %s", priority)
		}

		task.Priority = priority
	}

	// Update the updated time
	task.Updated = time.Now()

	// Save to context
	if err := m.saveToContext(); err != nil {
		return nil, fmt.Errorf("failed to save task update: %w", err)
	}

	return task, nil
}

// DeleteTask removes a task
func (m *TaskManager) DeleteTask(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, task := range m.tasks {
		if task.ID == id {
			// Remove the task from the slice
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)

			// Save to context
			if err := m.saveToContext(); err != nil {
				return fmt.Errorf("failed to save task deletion: %w", err)
			}
			return nil
		}
	}

	return errors.New("task not found")
}

// ListTasks returns all tasks, optionally filtered by status
func (m *TaskManager) ListTasks(status string) ([]*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If no status filter, return all tasks
	if status == "" {
		result := make([]*Task, len(m.tasks))
		copy(result, m.tasks)
		return result, nil
	}

	// Filter by status
	var result []*Task
	for _, task := range m.tasks {
		if task.Status == status {
			result = append(result, task)
		}
	}

	return result, nil
}

// saveToContext saves the current tasks to the context store
func (m *TaskManager) saveToContext() error {
	if m.contextStore == nil {
		return errors.New("no context store available")
	}

	fmt.Printf("Saving tasks to context ID: %s\n", m.contextID)

	// Get existing context
	ctx, err := m.contextStore.Get(m.contextID)
	if err != nil && err != contextPkg.ErrContextNotFound {
		return fmt.Errorf("failed to get context: %w", err)
	}

	// If context doesn't exist, create it
	if err == contextPkg.ErrContextNotFound {
		ctx = &contextPkg.ContextData{
			ID:      m.contextID,
			Type:    contextPkg.SessionContext,
			Name:    "Task List",
			Data:    make(map[string]interface{}),
			Tags:    []string{"tasks"},
			Created: time.Now(),
			Updated: time.Now(),
		}
	}

	// Store tasks in context
	ctx.Data["tasks"] = m.tasks
	ctx.Updated = time.Now()

	fmt.Printf("Storing %d tasks in context\n", len(m.tasks))
	for i, task := range m.tasks {
		fmt.Printf("Task %d: %s (Status: %s, Priority: %s)\n", i+1, task.Content, task.Status, task.Priority)
	}

	// Save context
	err = m.contextStore.Set(ctx)
	if err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// If we have a persistent store, make sure it's saved to disk
	if ps, ok := m.contextStore.(contextPkg.PersistentContextStore); ok {
		fmt.Println("Saving to persistent store...")
		if err := ps.Save(); err != nil {
			return fmt.Errorf("failed to save to persistent store: %w", err)
		}
	}

	return nil
}

// LoadFromContext loads tasks from the context store
func (m *TaskManager) LoadFromContext() error {
	if m.contextStore == nil {
		return errors.New("no context store available")
	}

	fmt.Printf("Loading tasks from context ID: %s\n", m.contextID)

	// Get the context
	ctx, err := m.contextStore.Get(m.contextID)
	if err != nil {
		if err == contextPkg.ErrContextNotFound {
			// No tasks stored yet, start with empty list
			fmt.Printf("Context not found, starting with empty task list\n")
			m.tasks = make([]*Task, 0)
			return nil
		}
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Extract tasks from context
	tasksData, ok := ctx.Data["tasks"]
	if !ok {
		// No tasks in context yet
		m.tasks = make([]*Task, 0)
		return nil
	}

	// Convert to JSON and back to handle interface{} conversion
	tasksJSON, err := json.Marshal(tasksData)
	if err != nil {
		return fmt.Errorf("failed to marshal tasks data: %w", err)
	}

	var tasks []*Task
	if err := json.Unmarshal(tasksJSON, &tasks); err != nil {
		return fmt.Errorf("failed to unmarshal tasks: %w", err)
	}

	m.mu.Lock()
	m.tasks = tasks
	m.mu.Unlock()

	return nil
}

// SaveToFile saves tasks to a file
func (m *TaskManager) SaveToFile(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prepare task list
	taskList := TaskList{
		Tasks: m.tasks,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(taskList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	return nil
}

// LoadFromFile loads tasks from a file
func (m *TaskManager) LoadFromFile(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // File doesn't exist, start with empty task list
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read tasks file: %w", err)
	}

	// Unmarshal JSON
	var taskList TaskList
	if err := json.Unmarshal(data, &taskList); err != nil {
		return fmt.Errorf("failed to unmarshal tasks: %w", err)
	}

	// Update tasks
	m.mu.Lock()
	m.tasks = taskList.Tasks
	m.mu.Unlock()

	return nil
}
