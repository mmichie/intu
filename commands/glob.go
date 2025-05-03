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

// InitGlobCommand initializes and adds the glob command to the root command
func InitGlobCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(RegisterGlobCommand())
}

// RegisterGlobCommand registers the glob command
func RegisterGlobCommand() *cobra.Command {
	globCmd := &cobra.Command{
		Use:   "glob [pattern]",
		Short: "Find files using glob patterns",
		Long:  `Find files using glob patterns with AI function calling capability.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runGlobCommand,
	}

	// Add path flag
	globCmd.Flags().StringP("path", "p", "", "Directory to search in (defaults to current directory)")

	return globCmd
}

func runGlobCommand(cmd *cobra.Command, args []string) error {
	// Get pattern from args or use default
	pattern := "**/*"
	if len(args) > 0 {
		pattern = args[0]
	}

	// Get path flag
	path, _ := cmd.Flags().GetString("path")

	// Create Claude provider
	provider, err := aikit.NewProvider("claude")
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Create Glob tool
	globTool := tools.NewGlobTool()

	// Register tool with provider
	err = provider.RegisterFunction(providers.FunctionDefinition{
		Name:        globTool.Name(),
		Description: globTool.Description(),
		Parameters:  globTool.ParameterSchema(),
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
	registry.Register(globTool)

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

	// Create prompt asking to find files
	prompt := fmt.Sprintf("Please find files using the glob pattern '%s'. "+
		"Only use the Glob function and return the results in a clear, organized format.", pattern)

	// Add path to the prompt if provided
	if path != "" {
		prompt += fmt.Sprintf(" Use the path: '%s'.", path)
	}

	// Call provider with function calling
	response, err := provider.GenerateResponseWithFunctions(context.Background(), prompt, functionExecutor)
	if err != nil {
		return fmt.Errorf("error generating response: %w", err)
	}

	// Print response
	fmt.Println(strings.TrimSpace(response))

	return nil
}
