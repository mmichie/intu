package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/providers"
	"github.com/mmichie/intu/pkg/security"
	"github.com/mmichie/intu/pkg/tools"
)

// InitEditCommand initializes and adds the edit command to the root command
func InitEditCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(RegisterEditCommand())
}

// RegisterEditCommand registers the edit command
func RegisterEditCommand() *cobra.Command {
	editCmd := &cobra.Command{
		Use:   "edit [file_path]",
		Short: "Edit a file by replacing text",
		Long:  `Edit a file by replacing specified text with new text. Handles string replacement with safety checks.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runEditCommand,
	}

	// Add flags
	editCmd.Flags().StringP("old", "o", "", "Text to replace (required)")
	editCmd.Flags().StringP("new", "n", "", "New text to insert (required)")
	editCmd.Flags().IntP("count", "c", 1, "Expected number of replacements")
	editCmd.Flags().BoolP("from-file", "f", false, "Read old and new text from files instead of parameters")
	editCmd.MarkFlagRequired("old")
	editCmd.MarkFlagRequired("new")

	return editCmd
}

func runEditCommand(cmd *cobra.Command, args []string) error {
	// Get file path from args
	filePath := args[0]

	// Get flags
	oldText, _ := cmd.Flags().GetString("old")
	newText, _ := cmd.Flags().GetString("new")
	count, _ := cmd.Flags().GetInt("count")
	fromFile, _ := cmd.Flags().GetBool("from-file")

	// If reading from files, load the content
	if fromFile {
		oldContent, err := ioutil.ReadFile(oldText)
		if err != nil {
			return fmt.Errorf("failed to read old text file: %w", err)
		}
		oldText = string(oldContent)

		newContent, err := ioutil.ReadFile(newText)
		if err != nil {
			return fmt.Errorf("failed to read new text file: %w", err)
		}
		newText = string(newContent)
	}

	// Create provider
	provider, err := aikit.NewProvider("claude")
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Create Edit tool
	editTool := tools.NewEditTool()

	// Register tool with provider
	err = provider.RegisterFunction(providers.FunctionDefinition{
		Name:        editTool.Name(),
		Description: editTool.Description(),
		Parameters:  editTool.ParameterSchema(),
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
	registry.Register(editTool)

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

	// Create prompt asking for the edit
	prompt := fmt.Sprintf("Please edit the file at '%s' by replacing the specified text. ", filePath)
	prompt += fmt.Sprintf("Replace '%s' with '%s'. ", oldText, newText)
	prompt += fmt.Sprintf("Expect %d replacements. ", count)
	prompt += "Only use the Edit function and return a success message with details of what was changed."

	// Call provider with function calling
	response, err := provider.GenerateResponseWithFunctions(context.Background(), prompt, functionExecutor)
	if err != nil {
		return fmt.Errorf("error generating response: %w", err)
	}

	// Print response
	fmt.Println(strings.TrimSpace(response))

	return nil
}
