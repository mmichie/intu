package provider

import (
	"context"
	"testing"

	"github.com/mmichie/intu/pkg/aikit/v2/config"
)

// mockFactory is a test factory
type mockFactory struct {
	name         string
	models       []string
	capabilities []string
	createError  error
}

func (f *mockFactory) Name() string {
	return f.name
}

func (f *mockFactory) Create(cfg config.Config) (Provider, error) {
	if f.createError != nil {
		return nil, f.createError
	}
	return &mockProvider{name: f.name}, nil
}

func (f *mockFactory) GetAvailableModels() []string {
	return f.models
}

func (f *mockFactory) GetCapabilities() []string {
	return f.capabilities
}

// mockProvider is a test provider
type mockProvider struct {
	name string
}

func (p *mockProvider) GenerateResponse(ctx context.Context, request Request) (Response, error) {
	return Response{}, nil
}

func (p *mockProvider) GenerateStreamingResponse(ctx context.Context, request Request, handler StreamHandler) error {
	return nil
}

func (p *mockProvider) Name() string {
	return p.name
}

func (p *mockProvider) Model() string {
	return "test-model"
}

func (p *mockProvider) Capabilities() []string {
	return []string{"test"}
}

func TestRegistry(t *testing.T) {
	// Create a new registry for testing
	reg := NewRegistry()

	// Test registering factories
	factory1 := &mockFactory{
		name:         "test1",
		models:       []string{"model1", "model2"},
		capabilities: []string{"cap1", "cap2"},
	}

	factory2 := &mockFactory{
		name:         "test2",
		models:       []string{"model3"},
		capabilities: []string{"cap3"},
	}

	// Register first factory
	err := reg.RegisterFactory(factory1)
	if err != nil {
		t.Errorf("Failed to register factory1: %v", err)
	}

	// First factory should be default
	defaultFactory, err := reg.GetDefaultFactory()
	if err != nil {
		t.Errorf("Failed to get default factory: %v", err)
	}
	if defaultFactory.Name() != "test1" {
		t.Errorf("Expected default factory to be test1, got %s", defaultFactory.Name())
	}

	// Register second factory
	err = reg.RegisterFactory(factory2)
	if err != nil {
		t.Errorf("Failed to register factory2: %v", err)
	}

	// Test duplicate registration
	err = reg.RegisterFactory(factory1)
	if err == nil {
		t.Error("Expected error when registering duplicate factory")
	}

	// Test listing providers
	providers := reg.ListProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Test getting factory
	f, err := reg.GetFactory("test2")
	if err != nil {
		t.Errorf("Failed to get factory: %v", err)
	}
	if f.Name() != "test2" {
		t.Errorf("Expected factory name test2, got %s", f.Name())
	}

	// Test getting non-existent factory
	_, err = reg.GetFactory("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent factory")
	}

	// Test setting default
	err = reg.SetDefaultFactory("test2")
	if err != nil {
		t.Errorf("Failed to set default factory: %v", err)
	}

	defaultFactory, err = reg.GetDefaultFactory()
	if err != nil {
		t.Errorf("Failed to get default factory: %v", err)
	}
	if defaultFactory.Name() != "test2" {
		t.Errorf("Expected default factory to be test2, got %s", defaultFactory.Name())
	}

	// Test creating provider
	cfg := config.Config{APIKey: "test-key"}
	provider, err := reg.CreateProvider("test1", cfg)
	if err != nil {
		t.Errorf("Failed to create provider: %v", err)
	}
	if provider.Name() != "test1" {
		t.Errorf("Expected provider name test1, got %s", provider.Name())
	}

	// Test provider info
	info, err := reg.GetProviderInfo("test1")
	if err != nil {
		t.Errorf("Failed to get provider info: %v", err)
	}
	if info.Name != "test1" {
		t.Errorf("Expected provider info name test1, got %s", info.Name)
	}
	if len(info.Models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(info.Models))
	}
	if info.IsDefault {
		t.Error("Expected test1 to not be default")
	}

	// Test all provider info
	allInfo := reg.GetAllProviderInfo()
	if len(allInfo) != 2 {
		t.Errorf("Expected 2 provider infos, got %d", len(allInfo))
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Save current state
	oldRegistry := globalRegistry
	defer func() {
		globalRegistry = oldRegistry
	}()

	// Use a fresh registry for testing
	globalRegistry = NewRegistry()

	// Test global functions
	factory := &mockFactory{
		name:         "global-test",
		models:       []string{"model1"},
		capabilities: []string{"cap1"},
	}

	err := Register(factory)
	if err != nil {
		t.Errorf("Failed to register factory: %v", err)
	}

	providers := List()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	f, err := Get("global-test")
	if err != nil {
		t.Errorf("Failed to get factory: %v", err)
	}
	if f.Name() != "global-test" {
		t.Errorf("Expected factory name global-test, got %s", f.Name())
	}

	// Test backward compatibility functions
	err = RegisterFactory(factory)
	if err == nil {
		t.Error("Expected error when registering duplicate factory")
	}

	availableProviders := GetAvailableProviders()
	if len(availableProviders) != 1 {
		t.Errorf("Expected 1 available provider, got %d", len(availableProviders))
	}

	if !IsProviderAvailable("global-test") {
		t.Error("Expected global-test to be available")
	}

	if IsProviderAvailable("nonexistent") {
		t.Error("Expected nonexistent to not be available")
	}
}
