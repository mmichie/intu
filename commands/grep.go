package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/providers"
	"github.com/mmichie/intu/pkg/security"
	"github.com/mmichie/intu/pkg/tools"
	"github.com/spf13/cobra"
)

// RegisterGrepCommand registers the grep command
func RegisterGrepCommand() *cobra.Command {
	grepCmd := &cobra.Command{
		Use:   "grep [pattern] [path]",
		Short: "Search file contents using AI function calling",
		Long:  `Search file contents for a pattern using AI function calling capability. This demonstrates the function calling API with the Grep tool.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runGrepCommand,
	}

	grepCmd.Flags().StringP("include", "i", "", "File pattern to include (e.g. '*.js', '*.{ts,tsx}')")

	return grepCmd
}

func runGrepCommand(cmd *cobra.Command, args []string) error {
	// Get pattern from args
	pattern := args[0]

	// Get path from args or use current directory
	path := "."
	if len(args) > 1 {
		path = args[1]
	}

	// Get include pattern from flags
	include, _ := cmd.Flags().GetString("include")

	// Create Claude provider
	provider, err := aikit.NewProvider("claude")
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Create Grep tool
	grepTool := tools.NewGrepTool()

	// Register tool with provider
	err = provider.RegisterFunction(providers.FunctionDefinition{
		Name:        grepTool.Name(),
		Description: grepTool.Description(),
		Parameters:  grepTool.ParameterSchema(),
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
	registry.Register(grepTool)

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

	// Create prompt asking to search for pattern
	prompt := fmt.Sprintf("Please search for the pattern '%s' in the directory at path '%s'", pattern, path)
	if include != "" {
		prompt += fmt.Sprintf(", including only files matching '%s'", include)
	}
	prompt += ". Only use the Grep function and return the results in a clear, organized format. Show the context around each match."

	// Call provider with function calling
	response, err := provider.GenerateResponseWithFunctions(context.Background(), prompt, functionExecutor)
	if err != nil {
		return fmt.Errorf("error generating response: %w", err)
	}

	// Print response
	fmt.Println(strings.TrimSpace(response))

	return nil
}

func InitGrepCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(RegisterGrepCommand())
}
