package commands

import (
	"fmt"

	"github.com/mmichie/intu/pkg/prompts"
	"github.com/spf13/cobra"
)

func runAICommand(cmd *cobra.Command, args []string) error {
	list, err := cmd.Flags().GetBool("list")
	if err != nil {
		return fmt.Errorf("error getting 'list' flag: %w", err)
	}
	if list {
		return listPrompts()
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	promptName := args[0]
	prompt, ok := prompts.GetPrompt(promptName)
	if !ok {
		return fmt.Errorf("unknown prompt: %s", promptName)
	}

	input, err := readInput(args[1:])
	if err != nil {
		return fmt.Errorf("error reading input for AI command (prompt: %s): %w", promptName, err)
	}

	formattedPrompt, err := prompt.Format(input)
	if err != nil {
		return fmt.Errorf("error formatting prompt '%s' for AI command: %w", promptName, err)
	}

	return processWithAI(cmd.Context(), input, formattedPrompt)
}

func runAskCommand(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ask command requires at least one argument for the prompt")
	}

	userPrompt := args[0]
	input, err := readInput(args[1:])
	if err != nil {
		return fmt.Errorf("error reading input for ask command: %w", err)
	}

	return processWithAI(cmd.Context(), input, userPrompt)
}
