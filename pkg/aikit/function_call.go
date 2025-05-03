// Package aikit provides AI interaction capabilities
package aikit

// This file is a compatibility layer that re-exports types from the providers package
// In a future refactoring, we should consolidate these definitions to avoid duplication

import (
	"github.com/mmichie/intu/pkg/aikit/providers"
)

// Re-export function calling types from the providers package
type (
	// FunctionDefinition is re-exported from providers package
	FunctionDefinition = providers.FunctionDefinition

	// FunctionCall is re-exported from providers package
	FunctionCall = providers.FunctionCall

	// FunctionResponse is re-exported from providers package
	FunctionResponse = providers.FunctionResponse

	// FunctionExecutorFunc is re-exported from providers package
	FunctionExecutorFunc = providers.FunctionExecutorFunc
)
