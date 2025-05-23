package pipeline_test

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
	"github.com/mmichie/intu/pkg/aikit/v2/pipeline"
	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

func ExampleFactory() {
	// Create a pipeline factory
	factory := pipeline.NewFactory().WithConfig(config.Config{
		APIKey: "your-api-key",
		Model:  "gpt-4",
	})

	// Create a simple pipeline
	simple, err := factory.CreateSimple("openai")
	if err != nil {
		log.Fatal(err)
	}

	// Use the pipeline
	ctx := context.Background()
	result, _ := simple.Execute(ctx, "Hello, world!")
	fmt.Println("Simple pipeline result:", result)
}

func ExampleFactory_serial() {
	factory := pipeline.NewFactory().WithConfig(config.Config{
		APIKey: "your-api-key",
	})

	// Create a serial pipeline that processes through multiple providers
	serial, err := factory.CreateSerial([]string{"openai", "claude", "gemini"})
	if err != nil {
		log.Fatal(err)
	}

	// Each provider processes the output of the previous one
	ctx := context.Background()
	result, _ := serial.Execute(ctx, "Translate to French: Hello")
	fmt.Println("Serial pipeline result:", result)
}

func ExampleFactory_parallel() {
	factory := pipeline.NewFactory().WithConfig(config.Config{
		APIKey: "your-api-key",
	})

	// Create a parallel pipeline with best picker
	parallel, err := factory.CreateParallelWithBestPicker(
		[]string{"openai", "claude"},
		"gemini", // Use Gemini to pick the best response
		pipeline.WithCache(300),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	result, _ := parallel.Execute(ctx, "Write a haiku about coding")
	fmt.Println("Best response selected:", result)
}

func ExampleBuilder() {
	// Use the builder pattern for complex pipelines
	pipeline, err := pipeline.NewBuilder().
		WithProvider("openai").
		WithProvider("claude").
		WithTransform(func(ctx context.Context, input string) (string, error) {
			// Add a prefix to the result
			return "Summary: " + input, nil
		}).
		WithOptions(
			pipeline.WithRetries(3),
			pipeline.WithCache(600),
		).
		BuildChain()

	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	result, _ := pipeline.Execute(ctx, "Explain quantum computing")
	fmt.Println(result)
}

func Example_combiners() {
	// Example of different combiners
	cfg := config.Config{APIKey: "test-key"}

	// Concatenate responses
	concat := pipeline.NewConcatCombiner("\n---\n")

	// Majority vote
	majority := pipeline.NewMajorityVoteCombiner()

	// Quality-based selection
	quality := pipeline.NewQualityScoreCombiner(100)

	// Use in a parallel pipeline
	providers, _ := createMockProviders(cfg)

	p1 := pipeline.NewParallelPipeline(providers, concat)
	p2 := pipeline.NewParallelPipeline(providers, majority)
	p3 := pipeline.NewParallelPipeline(providers, quality)

	ctx := context.Background()
	r1, _ := p1.Execute(ctx, "test")
	r2, _ := p2.Execute(ctx, "test")
	r3, _ := p3.Execute(ctx, "test")

	fmt.Println("Concatenated:", r1)
	fmt.Println("Majority vote:", r2)
	fmt.Println("Quality selected:", r3)
}

func ExampleNestedPipeline() {
	// Create a complex nested pipeline

	// Stage 1: Generate initial response
	stage1 := pipeline.NewFunctionAdapter("generate", func(ctx context.Context, input string) (string, error) {
		return "Generated: " + input, nil
	})

	// Stage 2: Enhance the response
	stage2 := pipeline.NewFunctionAdapter("enhance", func(ctx context.Context, input string) (string, error) {
		return strings.ReplaceAll(input, "Generated", "Enhanced"), nil
	})

	// Stage 3: Format the output
	stage3 := pipeline.NewFunctionAdapter("format", func(ctx context.Context, input string) (string, error) {
		return fmt.Sprintf("**%s**", input), nil
	})

	// Create nested pipeline
	nested := pipeline.NewNestedPipeline([]pipeline.Pipeline{stage1, stage2, stage3})

	ctx := context.Background()
	result, _ := nested.Execute(ctx, "Hello")
	fmt.Println("Nested result:", result)
	// Output: Nested result: **Enhanced: Hello**
}

func ExampleTransformAdapter() {
	// Create a pipeline with transformations

	// Base pipeline
	base := pipeline.NewFunctionAdapter("base", func(ctx context.Context, input string) (string, error) {
		return "Result: " + input, nil
	})

	// Input transformation: convert to uppercase
	inputTransform := func(ctx context.Context, input string) (string, error) {
		return strings.ToUpper(input), nil
	}

	// Output transformation: add timestamp
	outputTransform := func(ctx context.Context, output string) (string, error) {
		return fmt.Sprintf("[%s] %s", "2024-01-01", output), nil
	}

	// Create transform adapter
	transformed := pipeline.NewTransformAdapter(base, inputTransform, outputTransform)

	ctx := context.Background()
	result, _ := transformed.Execute(ctx, "hello world")
	fmt.Println(result)
	// Output: [2024-01-01] Result: HELLO WORLD
}

func ExampleCreateHighAvailabilityPipeline() {
	// Create a high-availability pipeline with automatic fallback
	cfg := config.Config{APIKey: "your-api-key"}

	ha, err := pipeline.CreateHighAvailabilityPipeline(cfg, []string{
		"openai", // Primary
		"claude", // First fallback
		"gemini", // Second fallback
		"grok",   // Third fallback
	})
	if err != nil {
		log.Fatal(err)
	}

	// The pipeline will automatically fall back if a provider fails
	ctx := context.Background()
	result, _ := ha.Execute(ctx, "What is the meaning of life?")
	fmt.Println("HA Pipeline result:", result)
}

func ExampleCreateConsensusPipeline() {
	// Create a consensus pipeline that synthesizes multiple responses
	cfg := config.Config{APIKey: "your-api-key"}

	consensus, err := pipeline.CreateConsensusPipeline(
		cfg,
		[]string{"openai", "claude", "gemini"}, // Providers to get responses from
		"gpt-4",                                // Provider to synthesize consensus
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	result, _ := consensus.Execute(ctx, "Explain photosynthesis")
	fmt.Println("Consensus result:", result)
}

// Helper function to create mock providers for examples
func createMockProviders(cfg config.Config) ([]provider.Provider, error) {
	// In real usage, you would use actual providers
	// This is just for example purposes
	return []provider.Provider{}, nil
}
