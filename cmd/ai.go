package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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
	if list, _ := cmd.Flags().GetBool("list"); list {
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

	// Read input from stdin
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return processWithAI(cmd, string(input), prompt.Format(string(input)))
}

func runAskCommand(cmd *cobra.Command, args []string) error {
	prompt := args[0]
	var input string

	// Check if input is provided as an argument
	if len(args) > 1 {
		input = strings.Join(args[1:], " ")
	} else {
		// If no input argument, try reading from stdin
		stdinInfo, err := os.Stdin.Stat()
		if err != nil {
			return fmt.Errorf("error checking stdin: %w", err)
		}

		if (stdinInfo.Mode() & os.ModeCharDevice) == 0 {
			// Data is being piped to stdin
			stdinData, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("error reading from stdin: %w", err)
			}
			input = string(stdinData)
		}
	}

	return processWithAI(cmd, input, prompt)
}

func processWithAI(cmd *cobra.Command, input, prompt string) error {
	// Create AI client
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}
	client := intu.NewClient(provider)

	// Process input with AI
	result, err := client.ProcessWithAI(cmd.Context(), input, prompt)
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
