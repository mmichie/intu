package aikit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PipelineConfig represents the configuration for a pipeline
type PipelineConfig struct {
	Type       string                   `json:"type"`
	Providers  []string                 `json:"providers,omitempty"`
	Jurors     []string                 `json:"jurors,omitempty"`
	Voting     string                   `json:"voting,omitempty"`
	Rounds     int                      `json:"rounds,omitempty"`
	Separator  string                   `json:"separator,omitempty"`
	Judge      string                   `json:"judge,omitempty"`
	Combiner   string                   `json:"combiner,omitempty"`
	Stages     []map[string]interface{} `json:"stages,omitempty"`
	Parameters map[string]interface{}   `json:"parameters,omitempty"`
}

// PipelineConfigs is a map of pipeline configurations
type PipelineConfigs map[string]PipelineConfig

// GetPipelineConfigDir returns the path to the pipeline configuration directory
func GetPipelineConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".intu")
	return configDir, nil
}

// GetPipelineConfigPath returns the path to the pipeline configuration file
func GetPipelineConfigPath() (string, error) {
	configDir, err := GetPipelineConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "pipelines.json"), nil
}

// LoadPipelineConfigs loads pipeline configurations from the pipeline configuration file
func LoadPipelineConfigs(ctx context.Context) (PipelineConfigs, error) {
	configPath, err := GetPipelineConfigPath()
	if err != nil {
		return nil, err
	}

	// If the file doesn't exist, return an empty configuration
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return PipelineConfigs{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline configuration file: %w", err)
	}

	var configs PipelineConfigs
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline configuration file: %w", err)
	}

	return configs, nil
}

// SavePipelineConfigs saves pipeline configurations to the pipeline configuration file
func SavePipelineConfigs(ctx context.Context, configs PipelineConfigs) error {
	configDir, err := GetPipelineConfigDir()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create configuration directory: %w", err)
	}

	configPath, err := GetPipelineConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline configurations: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write pipeline configuration file: %w", err)
	}

	return nil
}

// AddOrUpdatePipelineConfig adds or updates a pipeline configuration
func AddOrUpdatePipelineConfig(ctx context.Context, name string, config PipelineConfig) error {
	configs, err := LoadPipelineConfigs(ctx)
	if err != nil {
		return err
	}

	configs[name] = config

	return SavePipelineConfigs(ctx, configs)
}

// GetPipelineConfig retrieves a pipeline configuration by name
func GetPipelineConfig(ctx context.Context, name string) (PipelineConfig, error) {
	configs, err := LoadPipelineConfigs(ctx)
	if err != nil {
		return PipelineConfig{}, err
	}

	config, ok := configs[name]
	if !ok {
		return PipelineConfig{}, fmt.Errorf("pipeline configuration not found: %s", name)
	}

	return config, nil
}

// DeletePipelineConfig deletes a pipeline configuration
func DeletePipelineConfig(ctx context.Context, name string) error {
	configs, err := LoadPipelineConfigs(ctx)
	if err != nil {
		return err
	}

	if _, ok := configs[name]; !ok {
		return fmt.Errorf("pipeline configuration not found: %s", name)
	}

	delete(configs, name)

	return SavePipelineConfigs(ctx, configs)
}

// ListPipelineConfigs lists all available pipeline configurations
func ListPipelineConfigs(ctx context.Context) ([]string, error) {
	configs, err := LoadPipelineConfigs(ctx)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(configs))
	for name := range configs {
		names = append(names, name)
	}

	return names, nil
}
