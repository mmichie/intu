package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/providers"
	"github.com/mmichie/intu/pkg/security"
	"github.com/mmichie/intu/pkg/tools"
)

// InitReadCommand initializes and adds the read command to the root command
func InitReadCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(RegisterReadCommand())
}

// RegisterReadCommand registers the read command
func RegisterReadCommand() *cobra.Command {
	readCmd := &cobra.Command{
		Use:   "read [file_path]",
		Short: "Read file contents",
		Long:  `Read the contents of a file from the filesystem with optional offset and limit parameters.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runReadCommand,
	}

	// Add flags
	readCmd.Flags().IntP("offset", "o", 0, "Line number to start reading from (0-based)")
	readCmd.Flags().IntP("limit", "l", 0, "Maximum number of lines to read (0 means use default of 2000)")

	return readCmd
}

func runReadCommand(cmd *cobra.Command, args []string) error {
	// Get file path from args
	filePath := args[0]

	// Get flags
	offset, _ := cmd.Flags().GetInt("offset")
	limit, _ := cmd.Flags().GetInt("limit")

	// Create Claude provider
	provider, err := aikit.NewProvider("claude")
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Create Read tool
	readTool := tools.NewReadTool()

	// Register tool with provider
	err = provider.RegisterFunction(providers.FunctionDefinition{
		Name:        readTool.Name(),
		Description: readTool.Description(),
		Parameters:  readTool.ParameterSchema(),
	})
	if err != nil {
		return fmt.Errorf("failed to register function: %w", err)
	}

	// Create permission manager with default terminal prompt
	permissionMgr, err := security.NewPermissionManager(security.DefaultPrompt())
	if err != nil {
		return fmt.Errorf("failed to create permission manager: %w", err)
	}

	// Create tool registry with permissions
	registry := tools.NewRegistryWithPermissions(permissionMgr)
	registry.Register(readTool)

	// Create function executor
	functionExecutor := func(call providers.FunctionCall) (providers.FunctionResponse, error) {
		result, _ := registry.ExecuteFunctionCall(context.Background(), aikit.FunctionCall{
			Name:       call.Name,
			Parameters: call.Parameters,
		})

		return providers.FunctionResponse{
			Name:    result.Name,
			Content: result.Content,
			Error:   result.Error,
		}, nil
	}

	// Create prompt
	prompt := fmt.Sprintf("Please read the file at path '%s'", filePath)
	if offset > 0 {
		prompt += fmt.Sprintf(" starting from line %d", offset)
	}
	if limit > 0 {
		prompt += fmt.Sprintf(" and read %d lines", limit)
	}
	prompt += ". Only use the Read function and display the file content in a clear format."

	// Call provider with function calling
	response, err := provider.GenerateResponseWithFunctions(context.Background(), prompt, functionExecutor)
	if err != nil {
		return fmt.Errorf("error generating response: %w", err)
	}

	// Print response
	fmt.Println(strings.TrimSpace(response))

	return nil
}
