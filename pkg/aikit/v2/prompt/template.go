// Package prompt provides a template system for AI prompts
package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	aierrors "github.com/mmichie/intu/pkg/aikit/v2/errors"
)

//go:embed templates/*.tmpl
var defaultTemplateFS embed.FS

// Template represents a prompt template
type Template struct {
	Name        string
	Description string
	Category    string
	Variables   []Variable
	template    *template.Template
	content     string
}

// Variable describes a template variable
type Variable struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

// Registry manages prompt templates
type Registry struct {
	mu        sync.RWMutex
	templates map[string]*Template
	funcs     template.FuncMap
}

// NewRegistry creates a new template registry
func NewRegistry() *Registry {
	return &Registry{
		templates: make(map[string]*Template),
		funcs:     defaultFuncs(),
	}
}

// defaultFuncs returns the default template functions
func defaultFuncs() template.FuncMap {
	return template.FuncMap{
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"title":     strings.Title,
		"trim":      strings.TrimSpace,
		"replace":   strings.ReplaceAll,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"repeat":    strings.Repeat,
		"join":      strings.Join,
		"split":     strings.Split,
		"indent":    indent,
		"dedent":    dedent,
		"wordwrap":  wordwrap,
	}
}

// Register adds a template to the registry
func (r *Registry) Register(tmpl *Template) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tmpl.Name == "" {
		return aierrors.New("prompt", "register",
			fmt.Errorf("template name cannot be empty"))
	}

	if _, exists := r.templates[tmpl.Name]; exists {
		return aierrors.New("prompt", "register",
			fmt.Errorf("template %q already registered", tmpl.Name))
	}

	// Parse the template with custom functions
	parsedTmpl, err := template.New(tmpl.Name).Funcs(r.funcs).Parse(tmpl.content)
	if err != nil {
		return aierrors.New("prompt", "register",
			fmt.Errorf("failed to parse template %q: %w", tmpl.Name, err))
	}

	tmpl.template = parsedTmpl
	r.templates[tmpl.Name] = tmpl

	return nil
}

// Get retrieves a template by name
func (r *Registry) Get(name string) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tmpl, exists := r.templates[name]
	if !exists {
		return nil, aierrors.New("prompt", "get",
			fmt.Errorf("template %q not found", name))
	}

	return tmpl, nil
}

// List returns all registered templates
func (r *Registry) List() []*Template {
	r.mu.RLock()
	defer r.mu.RUnlock()

	templates := make([]*Template, 0, len(r.templates))
	for _, tmpl := range r.templates {
		templates = append(templates, tmpl)
	}

	return templates
}

// ListByCategory returns templates in a specific category
func (r *Registry) ListByCategory(category string) []*Template {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var templates []*Template
	for _, tmpl := range r.templates {
		if tmpl.Category == category {
			templates = append(templates, tmpl)
		}
	}

	return templates
}

// AddFunc adds a custom template function
func (r *Registry) AddFunc(name string, fn interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.funcs[name] = fn
}

// LoadFromFS loads templates from a filesystem
func (r *Registry) LoadFromFS(fsys fs.FS, pattern string) error {
	matches, err := fs.Glob(fsys, pattern)
	if err != nil {
		return aierrors.New("prompt", "load_from_fs",
			fmt.Errorf("failed to glob pattern %q: %w", pattern, err))
	}

	for _, match := range matches {
		content, err := fs.ReadFile(fsys, match)
		if err != nil {
			return aierrors.New("prompt", "load_from_fs",
				fmt.Errorf("failed to read file %q: %w", match, err))
		}

		// Extract name from filename
		name := strings.TrimSuffix(filepath.Base(match), filepath.Ext(match))

		// Parse template metadata from content
		tmpl, err := parseTemplate(name, string(content))
		if err != nil {
			return aierrors.New("prompt", "load_from_fs",
				fmt.Errorf("failed to parse template %q: %w", name, err))
		}

		if err := r.Register(tmpl); err != nil {
			return err
		}
	}

	return nil
}

// Execute renders a template with the given data
func (t *Template) Execute(data interface{}) (string, error) {
	if t.template == nil {
		return "", aierrors.New("prompt", "execute",
			fmt.Errorf("template %q not parsed", t.Name))
	}

	var buf bytes.Buffer
	err := t.template.Execute(&buf, data)
	if err != nil {
		return "", aierrors.New("prompt", "execute",
			fmt.Errorf("failed to execute template %q: %w", t.Name, err))
	}

	return buf.String(), nil
}

// ExecuteMap executes the template with a map of values
func (t *Template) ExecuteMap(values map[string]interface{}) (string, error) {
	// Apply defaults for missing required variables
	data := make(map[string]interface{})
	for _, v := range t.Variables {
		if val, ok := values[v.Name]; ok {
			data[v.Name] = val
		} else if v.Default != "" {
			data[v.Name] = v.Default
		} else if v.Required {
			return "", aierrors.New("prompt", "execute_map",
				fmt.Errorf("missing required variable %q", v.Name))
		}
	}

	// Copy any extra values
	for k, v := range values {
		if _, exists := data[k]; !exists {
			data[k] = v
		}
	}

	return t.Execute(data)
}

// ExecuteSimple executes a template with a single "Input" variable
func (t *Template) ExecuteSimple(input string) (string, error) {
	return t.ExecuteMap(map[string]interface{}{
		"Input": input,
	})
}

// parseTemplate parses template content and extracts metadata
func parseTemplate(name, content string) (*Template, error) {
	// Default template
	tmpl := &Template{
		Name:        name,
		Description: fmt.Sprintf("Template for %s", name),
		Category:    "general",
		content:     content,
		Variables:   []Variable{},
	}

	// Look for metadata in comments at the beginning
	lines := strings.Split(content, "\n")
	contentStart := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{{/*") && strings.HasSuffix(line, "*/}}") {
			// Parse metadata comment
			meta := strings.TrimSpace(line[4 : len(line)-4])
			if strings.HasPrefix(meta, "@description:") {
				tmpl.Description = strings.TrimSpace(meta[13:])
			} else if strings.HasPrefix(meta, "@category:") {
				tmpl.Category = strings.TrimSpace(meta[10:])
			} else if strings.HasPrefix(meta, "@var:") {
				// Parse variable definition: @var: name (required) - description
				varDef := strings.TrimSpace(meta[5:])
				if v := parseVariable(varDef); v != nil {
					tmpl.Variables = append(tmpl.Variables, *v)
				}
			}
			contentStart = i + 1
		} else if line != "" && !strings.HasPrefix(line, "{{/*") {
			// End of metadata section
			break
		}
	}

	// Update content to exclude metadata
	if contentStart > 0 && contentStart < len(lines) {
		tmpl.content = strings.Join(lines[contentStart:], "\n")
	}

	return tmpl, nil
}

// parseVariable parses a variable definition
func parseVariable(def string) *Variable {
	// Format: name (required) - description
	// or: name - description
	parts := strings.SplitN(def, " - ", 2)
	if len(parts) != 2 {
		return nil
	}

	namePart := strings.TrimSpace(parts[0])
	description := strings.TrimSpace(parts[1])

	required := false
	name := namePart

	// Check for (required) marker
	if strings.HasSuffix(namePart, "(required)") {
		required = true
		name = strings.TrimSpace(strings.TrimSuffix(namePart, "(required)"))
	}

	// Check for default value in description
	defaultVal := ""
	if idx := strings.Index(description, "[default:"); idx >= 0 {
		endIdx := strings.Index(description[idx:], "]")
		if endIdx > 0 {
			defaultVal = strings.TrimSpace(description[idx+9 : idx+endIdx])
			description = strings.TrimSpace(description[:idx] + description[idx+endIdx+1:])
		}
	}

	return &Variable{
		Name:        name,
		Description: description,
		Required:    required,
		Default:     defaultVal,
	}
}

// Template helper functions

func indent(spaces int, text string) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

func dedent(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}

	// Find minimum indentation
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return text
	}

	// Remove the minimum indentation from all lines
	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}

	return strings.Join(lines, "\n")
}

func wordwrap(width int, text string) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if len(line) <= width {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		words := strings.Fields(line)
		currentLine := ""

		for _, word := range words {
			if currentLine == "" {
				currentLine = word
			} else if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				result.WriteString(currentLine)
				result.WriteString("\n")
				currentLine = word
			}
		}

		if currentLine != "" {
			result.WriteString(currentLine)
			result.WriteString("\n")
		}
	}

	return strings.TrimSuffix(result.String(), "\n")
}

// Global registry instance
var globalRegistry = NewRegistry()

// Register adds a template to the global registry
func Register(tmpl *Template) error {
	return globalRegistry.Register(tmpl)
}

// Get retrieves a template from the global registry
func Get(name string) (*Template, error) {
	return globalRegistry.Get(name)
}

// List returns all templates from the global registry
func List() []*Template {
	return globalRegistry.List()
}

// ListByCategory returns templates by category from the global registry
func ListByCategory(category string) []*Template {
	return globalRegistry.ListByCategory(category)
}

// LoadDefaults loads the default templates
func LoadDefaults() error {
	return globalRegistry.LoadFromFS(defaultTemplateFS, "templates/*.tmpl")
}
