package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	contextPkg "github.com/mmichie/intu/pkg/context"
	"github.com/mmichie/intu/pkg/taskmanager"
	"github.com/mmichie/intu/pkg/tools"
	"github.com/spf13/cobra"
)

// todoReadToolCmd represents the todoread command
var todoReadToolCmd = &cobra.Command{
	Use:   "todoread",
	Short: "Read the current todo list",
	Long:  `Read and display the current todo list for tracking tasks.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get user home directory for storing context data
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		// Create context storage
		storePath := filepath.Join(homeDir, ".intu", "contexts")
		store, err := createContextStore(storePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating context store: %v\n", err)
			os.Exit(1)
		}

		// Create task manager with a consistent context ID
		taskMgr := taskmanager.NewTaskManager(store, "task-list")
		fmt.Printf("Using context ID: %s\n", "task-list")

		// Create TodoRead tool
		todoReadTool := tools.NewTodoReadTool(taskMgr)

		// Execute the tool
		result, err := todoReadTool.Execute(context.Background(), json.RawMessage("{}"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing TodoRead: %v\n", err)
			os.Exit(1)
		}

		// Format and display the result
		todoResult, ok := result.(tools.TodoReadResult)
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: unexpected result type\n")
			os.Exit(1)
		}

		// If there are no todos
		if len(todoResult.Todos) == 0 {
			fmt.Println("No todo items found.")
			return
		}

		// Print todos in a formatted way
		fmt.Println("TODO LIST:")
		fmt.Println("----------")
		for _, todo := range todoResult.Todos {
			// Format the status
			statusStr := ""
			switch todo.Status {
			case taskmanager.StatusPending:
				statusStr = "[ ] "
			case taskmanager.StatusInProgress:
				statusStr = "[>] "
			case taskmanager.StatusCompleted:
				statusStr = "[✓] "
			case taskmanager.StatusCancelled:
				statusStr = "[X] "
			}

			// Format the priority
			priorityStr := ""
			switch todo.Priority {
			case taskmanager.PriorityHigh:
				priorityStr = "(high)"
			case taskmanager.PriorityMedium:
				priorityStr = "(medium)"
			case taskmanager.PriorityLow:
				priorityStr = "(low)"
			}

			fmt.Printf("%s %s %s: %s\n", todo.ID, statusStr, priorityStr, todo.Content)
		}
	},
}

// todoWriteToolCmd represents the todowrite command
var todoWriteToolCmd = &cobra.Command{
	Use:   "todowrite",
	Short: "Create or update todo items",
	Long:  `Create or update todo items in the todo list.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get flags
		id, _ := cmd.Flags().GetString("id")
		content, _ := cmd.Flags().GetString("content")
		status, _ := cmd.Flags().GetString("status")
		priority, _ := cmd.Flags().GetString("priority")

		if content == "" {
			fmt.Fprintf(os.Stderr, "Error: content is required\n")
			os.Exit(1)
		}

		// Validate status
		if status != "" && status != taskmanager.StatusPending &&
			status != taskmanager.StatusInProgress &&
			status != taskmanager.StatusCompleted &&
			status != taskmanager.StatusCancelled {
			fmt.Fprintf(os.Stderr, "Error: status must be one of pending, in_progress, completed, cancelled\n")
			os.Exit(1)
		}

		// If no status provided, default to pending
		if status == "" {
			status = taskmanager.StatusPending
		}

		// Validate priority
		if priority != "" && priority != taskmanager.PriorityHigh &&
			priority != taskmanager.PriorityMedium &&
			priority != taskmanager.PriorityLow {
			fmt.Fprintf(os.Stderr, "Error: priority must be one of high, medium, low\n")
			os.Exit(1)
		}

		// If no priority provided, default to medium
		if priority == "" {
			priority = taskmanager.PriorityMedium
		}

		// Get user home directory for storing context data
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		// Create context storage
		storePath := filepath.Join(homeDir, ".intu", "contexts")
		store, err := createContextStore(storePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating context store: %v\n", err)
			os.Exit(1)
		}

		// Create task manager with a consistent context ID
		taskMgr := taskmanager.NewTaskManager(store, "task-list")
		fmt.Printf("Using context ID: %s\n", "task-list")

		// Load existing tasks
		if err := taskMgr.LoadFromContext(); err != nil {
			// Ignore "not found" errors
			if !contextPkg.IsContextNotFound(err) {
				fmt.Fprintf(os.Stderr, "Error loading tasks: %v\n", err)
				os.Exit(1)
			}
		}

		// Prepare the todo item
		todoItem := tools.TodoItemParams{
			ID:       id,
			Content:  content,
			Status:   status,
			Priority: priority,
		}

		// If ID is empty, it will be a new task with an auto-generated ID
		if todoItem.ID == "" {
			// Get the list of existing tasks to determine the next ID
			tasks, err := taskMgr.ListTasks("")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing tasks: %v\n", err)
				os.Exit(1)
			}

			// Generate a simple numeric ID
			todoItem.ID = fmt.Sprintf("%d", len(tasks)+1)
		}

		// Prepare TodoWriteParams
		params := tools.TodoWriteParams{
			Todos: []tools.TodoItemParams{todoItem},
		}

		// Convert to JSON
		paramsJson, err := json.Marshal(params)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding parameters: %v\n", err)
			os.Exit(1)
		}

		// Create TodoWrite tool
		todoWriteTool := tools.NewTodoWriteTool(taskMgr)

		// Execute the tool
		result, err := todoWriteTool.Execute(context.Background(), paramsJson)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing TodoWrite: %v\n", err)
			os.Exit(1)
		}

		// Format and display the result
		writeResult, ok := result.(tools.TodoWriteResult)
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: unexpected result type\n")
			os.Exit(1)
		}

		if writeResult.Success {
			fmt.Println("Todo item successfully saved.")
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", writeResult.Message)
			os.Exit(1)
		}
	},
}

func createContextStore(storePath string) (contextPkg.PersistentContextStore, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create context storage directory: %w", err)
	}

	// Create a persistent store with options
	options := contextPkg.DefaultPersistentStoreOptions()
	options.StoragePath = filepath.Join(storePath, "contexts.json")
	fmt.Printf("Using context storage path: %s\n", options.StoragePath)
	persStore, err := contextPkg.NewPersistentHierarchicalStore(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create persistent store: %w", err)
	}

	// Storage path is already set in options

	// Load existing contexts
	fmt.Println("Loading existing contexts...")
	if err := persStore.Load(); err != nil {
		// Ignore file not found errors
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load contexts: %w", err)
		}
		fmt.Println("No existing context file found.")
	} else {
		fmt.Println("Successfully loaded existing contexts.")
	}

	return persStore, nil
}

// InitTodoToolsCommand registers the todo tools commands
func InitTodoToolsCommand(rootCmd *cobra.Command) {
	// Add flags to todoWriteToolCmd
	todoWriteToolCmd.Flags().StringP("id", "i", "", "ID of the todo item (leave empty for new todos)")
	todoWriteToolCmd.Flags().StringP("content", "c", "", "Content of the todo item")
	todoWriteToolCmd.Flags().StringP("status", "s", "", "Status of the todo item (pending, in_progress, completed, cancelled)")
	todoWriteToolCmd.Flags().StringP("priority", "p", "", "Priority of the todo item (high, medium, low)")

	// Mark content as required
	todoWriteToolCmd.MarkFlagRequired("content")

	// Add commands to root
	rootCmd.AddCommand(todoReadToolCmd)
	rootCmd.AddCommand(todoWriteToolCmd)
}
