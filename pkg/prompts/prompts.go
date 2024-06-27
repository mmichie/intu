package prompts

import (
	"embed"
	"fmt"
	"log"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type Prompt struct {
	Name        string
	Description string
	Template    *template.Template
}

func (p Prompt) Format(input string) (string, error) {
	var data map[string]string
	switch p.Name {
	case "commit":
		data = map[string]string{"Changes": input}
	case "summarize":
		data = map[string]string{"TextToSummarize": input}
	case "readme":
		data = map[string]string{"Code": input}
	case "codereview":
		data = map[string]string{"CodeToReview": input}
	default:
		return "", fmt.Errorf("unknown prompt type: %s", p.Name)
	}

	var result strings.Builder
	err := p.Template.Execute(&result, data)
	if err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}
	return result.String(), nil
}

func loadTemplate(name string) (*template.Template, error) {
	tmplContent, err := templateFS.ReadFile(fmt.Sprintf("templates/%s.tmpl", name))
	if err != nil {
		log.Printf("Error reading template file %s: %v", name, err)
		return nil, err
	}

	tmpl, err := template.New(name).Parse(string(tmplContent))
	if err != nil {
		log.Printf("Error parsing template %s: %v", name, err)
		return nil, err
	}
	return tmpl, nil
}

var (
	Commit, _     = loadTemplate("commit")
	Summarize, _  = loadTemplate("summarize")
	Readme, _     = loadTemplate("readme")
	CodeReview, _ = loadTemplate("codereview")

	AllPrompts = []Prompt{
		{Name: "commit", Description: "Generate a git commit message", Template: Commit},
		{Name: "summarize", Description: "Summarize the given text", Template: Summarize},
		{Name: "readme", Description: "Generate a README from code", Template: Readme},
		{Name: "codereview", Description: "Generate code review comments", Template: CodeReview},
	}
)

func GetPrompt(name string) (Prompt, bool) {
	for _, p := range AllPrompts {
		if p.Name == name {
			return p, true
		}
	}
	return Prompt{}, false
}
