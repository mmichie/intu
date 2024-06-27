package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/mmichie/intu/internal/ai"
	"github.com/spf13/viper"
)

func selectProvider() (ai.Provider, error) {
	providerName := viper.GetString("provider")
	return ai.SelectProvider(providerName)
}

func readInput(args []string) (string, error) {
	// Check if there's input from stdin
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Data is being piped to stdin
		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("error reading from stdin: %w", err)
		}
		return strings.TrimSpace(string(bytes)), nil
	}

	// If no stdin input, use args if provided
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}

	// No input provided
	return "", nil
}
