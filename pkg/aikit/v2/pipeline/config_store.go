package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// ConfigStore manages pipeline configurations
type ConfigStore struct {
	mu       sync.RWMutex
	filePath string
	configs  map[string]*PipelineConfig
}

// NewConfigStore creates a new configuration store
func NewConfigStore(filePath string) *ConfigStore {
	if filePath == "" {
		home, _ := os.UserHomeDir()
		filePath = filepath.Join(home, ".intu", "pipelines.json")
	}

	return &ConfigStore{
		filePath: filePath,
		configs:  make(map[string]*PipelineConfig),
	}
}

// DefaultConfigStore returns the default configuration store
func DefaultConfigStore() *ConfigStore {
	return NewConfigStore("")
}

// Load loads configurations from disk
func (cs *ConfigStore) Load() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(cs.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read file
	file, err := os.Open(cs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, that's OK
			cs.configs = make(map[string]*PipelineConfig)
			return nil
		}
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Parse JSON
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if len(data) == 0 {
		cs.configs = make(map[string]*PipelineConfig)
		return nil
	}

	var configs map[string]*PipelineConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate all configs
	for name, config := range configs {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid config '%s': %w", name, err)
		}
		// Ensure name matches
		config.Name = name
	}

	cs.configs = configs
	return nil
}

// Save saves configurations to disk
func (cs *ConfigStore) Save() error {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.saveInternal()
}

// saveInternal saves without acquiring locks - must be called with lock held
func (cs *ConfigStore) saveInternal() error {

	// Ensure directory exists
	dir := filepath.Dir(cs.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(cs.configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configs: %w", err)
	}

	// Write atomically
	tmpFile := cs.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if err := os.Rename(tmpFile, cs.filePath); err != nil {
		os.Remove(tmpFile) // Clean up
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}

// Get retrieves a pipeline configuration by name
func (cs *ConfigStore) Get(name string) (*PipelineConfig, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	config, exists := cs.configs[name]
	if !exists {
		return nil, fmt.Errorf("pipeline config '%s' not found", name)
	}

	// Return a copy to prevent external modification
	return config.Clone(), nil
}

// Add adds or updates a pipeline configuration
func (cs *ConfigStore) Add(config *PipelineConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	cs.mu.Lock()
	cs.configs[config.Name] = config.Clone()
	err := cs.saveInternal()
	cs.mu.Unlock()

	return err
}

// Delete removes a pipeline configuration
func (cs *ConfigStore) Delete(name string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.configs[name]; !exists {
		return fmt.Errorf("pipeline config '%s' not found", name)
	}

	delete(cs.configs, name)
	return cs.saveInternal()
}

// List returns all configuration names
func (cs *ConfigStore) List() []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	names := make([]string, 0, len(cs.configs))
	for name := range cs.configs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListConfigs returns all configurations
func (cs *ConfigStore) ListConfigs() []*PipelineConfig {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	configs := make([]*PipelineConfig, 0, len(cs.configs))
	for _, config := range cs.configs {
		configs = append(configs, config.Clone())
	}

	// Sort by name
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Name < configs[j].Name
	})

	return configs
}

// Exists checks if a configuration exists
func (cs *ConfigStore) Exists(name string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	_, exists := cs.configs[name]
	return exists
}

// Export exports a configuration to JSON
func (cs *ConfigStore) Export(name string) ([]byte, error) {
	config, err := cs.Get(name)
	if err != nil {
		return nil, err
	}
	return config.ToJSON()
}

// Import imports a configuration from JSON
func (cs *ConfigStore) Import(data []byte) error {
	var config PipelineConfig
	if err := config.FromJSON(data); err != nil {
		return err
	}
	return cs.Add(&config)
}

// Clear removes all configurations
func (cs *ConfigStore) Clear() error {
	cs.mu.Lock()
	cs.configs = make(map[string]*PipelineConfig)
	err := cs.saveInternal()
	cs.mu.Unlock()

	return err
}

// Backup creates a backup of the configuration file
func (cs *ConfigStore) Backup() error {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Read current file
	data, err := os.ReadFile(cs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to backup
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Create backup with timestamp
	backupPath := fmt.Sprintf("%s.backup.%d", cs.filePath, os.Getpid())
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	return nil
}

// Restore restores from a backup file
func (cs *ConfigStore) Restore(backupPath string) error {
	// Read backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Parse to validate
	var configs map[string]*PipelineConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("invalid backup file: %w", err)
	}

	// Validate all configs
	for name, config := range configs {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid config '%s' in backup: %w", name, err)
		}
		config.Name = name
	}

	// Replace current configs
	cs.mu.Lock()
	cs.configs = configs
	cs.mu.Unlock()

	return cs.Save()
}

// FilterByType returns all configurations of a specific type
func (cs *ConfigStore) FilterByType(pipelineType PipelineType) []*PipelineConfig {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var configs []*PipelineConfig
	for _, config := range cs.configs {
		if config.Type == pipelineType {
			configs = append(configs, config.Clone())
		}
	}

	// Sort by name
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Name < configs[j].Name
	})

	return configs
}

// FilterByProvider returns all configurations using a specific provider
func (cs *ConfigStore) FilterByProvider(provider string) []*PipelineConfig {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var configs []*PipelineConfig
	for _, config := range cs.configs {
		providers := config.GetProviders()
		for _, p := range providers {
			if p == provider {
				configs = append(configs, config.Clone())
				break
			}
		}
	}

	// Sort by name
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Name < configs[j].Name
	})

	return configs
}
