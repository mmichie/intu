package commands

import (
	"fmt"

	"github.com/mmichie/intu/pkg/aikit"
	"github.com/mmichie/intu/pkg/aikit/prompt"
	"github.com/spf13/cobra"
)

func runAICommand(cmd *cobra.Command, args []string) error {
	list, err := cmd.Flags().GetBool("list")
	if err != nil {
		return fmt.Errorf("error getting 'list' flag: %w", err)
	}
	if list {
		return listPrompts()
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	promptName := args[0]
	p, ok := prompt.GetPrompt(promptName)
	if !ok {
		return fmt.Errorf("unknown prompt: %s", promptName)
	}

	input, err := readInput(args[1:])
	if err != nil {
		return fmt.Errorf("error reading input for AI command (prompt: %s): %w", promptName, err)
	}

	formattedPrompt, err := p.Format(input)
	if err != nil {
		return fmt.Errorf("error formatting prompt '%s' for AI command: %w", promptName, err)
	}

	return processWithAI(cmd.Context(), input, formattedPrompt)
}

func runAskCommand(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ask command requires at least one argument for the prompt")
	}

	userPrompt := args[0]

	parallel, _ := cmd.Flags().GetStringSlice("parallel")
	serial, _ := cmd.Flags().GetStringSlice("serial")
	bestPicker, _ := cmd.Flags().GetBool("best")
	separator, _ := cmd.Flags().GetString("separator")

	var pipeline aikit.Pipeline
	if len(parallel) > 0 {
		providers := make([]aikit.Provider, len(parallel))
		for i, name := range parallel {
			provider, err := aikit.NewProvider(name)
			if err != nil {
				return err
			}
			providers[i] = provider
		}

		var combiner aikit.ResultCombiner
		if bestPicker {
			defaultProvider, err := selectProvider()
			if err != nil {
				return err
			}
			combiner = aikit.NewBestPickerCombiner(defaultProvider)
		} else {
			combiner = aikit.NewConcatCombiner(separator)
		}

		pipeline = aikit.NewParallelPipeline(providers, combiner)
	} else if len(serial) > 0 {
		providers := make([]aikit.Provider, len(serial))
		for i, name := range serial {
			provider, err := aikit.NewProvider(name)
			if err != nil {
				return err
			}
			providers[i] = provider
		}
		pipeline = aikit.NewSerialPipeline(providers)
	} else {
		provider, err := selectProvider()
		if err != nil {
			return err
		}
		pipeline = aikit.NewSerialPipeline([]aikit.Provider{provider})
	}

	result, err := pipeline.Execute(cmd.Context(), userPrompt)
	if err != nil {
		return fmt.Errorf("error executing pipeline: %w", err)
	}

	fmt.Println(result)
	return nil
}