package prompts

import "fmt"

type Prompt struct {
	Name        string
	Description string
	Template    string
}

func (p Prompt) Format(input string) string {
	return fmt.Sprintf(p.Template, input)
}

var (
	Commit = Prompt{
		Name:        "commit",
		Description: "Generate a git commit message",
		Template: `Generate a concise git commit message in conventional style for the following diff:

%s

Provide a short summary in the first line, followed by a blank line and a more detailed description using bullet points.
Optimize for a FAANG engineer experienced with the code. Keep line width to about 79 characters.`,
	}

	// Add more pre-canned prompts here
	Summarize = Prompt{
		Name:        "summarize",
		Description: "Summarize the given text",
		Template: `Provide a concise summary of the following text:

%s

The summary should capture the main points and be no longer than 3-4 sentences.`,
	}

	Readme = Prompt{
		Name:        "readme",
		Description: "Generate a README from code",
		Template: `Generate a README.md file for a project based on the following code:

%s

Include sections for:
1. Project Title
2. Brief Description
3. Installation
4. Usage
5. Main Features
6. Dependencies (if any can be inferred from the code)

Use markdown formatting and keep it concise but informative.`,
	}
)

var AllPrompts = []Prompt{Commit, Summarize, Readme}

func GetPrompt(name string) (Prompt, bool) {
	for _, p := range AllPrompts {
		if p.Name == name {
			return p, true
		}
	}
	return Prompt{}, false
}
