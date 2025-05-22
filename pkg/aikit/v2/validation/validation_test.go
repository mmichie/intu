package validation

import (
	"strings"
	"testing"
)

func TestRequiredValidator(t *testing.T) {
	validator := Required()

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid string", "hello", false},
		{"empty string", "", true},
		{"nil value", nil, true},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"empty slice", []string{}, true},
		{"non-empty slice", []string{"a"}, false},
		{"nil pointer", (*string)(nil), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Required.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinLengthValidator(t *testing.T) {
	validator := MinLength(3)

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid string", "hello", false},
		{"short string", "hi", true},
		{"exact length", "abc", false},
		{"valid slice", []int{1, 2, 3, 4}, false},
		{"short slice", []int{1}, true},
		{"invalid type", 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinLength.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxLengthValidator(t *testing.T) {
	validator := MaxLength(5)

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid string", "hello", false},
		{"long string", "hello world", true},
		{"exact length", "12345", false},
		{"valid slice", []int{1, 2, 3}, false},
		{"long slice", []int{1, 2, 3, 4, 5, 6}, true},
		{"invalid type", 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxLength.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRangeValidator(t *testing.T) {
	validator := Range(0, 100)

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid int", 50, false},
		{"min value", 0, false},
		{"max value", 100, false},
		{"too small", -1, true},
		{"too large", 101, true},
		{"valid float", 50.5, false},
		{"invalid type", "50", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Range.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatternValidator(t *testing.T) {
	validator, err := Pattern(`^[a-z]+$`, "must be lowercase letters only")
	if err != nil {
		t.Fatalf("Failed to create pattern validator: %v", err)
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid match", "hello", false},
		{"invalid match", "Hello", true},
		{"numbers", "123", true},
		{"invalid type", 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Pattern.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOneOfValidator(t *testing.T) {
	validator := OneOf("red", "green", "blue")

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid value 1", "red", false},
		{"valid value 2", "green", false},
		{"valid value 3", "blue", false},
		{"invalid value", "yellow", true},
		{"wrong type", 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("OneOf.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFieldValidator(t *testing.T) {
	fv := NewFieldValidator().
		Add(Required()).
		Add(MinLength(3)).
		Add(MaxLength(10))

	tests := []struct {
		name       string
		value      interface{}
		errorCount int
	}{
		{"valid value", "hello", 0},
		{"empty value", "", 2},          // Required + MinLength
		{"too short", "hi", 1},          // MinLength
		{"too long", "hello world!", 1}, // MaxLength
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := fv.Validate("testField", tt.value)
			if len(errors) != tt.errorCount {
				t.Errorf("Expected %d errors, got %d: %v", tt.errorCount, len(errors), errors)
			}
		})
	}
}

type TestStruct struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

func TestStructValidator(t *testing.T) {
	emailPattern, _ := Pattern(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, "invalid email format")

	sv := NewStructValidator()
	sv.Field("Name").Add(Required()).Add(MinLength(2))
	sv.Field("Age").Add(Required()).Add(Range(0, 150))
	sv.Field("Email").Add(Required()).Add(emailPattern)

	tests := []struct {
		name       string
		obj        TestStruct
		errorCount int
	}{
		{
			"valid struct",
			TestStruct{Name: "John", Age: 30, Email: "john@example.com"},
			0,
		},
		{
			"empty name",
			TestStruct{Name: "", Age: 30, Email: "john@example.com"},
			2, // Required + MinLength
		},
		{
			"invalid age",
			TestStruct{Name: "John", Age: 200, Email: "john@example.com"},
			1, // Range
		},
		{
			"invalid email",
			TestStruct{Name: "John", Age: 30, Email: "invalid"},
			1, // Pattern
		},
		{
			"multiple errors",
			TestStruct{Name: "", Age: -5, Email: ""},
			5, // Name: Required + MinLength, Age: Range, Email: Required + Pattern
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := sv.Validate(tt.obj)
			if len(errors) != tt.errorCount {
				t.Errorf("Expected %d errors, got %d: %v", tt.errorCount, len(errors), errors)
			}
		})
	}
}

func TestJSONSchemaValidator(t *testing.T) {
	schema := map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"name", "age"},
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
			"age": map[string]interface{}{
				"type": "integer",
			},
			"email": map[string]interface{}{
				"type": "string",
			},
		},
	}

	validator := JSONSchema(schema)

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{
			"valid object",
			map[string]interface{}{
				"name":  "John",
				"age":   30,
				"email": "john@example.com",
			},
			false,
		},
		{
			"missing required field",
			map[string]interface{}{
				"name": "John",
			},
			true,
		},
		{
			"wrong type",
			map[string]interface{}{
				"name": "John",
				"age":  "thirty", // Should be integer
			},
			true,
		},
		{
			"extra fields allowed",
			map[string]interface{}{
				"name":  "John",
				"age":   30,
				"extra": "field",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONSchema.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "email",
		Value:   "invalid",
		Rule:    "pattern",
		Message: "invalid email format",
	}

	expected := "validation failed for field 'email': invalid email format"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test without field
	err2 := ValidationError{
		Message: "general validation error",
	}

	expected2 := "validation failed: general validation error"
	if err2.Error() != expected2 {
		t.Errorf("Expected error message '%s', got '%s'", expected2, err2.Error())
	}
}

func TestValidationErrors(t *testing.T) {
	// Test empty errors
	var errors ValidationErrors
	if errors.Error() != "no validation errors" {
		t.Errorf("Expected 'no validation errors', got '%s'", errors.Error())
	}

	// Test single error
	errors = append(errors, ValidationError{
		Field:   "name",
		Message: "is required",
	})

	if !strings.Contains(errors.Error(), "name") {
		t.Errorf("Expected error to contain 'name', got '%s'", errors.Error())
	}

	// Test multiple errors
	errors = append(errors, ValidationError{
		Field:   "age",
		Message: "must be positive",
	})

	errStr := errors.Error()
	if !strings.Contains(errStr, "multiple validation errors") {
		t.Errorf("Expected 'multiple validation errors', got '%s'", errStr)
	}
	if !strings.Contains(errStr, "name") || !strings.Contains(errStr, "age") {
		t.Errorf("Expected both field names in error, got '%s'", errStr)
	}
}
