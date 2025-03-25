package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

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

func runModelsCommand(cmd *cobra.Command, args []string) error {
	modelsMap, err := aikit.GetProviderModels()
	if err != nil {
		return fmt.Errorf("error getting models: %w", err)
	}

	// Get a sorted list of provider names for consistent output
	providers := make([]string, 0, len(modelsMap))
	for provider := range modelsMap {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	// Display models for each provider
	for _, provider := range providers {
		models := modelsMap[provider]
		if len(models) == 0 {
			fmt.Printf("Provider: %s (no models available or API key not set)\n", provider)
			continue
		}

		fmt.Printf("Provider: %s\n", provider)
		for _, model := range models {
			fmt.Printf("  - %s\n", model)
		}
		fmt.Println()
	}

	return nil
}

// createProviders creates a slice of providers from a slice of provider names
func createProviders(providerNames []string) ([]aikit.Provider, error) {
	providers := make([]aikit.Provider, len(providerNames))
	for i, name := range providerNames {
		provider, err := aikit.NewProvider(name)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider '%s': %w", name, err)
		}
		providers[i] = provider
	}
	return providers, nil
}

func runAskCommand(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("ask command requires at least one argument for the prompt")
	}

	userPrompt := args[0]

	parallel, err := cmd.Flags().GetStringSlice("parallel")
	if err != nil {
		fmt.Printf("Warning: error getting 'parallel' flag: %v\n", err)
	}

	serial, err := cmd.Flags().GetStringSlice("serial")
	if err != nil {
		fmt.Printf("Warning: error getting 'serial' flag: %v\n", err)
	}

	bestPicker, err := cmd.Flags().GetBool("best")
	if err != nil {
		fmt.Printf("Warning: error getting 'best' flag: %v\n", err)
	}

	separator, err := cmd.Flags().GetString("separator")
	if err != nil {
		fmt.Printf("Warning: error getting 'separator' flag: %v\n", err)
	}

	var pipeline aikit.Pipeline
	if len(parallel) > 0 {
		providers, err := createProviders(parallel)
		if err != nil {
			return err
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
		providers, err := createProviders(serial)
		if err != nil {
			return err
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

func runJuryCommand(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("jury command requires at least one argument for the prompt")
	}

	userPrompt := args[0]

	// Get providers that will generate responses
	providers, err := cmd.Flags().GetStringSlice("providers")
	if err != nil || len(providers) == 0 {
		return fmt.Errorf("jury command requires at least one provider via --providers flag")
	}

	// Get jury members
	jurors, err := cmd.Flags().GetStringSlice("jurors")
	if err != nil || len(jurors) == 0 {
		// If no jury members specified, use all providers as jurors
		jurors = providers
	}

	// Get voting method
	votingMethod, err := cmd.Flags().GetString("voting")
	if err != nil {
		fmt.Printf("Warning: error getting 'voting' flag: %v\n", err)
		votingMethod = "majority"
	}

	// Create provider instances
	providerInstances, err := createProviders(providers)
	if err != nil {
		return err
	}

	// Create juror instances
	jurorInstances, err := createProviders(jurors)
	if err != nil {
		return err
	}

	// Create the jury combiner
	juryCombiner := aikit.NewJuryCombiner(jurorInstances, votingMethod)

	// Create and execute the pipeline
	pipeline := aikit.NewParallelPipeline(providerInstances, juryCombiner)
	result, err := pipeline.Execute(cmd.Context(), userPrompt)
	if err != nil {
		return fmt.Errorf("error executing jury pipeline: %w", err)
	}

	fmt.Println(result)
	return nil
}

func runCollabCommand(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("collab command requires at least one argument for the prompt")
	}

	userPrompt := args[0]

	// Get providers
	providers, err := cmd.Flags().GetStringSlice("providers")
	if err != nil || len(providers) == 0 {
		return fmt.Errorf("collab command requires at least one provider via --providers flag")
	}

	// Get number of rounds
	rounds, err := cmd.Flags().GetInt("rounds")
	if err != nil {
		rounds = 3
	}

	// Create provider instances
	providerInstances, err := createProviders(providers)
	if err != nil {
		return err
	}

	// Create and execute the pipeline
	pipeline := aikit.NewCollaborativePipeline(providerInstances, rounds)
	result, err := pipeline.Execute(cmd.Context(), userPrompt)
	if err != nil {
		return fmt.Errorf("error executing collaborative pipeline: %w", err)
	}

	fmt.Println(result)
	return nil
}

func runPipelineCommand(cmd *cobra.Command, args []string) error {
	// Check if we're listing pipelines
	list, err := cmd.Flags().GetBool("list")
	if err != nil {
		fmt.Printf("Warning: error getting 'list' flag: %v\n", err)
	}

	if list {
		return listPipelines(cmd.Context())
	}

	// Check for create operation
	create, err := cmd.Flags().GetBool("create")
	if err != nil {
		fmt.Printf("Warning: error getting 'create' flag: %v\n", err)
	}

	if create {
		if len(args) < 1 {
			return fmt.Errorf("pipeline create requires a name")
		}
		return createPipeline(cmd)
	}

	// Default run operation
	if len(args) < 2 {
		return fmt.Errorf("pipeline command requires a pipeline name and a prompt")
	}

	pipelineName := args[0]
	userPrompt := args[1]

	// Get the pipeline configuration
	pipelineConfig, err := aikit.GetPipelineConfig(cmd.Context(), pipelineName)
	if err != nil {
		return fmt.Errorf("error loading pipeline configuration: %w", err)
	}

	// Create the pipeline
	pipeline, err := createPipelineFromConfig(pipelineConfig)
	if err != nil {
		return fmt.Errorf("error creating pipeline: %w", err)
	}

	// Execute the pipeline
	result, err := pipeline.Execute(cmd.Context(), userPrompt)
	if err != nil {
		return fmt.Errorf("error executing pipeline: %w", err)
	}

	fmt.Println(result)
	return nil
}

// listPipelines lists all available pipelines
func listPipelines(ctx context.Context) error {
	pipelineNames, err := aikit.ListPipelineConfigs(ctx)
	if err != nil {
		return fmt.Errorf("error listing pipelines: %w", err)
	}

	if len(pipelineNames) == 0 {
		fmt.Println("No pipelines found. Use 'intu ai pipeline --create' to create one.")
		return nil
	}

	fmt.Println("Available pipelines:")
	for _, name := range pipelineNames {
		config, err := aikit.GetPipelineConfig(ctx, name)
		if err != nil {
			fmt.Printf("- %s (error loading config: %v)\n", name, err)
			continue
		}
		fmt.Printf("- %s (type: %s)\n", name, config.Type)
	}

	return nil
}

// createPipeline creates a new pipeline configuration
func createPipeline(cmd *cobra.Command) error {
	pipelineName := cmd.Flags().Arg(0)
	if pipelineName == "" {
		return fmt.Errorf("pipeline name is required")
	}

	// Get pipeline type
	pipelineType, err := cmd.Flags().GetString("type")
	if err != nil || pipelineType == "" {
		return fmt.Errorf("pipeline type is required (--type flag)")
	}

	config := aikit.PipelineConfig{
		Type: pipelineType,
	}

	// Process common parameters
	providers, err := cmd.Flags().GetStringSlice("providers")
	if err == nil && len(providers) > 0 {
		config.Providers = providers
	}

	// Process type-specific parameters
	switch pipelineType {
	case "serial":
		// Serial pipeline just needs providers, which we already set

	case "parallel":
		// Get combiner type
		combiner, err := cmd.Flags().GetString("combiner")
		if err == nil && combiner != "" {
			config.Combiner = combiner
		}

		// Get judge for best-picker combiner
		judge, err := cmd.Flags().GetString("judge")
		if err == nil && judge != "" {
			config.Judge = judge
		}

		// Get jurors for jury combiner
		jurors, err := cmd.Flags().GetStringSlice("jurors")
		if err == nil && len(jurors) > 0 {
			config.Jurors = jurors
		}

		// Get voting method for jury combiner
		voting, err := cmd.Flags().GetString("voting")
		if err == nil && voting != "" {
			config.Voting = voting
		}

		// Get separator for concat combiner
		separator, err := cmd.Flags().GetString("separator")
		if err == nil && separator != "" {
			config.Separator = separator
		}

	case "collaborative":
		// Get number of rounds
		rounds, err := cmd.Flags().GetInt("rounds")
		if err == nil && rounds > 0 {
			config.Rounds = rounds
		}

	case "nested":
		// Nested pipelines are more complex and would typically be defined in a configuration file
		// This is just a placeholder for CLI-based creation
		return fmt.Errorf("nested pipelines cannot be created via CLI flags; please edit the JSON configuration file directly")

	default:
		return fmt.Errorf("unknown pipeline type: %s", pipelineType)
	}

	// Save the configuration
	if err := aikit.AddOrUpdatePipelineConfig(cmd.Context(), pipelineName, config); err != nil {
		return fmt.Errorf("error saving pipeline configuration: %w", err)
	}

	fmt.Printf("Pipeline '%s' created successfully.\n", pipelineName)
	return nil
}

// Helper function to create a pipeline from a configuration
func createPipelineFromConfig(config aikit.PipelineConfig) (aikit.Pipeline, error) {
	switch config.Type {
	case "serial":
		if len(config.Providers) == 0 {
			return nil, fmt.Errorf("serial pipeline configuration missing providers")
		}
		providers, err := createProviders(config.Providers)
		if err != nil {
			return nil, err
		}
		return aikit.NewSerialPipeline(providers), nil

	case "parallel":
		if len(config.Providers) == 0 {
			return nil, fmt.Errorf("parallel pipeline configuration missing providers")
		}
		providers, err := createProviders(config.Providers)
		if err != nil {
			return nil, err
		}

		var combiner aikit.ResultCombiner
		switch config.Combiner {
		case "best-picker":
			judgeProvider := config.Judge
			if judgeProvider == "" {
				judgeProvider = config.Providers[0]
			}
			judge, err := aikit.NewProvider(judgeProvider)
			if err != nil {
				return nil, err
			}
			combiner = aikit.NewBestPickerCombiner(judge)

		case "jury":
			jurorNames := config.Jurors
			if len(jurorNames) == 0 {
				jurorNames = config.Providers
			}
			jurors, err := createProviders(jurorNames)
			if err != nil {
				return nil, err
			}
			votingMethod := config.Voting
			if votingMethod == "" {
				votingMethod = "majority"
			}
			combiner = aikit.NewJuryCombiner(jurors, votingMethod)

		default:
			// Default to concat combiner
			separator := config.Separator
			if separator == "" {
				separator = "\n\n"
			}
			combiner = aikit.NewConcatCombiner(separator)
		}

		return aikit.NewParallelPipeline(providers, combiner), nil

	case "collaborative":
		if len(config.Providers) == 0 {
			return nil, fmt.Errorf("collaborative pipeline configuration missing providers")
		}
		providers, err := createProviders(config.Providers)
		if err != nil {
			return nil, err
		}

		rounds := config.Rounds
		if rounds <= 0 {
			rounds = 3
		}

		return aikit.NewCollaborativePipeline(providers, rounds), nil

	case "nested":
		if len(config.Stages) == 0 {
			return nil, fmt.Errorf("nested pipeline configuration missing stages")
		}

		// Convert stages from map[string]interface{} to PipelineConfig
		pipelineConfigs := make([]aikit.PipelineConfig, len(config.Stages))
		for i, stageConfig := range config.Stages {
			// Convert map to JSON and back to PipelineConfig
			jsonData, err := json.Marshal(stageConfig)
			if err != nil {
				return nil, fmt.Errorf("error marshaling stage %d: %w", i, err)
			}

			var pc aikit.PipelineConfig
			if err := json.Unmarshal(jsonData, &pc); err != nil {
				return nil, fmt.Errorf("error unmarshaling stage %d: %w", i, err)
			}

			pipelineConfigs[i] = pc
		}

		// Create pipeline for each stage
		stages := make([]aikit.Pipeline, len(pipelineConfigs))
		for i, stageConfig := range pipelineConfigs {
			stage, err := createPipelineFromConfig(stageConfig)
			if err != nil {
				return nil, fmt.Errorf("error creating stage %d: %w", i, err)
			}
			stages[i] = stage
		}

		return aikit.NewNestedPipeline(stages), nil

	default:
		return nil, fmt.Errorf("unknown pipeline type: %s", config.Type)
	}
}
