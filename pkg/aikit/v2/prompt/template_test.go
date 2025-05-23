package prompt

import (
	"strings"
	"testing"
)

func TestRegistry(t *testing.T) {
	reg := NewRegistry()

	// Test registering a template
	tmpl := &Template{
		Name:        "test",
		Description: "Test template",
		Category:    "testing",
		content:     "Hello {{.Name}}!",
		Variables: []Variable{
			{Name: "Name", Description: "User name", Required: true},
		},
	}

	err := reg.Register(tmpl)
	if err != nil {
		t.Errorf("Failed to register template: %v", err)
	}

	// Test getting the template
	retrieved, err := reg.Get("test")
	if err != nil {
		t.Errorf("Failed to get template: %v", err)
	}
	if retrieved.Name != "test" {
		t.Errorf("Expected template name 'test', got '%s'", retrieved.Name)
	}

	// Test duplicate registration
	err = reg.Register(tmpl)
	if err == nil {
		t.Error("Expected error when registering duplicate template")
	}

	// Test getting non-existent template
	_, err = reg.Get("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent template")
	}

	// Test listing templates
	templates := reg.List()
	if len(templates) != 1 {
		t.Errorf("Expected 1 template, got %d", len(templates))
	}

	// Test listing by category
	testingTemplates := reg.ListByCategory("testing")
	if len(testingTemplates) != 1 {
		t.Errorf("Expected 1 testing template, got %d", len(testingTemplates))
	}

	noTemplates := reg.ListByCategory("nonexistent")
	if len(noTemplates) != 0 {
		t.Errorf("Expected 0 templates for nonexistent category, got %d", len(noTemplates))
	}
}

func TestTemplateExecution(t *testing.T) {
	reg := NewRegistry()

	tmpl := &Template{
		Name:        "greeting",
		Description: "Greeting template",
		Category:    "testing",
		content:     "Hello {{.Name}}, welcome to {{.Place}}!",
		Variables: []Variable{
			{Name: "Name", Description: "User name", Required: true},
			{Name: "Place", Description: "Location", Required: false, Default: "Earth"},
		},
	}

	err := reg.Register(tmpl)
	if err != nil {
		t.Fatalf("Failed to register template: %v", err)
	}

	// Test with all variables
	result, err := tmpl.ExecuteMap(map[string]interface{}{
		"Name":  "Alice",
		"Place": "Wonderland",
	})
	if err != nil {
		t.Errorf("Failed to execute template: %v", err)
	}
	expected := "Hello Alice, welcome to Wonderland!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with default value
	result, err = tmpl.ExecuteMap(map[string]interface{}{
		"Name": "Bob",
	})
	if err != nil {
		t.Errorf("Failed to execute template with default: %v", err)
	}
	expected = "Hello Bob, welcome to Earth!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test missing required variable
	_, err = tmpl.ExecuteMap(map[string]interface{}{
		"Place": "Mars",
	})
	if err == nil {
		t.Error("Expected error when missing required variable")
	}
}

func TestTemplateSimpleExecution(t *testing.T) {
	reg := NewRegistry()

	tmpl := &Template{
		Name:        "simple",
		Description: "Simple template",
		Category:    "testing",
		content:     "Processing: {{.Input}}",
		Variables: []Variable{
			{Name: "Input", Description: "Input text", Required: true},
		},
	}

	err := reg.Register(tmpl)
	if err != nil {
		t.Fatalf("Failed to register template: %v", err)
	}

	result, err := tmpl.ExecuteSimple("test input")
	if err != nil {
		t.Errorf("Failed to execute simple template: %v", err)
	}
	expected := "Processing: test input"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestParseTemplate(t *testing.T) {
	content := `{{/* @description: Test template with metadata */}}
{{/* @category: testing */}}
{{/* @var: Name (required) - User's name */}}
{{/* @var: Age - User's age [default: 18] */}}

Hello {{.Name}}, you are {{.Age}} years old.`

	tmpl, err := parseTemplate("test", content)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	if tmpl.Description != "Test template with metadata" {
		t.Errorf("Expected description 'Test template with metadata', got '%s'", tmpl.Description)
	}

	if tmpl.Category != "testing" {
		t.Errorf("Expected category 'testing', got '%s'", tmpl.Category)
	}

	if len(tmpl.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(tmpl.Variables))
	}

	// Check first variable
	if tmpl.Variables[0].Name != "Name" {
		t.Errorf("Expected first variable name 'Name', got '%s'", tmpl.Variables[0].Name)
	}
	if !tmpl.Variables[0].Required {
		t.Error("Expected first variable to be required")
	}

	// Check second variable
	if tmpl.Variables[1].Name != "Age" {
		t.Errorf("Expected second variable name 'Age', got '%s'", tmpl.Variables[1].Name)
	}
	if tmpl.Variables[1].Required {
		t.Error("Expected second variable to not be required")
	}
	if tmpl.Variables[1].Default != "18" {
		t.Errorf("Expected default value '18', got '%s'", tmpl.Variables[1].Default)
	}

	// Check that metadata is stripped from content
	if strings.Contains(tmpl.content, "@description") {
		t.Error("Template content should not contain metadata")
	}
}

func TestCustomFunctions(t *testing.T) {
	reg := NewRegistry()

	// Add a custom function
	reg.AddFunc("double", func(n int) int {
		return n * 2
	})

	tmpl := &Template{
		Name:    "custom",
		content: "Double of {{.Number}} is {{double .Number}}",
	}

	err := reg.Register(tmpl)
	if err != nil {
		t.Fatalf("Failed to register template: %v", err)
	}

	result, err := tmpl.Execute(map[string]interface{}{
		"Number": 5,
	})
	if err != nil {
		t.Errorf("Failed to execute template with custom function: %v", err)
	}
	expected := "Double of 5 is 10"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test indent
	text := "line1\nline2\nline3"
	indented := indent(2, text)
	expected := "  line1\n  line2\n  line3"
	if indented != expected {
		t.Errorf("Indent failed: expected '%s', got '%s'", expected, indented)
	}

	// Test dedent
	indentedText := "    line1\n    line2\n      line3"
	dedented := dedent(indentedText)
	expected = "line1\nline2\n  line3"
	if dedented != expected {
		t.Errorf("Dedent failed: expected '%s', got '%s'", expected, dedented)
	}

	// Test wordwrap
	longText := "This is a very long line that should be wrapped at a specific width"
	wrapped := wordwrap(20, longText)
	lines := strings.Split(wrapped, "\n")
	for _, line := range lines {
		if len(line) > 20 {
			t.Errorf("Line exceeds wrap width: '%s'", line)
		}
	}
}

func TestBuiltinFunctions(t *testing.T) {
	reg := NewRegistry()

	tmpl := &Template{
		Name: "builtin-test",
		content: `Original: {{.Text}}
Lower: {{lower .Text}}
Upper: {{upper .Text}}
Title: {{title .Text}}
Trimmed: "{{trim .Spaces}}"
Replaced: {{replace .Text "world" "universe"}}
Contains 'hello': {{contains .Text "hello"}}
Indented:
{{indent 4 .Text}}`,
	}

	err := reg.Register(tmpl)
	if err != nil {
		t.Fatalf("Failed to register template: %v", err)
	}

	result, err := tmpl.Execute(map[string]interface{}{
		"Text":   "hello world",
		"Spaces": "  spaced  ",
	})
	if err != nil {
		t.Errorf("Failed to execute template with builtin functions: %v", err)
	}

	// Check various parts of the result
	if !strings.Contains(result, "Lower: hello world") {
		t.Error("Lower case function failed")
	}
	if !strings.Contains(result, "Upper: HELLO WORLD") {
		t.Error("Upper case function failed")
	}
	if !strings.Contains(result, "Title: Hello World") {
		t.Error("Title case function failed")
	}
	if !strings.Contains(result, "Trimmed: \"spaced\"") {
		t.Error("Trim function failed")
	}
	if !strings.Contains(result, "Replaced: hello universe") {
		t.Error("Replace function failed")
	}
	if !strings.Contains(result, "Contains 'hello': true") {
		t.Error("Contains function failed")
	}
	if !strings.Contains(result, "    hello world") {
		t.Error("Indent function failed")
	}
}

func TestLoadDefaults(t *testing.T) {
	// Create a new registry for testing
	oldRegistry := globalRegistry
	defer func() {
		globalRegistry = oldRegistry
	}()
	globalRegistry = NewRegistry()

	// Load default templates
	err := LoadDefaults()
	if err != nil {
		t.Fatalf("Failed to load default templates: %v", err)
	}

	// Check that templates were loaded
	templates := List()
	if len(templates) == 0 {
		t.Error("No templates loaded")
	}

	// Check for specific templates
	expectedTemplates := []string{
		"commit",
		"codereview",
		"summarize",
		"security_review",
		"unit_test",
		"readme",
		"code_summary",
	}

	for _, name := range expectedTemplates {
		_, err := Get(name)
		if err != nil {
			t.Errorf("Expected template '%s' not found: %v", name, err)
		}
	}

	// Test executing a loaded template
	commitTemplate, err := Get("commit")
	if err != nil {
		t.Fatalf("Failed to get commit template: %v", err)
	}

	result, err := commitTemplate.ExecuteMap(map[string]interface{}{
		"Changes": "Added new feature X",
	})
	if err != nil {
		t.Errorf("Failed to execute commit template: %v", err)
	}

	if !strings.Contains(result, "Added new feature X") {
		t.Error("Commit template did not include changes")
	}
}

func TestRegisterBuiltins(t *testing.T) {
	// Create a new registry for testing
	oldRegistry := globalRegistry
	defer func() {
		globalRegistry = oldRegistry
	}()
	globalRegistry = NewRegistry()

	err := RegisterBuiltins()
	if err != nil {
		t.Fatalf("Failed to register builtins: %v", err)
	}

	// Check that quick templates were registered
	quickTemplateNames := []string{
		"explain",
		"improve",
		"convert",
		"debug",
		"optimize",
		"document",
	}

	for _, name := range quickTemplateNames {
		tmpl, err := Get(name)
		if err != nil {
			t.Errorf("Expected quick template '%s' not found: %v", name, err)
		}
		if tmpl.Category == "" {
			t.Errorf("Template '%s' has no category", name)
		}
	}

	// Test executing a quick template
	explainTemplate, err := Get("explain")
	if err != nil {
		t.Fatalf("Failed to get explain template: %v", err)
	}

	result, err := explainTemplate.ExecuteSimple("Binary search algorithm")
	if err != nil {
		t.Errorf("Failed to execute explain template: %v", err)
	}

	if !strings.Contains(result, "Binary search algorithm") {
		t.Error("Explain template did not include input")
	}
}

func TestParseVariable(t *testing.T) {
	tests := []struct {
		input    string
		expected *Variable
	}{
		{
			input: "Name (required) - User's name",
			expected: &Variable{
				Name:        "Name",
				Description: "User's name",
				Required:    true,
			},
		},
		{
			input: "Age - User's age [default: 18]",
			expected: &Variable{
				Name:        "Age",
				Description: "User's age",
				Required:    false,
				Default:     "18",
			},
		},
		{
			input: "Count - Number of items",
			expected: &Variable{
				Name:        "Count",
				Description: "Number of items",
				Required:    false,
			},
		},
		{
			input:    "Invalid format",
			expected: nil,
		},
	}

	for _, test := range tests {
		result := parseVariable(test.input)
		if test.expected == nil {
			if result != nil {
				t.Errorf("Expected nil for input '%s', got %+v", test.input, result)
			}
			continue
		}
		if result == nil {
			t.Errorf("Expected variable for input '%s', got nil", test.input)
			continue
		}
		if result.Name != test.expected.Name {
			t.Errorf("Expected name '%s', got '%s'", test.expected.Name, result.Name)
		}
		if result.Description != test.expected.Description {
			t.Errorf("Expected description '%s', got '%s'", test.expected.Description, result.Description)
		}
		if result.Required != test.expected.Required {
			t.Errorf("Expected required %v, got %v", test.expected.Required, result.Required)
		}
		if result.Default != test.expected.Default {
			t.Errorf("Expected default '%s', got '%s'", test.expected.Default, result.Default)
		}
	}
}
