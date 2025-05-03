package aikit

import (
	"fmt"
	"sync"
)

// ProviderRegistry manages the available AI providers
type ProviderRegistry struct {
	mu             sync.RWMutex
	defaultFactory string
	factories      map[string]ProviderFactory
}

// ProviderFactory creates Provider instances
type ProviderFactory interface {
	// Create returns a new Provider instance
	Create() (Provider, error)

	// GetAvailableModels returns a list of available models for this provider
	GetAvailableModels() []string

	// GetCapabilities returns a list of capabilities supported by this provider
	GetCapabilities() []ProviderCapability
}

// Use the Provider interface from the existing definition

// ProviderCapability represents a capability that a provider may support
type ProviderCapability string

const (
	// CapabilityFunctionCalling indicates support for function calling
	CapabilityFunctionCalling ProviderCapability = "function_calling"

	// CapabilityVision indicates support for vision/image input
	CapabilityVision ProviderCapability = "vision"

	// CapabilityStreaming indicates support for streaming responses
	CapabilityStreaming ProviderCapability = "streaming"

	// CapabilityMultimodal indicates support for multiple input modalities
	CapabilityMultimodal ProviderCapability = "multimodal"
)

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		factories: make(map[string]ProviderFactory),
	}
}

// RegisterFactory adds a provider factory to the registry
func (r *ProviderRegistry) RegisterFactory(name string, factory ProviderFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("provider factory %q already registered", name)
	}

	r.factories[name] = factory

	// If this is the first factory, make it the default
	if r.defaultFactory == "" {
		r.defaultFactory = name
	}

	return nil
}

// SetDefaultFactory sets the default provider factory
func (r *ProviderRegistry) SetDefaultFactory(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; !exists {
		return fmt.Errorf("provider factory %q not registered", name)
	}

	r.defaultFactory = name
	return nil
}

// GetFactory returns a provider factory by name
func (r *ProviderRegistry) GetFactory(name string) (ProviderFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	return factory, exists
}

// GetDefaultFactory returns the default provider factory
func (r *ProviderRegistry) GetDefaultFactory() (ProviderFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.defaultFactory == "" {
		return nil, fmt.Errorf("no default provider factory set")
	}

	factory, exists := r.factories[r.defaultFactory]
	if !exists {
		return nil, fmt.Errorf("default provider factory %q not found", r.defaultFactory)
	}

	return factory, nil
}

// CreateProvider creates a provider instance using the specified factory
func (r *ProviderRegistry) CreateProvider(name string) (Provider, error) {
	factory, exists := r.GetFactory(name)
	if !exists {
		return nil, fmt.Errorf("provider factory %q not registered", name)
	}

	return factory.Create()
}

// CreateDefaultProvider creates a provider instance using the default factory
func (r *ProviderRegistry) CreateDefaultProvider() (Provider, error) {
	factory, err := r.GetDefaultFactory()
	if err != nil {
		return nil, err
	}

	return factory.Create()
}

// ListProviders returns a list of registered provider names
func (r *ProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(r.factories))
	for name := range r.factories {
		result = append(result, name)
	}

	return result
}
