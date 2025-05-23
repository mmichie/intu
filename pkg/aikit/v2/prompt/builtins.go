package prompt

// BuiltinTemplates provides convenient access to commonly used templates
var BuiltinTemplates = struct {
	Commit         string
	CodeReview     string
	SecurityReview string
	UnitTest       string
	Readme         string
	Summarize      string
	CodeSummary    string
}{
	Commit:         "commit",
	CodeReview:     "codereview",
	SecurityReview: "security_review",
	UnitTest:       "unit_test",
	Readme:         "readme",
	Summarize:      "summarize",
	CodeSummary:    "code_summary",
}

// Categories defines template categories
var Categories = struct {
	Development   string
	Documentation string
	Security      string
	General       string
}{
	Development:   "development",
	Documentation: "documentation",
	Security:      "security",
	General:       "general",
}

// QuickTemplates provides simple, inline templates for common tasks
var QuickTemplates = map[string]*Template{
	"explain": {
		Name:        "explain",
		Description: "Explain code or concept in simple terms",
		Category:    Categories.General,
		Variables: []Variable{
			{Name: "Input", Description: "Code or concept to explain", Required: true},
		},
		content: `Explain the following in simple, clear terms:

{{.Input}}

Provide a clear explanation that:
- Uses simple language
- Includes examples if helpful
- Breaks down complex concepts
- Highlights key points`,
	},
	"improve": {
		Name:        "improve",
		Description: "Suggest improvements for code",
		Category:    Categories.Development,
		Variables: []Variable{
			{Name: "Code", Description: "Code to improve", Required: true},
		},
		content: `Review the following code and suggest improvements:

{{.Code}}

Focus on:
- Code clarity and readability
- Performance optimizations
- Best practices
- Error handling
- Documentation

Provide specific, actionable suggestions.`,
	},
	"convert": {
		Name:        "convert",
		Description: "Convert code between languages or formats",
		Category:    Categories.Development,
		Variables: []Variable{
			{Name: "Code", Description: "Code to convert", Required: true},
			{Name: "From", Description: "Source language/format", Required: true},
			{Name: "To", Description: "Target language/format", Required: true},
		},
		content: `Convert the following {{.From}} code to {{.To}}:

{{.Code}}

Requirements:
- Maintain the same functionality
- Use idiomatic {{.To}} patterns
- Include necessary imports/dependencies
- Add appropriate comments
- Handle errors properly in the target language`,
	},
	"debug": {
		Name:        "debug",
		Description: "Help debug code issues",
		Category:    Categories.Development,
		Variables: []Variable{
			{Name: "Code", Description: "Code with issues", Required: true},
			{Name: "Error", Description: "Error message or behavior", Required: false},
			{Name: "Expected", Description: "Expected behavior", Required: false},
		},
		content: `Help debug the following code:

{{.Code}}

{{if .Error}}Error/Issue: {{.Error}}{{end}}
{{if .Expected}}Expected behavior: {{.Expected}}{{end}}

Please:
1. Identify potential issues
2. Explain why the error occurs
3. Provide a corrected version
4. Suggest debugging strategies`,
	},
	"optimize": {
		Name:        "optimize",
		Description: "Optimize code for performance",
		Category:    Categories.Development,
		Variables: []Variable{
			{Name: "Code", Description: "Code to optimize", Required: true},
			{Name: "Metric", Description: "Optimization target (speed/memory/readability)", Default: "speed"},
		},
		content: `Optimize the following code for {{.Metric}}:

{{.Code}}

Provide:
1. Analysis of current performance characteristics
2. Identified bottlenecks or issues
3. Optimized version of the code
4. Explanation of optimizations made
5. Trade-offs of the optimization`,
	},
	"document": {
		Name:        "document",
		Description: "Generate documentation for code",
		Category:    Categories.Documentation,
		Variables: []Variable{
			{Name: "Code", Description: "Code to document", Required: true},
			{Name: "Style", Description: "Documentation style", Default: "inline"},
		},
		content: `Generate comprehensive documentation for the following code:

{{.Code}}

Documentation style: {{.Style}}

Include:
- Purpose and overview
- Function/method descriptions
- Parameter explanations
- Return value descriptions
- Usage examples
- Any important notes or warnings`,
	},
}

// RegisterBuiltins registers all built-in quick templates
func RegisterBuiltins() error {
	for _, tmpl := range QuickTemplates {
		if err := Register(tmpl); err != nil {
			return err
		}
	}
	return nil
}

// MustLoadDefaults loads default templates and panics on error
func MustLoadDefaults() {
	if err := LoadDefaults(); err != nil {
		panic("failed to load default templates: " + err.Error())
	}
	if err := RegisterBuiltins(); err != nil {
		panic("failed to register builtin templates: " + err.Error())
	}
}
