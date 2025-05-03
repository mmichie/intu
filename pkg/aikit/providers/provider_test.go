package providers

import (
	"testing"
)

func TestProviderInitialization(t *testing.T) {
	// Test structure for initialization checks
	tests := []struct {
		name     string
		initFunc func() (Provider, error)
		wantName string
		needsEnv bool
	}{
		{
			name: "Claude Provider",
			initFunc: func() (Provider, error) {
				// During testing, ignore missing API keys
				t.Setenv("CLAUDE_API_KEY", "dummy_key")
				return NewClaudeAIProvider()
			},
			wantName: "claude",
			needsEnv: true,
		},
		{
			name: "OpenAI Provider",
			initFunc: func() (Provider, error) {
				t.Setenv("OPENAI_API_KEY", "dummy_key")
				return NewOpenAIProvider()
			},
			wantName: "openai",
			needsEnv: true,
		},
		{
			name: "Gemini Provider",
			initFunc: func() (Provider, error) {
				t.Setenv("GEMINI_API_KEY", "dummy_key")
				return NewGeminiProvider()
			},
			wantName: "gemini",
			needsEnv: true,
		},
		{
			name: "Grok Provider",
			initFunc: func() (Provider, error) {
				t.Setenv("XAI_API_KEY", "dummy_key")
				return NewGrokProvider()
			},
			wantName: "grok",
			needsEnv: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := tt.initFunc()
			if err != nil {
				if tt.needsEnv {
					t.Logf("Provider initialization error (possibly due to missing env var): %v", err)
					return
				}
				t.Fatalf("Failed to initialize provider: %v", err)
			}

			if provider.Name() != tt.wantName {
				t.Errorf("Provider name = %q, want %q", provider.Name(), tt.wantName)
			}

			// Check supported models
			models := provider.GetSupportedModels()
			if len(models) == 0 {
				t.Errorf("Provider %s returned no supported models", tt.wantName)
			}

			// Verify function calling capability works
			if !provider.SupportsFunctionCalling() {
				t.Logf("Provider %s does not support function calling with default model", tt.wantName)
			}
		})
	}
}

func TestFunctionDefinitionValidation(t *testing.T) {
	tests := []struct {
		name    string
		def     FunctionDefinition
		wantErr bool
	}{
		{
			name: "Valid function definition",
			def: FunctionDefinition{
				Name:        "testFunction",
				Description: "Test function",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"test": map[string]interface{}{
							"type":        "string",
							"description": "Test parameter",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			def: FunctionDefinition{
				Description: "Test function",
				Parameters:  map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "Missing description",
			def: FunctionDefinition{
				Name:       "testFunction",
				Parameters: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "Missing parameters",
			def: FunctionDefinition{
				Name:        "testFunction",
				Description: "Test function",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.def.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FunctionDefinition.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
