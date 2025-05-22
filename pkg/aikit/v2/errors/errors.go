// Package errors provides domain-specific error types for aikit
package errors

import (
	"errors"
	"fmt"
)

// Standard errors that can be used with errors.Is()
var (
	// ErrInvalidConfig indicates a configuration error
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrRateLimit indicates provider rate limiting
	ErrRateLimit = errors.New("rate limit exceeded")

	// ErrModelContext indicates context length exceeded
	ErrModelContext = errors.New("context limit exceeded")

	// ErrAuthentication indicates authentication failure
	ErrAuthentication = errors.New("authentication failed")

	// ErrModelUnavailable indicates the model is not available
	ErrModelUnavailable = errors.New("model unavailable")

	// ErrProviderUnavailable indicates the provider is not available
	ErrProviderUnavailable = errors.New("provider unavailable")

	// ErrFunctionNotFound indicates a requested function was not found
	ErrFunctionNotFound = errors.New("function not found")

	// ErrFunctionExecutionFailed indicates a function execution failure
	ErrFunctionExecutionFailed = errors.New("function execution failed")
)

// ProviderError wraps provider-related errors with context
type ProviderError struct {
	// Provider is the name of the provider (e.g., "claude", "openai")
	Provider string

	// Operation being performed (e.g., "generate_response", "stream")
	Op string

	// Underlying error
	Err error
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider %s: %s: %v", e.Provider, e.Op, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As support
func (e *ProviderError) Unwrap() error {
	return e.Err
}

// New creates a new ProviderError
func New(provider, op string, err error) error {
	return &ProviderError{
		Provider: provider,
		Op:       op,
		Err:      err,
	}
}

// Wrap adds provider context to an existing error
func Wrap(err error, provider, op string) error {
	if err == nil {
		return nil
	}
	return &ProviderError{
		Provider: provider,
		Op:       op,
		Err:      err,
	}
}

// Is enables custom error matching
func (e *ProviderError) Is(target error) bool {
	// Embed standard error check
	if errors.Is(e.Err, target) {
		return true
	}

	// Compare with another ProviderError
	t, ok := target.(*ProviderError)
	if !ok {
		return false
	}

	// Match on specific fields if provided
	if t.Provider != "" && t.Provider != e.Provider {
		return false
	}
	if t.Op != "" && t.Op != e.Op {
		return false
	}

	// If we got here with specific fields, it's a match
	if t.Provider != "" || t.Op != "" {
		return true
	}

	// Otherwise, check the wrapped error
	return errors.Is(e.Err, t.Err)
}
