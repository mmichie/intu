// Package function provides function calling interfaces for AI providers
package function

import (
	"encoding/json"
	"errors"
)

// FunctionDefinition represents a function that can be called by an AI model
type FunctionDefinition struct {
	// Name is the unique identifier for the function
	Name string `json:"name"`

	// Description explains what the function does
	Description string `json:"description"`

	// Parameters defines the expected input parameters in JSON Schema format
	// This should be a valid JSON Schema object
	Parameters map[string]interface{} `json:"parameters"`
}

// FunctionCall represents a function call requested by an AI model
type FunctionCall struct {
	// Name is the function being called
	Name string `json:"name"`

	// Parameters contains the serialized parameters for the function call
	Parameters json.RawMessage `json:"parameters"`
}

// FunctionResponse represents the response from a function execution
type FunctionResponse struct {
	// Name is the function that was called
	Name string `json:"name"`

	// Content contains the function result
	Content interface{} `json:"content"`

	// Error contains an error message if the function failed
	Error string `json:"error,omitempty"`

	// Metadata contains optional additional information
	Metadata interface{} `json:"metadata,omitempty"`
}

// FunctionExecutor processes function calls and returns responses
type FunctionExecutor func(call FunctionCall) (FunctionResponse, error)

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

	// Could add more validation for JSON Schema correctness here

	return nil
}

// Registry manages a collection of function definitions
type Registry struct {
	functions map[string]FunctionDefinition
}

// NewRegistry creates a new function registry
func NewRegistry() *Registry {
	return &Registry{
		functions: make(map[string]FunctionDefinition),
	}
}

// Register adds a function definition to the registry
// Returns error if validation fails or function already exists
func (r *Registry) Register(def FunctionDefinition) error {
	if err := def.Validate(); err != nil {
		return err
	}

	if _, exists := r.functions[def.Name]; exists {
		return errors.New("function already registered: " + def.Name)
	}

	r.functions[def.Name] = def
	return nil
}

// RegisterMany adds multiple function definitions
// Returns on first error
func (r *Registry) RegisterMany(defs []FunctionDefinition) error {
	for _, def := range defs {
		if err := r.Register(def); err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves a function definition by name
// Returns the definition and a boolean indicating if it was found
func (r *Registry) Get(name string) (FunctionDefinition, bool) {
	def, found := r.functions[name]
	return def, found
}

// List returns all registered function definitions
func (r *Registry) List() []FunctionDefinition {
	result := make([]FunctionDefinition, 0, len(r.functions))
	for _, def := range r.functions {
		result = append(result, def)
	}
	return result
}

// CreateExecutor creates a function executor that uses this registry
func (r *Registry) CreateExecutor(handler func(name string, params json.RawMessage) (interface{}, error)) FunctionExecutor {
	return func(call FunctionCall) (FunctionResponse, error) {
		// Check if function exists
		if _, found := r.Get(call.Name); !found {
			return FunctionResponse{
				Name:  call.Name,
				Error: "function not found: " + call.Name,
			}, errors.New("function not found: " + call.Name)
		}

		// Execute function
		result, err := handler(call.Name, call.Parameters)
		if err != nil {
			return FunctionResponse{
				Name:  call.Name,
				Error: err.Error(),
			}, err
		}

		// Return successful response
		return FunctionResponse{
			Name:    call.Name,
			Content: result,
		}, nil
	}
}
