package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigDefaults(t *testing.T) {
	// Test default configuration
	config := NewConfig()

	// Check default values
	if config.MaxTokens != 4096 {
		t.Errorf("Expected default MaxTokens 4096, got %d", config.MaxTokens)
	}

	if config.Temperature != 0.7 {
		t.Errorf("Expected default Temperature 0.7, got %f", config.Temperature)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected default Timeout 30s, got %v", config.Timeout)
	}
}

func TestConfigOptions(t *testing.T) {
	// Test configuration with options
	config := NewConfig(
		WithAPIKey("test-api-key"),
		WithModel("test-model"),
		WithMaxTokens(1000),
		WithTemperature(0.5),
		WithBaseURL("https://test.example.com"),
		WithTimeout(10*time.Second),
	)

	// Check applied options
	if config.APIKey != "test-api-key" {
		t.Errorf("Expected APIKey 'test-api-key', got %q", config.APIKey)
	}

	if config.Model != "test-model" {
		t.Errorf("Expected Model 'test-model', got %q", config.Model)
	}

	if config.MaxTokens != 1000 {
		t.Errorf("Expected MaxTokens 1000, got %d", config.MaxTokens)
	}

	if config.Temperature != 0.5 {
		t.Errorf("Expected Temperature 0.5, got %f", config.Temperature)
	}

	if config.BaseURL != "https://test.example.com" {
		t.Errorf("Expected BaseURL 'https://test.example.com', got %q", config.BaseURL)
	}

	if config.Timeout != 10*time.Second {
		t.Errorf("Expected Timeout 10s, got %v", config.Timeout)
	}
}

func TestFromEnvironment(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("TEST_API_KEY", "env-api-key")
	os.Setenv("TEST_MODEL", "env-model")
	os.Setenv("TEST_BASE_URL", "https://env.example.com")
	os.Setenv("TEST_MAX_TOKENS", "2000")
	os.Setenv("TEST_TEMPERATURE", "0.3")
	os.Setenv("TEST_TIMEOUT", "20s")
	defer func() {
		// Clean up environment variables
		os.Unsetenv("TEST_API_KEY")
		os.Unsetenv("TEST_MODEL")
		os.Unsetenv("TEST_BASE_URL")
		os.Unsetenv("TEST_MAX_TOKENS")
		os.Unsetenv("TEST_TEMPERATURE")
		os.Unsetenv("TEST_TIMEOUT")
	}()

	// Load configuration from environment
	config := FromEnvironment("TEST")

	// Check loaded values
	if config.APIKey != "env-api-key" {
		t.Errorf("Expected APIKey 'env-api-key', got %q", config.APIKey)
	}

	if config.Model != "env-model" {
		t.Errorf("Expected Model 'env-model', got %q", config.Model)
	}

	if config.BaseURL != "https://env.example.com" {
		t.Errorf("Expected BaseURL 'https://env.example.com', got %q", config.BaseURL)
	}

	if config.MaxTokens != 2000 {
		t.Errorf("Expected MaxTokens 2000, got %d", config.MaxTokens)
	}

	if config.Temperature != 0.3 {
		t.Errorf("Expected Temperature 0.3, got %f", config.Temperature)
	}

	if config.Timeout != 20*time.Second {
		t.Errorf("Expected Timeout 20s, got %v", config.Timeout)
	}
}

func TestMergeConfig(t *testing.T) {
	// Create base configuration
	base := Config{
		APIKey:      "base-api-key",
		Model:       "base-model",
		BaseURL:     "https://base.example.com",
		MaxTokens:   1000,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
	}

	// Create override configuration (with some fields empty)
	override := Config{
		APIKey:    "override-api-key",
		Model:     "override-model",
		MaxTokens: 2000,
		// BaseURL, Temperature and Timeout left as zero values
	}

	// Merge configurations
	merged := base.Merge(override)

	// Check merged values
	if merged.APIKey != "override-api-key" {
		t.Errorf("Expected merged APIKey 'override-api-key', got %q", merged.APIKey)
	}

	if merged.Model != "override-model" {
		t.Errorf("Expected merged Model 'override-model', got %q", merged.Model)
	}

	if merged.BaseURL != "https://base.example.com" {
		t.Errorf("Expected merged BaseURL to remain 'https://base.example.com', got %q",
			merged.BaseURL)
	}

	if merged.MaxTokens != 2000 {
		t.Errorf("Expected merged MaxTokens 2000, got %d", merged.MaxTokens)
	}

	if merged.Temperature != 0.7 {
		t.Errorf("Expected merged Temperature to remain 0.7, got %f", merged.Temperature)
	}

	if merged.Timeout != 30*time.Second {
		t.Errorf("Expected merged Timeout to remain 30s, got %v", merged.Timeout)
	}
}

func TestWithOptions(t *testing.T) {
	// Create base configuration
	base := Config{
		APIKey:      "base-api-key",
		Model:       "base-model",
		BaseURL:     "https://base.example.com",
		MaxTokens:   1000,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
	}

	// Apply options using WithOptions
	updated := base.WithOptions(
		WithAPIKey("updated-api-key"),
		WithModel("updated-model"),
	)

	// Check updated values
	if updated.APIKey != "updated-api-key" {
		t.Errorf("Expected updated APIKey 'updated-api-key', got %q", updated.APIKey)
	}

	if updated.Model != "updated-model" {
		t.Errorf("Expected updated Model 'updated-model', got %q", updated.Model)
	}

	// Check unchanged values
	if updated.BaseURL != "https://base.example.com" {
		t.Errorf("Expected BaseURL to remain unchanged, got %q", updated.BaseURL)
	}

	// Verify original config is unchanged (immutability)
	if base.APIKey != "base-api-key" || base.Model != "base-model" {
		t.Error("Original configuration should not be modified")
	}
}
