package commands

import (
	"fmt"

	"github.com/mmichie/intu/pkg/ai"
	"github.com/mmichie/intu/pkg/prompts"
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
	prompt, ok := prompts.GetPrompt(promptName)
	if !ok {
		return fmt.Errorf("unknown prompt: %s", promptName)
	}

	input, err := readInput(args[1:])
	if err != nil {
		return fmt.Errorf("error reading input for AI command (prompt: %s): %w", promptName, err)
	}

	formattedPrompt, err := prompt.Format(input)
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

	var pipeline ai.Pipeline
	if len(parallel) > 0 {
		providers := make([]ai.Provider, len(parallel))
		for i, name := range parallel {
			provider, err := ai.NewProvider(name)
			if err != nil {
				return err
			}
			providers[i] = provider
		}

		var combiner ai.ResultCombiner
		if bestPicker {
			defaultProvider, err := selectProvider()
			if err != nil {
				return err
			}
			combiner = ai.NewBestPickerCombiner(defaultProvider)
		} else {
			combiner = ai.NewConcatCombiner(separator)
		}

		pipeline = ai.NewParallelPipeline(providers, combiner)
	} else if len(serial) > 0 {
		providers := make([]ai.Provider, len(serial))
		for i, name := range serial {
			provider, err := ai.NewProvider(name)
			if err != nil {
				return err
			}
			providers[i] = provider
		}
		pipeline = ai.NewSerialPipeline(providers)
	} else {
		provider, err := selectProvider()
		if err != nil {
			return err
		}
		pipeline = ai.NewSerialPipeline([]ai.Provider{provider})
	}

	result, err := pipeline.Execute(cmd.Context(), userPrompt)
	if err != nil {
		return fmt.Errorf("error executing pipeline: %w", err)
	}

	fmt.Println(result)
	return nil
}
