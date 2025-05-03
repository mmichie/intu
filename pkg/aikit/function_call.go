// Package aikit provides AI interaction capabilities
package aikit

import (
	"encoding/json"
	"errors"
)

// FunctionDefinition represents a function that can be called by the LLM
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// FunctionCall represents a function call from the LLM
type FunctionCall struct {
	Name       string          `json:"name"`
	Parameters json.RawMessage `json:"parameters"`
}

// FunctionResponse represents the response from a function call
type FunctionResponse struct {
	Name     string      `json:"name"`
	Content  interface{} `json:"content"`
	Error    string      `json:"error,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// Validate ensures a function definition is properly formed
func (fd *FunctionDefinition) Validate() error {
	if fd.Name == "" {
		return errors.New("function definition must have a name")
	}
	
	if fd.Description == "" {
		return errors.New("function definition must have a description")
	}
	
	if fd.Parameters == nil {
		return errors.New("function definition must have parameters schema (use empty object for no parameters)")
	}
	
	return nil
}

// ToMap converts a function definition to a map structure
// that can be easily serialized for different provider formats
func (fd *FunctionDefinition) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"name":        fd.Name,
		"description": fd.Description,
		"parameters":  fd.Parameters,
	}
}

// FunctionExecutorFunc defines a function that executes function calls
type FunctionExecutorFunc func(call FunctionCall) (FunctionResponse, error)