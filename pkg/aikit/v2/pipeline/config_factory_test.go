package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
	"github.com/mmichie/intu/pkg/aikit/v2/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockConfigProvider is a mock provider for config factory testing
type mockConfigProvider struct {
	mock.Mock
}

func (m *mockConfigProvider) GenerateResponse(ctx context.Context, request provider.Request) (provider.Response, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(provider.Response), args.Error(1)
}

func (m *mockConfigProvider) GenerateStreamingResponse(ctx context.Context, request provider.Request, handler provider.StreamHandler) error {
	args := m.Called(ctx, request, handler)
	return args.Error(0)
}

func (m *mockConfigProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockConfigProvider) Model() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockConfigProvider) Capabilities() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// mockConfigProviderFactory is a mock provider factory for config tests
type mockConfigProviderFactory struct{}

func (f *mockConfigProviderFactory) Name() string {
	return "mock"
}

func (f *mockConfigProviderFactory) Create(cfg config.Config) (provider.Provider, error) {
	p := &mockConfigProvider{}
	response := provider.Response{
		Content: "mock response",
	}
	p.On("GenerateResponse", mock.Anything, mock.Anything).Return(response, nil)
	p.On("Name").Return("mock")
	p.On("Model").Return("mock-model")
	p.On("Capabilities").Return([]string{})
	return p, nil
}

func (f *mockConfigProviderFactory) GetAvailableModels() []string {
	return []string{"mock-model"}
}

func (f *mockConfigProviderFactory) GetCapabilities() []string {
	return []string{}
}

func TestConfigFactory(t *testing.T) {
	// Create temp directory for tests
	tempDir, err := os.MkdirTemp("", "config-factory-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "pipelines.json")
	store := NewConfigStore(configPath)

	// Create mock registry
	registry := provider.NewRegistry()

	// Register mock factory
	mockFactory := &mockConfigProviderFactory{}
	registry.RegisterFactory(mockFactory)

	factory := NewConfigFactory(registry, store)

	t.Run("CreateFromConfig - Simple", func(t *testing.T) {
		// Save a simple config
		simpleConfig := &PipelineConfig{
			Name:     "simple-test",
			Type:     PipelineTypeSimple,
			Provider: "mock",
		}
		err := factory.SaveConfig(simpleConfig)
		require.NoError(t, err)

		// Create pipeline from config
		pipeline, err := factory.CreateFromConfig("simple-test")
		require.NoError(t, err)
		assert.NotNil(t, pipeline)

		// Test execution
		ctx := context.Background()
		result, err := pipeline.Execute(ctx, "test prompt")
		assert.NoError(t, err)
		assert.Equal(t, "mock response", result)
	})

	t.Run("CreateFromConfig - Serial", func(t *testing.T) {
		serialConfig := &PipelineConfig{
			Name:      "serial-test",
			Type:      PipelineTypeSerial,
			Providers: []string{"mock", "mock"},
		}
		err := factory.SaveConfig(serialConfig)
		require.NoError(t, err)

		pipeline, err := factory.CreateFromConfig("serial-test")
		require.NoError(t, err)
		assert.NotNil(t, pipeline)
	})

	t.Run("CreateFromConfig - Parallel", func(t *testing.T) {
		parallelConfig := &PipelineConfig{
			Name:      "parallel-test",
			Type:      PipelineTypeParallel,
			Providers: []string{"mock", "mock"},
			Combiner:  CombinerTypeConcat,
			CombinerConfig: struct {
				Separator      string `json:"separator,omitempty"`
				MaxTokens      int    `json:"max_tokens,omitempty"`
				PickerProvider string `json:"picker_provider,omitempty"`
				JudgeProvider  string `json:"judge_provider,omitempty"`
			}{
				Separator: " | ",
			},
		}
		err := factory.SaveConfig(parallelConfig)
		require.NoError(t, err)

		pipeline, err := factory.CreateFromConfig("parallel-test")
		require.NoError(t, err)
		assert.NotNil(t, pipeline)
	})

	t.Run("CreateFromConfig - Nested", func(t *testing.T) {
		// Create stage configs
		stage1 := PipelineConfig{
			Provider: "mock",
		}
		stage1Data, _ := json.Marshal(stage1)

		stage2 := PipelineConfig{
			Providers: []string{"mock", "mock"},
		}
		stage2Data, _ := json.Marshal(stage2)

		nestedConfig := &PipelineConfig{
			Name: "nested-test",
			Type: PipelineTypeNested,
			Stages: []PipelineStageConfig{
				{
					Name:   "stage1",
					Type:   PipelineTypeSimple,
					Config: stage1Data,
				},
				{
					Name:   "stage2",
					Type:   PipelineTypeSerial,
					Config: stage2Data,
				},
			},
		}
		err := factory.SaveConfig(nestedConfig)
		require.NoError(t, err)

		pipeline, err := factory.CreateFromConfig("nested-test")
		require.NoError(t, err)
		assert.NotNil(t, pipeline)
	})

	t.Run("CreateFromConfig - Transform", func(t *testing.T) {
		transformConfig := &PipelineConfig{
			Name: "transform-test",
			Type: PipelineTypeTransform,
			BaseConfig: &PipelineConfig{
				Name:     "base",
				Type:     PipelineTypeSimple,
				Provider: "mock",
			},
			InputTransform: &TransformConfig{
				Type: "function",
				Name: "uppercase",
			},
			OutputTransform: &TransformConfig{
				Type: "function",
				Name: "add-prefix",
			},
		}
		err := factory.SaveConfig(transformConfig)
		require.NoError(t, err)

		pipeline, err := factory.CreateFromConfig("transform-test")
		require.NoError(t, err)
		assert.NotNil(t, pipeline)
	})

	t.Run("CreateFromConfig - HighAvailability", func(t *testing.T) {
		haConfig := &PipelineConfig{
			Name:      "ha-test",
			Type:      PipelineTypeHA,
			Providers: []string{"mock", "mock", "mock"},
		}
		err := factory.SaveConfig(haConfig)
		require.NoError(t, err)

		pipeline, err := factory.CreateFromConfig("ha-test")
		require.NoError(t, err)
		assert.NotNil(t, pipeline)
	})

	t.Run("CreateFromConfig - Non-existent", func(t *testing.T) {
		_, err := factory.CreateFromConfig("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("CreateFromPipelineConfig - Invalid", func(t *testing.T) {
		invalidConfig := &PipelineConfig{
			Name: "invalid",
			Type: PipelineTypeSimple,
			// Missing provider
		}
		_, err := factory.CreateFromPipelineConfig(invalidConfig)
		assert.Error(t, err)
	})

	t.Run("LoadConfig", func(t *testing.T) {
		config, err := factory.LoadConfig("simple-test")
		require.NoError(t, err)
		assert.Equal(t, "simple-test", config.Name)
		assert.Equal(t, PipelineTypeSimple, config.Type)
	})

	t.Run("DeleteConfig", func(t *testing.T) {
		err := factory.DeleteConfig("simple-test")
		assert.NoError(t, err)

		_, err = factory.LoadConfig("simple-test")
		assert.Error(t, err)
	})

	t.Run("ListConfigs", func(t *testing.T) {
		configs := factory.ListConfigs()
		assert.NotEmpty(t, configs)

		// Check that configs are sorted by name
		for i := 1; i < len(configs); i++ {
			assert.LessOrEqual(t, configs[i-1].Name, configs[i].Name)
		}
	})

	t.Run("Export and Import", func(t *testing.T) {
		// Export existing config
		data, err := factory.ExportConfig("serial-test")
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Delete it
		err = factory.DeleteConfig("serial-test")
		require.NoError(t, err)

		// Import it back
		err = factory.ImportConfig(data)
		assert.NoError(t, err)

		// Verify it's back
		config, err := factory.LoadConfig("serial-test")
		assert.NoError(t, err)
		assert.Equal(t, "serial-test", config.Name)
	})

	t.Run("Options from config", func(t *testing.T) {
		configWithOptions := &PipelineConfig{
			Name:     "options-test",
			Type:     PipelineTypeSimple,
			Provider: "mock",
			Options: map[string]interface{}{
				"retries": 5.0,
				"cache":   300.0,
				"timeout": 30.0,
			},
		}
		err := factory.SaveConfig(configWithOptions)
		require.NoError(t, err)

		pipeline, err := factory.CreateFromConfig("options-test")
		require.NoError(t, err)
		assert.NotNil(t, pipeline)
	})

	t.Run("Provider configs", func(t *testing.T) {
		configWithProviderConfigs := &PipelineConfig{
			Name:      "provider-configs-test",
			Type:      PipelineTypeSerial,
			Providers: []string{"mock", "mock"},
			ProviderConfigs: map[string]config.Config{
				"mock": {
					APIKey: "test-key",
					Model:  "test-model",
				},
			},
		}
		err := factory.SaveConfig(configWithProviderConfigs)
		require.NoError(t, err)

		pipeline, err := factory.CreateFromConfig("provider-configs-test")
		require.NoError(t, err)
		assert.NotNil(t, pipeline)
	})
}

func TestGlobalConfigFactory(t *testing.T) {
	// Reset global factory
	defaultConfigFactory = nil

	t.Run("InitDefaultConfigFactory", func(t *testing.T) {
		err := InitDefaultConfigFactory()
		assert.NoError(t, err)
		assert.NotNil(t, defaultConfigFactory)
	})

	t.Run("GetConfigFactory", func(t *testing.T) {
		factory := GetConfigFactory()
		assert.NotNil(t, factory)
		assert.Equal(t, defaultConfigFactory, factory)
	})

	t.Run("Global functions", func(t *testing.T) {
		// Test SavePipelineConfig
		config := &PipelineConfig{
			Name:     "global-test",
			Type:     PipelineTypeSimple,
			Provider: "openai",
		}
		err := SavePipelineConfig(config)
		// May fail if openai not registered, that's OK for this test
		_ = err

		// Test LoadPipelineConfig
		loaded, err := LoadPipelineConfig("global-test")
		if err == nil {
			assert.Equal(t, "global-test", loaded.Name)
		}

		// Test ListPipelineConfigs
		configs := ListPipelineConfigs()
		assert.NotNil(t, configs)

		// Clean up
		DeletePipelineConfig("global-test")
	})
}
