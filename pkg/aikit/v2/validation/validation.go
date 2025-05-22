// Package validation provides request and response validation for the AI kit
package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Validator interface for custom validation logic
type Validator interface {
	Validate(value interface{}) error
	Name() string
}

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string
	Value   interface{}
	Rule    string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed: %s", e.Message)
}

// ValidationErrors represents multiple validation failures
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}

	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return fmt.Sprintf("multiple validation errors:\n%s", strings.Join(msgs, "\n"))
}

// FieldValidator validates a specific field
type FieldValidator struct {
	validators []Validator
}

// NewFieldValidator creates a new field validator
func NewFieldValidator() *FieldValidator {
	return &FieldValidator{
		validators: make([]Validator, 0),
	}
}

// Add adds a validator to the field
func (fv *FieldValidator) Add(validator Validator) *FieldValidator {
	fv.validators = append(fv.validators, validator)
	return fv
}

// Validate runs all validators on the value
func (fv *FieldValidator) Validate(fieldName string, value interface{}) ValidationErrors {
	var errors ValidationErrors

	for _, validator := range fv.validators {
		if err := validator.Validate(value); err != nil {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Value:   value,
				Rule:    validator.Name(),
				Message: err.Error(),
			})
		}
	}

	return errors
}

// StructValidator validates entire structs
type StructValidator struct {
	fields map[string]*FieldValidator
}

// NewStructValidator creates a new struct validator
func NewStructValidator() *StructValidator {
	return &StructValidator{
		fields: make(map[string]*FieldValidator),
	}
}

// Field gets or creates a field validator
func (sv *StructValidator) Field(name string) *FieldValidator {
	if fv, exists := sv.fields[name]; exists {
		return fv
	}

	fv := NewFieldValidator()
	sv.fields[name] = fv
	return fv
}

// Validate validates a struct
func (sv *StructValidator) Validate(obj interface{}) ValidationErrors {
	var allErrors ValidationErrors

	// Use reflection to get field values
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		allErrors = append(allErrors, ValidationError{
			Message: "expected struct type",
		})
		return allErrors
	}

	t := v.Type()

	// Validate each field
	for fieldName, fieldValidator := range sv.fields {
		// Find the field
		field, found := t.FieldByName(fieldName)
		if !found {
			// Try json tag
			found = false
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				if jsonTag := f.Tag.Get("json"); jsonTag != "" {
					tagName := strings.Split(jsonTag, ",")[0]
					if tagName == fieldName {
						field = f
						found = true
						break
					}
				}
			}

			if !found {
				allErrors = append(allErrors, ValidationError{
					Field:   fieldName,
					Message: "field not found in struct",
				})
				continue
			}
		}

		// Get field value
		fieldValue := v.FieldByIndex(field.Index)

		// Run validators
		errors := fieldValidator.Validate(fieldName, fieldValue.Interface())
		allErrors = append(allErrors, errors...)
	}

	return allErrors
}

// Common validators

// Required validator checks if a value is not empty
type RequiredValidator struct{}

func (r RequiredValidator) Name() string { return "required" }

func (r RequiredValidator) Validate(value interface{}) error {
	if value == nil {
		return errors.New("value is required")
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		if v.String() == "" {
			return errors.New("value is required")
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if v.Len() == 0 {
			return errors.New("value is required")
		}
	case reflect.Ptr:
		if v.IsNil() {
			return errors.New("value is required")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() == 0 {
			return errors.New("value is required")
		}
	case reflect.Float32, reflect.Float64:
		if v.Float() == 0 {
			return errors.New("value is required")
		}
	}

	return nil
}

// MinLength validator checks minimum length
type MinLengthValidator struct {
	Min int
}

func (m MinLengthValidator) Name() string { return fmt.Sprintf("minLength(%d)", m.Min) }

func (m MinLengthValidator) Validate(value interface{}) error {
	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.String:
		if len(v.String()) < m.Min {
			return fmt.Errorf("minimum length is %d", m.Min)
		}
	case reflect.Slice, reflect.Array:
		if v.Len() < m.Min {
			return fmt.Errorf("minimum length is %d", m.Min)
		}
	default:
		return fmt.Errorf("minLength validator not applicable to type %T", value)
	}

	return nil
}

// MaxLength validator checks maximum length
type MaxLengthValidator struct {
	Max int
}

func (m MaxLengthValidator) Name() string { return fmt.Sprintf("maxLength(%d)", m.Max) }

func (m MaxLengthValidator) Validate(value interface{}) error {
	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.String:
		if len(v.String()) > m.Max {
			return fmt.Errorf("maximum length is %d", m.Max)
		}
	case reflect.Slice, reflect.Array:
		if v.Len() > m.Max {
			return fmt.Errorf("maximum length is %d", m.Max)
		}
	default:
		return fmt.Errorf("maxLength validator not applicable to type %T", value)
	}

	return nil
}

// Range validator checks numeric ranges
type RangeValidator struct {
	Min float64
	Max float64
}

func (r RangeValidator) Name() string { return fmt.Sprintf("range(%v,%v)", r.Min, r.Max) }

func (r RangeValidator) Validate(value interface{}) error {
	v := reflect.ValueOf(value)

	var num float64
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		num = float64(v.Int())
	case reflect.Float32, reflect.Float64:
		num = v.Float()
	default:
		return fmt.Errorf("range validator not applicable to type %T", value)
	}

	if num < r.Min || num > r.Max {
		return fmt.Errorf("value must be between %v and %v", r.Min, r.Max)
	}

	return nil
}

// Pattern validator checks against regex
type PatternValidator struct {
	Pattern *regexp.Regexp
	Message string
}

func NewPatternValidator(pattern, message string) (PatternValidator, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return PatternValidator{}, err
	}

	return PatternValidator{
		Pattern: re,
		Message: message,
	}, nil
}

func (p PatternValidator) Name() string { return "pattern" }

func (p PatternValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("pattern validator requires string type, got %T", value)
	}

	if !p.Pattern.MatchString(str) {
		if p.Message != "" {
			return errors.New(p.Message)
		}
		return fmt.Errorf("value does not match pattern %s", p.Pattern.String())
	}

	return nil
}

// OneOf validator checks if value is in a set
type OneOfValidator struct {
	Values []interface{}
}

func (o OneOfValidator) Name() string { return "oneOf" }

func (o OneOfValidator) Validate(value interface{}) error {
	for _, allowed := range o.Values {
		if reflect.DeepEqual(value, allowed) {
			return nil
		}
	}

	return fmt.Errorf("value must be one of %v", o.Values)
}

// JSONSchema validator validates against JSON schema
type JSONSchemaValidator struct {
	Schema map[string]interface{}
}

func (j JSONSchemaValidator) Name() string { return "jsonSchema" }

func (j JSONSchemaValidator) Validate(value interface{}) error {
	// Convert value to JSON and back to ensure proper type handling
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	var jsonValue interface{}
	if err := json.Unmarshal(jsonBytes, &jsonValue); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	// Basic JSON schema validation
	return j.validateAgainstSchema(jsonValue, j.Schema)
}

func (j JSONSchemaValidator) validateAgainstSchema(value interface{}, schema map[string]interface{}) error {
	// Check type
	if schemaType, ok := schema["type"].(string); ok {
		if !j.checkType(value, schemaType) {
			return fmt.Errorf("expected type %s", schemaType)
		}
	}

	// Check required properties for objects
	if schemaType, _ := schema["type"].(string); schemaType == "object" {
		objValue, ok := value.(map[string]interface{})
		if !ok {
			return errors.New("expected object type")
		}

		// Check required fields
		if required, ok := schema["required"].([]interface{}); ok {
			for _, req := range required {
				reqStr, _ := req.(string)
				if _, exists := objValue[reqStr]; !exists {
					return fmt.Errorf("required property '%s' is missing", reqStr)
				}
			}
		}

		// Check properties
		if properties, ok := schema["properties"].(map[string]interface{}); ok {
			for propName, propValue := range objValue {
				if propSchema, exists := properties[propName]; exists {
					if propSchemaMap, ok := propSchema.(map[string]interface{}); ok {
						if err := j.validateAgainstSchema(propValue, propSchemaMap); err != nil {
							return fmt.Errorf("property '%s': %w", propName, err)
						}
					}
				}
			}
		}
	}

	return nil
}

func (j JSONSchemaValidator) checkType(value interface{}, schemaType string) bool {
	switch schemaType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok1 := value.(float64)
		_, ok2 := value.(int)
		return ok1 || ok2
	case "integer":
		_, ok1 := value.(int)
		_, ok2 := value.(float64)
		if ok2 {
			// Check if float is actually an integer
			f := value.(float64)
			return f == float64(int(f))
		}
		return ok1
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "null":
		return value == nil
	default:
		return false
	}
}

// Helper functions

// Required creates a required validator
func Required() Validator {
	return RequiredValidator{}
}

// MinLength creates a min length validator
func MinLength(min int) Validator {
	return MinLengthValidator{Min: min}
}

// MaxLength creates a max length validator
func MaxLength(max int) Validator {
	return MaxLengthValidator{Max: max}
}

// Range creates a range validator
func Range(min, max float64) Validator {
	return RangeValidator{Min: min, Max: max}
}

// Pattern creates a pattern validator
func Pattern(pattern, message string) (Validator, error) {
	return NewPatternValidator(pattern, message)
}

// OneOf creates a one-of validator
func OneOf(values ...interface{}) Validator {
	return OneOfValidator{Values: values}
}

// JSONSchema creates a JSON schema validator
func JSONSchema(schema map[string]interface{}) Validator {
	return JSONSchemaValidator{Schema: schema}
}
