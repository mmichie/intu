package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mmichie/intu/internal/ai"
	"github.com/spf13/viper"
)

func selectProvider() (ai.Provider, error) {
	providerName := viper.GetString("provider")
	return ai.SelectProvider(providerName)
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
			return "", fmt.Errorf("error reading input: %w", err)
		}
		input.WriteString(line)
	}

	return input.String(), nil
}
