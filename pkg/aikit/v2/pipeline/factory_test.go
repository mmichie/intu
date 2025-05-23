package pipeline

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// Mock provider for testing
type mockProvider struct {
	name     string
	response string
	err      error
}

func (m *mockProvider) GenerateResponse(ctx context.Context, request provider.Request) (provider.Response, error) {
	if m.err != nil {
		return provider.Response{}, m.err
	}
	return provider.Response{
		Content:  m.response,
		Provider: m.name,
		Model:    "mock",
	}, nil
}

func (m *mockProvider) GenerateStreamingResponse(ctx context.Context, request provider.Request, handler provider.StreamHandler) error {
	return fmt.Errorf("streaming not implemented")
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Model() string {
	return "mock"
}

func (m *mockProvider) Capabilities() []string {
	return []string{"test"}
}

// Mock provider factory
type mockProviderFactory struct {
	providers map[string]provider.Provider
}

func (f *mockProviderFactory) Name() string {
	return "mock"
}

func (f *mockProviderFactory) Create(cfg config.Config) (provider.Provider, error) {
	// Return different mock providers based on model
	if p, ok := f.providers[cfg.Model]; ok {
		return p, nil
	}
	return &mockProvider{name: "mock", response: "mock response"}, nil
}

func (f *mockProviderFactory) GetAvailableModels() []string {
	models := make([]string, 0, len(f.providers))
	for model := range f.providers {
		models = append(models, model)
	}
	return models
}

func (f *mockProviderFactory) GetCapabilities() []string {
	return []string{"test"}
}

func TestFactory(t *testing.T) {
	// Create a factory with mock registry
	registry := provider.NewRegistry()

	// Register mock providers
	mockFactory := &mockProviderFactory{
		providers: map[string]provider.Provider{
			"mock1": &mockProvider{name: "mock1", response: "response1"},
			"mock2": &mockProvider{name: "mock2", response: "response2"},
			"mock3": &mockProvider{name: "mock3", response: "response3"},
		},
	}

	registry.RegisterFactory(mockFactory)

	factory := NewFactoryWithRegistry(registry).WithConfig(config.Config{Model: "mock1"})

	t.Run("CreateSimple", func(t *testing.T) {
		pipeline, err := factory.CreateSimple("mock")
		if err != nil {
			t.Fatalf("Failed to create simple pipeline: %v", err)
		}

		result, err := pipeline.Execute(context.Background(), "test")
		if err != nil {
			t.Fatalf("Failed to execute pipeline: %v", err)
		}

		if result != "response1" {
			t.Errorf("Expected 'response1', got '%s'", result)
		}
	})

	t.Run("CreateSerial", func(t *testing.T) {
		// For serial, we need different configs for different providers
		factory2 := NewFactoryWithRegistry(registry).WithConfig(config.Config{Model: "mock2"})
		factory3 := NewFactoryWithRegistry(registry).WithConfig(config.Config{Model: "mock3"})

		// Create providers individually
		p1, _ := factory.createProvider("mock")
		p2, _ := factory2.createProvider("mock")
		p3, _ := factory3.createProvider("mock")

		pipeline := NewSerialPipeline([]provider.Provider{p1, p2, p3})

		result, err := pipeline.Execute(context.Background(), "test")
		if err != nil {
			t.Fatalf("Failed to execute serial pipeline: %v", err)
		}

		// Serial pipeline returns the last result
		if result != "response3" {
			t.Errorf("Expected 'response3', got '%s'", result)
		}
	})

	t.Run("CreateParallelWithCombiner", func(t *testing.T) {
		factory2 := NewFactoryWithRegistry(registry).WithConfig(config.Config{Model: "mock2"})

		p1, _ := factory.createProvider("mock")
		p2, _ := factory2.createProvider("mock")

		combiner := NewConcatCombiner(" | ")
		pipeline := NewParallelPipeline([]provider.Provider{p1, p2}, combiner)

		result, err := pipeline.Execute(context.Background(), "test")
		if err != nil {
			t.Fatalf("Failed to execute parallel pipeline: %v", err)
		}

		if !strings.Contains(result, "response1") || !strings.Contains(result, "response2") {
			t.Errorf("Expected concatenated responses, got '%s'", result)
		}
	})
}

func TestBuilder(t *testing.T) {
	// Create mock providers
	p1 := &mockProvider{name: "p1", response: "response1"}
	p2 := &mockProvider{name: "p2", response: "response2"}

	// Create a mock registry
	registry := provider.NewRegistry()
	mockFactory := &mockProviderFactory{
		providers: map[string]provider.Provider{
			"p1": p1,
			"p2": p2,
		},
	}
	registry.RegisterFactory(mockFactory)

	factory := NewFactoryWithRegistry(registry)

	t.Run("BuildSimple", func(t *testing.T) {
		pipeline, err := NewBuilder().
			WithFactory(factory.WithConfig(config.Config{Model: "p1"})).
			WithProvider("mock").
			BuildSimple()

		if err != nil {
			t.Fatalf("Failed to build simple pipeline: %v", err)
		}

		result, err := pipeline.Execute(context.Background(), "test")
		if err != nil {
			t.Fatalf("Failed to execute pipeline: %v", err)
		}

		if result != "response1" {
			t.Errorf("Expected 'response1', got '%s'", result)
		}
	})

	t.Run("BuildWithTransforms", func(t *testing.T) {
		// Create transform functions
		addPrefix := func(ctx context.Context, input string) (string, error) {
			return "PREFIX: " + input, nil
		}

		addSuffix := func(ctx context.Context, input string) (string, error) {
			return input + " :SUFFIX", nil
		}

		pipeline, err := NewBuilder().
			WithFactory(factory.WithConfig(config.Config{Model: "p1"})).
			WithProvider("mock").
			WithTransform(addPrefix).
			WithTransform(addSuffix).
			BuildChain()

		if err != nil {
			t.Fatalf("Failed to build chain pipeline: %v", err)
		}

		result, err := pipeline.Execute(context.Background(), "test")
		if err != nil {
			t.Fatalf("Failed to execute pipeline: %v", err)
		}

		// The transforms should be applied to the provider's response
		expected := "PREFIX: response1 :SUFFIX"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})
}

func TestCombiners(t *testing.T) {
	// Create test responses
	responses := []provider.Response{
		{Content: "response1", Provider: "p1"},
		{Content: "response2", Provider: "p2"},
		{Content: "response1", Provider: "p3"}, // Duplicate for majority vote
	}

	ctx := context.Background()

	t.Run("ConcatCombiner", func(t *testing.T) {
		combiner := NewConcatCombiner(" | ")
		result, err := combiner.Combine(ctx, responses)

		if err != nil {
			t.Fatalf("Combine failed: %v", err)
		}

		expected := "response1 | response2 | response1"
		if result.Content != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result.Content)
		}
	})

	t.Run("MajorityVoteCombiner", func(t *testing.T) {
		combiner := NewMajorityVoteCombiner()
		result, err := combiner.Combine(ctx, responses)

		if err != nil {
			t.Fatalf("Combine failed: %v", err)
		}

		if result.Content != "response1" {
			t.Errorf("Expected 'response1' (majority), got '%s'", result.Content)
		}

		// Check metadata
		if votes, ok := result.Metadata["votes"].(int); !ok || votes != 2 {
			t.Errorf("Expected 2 votes, got %v", result.Metadata["votes"])
		}
	})

	t.Run("LongestResponseCombiner", func(t *testing.T) {
		longerResponses := []provider.Response{
			{Content: "short", Provider: "p1"},
			{Content: "this is a much longer response", Provider: "p2"},
			{Content: "medium length", Provider: "p3"},
		}

		combiner := NewLongestResponseCombiner()
		result, err := combiner.Combine(ctx, longerResponses)

		if err != nil {
			t.Fatalf("Combine failed: %v", err)
		}

		if result.Provider != "p2" {
			t.Errorf("Expected longest response from p2, got %s", result.Provider)
		}
	})

	t.Run("QualityScoreCombiner", func(t *testing.T) {
		qualityResponses := []provider.Response{
			{Content: "short", Provider: "p1"},
			{Content: "This is a well-structured response.\n\nWith multiple paragraphs.\n\n- And a list item", Provider: "p2"},
			{Content: "Medium length response without structure", Provider: "p3"},
		}

		combiner := NewQualityScoreCombiner(10)
		result, err := combiner.Combine(ctx, qualityResponses)

		if err != nil {
			t.Fatalf("Combine failed: %v", err)
		}

		if result.Provider != "p2" {
			t.Errorf("Expected highest quality response from p2, got %s", result.Provider)
		}

		if score, ok := result.Metadata["quality_score"].(float64); !ok || score < 50 {
			t.Errorf("Expected quality score > 50, got %v", result.Metadata["quality_score"])
		}
	})
}

func TestNestedPipeline(t *testing.T) {
	// Create mock pipelines
	stage1 := NewFunctionAdapter("stage1", func(ctx context.Context, input string) (string, error) {
		return input + " -> stage1", nil
	})

	stage2 := NewFunctionAdapter("stage2", func(ctx context.Context, input string) (string, error) {
		return input + " -> stage2", nil
	})

	stage3 := NewFunctionAdapter("stage3", func(ctx context.Context, input string) (string, error) {
		return input + " -> stage3", nil
	})

	nested := NewNestedPipeline([]Pipeline{stage1, stage2, stage3})

	result, err := nested.Execute(context.Background(), "input")
	if err != nil {
		t.Fatalf("Failed to execute nested pipeline: %v", err)
	}

	expected := "input -> stage1 -> stage2 -> stage3"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestTransformAdapter(t *testing.T) {
	// Create a base pipeline
	base := NewFunctionAdapter("base", func(ctx context.Context, input string) (string, error) {
		return "processed: " + input, nil
	})

	// Create transforms
	inputTransform := func(ctx context.Context, input string) (string, error) {
		return strings.ToUpper(input), nil
	}

	outputTransform := func(ctx context.Context, output string) (string, error) {
		return "[" + output + "]", nil
	}

	// Create transform adapter
	transformed := NewTransformAdapter(base, inputTransform, outputTransform)

	result, err := transformed.Execute(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Failed to execute transform adapter: %v", err)
	}

	// Input "hello" -> "HELLO" -> "processed: HELLO" -> "[processed: HELLO]"
	expected := "[processed: HELLO]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
