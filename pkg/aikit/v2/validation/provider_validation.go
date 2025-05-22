package validation

import (
	"errors"
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

// RequestValidator validates provider requests
type RequestValidator struct {
	validator *StructValidator
}

// NewRequestValidator creates a validator for provider requests
func NewRequestValidator() *RequestValidator {
	sv := NewStructValidator()

	// Prompt is required and must be non-empty
	sv.Field("Prompt").
		Add(Required()).
		Add(MinLength(1))

	// Temperature must be between 0 and 1
	sv.Field("Temperature").
		Add(Range(0, 1))

	// MaxTokens must be positive
	sv.Field("MaxTokens").
		Add(Range(1, 1000000))

	return &RequestValidator{
		validator: sv,
	}
}

// Validate validates a provider request
func (rv *RequestValidator) Validate(request provider.Request) ValidationErrors {
	return rv.validator.Validate(request)
}

// ResponseValidator validates provider responses
type ResponseValidator struct {
	validator *StructValidator
}

// NewResponseValidator creates a validator for provider responses
func NewResponseValidator() *ResponseValidator {
	sv := NewStructValidator()

	// Content should not be empty if no function call
	// This is a conditional validation, so we'll handle it separately

	// Model must be provided
	sv.Field("Model").
		Add(Required()).
		Add(MinLength(1))

	// Provider must be provided
	sv.Field("Provider").
		Add(Required()).
		Add(MinLength(1))

	return &ResponseValidator{
		validator: sv,
	}
}

// Validate validates a provider response
func (rv *ResponseValidator) Validate(response provider.Response) ValidationErrors {
	errors := rv.validator.Validate(response)

	// Custom validation: Content should not be empty if no function call
	if response.FunctionCall == nil && response.Content == "" {
		errors = append(errors, ValidationError{
			Field:   "Content",
			Rule:    "required_without_function",
			Message: "content is required when no function call is present",
		})
	}

	return errors
}

// FunctionDefinitionValidator validates function definitions
type FunctionDefinitionValidator struct {
	validator *StructValidator
}

// NewFunctionDefinitionValidator creates a validator for function definitions
func NewFunctionDefinitionValidator() *FunctionDefinitionValidator {
	sv := NewStructValidator()

	// Name is required and must follow naming conventions
	pattern, _ := Pattern(`^[a-zA-Z][a-zA-Z0-9_]*$`, "must start with letter and contain only letters, numbers, and underscores")
	sv.Field("Name").
		Add(Required()).
		Add(MinLength(1)).
		Add(MaxLength(64)).
		Add(pattern)

	// Description is required
	sv.Field("Description").
		Add(Required()).
		Add(MinLength(10)).
		Add(MaxLength(500))

	// Parameters must be a valid JSON Schema
	sv.Field("Parameters").
		Add(Required()).
		Add(JSONSchemaValidator{
			Schema: map[string]interface{}{
				"type": "object",
			},
		})

	return &FunctionDefinitionValidator{
		validator: sv,
	}
}

// Validate validates a function definition
func (fv *FunctionDefinitionValidator) Validate(def interface{}) ValidationErrors {
	return fv.validator.Validate(def)
}

// StreamingRequestValidator validates streaming-specific requirements
type StreamingRequestValidator struct {
	*RequestValidator
}

// NewStreamingRequestValidator creates a validator for streaming requests
func NewStreamingRequestValidator() *StreamingRequestValidator {
	rv := NewRequestValidator()

	// For streaming, we might want different constraints
	// For example, no function calls in streaming mode

	return &StreamingRequestValidator{
		RequestValidator: rv,
	}
}

// Validate validates a streaming request
func (srv *StreamingRequestValidator) Validate(request provider.Request) ValidationErrors {
	// First run base validation
	errors := srv.RequestValidator.Validate(request)

	// Add streaming-specific validations
	if request.Stream && request.FunctionRegistry != nil {
		errors = append(errors, ValidationError{
			Field:   "FunctionRegistry",
			Rule:    "no_functions_in_stream",
			Message: "function calling is not supported in streaming mode",
		})
	}

	return errors
}

// ModelNameValidator validates model names against known models
type ModelNameValidator struct {
	knownModels map[string]bool
}

// NewModelNameValidator creates a validator for model names
func NewModelNameValidator(models []string) *ModelNameValidator {
	knownModels := make(map[string]bool)
	for _, model := range models {
		knownModels[model] = true
	}

	return &ModelNameValidator{
		knownModels: knownModels,
	}
}

func (m ModelNameValidator) Name() string { return "model_name" }

func (m ModelNameValidator) Validate(value interface{}) error {
	model, ok := value.(string)
	if !ok {
		return errors.New("model name must be a string")
	}

	if !m.knownModels[model] {
		return fmt.Errorf("unknown model: %s", model)
	}

	return nil
}

// CreateProviderValidator creates a validator with provider-specific rules
func CreateProviderValidator(providerName string, knownModels []string) *StructValidator {
	sv := NewStructValidator()

	// Common validations
	sv.Field("Prompt").
		Add(Required()).
		Add(MinLength(1))

	sv.Field("Model").
		Add(NewModelNameValidator(knownModels))

	// Provider-specific validations
	switch providerName {
	case "openai":
		// OpenAI specific limits
		sv.Field("MaxTokens").
			Add(Range(1, 32768))

	case "claude":
		// Claude specific limits
		sv.Field("MaxTokens").
			Add(Range(1, 200000))

	case "gemini":
		// Gemini specific limits
		sv.Field("MaxTokens").
			Add(Range(1, 100000))
	}

	return sv
}
