package cmd

import (
	"context"
	"fmt"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate a commit message",
	Long: `Generate a commit message based on the provided git diff.
	
	This command can be used in several ways:
	1. Pipe git diff directly: git diff --staged | intu commit
	2. Use in a git hook: Add 'intu commit' to your prepare-commit-msg hook
	3. Manual input: intu commit (then type or paste the diff, press Ctrl+D when done)`,
	RunE: runCommitCommand,
}

func init() {
	rootCmd.AddCommand(commitCmd)
}

func runCommitCommand(cmd *cobra.Command, args []string) error {
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}

	client := intu.NewClient(provider)

	diffOutput, err := readInput()
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	if diffOutput == "" {
		fmt.Println("No input received. Please provide git diff output.")
		fmt.Println("Usage: git diff --staged | intu commit")
		return nil
	}

	message, err := client.GenerateCommitMessage(context.Background(), diffOutput)
	if err != nil {
		return fmt.Errorf("error generating commit message: %w", err)
	}

	fmt.Println(message)
	return nil
}
