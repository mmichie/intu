package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// ConfigFactory extends Factory with configuration support
type ConfigFactory struct {
	*Factory
	store *ConfigStore
}

// NewConfigFactory creates a new configuration-aware factory
func NewConfigFactory(registry *provider.Registry, store *ConfigStore) *ConfigFactory {
	if store == nil {
		store = DefaultConfigStore()
	}

	return &ConfigFactory{
		Factory: NewFactoryWithRegistry(registry),
		store:   store,
	}
}

// CreateFromConfig creates a pipeline from a configuration name
func (cf *ConfigFactory) CreateFromConfig(name string) (Pipeline, error) {
	// Load configuration
	pipelineConfig, err := cf.store.Get(name)
	if err != nil {
		return nil, err
	}

	return cf.CreateFromPipelineConfig(pipelineConfig)
}

// CreateFromPipelineConfig creates a pipeline from a configuration object
func (cf *ConfigFactory) CreateFromPipelineConfig(pc *PipelineConfig) (Pipeline, error) {
	if err := pc.Validate(); err != nil {
		return nil, err
	}

	// Create builder from config
	builder, err := BuilderFromConfig(pc)
	if err != nil {
		return nil, err
	}
	builder.factory = cf.Factory

	// Build based on type
	switch pc.Type {
	case PipelineTypeSimple:
		return cf.createSimpleFromConfig(pc, builder)

	case PipelineTypeSerial:
		return cf.createSerialFromConfig(pc, builder)

	case PipelineTypeParallel:
		return cf.createParallelFromConfig(pc, builder)

	case PipelineTypeNested:
		return cf.createNestedFromConfig(pc, builder)

	case PipelineTypeTransform:
		return cf.createTransformFromConfig(pc, builder)

	case PipelineTypeHA:
		return cf.createHAFromConfig(pc, builder)

	case PipelineTypeConsensus:
		return cf.createConsensusFromConfig(pc, builder)

	default:
		return nil, fmt.Errorf("unsupported pipeline type: %s", pc.Type)
	}
}

// createSimpleFromConfig creates a simple pipeline from config
func (cf *ConfigFactory) createSimpleFromConfig(pc *PipelineConfig, builder *Builder) (Pipeline, error) {
	// Apply provider config if exists
	if providerConfig, exists := pc.ProviderConfigs[pc.Provider]; exists {
		cf.config = providerConfig
	}

	return cf.CreateSimple(pc.Provider, builder.options...)
}

// createSerialFromConfig creates a serial pipeline from config
func (cf *ConfigFactory) createSerialFromConfig(pc *PipelineConfig, builder *Builder) (Pipeline, error) {
	// Use the factory's CreateSerial method which handles provider creation
	return cf.CreateSerial(pc.Providers, builder.options...)
}

// createParallelFromConfig creates a parallel pipeline from config
func (cf *ConfigFactory) createParallelFromConfig(pc *PipelineConfig, builder *Builder) (Pipeline, error) {
	// Create combiner based on type
	var combiner ResultCombiner

	switch pc.Combiner {
	case CombinerTypeConcat:
		separator := pc.CombinerConfig.Separator
		if separator == "" {
			separator = "\n"
		}
		combiner = NewConcatCombiner(separator)

	case CombinerTypeMajorityVote:
		combiner = NewMajorityVoteCombiner()

	case CombinerTypeLongest:
		combiner = NewLongestResponseCombiner()

	case CombinerTypeQualityScore:
		maxTokens := pc.CombinerConfig.MaxTokens
		if maxTokens == 0 {
			maxTokens = 100
		}
		combiner = NewQualityScoreCombiner(maxTokens)

	case CombinerTypeBestPicker:
		if pc.CombinerConfig.PickerProvider == "" {
			return nil, fmt.Errorf("picker provider is required for best picker combiner")
		}
		return cf.CreateParallelWithBestPicker(pc.Providers, pc.CombinerConfig.PickerProvider, builder.options...)

	case CombinerTypeConsensus:
		if pc.CombinerConfig.JudgeProvider == "" {
			return nil, fmt.Errorf("judge provider is required for consensus combiner")
		}
		consensusCombiner, err := cf.createConsensusCombiner(pc.CombinerConfig.JudgeProvider)
		if err != nil {
			return nil, err
		}
		combiner = consensusCombiner

	default:
		return nil, fmt.Errorf("unsupported combiner type: %s", pc.Combiner)
	}

	return cf.CreateParallel(pc.Providers, combiner, builder.options...)
}

// createNestedFromConfig creates a nested pipeline from config
func (cf *ConfigFactory) createNestedFromConfig(pc *PipelineConfig, builder *Builder) (Pipeline, error) {
	stages := make([]Pipeline, 0, len(pc.Stages))

	for _, stageConfig := range pc.Stages {
		// Parse stage configuration
		var stagePipelineConfig PipelineConfig
		if err := json.Unmarshal(stageConfig.Config, &stagePipelineConfig); err != nil {
			return nil, fmt.Errorf("failed to parse stage '%s' config: %w", stageConfig.Name, err)
		}

		// Set name from stage
		stagePipelineConfig.Name = stageConfig.Name
		stagePipelineConfig.Type = stageConfig.Type

		// Create stage pipeline
		stage, err := cf.CreateFromPipelineConfig(&stagePipelineConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create stage '%s': %w", stageConfig.Name, err)
		}

		stages = append(stages, stage)
	}

	nested := NewNestedPipeline(stages, builder.options...)
	return nested, nil
}

// createTransformFromConfig creates a transform pipeline from config
func (cf *ConfigFactory) createTransformFromConfig(pc *PipelineConfig, builder *Builder) (Pipeline, error) {
	// Create base pipeline
	basePipeline, err := cf.CreateFromPipelineConfig(pc.BaseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create base pipeline: %w", err)
	}

	// Create transforms
	var inputTransform, outputTransform ProcessFunc

	if pc.InputTransform != nil {
		inputTransform = func(ctx context.Context, input string) (string, error) {
			// For now, just return as-is. In future, could support custom transforms
			return input, nil
		}
	}

	if pc.OutputTransform != nil {
		outputTransform = func(ctx context.Context, output string) (string, error) {
			// For now, just return as-is. In future, could support custom transforms
			return output, nil
		}
	}

	return NewTransformAdapter(basePipeline, inputTransform, outputTransform, builder.options...), nil
}

// createHAFromConfig creates a high availability pipeline from config
func (cf *ConfigFactory) createHAFromConfig(pc *PipelineConfig, builder *Builder) (Pipeline, error) {
	// Apply provider configs
	var cfg config.Config = cf.config
	for _, provider := range pc.Providers {
		if providerConfig, exists := pc.ProviderConfigs[provider]; exists {
			// Use first provider's config as base
			if cfg.APIKey == "" {
				cfg = providerConfig
			}
		}
	}

	// Create HA pipeline using factory's CreateFallback
	if len(pc.Providers) == 0 {
		return nil, fmt.Errorf("at least one provider required for high availability")
	}

	primary := pc.Providers[0]
	fallbacks := pc.Providers[1:]

	return cf.CreateFallback(primary, fallbacks, append(builder.options, WithRetries(3), WithCache(300))...)
}

// createConsensusFromConfig creates a consensus pipeline from config
func (cf *ConfigFactory) createConsensusFromConfig(pc *PipelineConfig, builder *Builder) (Pipeline, error) {
	// Apply provider configs
	cfg := cf.config
	for _, provider := range pc.Providers {
		if providerConfig, exists := pc.ProviderConfigs[provider]; exists {
			// Use first provider's config as base
			if cfg.APIKey == "" {
				cfg = providerConfig
			}
		}
	}

	// Create consensus combiner
	consensusCombiner, err := cf.createConsensusCombiner(pc.CombinerConfig.JudgeProvider)
	if err != nil {
		return nil, err
	}

	// Create parallel pipeline with consensus combiner
	return cf.CreateParallel(pc.Providers, consensusCombiner, builder.options...)
}

// createConsensusCombiner creates a consensus combiner
func (cf *ConfigFactory) createConsensusCombiner(judgeProvider string) (ResultCombiner, error) {
	judge, err := cf.createProvider(judgeProvider)
	if err != nil {
		return nil, err
	}

	return NewConsensusCombiner(judge), nil
}

// SaveConfig saves a pipeline configuration
func (cf *ConfigFactory) SaveConfig(config *PipelineConfig) error {
	return cf.store.Add(config)
}

// LoadConfig loads a pipeline configuration
func (cf *ConfigFactory) LoadConfig(name string) (*PipelineConfig, error) {
	return cf.store.Get(name)
}

// DeleteConfig deletes a pipeline configuration
func (cf *ConfigFactory) DeleteConfig(name string) error {
	return cf.store.Delete(name)
}

// ListConfigs returns all saved configurations
func (cf *ConfigFactory) ListConfigs() []*PipelineConfig {
	return cf.store.ListConfigs()
}

// ExportConfig exports a configuration to JSON
func (cf *ConfigFactory) ExportConfig(name string) ([]byte, error) {
	return cf.store.Export(name)
}

// ImportConfig imports a configuration from JSON
func (cf *ConfigFactory) ImportConfig(data []byte) error {
	return cf.store.Import(data)
}

// Global config factory instance
var defaultConfigFactory *ConfigFactory

// InitDefaultConfigFactory initializes the default config factory
func InitDefaultConfigFactory() error {
	store := DefaultConfigStore()
	if err := store.Load(); err != nil {
		return err
	}

	defaultConfigFactory = NewConfigFactory(nil, store)
	return nil
}

// GetConfigFactory returns the default config factory
func GetConfigFactory() *ConfigFactory {
	if defaultConfigFactory == nil {
		InitDefaultConfigFactory()
	}
	return defaultConfigFactory
}

// CreatePipelineFromConfig creates a pipeline from a saved configuration
func CreatePipelineFromConfig(name string) (Pipeline, error) {
	return GetConfigFactory().CreateFromConfig(name)
}

// SavePipelineConfig saves a pipeline configuration
func SavePipelineConfig(config *PipelineConfig) error {
	return GetConfigFactory().SaveConfig(config)
}

// LoadPipelineConfig loads a pipeline configuration
func LoadPipelineConfig(name string) (*PipelineConfig, error) {
	return GetConfigFactory().LoadConfig(name)
}

// DeletePipelineConfig deletes a pipeline configuration
func DeletePipelineConfig(name string) error {
	return GetConfigFactory().DeleteConfig(name)
}

// ListPipelineConfigs returns all saved configurations
func ListPipelineConfigs() []*PipelineConfig {
	return GetConfigFactory().ListConfigs()
}
