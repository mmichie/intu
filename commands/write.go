package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/providers"
	"github.com/mmichie/intu/pkg/security"
	"github.com/mmichie/intu/pkg/tools"
)

// InitWriteCommand initializes and adds the write command to the root command
func InitWriteCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(RegisterWriteCommand())
}

// RegisterWriteCommand registers the write command
func RegisterWriteCommand() *cobra.Command {
	writeCmd := &cobra.Command{
		Use:   "write [file_path]",
		Short: "Write content to a file",
		Long:  `Write content to a file, creating or overwriting as needed.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runWriteCommand,
	}

	// Add flags
	writeCmd.Flags().StringP("content", "c", "", "Content to write to the file")
	writeCmd.Flags().StringP("from-file", "f", "", "Read content from this file instead of parameter")
	writeCmd.Flags().BoolP("from-stdin", "s", false, "Read content from stdin")

	return writeCmd
}

func runWriteCommand(cmd *cobra.Command, args []string) error {
	// Get file path from args
	filePath := args[0]

	// Get content from flags (prioritize in this order: stdin > file > parameter)
	content, _ := cmd.Flags().GetString("content")
	fromFile, _ := cmd.Flags().GetString("from-file")
	fromStdin, _ := cmd.Flags().GetBool("from-stdin")

	// Get content from stdin if requested
	if fromStdin {
		stdinContent, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		content = string(stdinContent)
	} else if fromFile != "" {
		// Get content from file if provided
		fileContent, err := ioutil.ReadFile(fromFile)
		if err != nil {
			return fmt.Errorf("failed to read from file %s: %w", fromFile, err)
		}
		content = string(fileContent)
	}

	// Verify content is provided
	if content == "" {
		return fmt.Errorf("content must be provided via --content, --from-file, or --from-stdin")
	}

	// Create provider
	provider, err := aikit.NewProvider("claude")
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Create Write tool
	writeTool := tools.NewWriteTool()

	// Register tool with provider
	err = provider.RegisterFunction(providers.FunctionDefinition{
		Name:        writeTool.Name(),
		Description: writeTool.Description(),
		Parameters:  writeTool.ParameterSchema(),
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
	registry.Register(writeTool)

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

	// Create prompt asking for the write
	prompt := fmt.Sprintf("Please write content to the file at '%s'. ", filePath)
	prompt += "The content is provided in this request. "
	prompt += "Only use the Write function and return a success message with details of what was written."

	// Call provider with function calling
	response, err := provider.GenerateResponseWithFunctions(context.Background(), prompt, functionExecutor)
	if err != nil {
		return fmt.Errorf("error generating response: %w", err)
	}

	// Print response
	fmt.Println(strings.TrimSpace(response))

	return nil
}
