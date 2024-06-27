package prompts

import (
	"strings"
)

type Prompt struct {
	Name        string
	Description string
	Template    string
}

func (p Prompt) Format(input string) string {
	return strings.Replace(p.Template, "{{CHANGES}}", input, 1)
}

var (
	Commit = Prompt{
		Name:        "commit",
		Description: "Generate a git commit message",
		Template: `You are tasked with writing a git commit message using the conventional style. Conventional commit messages have a specific structure that includes a type, an optional scope, and a description. The format is as follows: <type>[optional scope]: <description>
Here are the changes made in this commit:
<changes>
{{CHANGES}}
</changes>
Analyze the changes provided above. Determine the primary purpose of these changes (e.g., fixing a bug, adding a feature, refactoring code, etc.).
Based on your analysis, select the most appropriate type prefix from the following list:
- feat: A new feature
- fix: A bug fix
- docs: Documentation only changes
- style: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- refactor: A code change that neither fixes a bug nor adds a feature
- perf: A code change that improves performance
- test: Adding missing tests or correcting existing tests
- build: Changes that affect the build system or external dependencies
- ci: Changes to our CI configuration files and scripts
- chore: Other changes that don't modify src or test files
Next, write a concise description (50 characters or less) that summarizes the change. The description should:
- Use the imperative mood ("Add feature" not "Added feature" or "Adds feature")
- Not capitalize the first letter
- Not end with a period
Your commit message should follow this format:
<type>: <description>
Now, write the commit message for the changes provided, following the conventional style and format described above. Place your commit message inside <commit_message> tags.`,
	}

	// Keep the existing Summarize and Readme prompts
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
