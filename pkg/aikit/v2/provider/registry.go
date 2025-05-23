package provider

import (
	"fmt"
	"sync"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
	aierrors "github.com/mmichie/intu/pkg/aikit/v2/errors"
)

// Registry manages the available AI provider factories
type Registry struct {
	mu             sync.RWMutex
	defaultFactory string
	factories      map[string]ProviderFactory
}

// globalRegistry is the default registry instance
var globalRegistry = NewRegistry()

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ProviderFactory),
	}
}

// RegisterFactory adds a provider factory to the registry
func (r *Registry) RegisterFactory(factory ProviderFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := factory.Name()
	if name == "" {
		return aierrors.New("registry", "register_factory",
			fmt.Errorf("provider factory name cannot be empty"))
	}

	if _, exists := r.factories[name]; exists {
		return aierrors.New("registry", "register_factory",
			fmt.Errorf("provider factory %q already registered", name))
	}

	r.factories[name] = factory

	// If this is the first factory, make it the default
	if r.defaultFactory == "" {
		r.defaultFactory = name
	}

	return nil
}

// SetDefaultFactory sets the default provider factory
func (r *Registry) SetDefaultFactory(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; !exists {
		return aierrors.New("registry", "set_default",
			fmt.Errorf("provider factory %q not registered", name))
	}

	r.defaultFactory = name
	return nil
}

// GetFactory returns a provider factory by name
func (r *Registry) GetFactory(name string) (ProviderFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, aierrors.New("registry", "get_factory",
			fmt.Errorf("provider factory %q not registered", name))
	}

	return factory, nil
}

// GetDefaultFactory returns the default provider factory
func (r *Registry) GetDefaultFactory() (ProviderFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.defaultFactory == "" {
		return nil, aierrors.New("registry", "get_default",
			fmt.Errorf("no default provider factory set"))
	}

	factory, exists := r.factories[r.defaultFactory]
	if !exists {
		return nil, aierrors.New("registry", "get_default",
			fmt.Errorf("default provider factory %q not found", r.defaultFactory))
	}

	return factory, nil
}

// CreateProvider creates a provider instance using the specified factory
func (r *Registry) CreateProvider(name string, cfg config.Config) (Provider, error) {
	factory, err := r.GetFactory(name)
	if err != nil {
		return nil, err
	}

	return factory.Create(cfg)
}

// CreateDefaultProvider creates a provider instance using the default factory
func (r *Registry) CreateDefaultProvider(cfg config.Config) (Provider, error) {
	factory, err := r.GetDefaultFactory()
	if err != nil {
		return nil, err
	}

	return factory.Create(cfg)
}

// ListProviders returns a list of registered provider names
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(r.factories))
	for name := range r.factories {
		result = append(result, name)
	}

	return result
}

// GetProviderInfo returns detailed information about a provider
func (r *Registry) GetProviderInfo(name string) (ProviderInfo, error) {
	factory, err := r.GetFactory(name)
	if err != nil {
		return ProviderInfo{}, err
	}

	return ProviderInfo{
		Name:         name,
		Models:       factory.GetAvailableModels(),
		Capabilities: factory.GetCapabilities(),
		IsDefault:    name == r.defaultFactory,
	}, nil
}

// GetAllProviderInfo returns information about all registered providers
func (r *Registry) GetAllProviderInfo() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ProviderInfo, 0, len(r.factories))
	for name, factory := range r.factories {
		result = append(result, ProviderInfo{
			Name:         name,
			Models:       factory.GetAvailableModels(),
			Capabilities: factory.GetCapabilities(),
			IsDefault:    name == r.defaultFactory,
		})
	}

	return result
}

// ProviderInfo contains information about a provider
type ProviderInfo struct {
	Name         string   `json:"name"`
	Models       []string `json:"models"`
	Capabilities []string `json:"capabilities"`
	IsDefault    bool     `json:"is_default"`
}

// Global registry functions for convenience

// Register adds a provider factory to the global registry
func Register(factory ProviderFactory) error {
	return globalRegistry.RegisterFactory(factory)
}

// SetDefault sets the default provider in the global registry
func SetDefault(name string) error {
	return globalRegistry.SetDefaultFactory(name)
}

// Get returns a provider factory from the global registry
func Get(name string) (ProviderFactory, error) {
	return globalRegistry.GetFactory(name)
}

// GetDefault returns the default provider factory from the global registry
func GetDefault() (ProviderFactory, error) {
	return globalRegistry.GetDefaultFactory()
}

// Create creates a provider instance using the global registry
func Create(name string, cfg config.Config) (Provider, error) {
	return globalRegistry.CreateProvider(name, cfg)
}

// CreateDefault creates a default provider instance using the global registry
func CreateDefault(cfg config.Config) (Provider, error) {
	return globalRegistry.CreateDefaultProvider(cfg)
}

// List returns all registered provider names from the global registry
func List() []string {
	return globalRegistry.ListProviders()
}

// Info returns information about a provider from the global registry
func Info(name string) (ProviderInfo, error) {
	return globalRegistry.GetProviderInfo(name)
}

// AllInfo returns information about all providers from the global registry
func AllInfo() []ProviderInfo {
	return globalRegistry.GetAllProviderInfo()
}
