package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/mmichie/intu/pkg/prompts"
	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai [prompt-name]",
	Short: "Process input with AI using pre-canned prompts",
	Long:  `Process input from stdin using an AI provider with pre-canned prompts.`,
	RunE:  runAICommand,
}

func init() {
	rootCmd.AddCommand(aiCmd)

	// Add a flag for listing available prompts
	aiCmd.Flags().BoolP("list", "l", false, "List available prompts")
}

func runAICommand(cmd *cobra.Command, args []string) error {
	// Check if the user wants to list prompts
	if list, _ := cmd.Flags().GetBool("list"); list {
		listPrompts()
		return nil
	}

	// Ensure a prompt name is provided if not listing
	if len(args) == 0 {
		return fmt.Errorf("prompt name is required when not using --list")
	}

	promptName := args[0]
	prompt, ok := prompts.GetPrompt(promptName)
	if !ok {
		return fmt.Errorf("unknown prompt: %s", promptName)
	}

	// Read input from stdin
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	// Create AI client
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}
	client := intu.NewClient(provider)

	// Process input with AI
	result, err := client.ProcessWithAI(cmd.Context(), string(input), prompt.Format(string(input)))
	if err != nil {
		return fmt.Errorf("error processing with AI: %w", err)
	}

	// Output result
	fmt.Println(result)
	return nil
}

func listPrompts() {
	fmt.Println("Available prompts:")
	for _, p := range prompts.AllPrompts {
		fmt.Printf("  %s: %s\n", p.Name, p.Description)
	}
}
