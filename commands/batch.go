package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mmichie/intu/pkg/security"
	"github.com/mmichie/intu/pkg/tools"
)

// InitBatchCommand initializes and adds the batch command to the root command
func InitBatchCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(RegisterBatchCommand())
}

// RegisterBatchCommand registers the batch command
func RegisterBatchCommand() *cobra.Command {
	batchCmd := &cobra.Command{
		Use:   "batch",
		Short: "Execute multiple tools in parallel",
		Long:  `Batch execution tool that runs multiple tools in parallel where possible.`,
		RunE:  runBatchCommand,
	}

	// Add flags
	batchCmd.Flags().StringP("file", "f", "", "JSON file containing batch invocations")
	batchCmd.Flags().StringP("description", "d", "Batch execution", "Description of the batch operation")

	return batchCmd
}

// runBatchCommand runs the batch command
func runBatchCommand(cmd *cobra.Command, args []string) error {
	// Get input from file or stdin
	inputFile, _ := cmd.Flags().GetString("file")
	description, _ := cmd.Flags().GetString("description")

	var invocations []tools.BatchInvocation
	var err error

	if inputFile != "" {
		// Read from file
		var fileContent []byte
		fileContent, err = os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %w", err)
		}

		// Parse JSON input
		err = json.Unmarshal(fileContent, &invocations)
		if err != nil {
			return fmt.Errorf("failed to parse JSON input: %w", err)
		}
	} else {
		// No explicit invocations provided
		return fmt.Errorf("batch invocations must be provided via --file")
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

	// Create batch tool (registry implements ToolRegistry interface)
	batchTool := tools.NewBatchTool(registry)

	// Create batch parameters
	batchParams := tools.BatchParams{
		Description: description,
		Invocations: invocations,
	}

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(batchParams)
	if err != nil {
		return fmt.Errorf("failed to marshal batch parameters: %w", err)
	}

	// Execute the batch
	result, err := batchTool.Execute(context.Background(), paramsJSON)
	if err != nil {
		return fmt.Errorf("failed to execute batch: %w", err)
	}

	// Format and print results
	batchResult, ok := result.(tools.BatchResult)
	if !ok {
		return fmt.Errorf("unexpected result type: %T", result)
	}

	// Print results in a readable format
	fmt.Println("Batch Execution Results:")
	fmt.Println("------------------------")
	for i, resultMap := range batchResult.Results {
		toolName, _ := resultMap["tool_name"].(string)
		fmt.Printf("[%d] Tool: %s\n", i+1, toolName)

		if errMsg, hasError := resultMap["error"]; hasError {
			fmt.Printf("    Error: %s\n", errMsg)
		} else if res, hasResult := resultMap["result"]; hasResult {
			resultJSON, _ := json.MarshalIndent(res, "    ", "  ")
			fmt.Printf("    Result: %s\n", resultJSON)
		}
		fmt.Println()
	}

	return nil
}

// registerAllTools registers all available tools to the registry
func registerAllTools(registry *tools.Registry) {
	// Register read-only tools
	registry.Register(tools.NewLSTool())
	registry.Register(tools.NewGrepTool())
	registry.Register(tools.NewGlobTool())
	registry.Register(tools.NewReadTool())

	// Register editing tools
	registry.Register(tools.NewEditTool())
	registry.Register(tools.NewWriteTool())

	// Register execution tools
	registry.Register(tools.NewBashTool())

	// Register other available tools
	// TODO: Add any other tools that should be available for batch execution

	// Don't register the batch tool itself to avoid infinite recursion
}
