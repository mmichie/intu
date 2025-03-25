package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/spf13/viper"
)

func selectProvider() (aikit.Provider, error) {
	// Get default provider from config
	providerName := viper.GetString("default_provider")
	if providerName == "" {
		providerName = "openai" // Fallback default
	}

	// If provider is specified in command line flags, use that instead
	if flagProvider := viper.GetString("provider"); flagProvider != "" {
		providerName = flagProvider
	}

	return aikit.NewProvider(providerName)
}

// Helper function to read input from args or stdin
func readInput(args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Data is being piped to stdin
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("error reading from stdin: %w", err)
		}
		return strings.TrimSpace(string(bytes)), nil
	}

	return "", nil
}

// readInputFromArgsOrStdin reads input from args or stdin
func readInputFromArgsOrStdin(args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Data is being piped to stdin
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("error reading from stdin: %w", err)
		}
		return strings.TrimSpace(string(bytes)), nil
	}

	return "", nil
}

// Helper function to check for empty input
func checkEmptyInput(input string) error {
	if input == "" {
		return fmt.Errorf("no input provided")
	}
	return nil
}
