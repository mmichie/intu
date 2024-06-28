package cmd

import (
	"errors"
	"fmt"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/mmichie/intu/pkg/prompts"
	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai [prompt-name]",
	Short: "AI-powered commands",
	Long:  `AI-powered commands for various tasks.`,
	RunE:  runAICommand,
}

var askCmd = &cobra.Command{
	Use:   "ask <prompt> [input]",
	Short: "Ask the AI a free-form question",
	Long:  `Ask the AI a free-form question or provide a custom prompt. Input can be provided via stdin or as an argument.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAskCommand,
}

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(askCmd)

	// Add a flag for listing available prompts
	aiCmd.PersistentFlags().BoolP("list", "l", false, "List available prompts")
}

func runAICommand(cmd *cobra.Command, args []string) error {
	// Check if the user wants to list prompts
	list, err := cmd.Flags().GetBool("list")
	if err != nil {
		return fmt.Errorf("error getting 'list' flag: %w", err)
	}
	if list {
		return listPrompts()
	}

	// If no arguments are provided, show help
	if len(args) == 0 {
		return cmd.Help()
	}

	// Handle pre-canned prompts
	promptName := args[0]
	prompt, ok := prompts.GetPrompt(promptName)
	if !ok {
		return fmt.Errorf("unknown prompt: %s", promptName)
	}

	input, err := readInput(args[1:])
	if err != nil {
		return err
	}

	formattedPrompt, err := prompt.Format(input)
	if err != nil {
		return fmt.Errorf("error formatting prompt: %w", err)
	}

	return processWithAI(cmd, input, formattedPrompt)
}

func runAskCommand(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("prompt is required")
	}

	userPrompt := args[0]
	input, err := readInput(args[1:])
	if err != nil {
		return err
	}

	// For the ask command, we don't use a pre-defined prompt template
	// Instead, we use the user's prompt directly
	return processWithAI(cmd, input, userPrompt)
}

func processWithAI(cmd *cobra.Command, input, promptText string) error {
	// Create AI client
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}
	client := intu.NewClient(provider)

	// Process input with AI
	result, err := client.ProcessWithAI(cmd.Context(), input, promptText)
	if err != nil {
		return fmt.Errorf("error processing with AI: %w", err)
	}

	// Output result
	fmt.Println(result)
	return nil
}

func listPrompts() error {
	fmt.Println("Available prompts:")
	for _, p := range prompts.AllPrompts {
		fmt.Printf("  %s: %s\n", p.Name, p.Description)
	}
	return nil
}
