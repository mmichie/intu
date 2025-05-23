package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  PipelineConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid simple pipeline",
			config: PipelineConfig{
				Name:     "test-simple",
				Type:     PipelineTypeSimple,
				Provider: "openai",
			},
			wantErr: false,
		},
		{
			name: "simple pipeline missing provider",
			config: PipelineConfig{
				Name: "test-simple",
				Type: PipelineTypeSimple,
			},
			wantErr: true,
			errMsg:  "provider",
		},
		{
			name: "valid serial pipeline",
			config: PipelineConfig{
				Name:      "test-serial",
				Type:      PipelineTypeSerial,
				Providers: []string{"openai", "claude"},
			},
			wantErr: false,
		},
		{
			name: "serial pipeline missing providers",
			config: PipelineConfig{
				Name: "test-serial",
				Type: PipelineTypeSerial,
			},
			wantErr: true,
			errMsg:  "at least one provider",
		},
		{
			name: "valid parallel pipeline",
			config: PipelineConfig{
				Name:      "test-parallel",
				Type:      PipelineTypeParallel,
				Providers: []string{"openai", "claude"},
				Combiner:  CombinerTypeConcat,
			},
			wantErr: false,
		},
		{
			name: "parallel pipeline missing combiner",
			config: PipelineConfig{
				Name:      "test-parallel",
				Type:      PipelineTypeParallel,
				Providers: []string{"openai", "claude"},
			},
			wantErr: true,
			errMsg:  "combiner",
		},
		{
			name: "best picker missing picker provider",
			config: PipelineConfig{
				Name:      "test-parallel",
				Type:      PipelineTypeParallel,
				Providers: []string{"openai", "claude"},
				Combiner:  CombinerTypeBestPicker,
			},
			wantErr: true,
			errMsg:  "picker provider",
		},
		{
			name: "valid best picker",
			config: PipelineConfig{
				Name:      "test-parallel",
				Type:      PipelineTypeParallel,
				Providers: []string{"openai", "claude"},
				Combiner:  CombinerTypeBestPicker,
				CombinerConfig: struct {
					Separator      string `json:"separator,omitempty"`
					MaxTokens      int    `json:"max_tokens,omitempty"`
					PickerProvider string `json:"picker_provider,omitempty"`
					JudgeProvider  string `json:"judge_provider,omitempty"`
				}{
					PickerProvider: "gemini",
				},
			},
			wantErr: false,
		},
		{
			name: "valid nested pipeline",
			config: PipelineConfig{
				Name: "test-nested",
				Type: PipelineTypeNested,
				Stages: []PipelineStageConfig{
					{
						Name:   "stage1",
						Type:   PipelineTypeSimple,
						Config: json.RawMessage(`{"provider": "openai"}`),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "nested pipeline missing stages",
			config: PipelineConfig{
				Name: "test-nested",
				Type: PipelineTypeNested,
			},
			wantErr: true,
			errMsg:  "at least one stage",
		},
		{
			name: "transform pipeline missing base config",
			config: PipelineConfig{
				Name: "test-transform",
				Type: PipelineTypeTransform,
			},
			wantErr: true,
			errMsg:  "base configuration",
		},
		{
			name: "valid transform pipeline",
			config: PipelineConfig{
				Name: "test-transform",
				Type: PipelineTypeTransform,
				BaseConfig: &PipelineConfig{
					Name:     "base",
					Type:     PipelineTypeSimple,
					Provider: "openai",
				},
			},
			wantErr: false,
		},
		{
			name: "high availability missing providers",
			config: PipelineConfig{
				Name: "test-ha",
				Type: PipelineTypeHA,
			},
			wantErr: true,
			errMsg:  "at least one provider",
		},
		{
			name: "valid high availability",
			config: PipelineConfig{
				Name:      "test-ha",
				Type:      PipelineTypeHA,
				Providers: []string{"openai", "claude", "gemini"},
			},
			wantErr: false,
		},
		{
			name: "consensus missing providers",
			config: PipelineConfig{
				Name:      "test-consensus",
				Type:      PipelineTypeConsensus,
				Providers: []string{"openai"},
			},
			wantErr: true,
			errMsg:  "two providers",
		},
		{
			name: "consensus missing judge",
			config: PipelineConfig{
				Name:      "test-consensus",
				Type:      PipelineTypeConsensus,
				Providers: []string{"openai", "claude"},
			},
			wantErr: true,
			errMsg:  "judge provider",
		},
		{
			name: "valid consensus",
			config: PipelineConfig{
				Name:      "test-consensus",
				Type:      PipelineTypeConsensus,
				Providers: []string{"openai", "claude"},
				CombinerConfig: struct {
					Separator      string `json:"separator,omitempty"`
					MaxTokens      int    `json:"max_tokens,omitempty"`
					PickerProvider string `json:"picker_provider,omitempty"`
					JudgeProvider  string `json:"judge_provider,omitempty"`
				}{
					JudgeProvider: "gemini",
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: PipelineConfig{
				Type:     PipelineTypeSimple,
				Provider: "openai",
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "missing type",
			config: PipelineConfig{
				Name:     "test",
				Provider: "openai",
			},
			wantErr: true,
			errMsg:  "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPipelineConfig_JSON(t *testing.T) {
	original := &PipelineConfig{
		Name:        "test-pipeline",
		Description: "A test pipeline",
		Type:        PipelineTypeParallel,
		Version:     "1.0",
		Providers:   []string{"openai", "claude"},
		Combiner:    CombinerTypeConcat,
		CombinerConfig: struct {
			Separator      string `json:"separator,omitempty"`
			MaxTokens      int    `json:"max_tokens,omitempty"`
			PickerProvider string `json:"picker_provider,omitempty"`
			JudgeProvider  string `json:"judge_provider,omitempty"`
		}{
			Separator: " | ",
		},
		Options: map[string]interface{}{
			"retries": 3.0,
			"cache":   600.0,
		},
		ProviderConfigs: map[string]config.Config{
			"openai": {
				APIKey: "test-key",
				Model:  "gpt-4",
			},
		},
	}

	// Test ToJSON
	data, err := original.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test FromJSON
	loaded := &PipelineConfig{}
	err = loaded.FromJSON(data)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Description, loaded.Description)
	assert.Equal(t, original.Type, loaded.Type)
	assert.Equal(t, original.Providers, loaded.Providers)
	assert.Equal(t, original.Combiner, loaded.Combiner)
	assert.Equal(t, original.CombinerConfig.Separator, loaded.CombinerConfig.Separator)
	assert.Equal(t, original.Options["retries"], loaded.Options["retries"])
}

func TestPipelineConfig_Clone(t *testing.T) {
	original := &PipelineConfig{
		Name:      "test",
		Type:      PipelineTypeSimple,
		Provider:  "openai",
		Providers: []string{"openai", "claude"},
		Options: map[string]interface{}{
			"timeout": 30.0,
		},
	}

	clone := original.Clone()

	// Test independence
	clone.Name = "modified"
	clone.Providers[0] = "gemini"
	clone.Options["timeout"] = 60.0

	assert.Equal(t, "test", original.Name)
	assert.Equal(t, "openai", original.Providers[0])
	assert.Equal(t, 30.0, original.Options["timeout"])
}

func TestPipelineConfig_GetProviders(t *testing.T) {
	tests := []struct {
		name     string
		config   PipelineConfig
		expected []string
	}{
		{
			name: "simple pipeline",
			config: PipelineConfig{
				Provider: "openai",
			},
			expected: []string{"openai"},
		},
		{
			name: "parallel pipeline",
			config: PipelineConfig{
				Providers: []string{"openai", "claude"},
			},
			expected: []string{"openai", "claude"},
		},
		{
			name: "best picker with picker provider",
			config: PipelineConfig{
				Providers: []string{"openai", "claude"},
				CombinerConfig: struct {
					Separator      string `json:"separator,omitempty"`
					MaxTokens      int    `json:"max_tokens,omitempty"`
					PickerProvider string `json:"picker_provider,omitempty"`
					JudgeProvider  string `json:"judge_provider,omitempty"`
				}{
					PickerProvider: "gemini",
				},
			},
			expected: []string{"openai", "claude", "gemini"},
		},
		{
			name: "consensus with judge",
			config: PipelineConfig{
				Providers: []string{"openai", "claude"},
				CombinerConfig: struct {
					Separator      string `json:"separator,omitempty"`
					MaxTokens      int    `json:"max_tokens,omitempty"`
					PickerProvider string `json:"picker_provider,omitempty"`
					JudgeProvider  string `json:"judge_provider,omitempty"`
				}{
					JudgeProvider: "gpt-4",
				},
			},
			expected: []string{"openai", "claude", "gpt-4"},
		},
		{
			name: "nested pipeline",
			config: PipelineConfig{
				Stages: []PipelineStageConfig{
					{
						Config: json.RawMessage(`{"provider": "openai"}`),
					},
					{
						Config: json.RawMessage(`{"providers": ["claude", "gemini"]}`),
					},
				},
			},
			expected: []string{"openai", "claude", "gemini"},
		},
		{
			name: "transform pipeline",
			config: PipelineConfig{
				BaseConfig: &PipelineConfig{
					Provider: "openai",
				},
			},
			expected: []string{"openai"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers := tt.config.GetProviders()
			// Sort for consistent comparison
			assert.ElementsMatch(t, tt.expected, providers)
		})
	}
}

func TestParsePipelineType(t *testing.T) {
	tests := []struct {
		input    string
		expected PipelineType
		wantErr  bool
	}{
		{"simple", PipelineTypeSimple, false},
		{"serial", PipelineTypeSerial, false},
		{"parallel", PipelineTypeParallel, false},
		{"nested", PipelineTypeNested, false},
		{"transform", PipelineTypeTransform, false},
		{"high_availability", PipelineTypeHA, false},
		{"high-availability", PipelineTypeHA, false},
		{"ha", PipelineTypeHA, false},
		{"consensus", PipelineTypeConsensus, false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParsePipelineType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseCombinerType(t *testing.T) {
	tests := []struct {
		input    string
		expected CombinerType
		wantErr  bool
	}{
		{"concat", CombinerTypeConcat, false},
		{"majority_vote", CombinerTypeMajorityVote, false},
		{"majority-vote", CombinerTypeMajorityVote, false},
		{"longest", CombinerTypeLongest, false},
		{"quality_score", CombinerTypeQualityScore, false},
		{"quality-score", CombinerTypeQualityScore, false},
		{"best_picker", CombinerTypeBestPicker, false},
		{"best-picker", CombinerTypeBestPicker, false},
		{"consensus", CombinerTypeConsensus, false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseCombinerType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBuilderFromConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		pc := &PipelineConfig{
			Name:     "test",
			Type:     PipelineTypeSimple,
			Provider: "openai",
			Options: map[string]interface{}{
				"retries": 3.0,
				"cache":   300.0,
				"timeout": 60.0,
			},
			ProviderConfigs: map[string]config.Config{
				"openai": {
					APIKey: "test-key",
					Model:  "gpt-4",
				},
			},
		}

		builder, err := BuilderFromConfig(pc)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Len(t, builder.options, 2) // retries, cache (timeout is commented out)
	})

	t.Run("invalid config", func(t *testing.T) {
		pc := &PipelineConfig{
			Name: "test",
			Type: PipelineTypeSimple,
			// Missing provider
		}

		builder, err := BuilderFromConfig(pc)
		assert.Error(t, err)
		assert.Nil(t, builder)
	})
}
