package errors

import (
	"errors"
	"testing"
)

func TestProviderError(t *testing.T) {
	// Test basic error construction
	baseErr := errors.New("base error")
	err := New("claude", "generate", baseErr)

	// Test error message
	expected := "provider claude: generate: base error"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}

	// Test unwrapping
	unwrapped := errors.Unwrap(err)
	if unwrapped != baseErr {
		t.Errorf("Expected unwrapped error %v, got %v", baseErr, unwrapped)
	}

	// Test errors.Is with standard errors
	rateErr := New("claude", "generate", ErrRateLimit)
	if !errors.Is(rateErr, ErrRateLimit) {
		t.Error("errors.Is failed with standard error")
	}

	// Test errors.Is with provider pattern matching
	patternErr := &ProviderError{Provider: "claude", Op: "", Err: nil}
	if !errors.Is(err, patternErr) {
		t.Error("errors.Is failed with provider pattern matching")
	}

	// Test that non-matching provider returns false
	wrongProvider := &ProviderError{Provider: "openai", Op: "", Err: nil}
	if errors.Is(err, wrongProvider) {
		t.Error("errors.Is incorrectly matched different provider")
	}
}

func TestWrap(t *testing.T) {
	// Test wrapping nil error
	if Wrap(nil, "claude", "generate") != nil {
		t.Error("Wrap(nil) should return nil")
	}

	// Test wrapping standard error
	wrapped := Wrap(ErrRateLimit, "claude", "generate")
	if !errors.Is(wrapped, ErrRateLimit) {
		t.Error("Wrapped error should match original with errors.Is")
	}

	// Check the error type
	var provErr *ProviderError
	if !errors.As(wrapped, &provErr) {
		t.Error("Wrapped error should be a ProviderError")
	}

	// Check the fields
	if provErr.Provider != "claude" || provErr.Op != "generate" {
		t.Errorf("Expected provider 'claude' and op 'generate', got %q and %q",
			provErr.Provider, provErr.Op)
	}
}
