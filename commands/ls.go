package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/providers"
	"github.com/mmichie/intu/pkg/tools"
	"github.com/spf13/cobra"
)

// RegisterLSCommand registers the ls command
func RegisterLSCommand() *cobra.Command {
	lsCmd := &cobra.Command{
		Use:   "ls [path]",
		Short: "List directory contents using AI function calling",
		Long:  `List directory contents using AI function calling capability. This demonstrates the function calling API.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runLSCommand,
	}

	return lsCmd
}

func runLSCommand(cmd *cobra.Command, args []string) error {
	// Get path from args or use current directory
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Create Claude provider
	provider, err := aikit.NewProvider("claude")
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Create LS tool
	lsTool := tools.NewLSTool()

	// Register tool with provider
	err = provider.RegisterFunction(providers.FunctionDefinition{
		Name:        lsTool.Name(),
		Description: lsTool.Description(),
		Parameters:  lsTool.ParameterSchema(),
	})
	if err != nil {
		return fmt.Errorf("failed to register function: %w", err)
	}

	// Create tool registry
	registry := tools.NewRegistry()
	registry.Register(lsTool)

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

	// Create prompt asking to list directory
	prompt := fmt.Sprintf("Please list the contents of the directory at path '%s'. "+
		"Only use the LS function and return the information in a clear, organized format.", path)

	// Call provider with function calling
	response, err := provider.GenerateResponseWithFunctions(context.Background(), prompt, functionExecutor)
	if err != nil {
		return fmt.Errorf("error generating response: %w", err)
	}

	// Print response
	fmt.Println(response)

	return nil
}

// FunctionCall represents a function call from the Claude provider to the tool
type FunctionCall struct {
	Name       string
	Parameters json.RawMessage
}
