package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mmichie/intu/pkg/aikit/v2/function"
)

func TestBaseProvider(t *testing.T) {
	// Create test model capabilities
	models := map[string]ModelCapabilities{
		"test-model-1": {
			Supported:        true,
			FunctionCalling:  true,
			VisionCapable:    true,
			MaxContextTokens: 100000,
			DefaultMaxTokens: 4096,
		},
		"test-model-2": {
			Supported:        true,
			FunctionCalling:  false,
			VisionCapable:    false,
			MaxContextTokens: 50000,
			DefaultMaxTokens: 2048,
		},
		"unsupported-model": {
			Supported: false,
		},
	}

	base := NewBaseProvider("test-provider", "test-key", "", "https://test.api", "test-model-1", models)

	t.Run("ValidateAndSetModel", func(t *testing.T) {
		// Test empty model defaults to default model
		err := base.ValidateAndSetModel("")
		if err != nil {
			t.Errorf("Expected no error for empty model, got: %v", err)
		}
		if base.GetModel() != "test-model-1" {
			t.Errorf("Expected model to be test-model-1, got: %s", base.GetModel())
		}

		// Test valid model
		err = base.ValidateAndSetModel("test-model-2")
		if err != nil {
			t.Errorf("Expected no error for valid model, got: %v", err)
		}
		if base.GetModel() != "test-model-2" {
			t.Errorf("Expected model to be test-model-2, got: %s", base.GetModel())
		}

		// Test unsupported model
		err = base.ValidateAndSetModel("unsupported-model")
		if err == nil {
			t.Error("Expected error for unsupported model")
		}

		// Test non-existent model
		err = base.ValidateAndSetModel("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent model")
		}
	})

	t.Run("Capabilities", func(t *testing.T) {
		// Set to model with full capabilities
		base.ValidateAndSetModel("test-model-1")

		if !base.SupportsFunctionCalling() {
			t.Error("Expected test-model-1 to support function calling")
		}
		if !base.SupportsVision() {
			t.Error("Expected test-model-1 to support vision")
		}

		// Set to model without capabilities
		base.ValidateAndSetModel("test-model-2")

		if base.SupportsFunctionCalling() {
			t.Error("Expected test-model-2 to not support function calling")
		}
		if base.SupportsVision() {
			t.Error("Expected test-model-2 to not support vision")
		}
	})

	t.Run("RequestDefaults", func(t *testing.T) {
		base.ValidateAndSetModel("test-model-1")

		request := Request{
			Prompt: "Test prompt",
		}

		base.PrepareRequestDefaults(&request)

		if request.Temperature != 0.7 {
			t.Errorf("Expected temperature to be set to 0.7, got %f", request.Temperature)
		}

		if request.MaxTokens != 4096 {
			t.Errorf("Expected max tokens to be set to 4096, got %d", request.MaxTokens)
		}

		// Test with pre-set values
		request2 := Request{
			Prompt:      "Test prompt",
			Temperature: 0.5,
			MaxTokens:   1000,
		}

		base.PrepareRequestDefaults(&request2)

		if request2.Temperature != 0.5 {
			t.Errorf("Expected temperature to remain 0.5, got %f", request2.Temperature)
		}

		if request2.MaxTokens != 1000 {
			t.Errorf("Expected max tokens to remain 1000, got %d", request2.MaxTokens)
		}
	})

	t.Run("FunctionExecution", func(t *testing.T) {
		// Set up function executor
		executed := false
		base.SetFunctionExecutor(func(call function.FunctionCall) (function.FunctionResponse, error) {
			executed = true
			return function.FunctionResponse{
				Name:    call.Name,
				Content: map[string]string{"result": "success"},
			}, nil
		})

		// Execute function call
		funcCall := function.FunctionCall{
			Name:       "test_function",
			Parameters: json.RawMessage(`{"param": "value"}`),
		}

		result, err := base.ExecuteFunctionCall(context.Background(), funcCall)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !executed {
			t.Error("Function executor was not called")
		}

		if !strings.Contains(result, "Successfully executed function") {
			t.Error("Expected success message in result")
		}

		if !strings.Contains(result, "test_function") {
			t.Error("Expected function name in result")
		}
	})
}

func TestHTTPOptions(t *testing.T) {
	base := NewBaseProvider("test", "key", "model", "url", "default", nil)

	t.Run("DefaultOptions", func(t *testing.T) {
		opts := base.GetDefaultHTTPOptions()

		if opts.Timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got: %v", opts.Timeout)
		}

		if opts.MaxRetries != 3 {
			t.Errorf("Expected 3 retries, got: %d", opts.MaxRetries)
		}

		if opts.RetryDelay != time.Second {
			t.Errorf("Expected 1s retry delay, got: %v", opts.RetryDelay)
		}
	})

	t.Run("StreamingOptions", func(t *testing.T) {
		opts := base.GetStreamingHTTPOptions()

		if opts.Timeout != 90*time.Second {
			t.Errorf("Expected timeout 90s, got: %v", opts.Timeout)
		}

		if opts.MaxRetries != 3 {
			t.Errorf("Expected 3 retries, got: %d", opts.MaxRetries)
		}
	})
}
