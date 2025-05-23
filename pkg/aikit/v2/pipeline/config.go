package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
)

// PipelineType represents the type of pipeline
type PipelineType string

const (
	PipelineTypeSimple    PipelineType = "simple"
	PipelineTypeSerial    PipelineType = "serial"
	PipelineTypeParallel  PipelineType = "parallel"
	PipelineTypeNested    PipelineType = "nested"
	PipelineTypeTransform PipelineType = "transform"
	PipelineTypeHA        PipelineType = "high_availability"
	PipelineTypeConsensus PipelineType = "consensus"
)

// CombinerType represents the type of result combiner for parallel pipelines
type CombinerType string

const (
	CombinerTypeConcat       CombinerType = "concat"
	CombinerTypeMajorityVote CombinerType = "majority_vote"
	CombinerTypeLongest      CombinerType = "longest"
	CombinerTypeQualityScore CombinerType = "quality_score"
	CombinerTypeBestPicker   CombinerType = "best_picker"
	CombinerTypeConsensus    CombinerType = "consensus"
)

// TransformConfig represents transformation functions for pipelines
type TransformConfig struct {
	Type   string          `json:"type"`   // "function" or "pipeline"
	Name   string          `json:"name"`   // Name of the transform
	Config json.RawMessage `json:"config"` // Transform-specific config
}

// PipelineConfig represents a saved pipeline configuration
type PipelineConfig struct {
	// Basic info
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        PipelineType `json:"type"`
	Version     string       `json:"version"`

	// Provider configuration
	Providers []string `json:"providers,omitempty"` // For serial/parallel pipelines
	Provider  string   `json:"provider,omitempty"`  // For simple pipeline

	// Parallel pipeline configuration
	Combiner       CombinerType `json:"combiner,omitempty"`
	CombinerConfig struct {
		Separator      string `json:"separator,omitempty"`       // For concat
		MaxTokens      int    `json:"max_tokens,omitempty"`      // For quality score
		PickerProvider string `json:"picker_provider,omitempty"` // For best picker
		JudgeProvider  string `json:"judge_provider,omitempty"`  // For consensus
	} `json:"combiner_config,omitempty"`

	// Nested pipeline configuration
	Stages []PipelineStageConfig `json:"stages,omitempty"`

	// Transform configuration
	BaseConfig      *PipelineConfig  `json:"base_config,omitempty"`
	InputTransform  *TransformConfig `json:"input_transform,omitempty"`
	OutputTransform *TransformConfig `json:"output_transform,omitempty"`

	// Advanced options
	Options map[string]interface{} `json:"options,omitempty"`

	// Provider-specific configuration
	ProviderConfigs map[string]config.Config `json:"provider_configs,omitempty"`
}

// PipelineStageConfig represents a stage in a nested pipeline
type PipelineStageConfig struct {
	Name   string          `json:"name"`
	Type   PipelineType    `json:"type"`
	Config json.RawMessage `json:"config"` // Embedded PipelineConfig
}

// Validate validates the pipeline configuration
func (pc *PipelineConfig) Validate() error {
	var validationErrors []string

	// Basic validation
	if pc.Name == "" {
		validationErrors = append(validationErrors, "name is required")
	}
	if pc.Type == "" {
		validationErrors = append(validationErrors, "type is required")
	}

	// Type-specific validation
	switch pc.Type {
	case PipelineTypeSimple:
		if pc.Provider == "" {
			validationErrors = append(validationErrors, "provider is required for simple pipeline")
		}

	case PipelineTypeSerial, PipelineTypeParallel:
		if len(pc.Providers) == 0 {
			validationErrors = append(validationErrors, "at least one provider is required")
		}

	case PipelineTypeNested:
		if len(pc.Stages) == 0 {
			validationErrors = append(validationErrors, "at least one stage is required")
		}

	case PipelineTypeTransform:
		if pc.BaseConfig == nil {
			validationErrors = append(validationErrors, "base configuration is required")
		}

	case PipelineTypeHA:
		if len(pc.Providers) == 0 {
			validationErrors = append(validationErrors, "at least one provider is required for high availability")
		}

	case PipelineTypeConsensus:
		if len(pc.Providers) < 2 {
			validationErrors = append(validationErrors, "at least two providers are required for consensus")
		}
		if pc.CombinerConfig.JudgeProvider == "" {
			validationErrors = append(validationErrors, "judge provider is required for consensus")
		}
	}

	// Validate combiner for parallel pipelines
	if pc.Type == PipelineTypeParallel && pc.Combiner == "" {
		validationErrors = append(validationErrors, "combiner type is required for parallel pipelines")
	}

	// Validate combiner config
	if pc.Combiner == CombinerTypeBestPicker && pc.CombinerConfig.PickerProvider == "" {
		validationErrors = append(validationErrors, "picker provider is required for best picker combiner")
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}

// ToJSON converts the configuration to JSON
func (pc *PipelineConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(pc, "", "  ")
}

// FromJSON loads configuration from JSON
func (pc *PipelineConfig) FromJSON(data []byte) error {
	if err := json.Unmarshal(data, pc); err != nil {
		return fmt.Errorf("failed to parse pipeline config: %w", err)
	}
	return pc.Validate()
}

// Clone creates a deep copy of the configuration
func (pc *PipelineConfig) Clone() *PipelineConfig {
	data, _ := json.Marshal(pc)
	var clone PipelineConfig
	json.Unmarshal(data, &clone)
	return &clone
}

// GetProviders returns all providers used by this configuration
func (pc *PipelineConfig) GetProviders() []string {
	providers := make(map[string]bool)

	// Add single provider
	if pc.Provider != "" {
		providers[pc.Provider] = true
	}

	// Add multiple providers
	for _, p := range pc.Providers {
		providers[p] = true
	}

	// Add combiner providers
	if pc.CombinerConfig.PickerProvider != "" {
		providers[pc.CombinerConfig.PickerProvider] = true
	}
	if pc.CombinerConfig.JudgeProvider != "" {
		providers[pc.CombinerConfig.JudgeProvider] = true
	}

	// Add nested providers
	for _, stage := range pc.Stages {
		var stageConfig PipelineConfig
		if err := json.Unmarshal(stage.Config, &stageConfig); err == nil {
			for _, p := range stageConfig.GetProviders() {
				providers[p] = true
			}
		}
	}

	// Add base config providers
	if pc.BaseConfig != nil {
		for _, p := range pc.BaseConfig.GetProviders() {
			providers[p] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(providers))
	for p := range providers {
		result = append(result, p)
	}
	return result
}

// BuilderFromConfig creates a pipeline builder from configuration
func BuilderFromConfig(pc *PipelineConfig) (*Builder, error) {
	if err := pc.Validate(); err != nil {
		return nil, err
	}

	builder := NewBuilder()

	// Provider configs are stored in the PipelineConfig
	// They will be used by the factory when creating providers

	// Apply options
	var opts []Option
	if pc.Options != nil {
		// Convert options map to pipeline options
		if retries, ok := pc.Options["retries"].(float64); ok {
			opts = append(opts, WithRetries(int(retries)))
		}
		if cache, ok := pc.Options["cache"].(float64); ok {
			opts = append(opts, WithCache(int64(cache)))
		}
		// Timeout option not yet implemented
		// if timeout, ok := pc.Options["timeout"].(float64); ok {
		//     opts = append(opts, WithTimeout(int(timeout)))
		// }
	}
	builder.WithOptions(opts...)

	return builder, nil
}

// ParsePipelineType parses a string into a PipelineType
func ParsePipelineType(s string) (PipelineType, error) {
	normalized := strings.ToLower(strings.ReplaceAll(s, "-", "_"))
	switch normalized {
	case "simple":
		return PipelineTypeSimple, nil
	case "serial":
		return PipelineTypeSerial, nil
	case "parallel":
		return PipelineTypeParallel, nil
	case "nested":
		return PipelineTypeNested, nil
	case "transform":
		return PipelineTypeTransform, nil
	case "high_availability", "ha":
		return PipelineTypeHA, nil
	case "consensus":
		return PipelineTypeConsensus, nil
	default:
		return "", fmt.Errorf("unknown pipeline type: %s", s)
	}
}

// ParseCombinerType parses a string into a CombinerType
func ParseCombinerType(s string) (CombinerType, error) {
	normalized := strings.ToLower(strings.ReplaceAll(s, "-", "_"))
	switch normalized {
	case "concat":
		return CombinerTypeConcat, nil
	case "majority_vote":
		return CombinerTypeMajorityVote, nil
	case "longest":
		return CombinerTypeLongest, nil
	case "quality_score":
		return CombinerTypeQualityScore, nil
	case "best_picker":
		return CombinerTypeBestPicker, nil
	case "consensus":
		return CombinerTypeConsensus, nil
	default:
		return "", fmt.Errorf("unknown combiner type: %s", s)
	}
}
