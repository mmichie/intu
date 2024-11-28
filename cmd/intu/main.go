package main

import (
	"fmt"
	"os"

	"github.com/mmichie/intu/internal/commands/root"
)

func main() {
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
