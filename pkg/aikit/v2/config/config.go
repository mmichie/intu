// Package config provides configuration structures for aikit
package config

import (
	"os"
	"strconv"
	"time"
)

// Config provides explicit configuration for AI providers
type Config struct {
	// API credentials
	APIKey string

	// Model configuration
	Model       string
	MaxTokens   int
	Temperature float64

	// Service configuration
	BaseURL string
	Timeout time.Duration
}

// ProviderOptions allow optional configuration updates
type ProviderOption func(*Config)

// WithAPIKey sets the API key
func WithAPIKey(apiKey string) ProviderOption {
	return func(c *Config) {
		c.APIKey = apiKey
	}
}

// WithModel sets the model name
func WithModel(model string) ProviderOption {
	return func(c *Config) {
		c.Model = model
	}
}

// WithMaxTokens sets the maximum tokens for responses
func WithMaxTokens(maxTokens int) ProviderOption {
	return func(c *Config) {
		c.MaxTokens = maxTokens
	}
}

// WithTemperature sets the temperature for sampling
func WithTemperature(temperature float64) ProviderOption {
	return func(c *Config) {
		c.Temperature = temperature
	}
}

// WithBaseURL sets the API base URL
func WithBaseURL(baseURL string) ProviderOption {
	return func(c *Config) {
		c.BaseURL = baseURL
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) ProviderOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// NewConfig creates a new configuration with defaults
func NewConfig(options ...ProviderOption) Config {
	// Create default configuration
	config := Config{
		MaxTokens:   4096,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
	}

	// Apply all provided options
	for _, option := range options {
		option(&config)
	}

	return config
}

// FromEnvironment loads configuration from environment variables
func FromEnvironment(prefix string) Config {
	if prefix != "" && prefix[len(prefix)-1] != '_' {
		prefix = prefix + "_"
	}

	config := Config{
		APIKey:      os.Getenv(prefix + "API_KEY"),
		Model:       os.Getenv(prefix + "MODEL"),
		BaseURL:     os.Getenv(prefix + "BASE_URL"),
		MaxTokens:   parseEnvInt(prefix+"MAX_TOKENS", 4096),
		Temperature: parseEnvFloat(prefix+"TEMPERATURE", 0.7),
		Timeout:     parseEnvDuration(prefix+"TIMEOUT", 30*time.Second),
	}

	return config
}

// Merge combines this configuration with another, with the other taking precedence
func (c Config) Merge(other Config) Config {
	// Only override values that are set in the other config
	result := c

	if other.APIKey != "" {
		result.APIKey = other.APIKey
	}

	if other.Model != "" {
		result.Model = other.Model
	}

	if other.BaseURL != "" {
		result.BaseURL = other.BaseURL
	}

	if other.MaxTokens != 0 {
		result.MaxTokens = other.MaxTokens
	}

	if other.Temperature != 0 {
		result.Temperature = other.Temperature
	}

	if other.Timeout != 0 {
		result.Timeout = other.Timeout
	}

	return result
}

// WithOptions returns a new Config with options applied
func (c Config) WithOptions(options ...ProviderOption) Config {
	result := c
	for _, option := range options {
		option(&result)
	}
	return result
}

// Utility functions for environment variable parsing

func parseEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func parseEnvFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}

	return value
}

func parseEnvDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
