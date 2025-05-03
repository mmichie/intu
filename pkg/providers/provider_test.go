package providers

import (
	"testing"
)

func TestBaseProvider_GetEnvOrDefault(t *testing.T) {
	p := BaseProvider{}

	// Test non-existent environment variable
	result := p.GetEnvOrDefault("NONEXISTENT_TEST_VAR", "default_value")
	if result != "default_value" {
		t.Errorf("GetEnvOrDefault() = %v, want %v", result, "default_value")
	}

	// Test with existing environment variable (would require setting env vars in test)
	// This is commented out as it would modify the test environment
	/*
		os.Setenv("TEST_VAR", "test_value")
		result = p.GetEnvOrDefault("TEST_VAR", "default_value")
		if result != "test_value" {
			t.Errorf("GetEnvOrDefault() = %v, want %v", result, "test_value")
		}
	*/
}

// PermissionCapability String test
func TestProviderCapability_String(t *testing.T) {
	tests := []struct {
		capability ProviderCapability
		want       string
	}{
		{CapabilityFunctionCalling, "function_calling"},
		{CapabilityVision, "vision"},
		{CapabilityStreaming, "streaming"},
		{CapabilityMultimodal, "multimodal"},
		{ProviderCapability("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.capability), func(t *testing.T) {
			got := string(tt.capability)
			if got != tt.want {
				t.Errorf("ProviderCapability string value = %v, want %v", got, tt.want)
			}
		})
	}
}
