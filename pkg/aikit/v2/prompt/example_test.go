package prompt_test

import (
	"context"
	"fmt"
	"log"

	"github.com/mmichie/intu/pkg/aikit/v2/agent"
	"github.com/mmichie/intu/pkg/aikit/v2/config"
	"github.com/mmichie/intu/pkg/aikit/v2/prompt"
	"github.com/mmichie/intu/pkg/aikit/v2/provider"
)

func ExampleTemplate() {
	// Load default templates
	if err := prompt.LoadDefaults(); err != nil {
		log.Fatal(err)
	}

	// Get a template
	tmpl, err := prompt.Get("commit")
	if err != nil {
		log.Fatal(err)
	}

	// Execute the template
	result, err := tmpl.ExecuteMap(map[string]interface{}{
		"Changes": "Added user authentication feature",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Generated prompt:", result[:50]+"...")
}

func ExampleRegistry() {
	// Create a custom registry
	registry := prompt.NewRegistry()

	// Register a custom template
	customTemplate := &prompt.Template{
		Name:        "bug_report",
		Description: "Generate a bug report",
		Category:    "development",
		Variables: []prompt.Variable{
			{Name: "Title", Description: "Bug title", Required: true},
			{Name: "Description", Description: "Bug description", Required: true},
			{Name: "Steps", Description: "Steps to reproduce", Required: false},
			{Name: "Expected", Description: "Expected behavior", Required: false},
			{Name: "Actual", Description: "Actual behavior", Required: false},
		},
		// Note: content should be set, but we're keeping this example simple
	}

	if err := registry.Register(customTemplate); err != nil {
		log.Fatal(err)
	}

	// List templates by category
	devTemplates := registry.ListByCategory("development")
	fmt.Printf("Development templates: %d\n", len(devTemplates))
}

func ExampleTemplateExecutor() {
	// This is a conceptual example - in real use you would have a real provider
	// Create an agent
	cfg := config.Config{APIKey: "test-key"}
	p, _ := provider.Create("openai", cfg)
	a := agent.New(p)

	// Create a template executor
	executor := prompt.NewTemplateExecutor(a)

	// Use it to execute templates
	ctx := context.Background()

	// Execute with simple input
	_, _ = executor.ExecuteSimple(ctx, "explain", "Binary search algorithm")

	// Execute with multiple values
	_, _ = executor.Execute(ctx, "convert", map[string]interface{}{
		"Code": "def hello(): print('Hello')",
		"From": "Python",
		"To":   "Go",
	})
}

func ExampleQuickTemplates() {
	// Register built-in quick templates
	if err := prompt.RegisterBuiltins(); err != nil {
		log.Fatal(err)
	}

	// Use a quick template
	tmpl, err := prompt.Get("explain")
	if err != nil {
		log.Fatal(err)
	}

	prompt, err := tmpl.ExecuteSimple("Recursion in programming")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Template variables:", len(tmpl.Variables))
	fmt.Println("Template category:", tmpl.Category)
	fmt.Println("Generated prompt preview:", prompt[:60]+"...")
}

func Example_workflowIntegration() {
	// Load all templates
	prompt.MustLoadDefaults()

	// Example: Code review workflow
	ctx := context.Background()

	// Assuming we have an agent configured
	cfg := config.Config{APIKey: "your-api-key"}
	p, _ := provider.Create("claude", cfg)
	a := agent.New(p)

	// Sample code to review
	code := `
func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
}
`

	// Perform code review
	_, _ = prompt.ReviewCode(ctx, a, code)

	// Generate unit tests
	_, _ = prompt.GenerateUnitTests(ctx, a, code, "go", "testing")

	// Analyze code structure
	_, _ = prompt.AnalyzeCode(ctx, a, code, true)

	fmt.Println("Workflow completed")
}
