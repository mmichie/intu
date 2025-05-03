package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/providers"
	"github.com/mmichie/intu/pkg/security"
	"github.com/mmichie/intu/pkg/tools"
)

// InitBashCommand initializes and adds the bash command to the root command
func InitBashCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(RegisterBashCommand())
}

// RegisterBashCommand registers the bash command
func RegisterBashCommand() *cobra.Command {
	bashCmd := &cobra.Command{
		Use:   "bash [command]",
		Short: "Execute shell commands",
		Long:  `Execute shell commands with AI assistance and security safeguards.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runBashCommand,
	}

	// Add flags
	bashCmd.Flags().IntP("timeout", "t", 120000, "Timeout in milliseconds (max 10 minutes)")
	bashCmd.Flags().BoolP("explain", "e", false, "Ask AI to explain the command's output")
	bashCmd.Flags().BoolP("raw", "r", false, "Show raw output without AI processing")

	return bashCmd
}

func runBashCommand(cmd *cobra.Command, args []string) error {
	// Join all args to get the full command
	command := strings.Join(args, " ")

	// Get flags
	timeout, _ := cmd.Flags().GetInt("timeout")
	explain, _ := cmd.Flags().GetBool("explain")
	raw, _ := cmd.Flags().GetBool("raw")

	// Cap timeout
	if timeout <= 0 {
		timeout = 120000 // 2 minutes default
	} else if timeout > 600000 {
		timeout = 600000 // 10 minutes max
	}

	// If raw output is requested, execute directly without AI
	if raw {
		return executeRawCommand(command, timeout)
	}

	// Create provider
	provider, err := aikit.NewProvider("claude")
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Create Bash tool
	bashTool := tools.NewBashTool()

	// Register tool with provider
	err = provider.RegisterFunction(providers.FunctionDefinition{
		Name:        bashTool.Name(),
		Description: bashTool.Description(),
		Parameters:  bashTool.ParameterSchema(),
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
	registry.Register(bashTool)

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

	// Create prompt based on mode
	prompt := fmt.Sprintf("Please execute the command '%s' using the Bash tool. ", command)
	if explain {
		prompt += "After running the command, explain what the command does and interpret its output in a clear, helpful way."
	} else {
		prompt += "Only show the command output with minimal additional text."
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

// executeRawCommand executes the command directly without AI assistance
func executeRawCommand(command string, timeoutMs int) error {
	// Create permission manager with default terminal prompt
	permissionMgr, err := security.NewPermissionManager(security.DefaultPrompt())
	if err != nil {
		return fmt.Errorf("failed to create permission manager: %w", err)
	}

	// Create tool registry with permissions
	registry := tools.NewRegistryWithPermissions(permissionMgr)
	bashTool := tools.NewBashTool()
	registry.Register(bashTool)

	// Execute the command directly through the tool
	params := tools.BashParams{
		Command: command,
		Timeout: timeoutMs,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	result, err := registry.ExecuteTool(ctx, bashTool.Name(), paramsJSON)
	if err != nil {
		return fmt.Errorf("error executing command: %w", err)
	}

	// Print result
	bashResult, ok := result.(tools.BashResult)
	if !ok {
		return fmt.Errorf("unexpected result type")
	}

	// Print stdout, then stderr if any
	fmt.Print(bashResult.Stdout)
	if bashResult.Stderr != "" {
		fmt.Fprintln(os.Stderr, bashResult.Stderr)
	}

	// Return an error if the command failed
	if bashResult.ExitCode != 0 {
		return fmt.Errorf("command failed with exit code %d", bashResult.ExitCode)
	}

	return nil
}
