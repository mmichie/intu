package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/spf13/cobra"
)

var (
	provider string
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate a commit message",
	Long: `Generate a commit message based on the provided git diff.
	
	This command can be used in several ways:
	1. Pipe git diff directly: git diff --staged | intu commit
	2. Use in a git hook: Add 'intu commit' to your prepare-commit-msg hook
	3. Manual input: intu commit (then type or paste the diff, press Ctrl+D when done)`,
	Run: runCommitCommand,
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().StringVarP(&provider, "provider", "p", "", "AI provider to use (openai or claude)")
}

func runCommitCommand(cmd *cobra.Command, args []string) {
	// Create a new IntuClient with the specified provider
	client, err := intu.NewIntuClient(provider)
	if err != nil {
		log.Fatalf("Error creating IntuClient: %v", err)
	}

	// Read input from stdin
	diffOutput, err := readInput()
	if err != nil {
		log.Fatalf("Error reading input: %v", err)
	}

	// If there's no input, inform the user and exit
	if diffOutput == "" {
		fmt.Println("No input received. Please provide git diff output.")
		fmt.Println("Usage: git diff --staged | intu commit")
		return
	}

	// Generate the commit message using the diff output
	message, err := client.GenerateCommitMessage(diffOutput)
	if err != nil {
		log.Fatalf("Error generating commit message: %v", err)
	}

	// Print the generated commit message to stdout
	fmt.Println(message)
}

func readInput() (string, error) {
	var input strings.Builder
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		input.WriteString(line)
	}

	return input.String(), nil
}
