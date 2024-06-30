package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/mmichie/intu/pkg/prompts"
	"github.com/mmichie/intu/ui/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start the Text User Interface",
	Long:  `Start an interactive Text User Interface for communicating with the AI assistant.`,
	RunE:  runTUICommand,
}

// InitAICommand initializes and adds the AI commands to the root command
func InitAICommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(askCmd)
	aiCmd.AddCommand(tuiCmd)
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

	// For the ask command, we don't use a pre-defined prompt template
	// Instead, we use the user's prompt directly
	return processWithAI(cmd.Context(), input, userPrompt)
}

func runTUICommand(cmd *cobra.Command, args []string) error {
	provider, err := selectProvider()
	if err != nil {
		return err
	}
	client := intu.NewClient(provider)

	// Get terminal size
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// If we can't get the terminal size, use default values
		width, height = 80, 24
	}

	return tui.StartTUI(cmd.Context(), client, width, height)
}

func processWithAI(ctx context.Context, input, promptText string) error {
	// Create AI client
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider for AI command: %w", err)
	}

	client := intu.NewClient(provider)

	// Process input with AI
	result, err := client.ProcessWithAI(ctx, input, promptText)
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
