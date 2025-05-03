package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/providers"
	"github.com/mmichie/intu/pkg/security"
	"github.com/mmichie/intu/pkg/tools"
)

// InitTaskCommand initializes and adds the task command to the root command
func InitTaskCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(RegisterTaskCommand())
}

// RegisterTaskCommand registers the task command
func RegisterTaskCommand() *cobra.Command {
	taskCmd := &cobra.Command{
		Use:   "task [prompt]",
		Short: "Execute a task with an AI agent",
		Long:  `Task execution tool that spawns an AI agent with access to tools to perform complex operations.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runTaskCommand,
	}

	// Add flags
	taskCmd.Flags().StringP("description", "d", "Task execution", "Short description of the task")
	taskCmd.Flags().StringP("file", "f", "", "Read prompt from file instead of arguments")
	taskCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return taskCmd
}

// runTaskCommand runs the task command
func runTaskCommand(cmd *cobra.Command, args []string) error {
	// Get input from arguments or file
	inputFile, _ := cmd.Flags().GetString("file")
	description, _ := cmd.Flags().GetString("description")

	var prompt string
	var err error

	if inputFile != "" {
		// Read from file
		var fileContent []byte
		fileContent, err = os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %w", err)
		}
		prompt = string(fileContent)
	} else {
		// Use arguments as prompt
		prompt = strings.Join(args, " ")
	}

	// Create permission manager with default terminal prompt
	permissionMgr, err := security.NewPermissionManager(security.DefaultPrompt())
	if err != nil {
		return fmt.Errorf("failed to create permission manager: %w", err)
	}

	// Create tool registry with permissions
	registry := tools.NewRegistry()
	registry.SetPermissionManager(permissionMgr)

	// Register all available tools to the registry
	registerAllTools(registry)

	// Use a simpler approach - use the factory pattern from the aikit package
	// Create default config
	providerName := viper.GetString("default_provider")
	if providerName == "" {
		providerName = "openai" // Default to OpenAI if not specified
	}

	// Create provider
	provider, err := aikit.NewProvider(providerName)
	if err != nil {
		return fmt.Errorf("failed to initialize provider: %w", err)
	}

	// Convert aikit.FunctionDefinition to providers.FunctionDefinition and register with provider
	providerFunctions := make([]providers.FunctionDefinition, 0, len(registry.GetFunctionDefinitions()))
	for _, fn := range registry.GetFunctionDefinitions() {
		providerFunctions = append(providerFunctions, providers.FunctionDefinition{
			Name:        fn.Name,
			Description: fn.Description,
			Parameters:  fn.Parameters,
		})
	}
	provider.RegisterFunctions(providerFunctions)

	// Create task tool
	taskTool := tools.NewTaskTool(registry, provider)

	// Create task parameters
	taskParams := tools.TaskParams{
		Description: description,
		Prompt:      prompt,
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(taskParams)
	if err != nil {
		return fmt.Errorf("failed to marshal task parameters: %w", err)
	}

	// Show progress message
	fmt.Println("📝 Task started: " + description)
	fmt.Println("🤖 Working on: " + prompt)
	fmt.Println("⏳ Please wait while the AI agent works on your task...")

	// Execute the task
	result, err := taskTool.Execute(context.Background(), paramsJSON)
	if err != nil {
		return fmt.Errorf("failed to execute task: %w", err)
	}

	// Format and print results
	taskResult, ok := result.(tools.TaskResult)
	if !ok {
		return fmt.Errorf("unexpected result type: %T", result)
	}

	// Print task result
	fmt.Println("\n✅ Task completed!")
	fmt.Println("\nResult:")
	fmt.Println("-------")
	fmt.Println(taskResult.Result)

	return nil
}
