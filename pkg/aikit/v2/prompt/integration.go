package prompt

import (
	"context"
	"fmt"

	"github.com/mmichie/intu/pkg/aikit/v2/agent"
)

// Note: WithTemplate would be implemented in the agent package
// This is just a documentation example of how it could work
//
// Example usage:
//   response, err := agent.Process(ctx, "",
//     prompt.WithTemplate("commit", map[string]interface{}{
//       "Changes": gitDiff,
//     }),
//   )

// ExecuteTemplate executes a template and processes it with an agent
func ExecuteTemplate(ctx context.Context, a *agent.Agent, templateName string, values map[string]interface{}) (string, error) {
	// Get the template
	tmpl, err := Get(templateName)
	if err != nil {
		return "", fmt.Errorf("template not found: %w", err)
	}

	// Execute the template
	prompt, err := tmpl.ExecuteMap(values)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Process with the agent
	return a.Process(ctx, prompt)
}

// ExecuteTemplateStreaming executes a template and streams the response
func ExecuteTemplateStreaming(ctx context.Context, a *agent.Agent, templateName string, values map[string]interface{}, handler agent.StreamHandler) error {
	// Get the template
	tmpl, err := Get(templateName)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Execute the template
	prompt, err := tmpl.ExecuteMap(values)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Process with streaming
	return a.ProcessStreaming(ctx, prompt, handler)
}

// TemplateExecutor provides a convenient interface for template execution
type TemplateExecutor struct {
	agent    *agent.Agent
	registry *Registry
}

// NewTemplateExecutor creates a new template executor
func NewTemplateExecutor(a *agent.Agent) *TemplateExecutor {
	return &TemplateExecutor{
		agent:    a,
		registry: globalRegistry,
	}
}

// NewTemplateExecutorWithRegistry creates a new template executor with a custom registry
func NewTemplateExecutorWithRegistry(a *agent.Agent, r *Registry) *TemplateExecutor {
	return &TemplateExecutor{
		agent:    a,
		registry: r,
	}
}

// Execute runs a template with the given values
func (te *TemplateExecutor) Execute(ctx context.Context, templateName string, values map[string]interface{}) (string, error) {
	tmpl, err := te.registry.Get(templateName)
	if err != nil {
		return "", err
	}

	prompt, err := tmpl.ExecuteMap(values)
	if err != nil {
		return "", err
	}

	return te.agent.Process(ctx, prompt)
}

// ExecuteSimple runs a template with a single input value
func (te *TemplateExecutor) ExecuteSimple(ctx context.Context, templateName string, input string) (string, error) {
	tmpl, err := te.registry.Get(templateName)
	if err != nil {
		return "", err
	}

	prompt, err := tmpl.ExecuteSimple(input)
	if err != nil {
		return "", err
	}

	return te.agent.Process(ctx, prompt)
}

// Common template execution helpers

// GenerateCommitMessage generates a commit message from changes
func GenerateCommitMessage(ctx context.Context, a *agent.Agent, changes string) (string, error) {
	return ExecuteTemplate(ctx, a, BuiltinTemplates.Commit, map[string]interface{}{
		"Changes": changes,
	})
}

// ReviewCode performs a code review
func ReviewCode(ctx context.Context, a *agent.Agent, code string) (string, error) {
	return ExecuteTemplate(ctx, a, BuiltinTemplates.CodeReview, map[string]interface{}{
		"CodeToReview": code,
	})
}

// ReviewSecurity performs a security review
func ReviewSecurity(ctx context.Context, a *agent.Agent, code string, language string) (string, error) {
	return ExecuteTemplate(ctx, a, BuiltinTemplates.SecurityReview, map[string]interface{}{
		"CodeToReview": code,
		"Language":     language,
	})
}

// GenerateUnitTests generates unit tests for code
func GenerateUnitTests(ctx context.Context, a *agent.Agent, code string, language string, framework string) (string, error) {
	values := map[string]interface{}{
		"Code":     code,
		"Language": language,
	}
	if framework != "" {
		values["Framework"] = framework
	}
	return ExecuteTemplate(ctx, a, BuiltinTemplates.UnitTest, values)
}

// GenerateReadme generates a README file
func GenerateReadme(ctx context.Context, a *agent.Agent, projectName, description string, opts ...ReadmeOption) (string, error) {
	values := map[string]interface{}{
		"ProjectName": projectName,
		"Description": description,
	}

	// Apply options
	for _, opt := range opts {
		opt(values)
	}

	return ExecuteTemplate(ctx, a, BuiltinTemplates.Readme, values)
}

// ReadmeOption is an option for README generation
type ReadmeOption func(map[string]interface{})

// WithCode adds code context to README generation
func WithCode(code string) ReadmeOption {
	return func(values map[string]interface{}) {
		values["Code"] = code
	}
}

// WithLanguage specifies the primary language
func WithLanguage(language string) ReadmeOption {
	return func(values map[string]interface{}) {
		values["Language"] = language
	}
}

// WithLicense specifies the license
func WithLicense(license string) ReadmeOption {
	return func(values map[string]interface{}) {
		values["License"] = license
	}
}

// SummarizeText summarizes text content
func SummarizeText(ctx context.Context, a *agent.Agent, text string, opts ...SummarizeOption) (string, error) {
	values := map[string]interface{}{
		"TextToSummarize": text,
	}

	// Apply options
	for _, opt := range opts {
		opt(values)
	}

	return ExecuteTemplate(ctx, a, BuiltinTemplates.Summarize, values)
}

// SummarizeOption is an option for text summarization
type SummarizeOption func(map[string]interface{})

// WithMaxLength sets the maximum summary length
func WithMaxLength(sentences int) SummarizeOption {
	return func(values map[string]interface{}) {
		values["MaxLength"] = sentences
	}
}

// WithStyle sets the summary style
func WithStyle(style string) SummarizeOption {
	return func(values map[string]interface{}) {
		values["Style"] = style
	}
}

// AnalyzeCode provides a detailed code analysis
func AnalyzeCode(ctx context.Context, a *agent.Agent, code string, detailed bool) (string, error) {
	level := "detailed"
	if !detailed {
		level = "brief"
	}
	return ExecuteTemplate(ctx, a, BuiltinTemplates.CodeSummary, map[string]interface{}{
		"Code":  code,
		"Level": level,
	})
}
