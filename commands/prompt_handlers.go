package commands

import (
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/prompt"
)

func listPrompts() error {
	fmt.Println("Available prompts:")
	for _, p := range prompt.AllPrompts {
		fmt.Printf("  %s: %s\n", p.Name, p.Description)
	}
	return nil
}
