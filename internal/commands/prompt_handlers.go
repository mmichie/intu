package commands

import (
	"fmt"

	"github.com/mmichie/intu/pkg/prompts"
)

func listPrompts() error {
	fmt.Println("Available prompts:")
	for _, p := range prompts.AllPrompts {
		fmt.Printf("  %s: %s\n", p.Name, p.Description)
	}
	return nil
}
