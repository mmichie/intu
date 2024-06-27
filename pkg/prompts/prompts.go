package prompts

import (
	"strings"
)

type Prompt struct {
	Name        string
	Description string
	Template    string
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

	Summarize = Prompt{
		Name:        "summarize",
		Description: "Summarize the given text",
		Template: `You are tasked with summarizing a given text. Here is the text you need to summarize:
<text_to_summarize>
{{TEXT_TO_SUMMARIZE}}
</text_to_summarize>
Please provide a concise summary of the above text. Your summary should:
1. Capture the main points and key ideas of the original text
2. Be no longer than 3-4 sentences
3. Be written in your own words (do not copy sentences directly from the original text)
4. Maintain the overall tone and intent of the original text
Focus on the most important information and central themes. Avoid including minor details or examples unless they are crucial to understanding the main points.
Present your final summary within <summary> tags. Before writing your summary, you may use <scratchpad> tags to organize your thoughts if needed, but this is optional for simpler texts.
Remember, the goal is to create a clear, concise, and accurate representation of the original text that can be quickly understood by a reader.`,
	}

	Readme = Prompt{
		Name:        "readme",
		Description: "Generate a README from code",
		Template: `You are tasked with generating a README.md file for a project based on the provided code. Here's the code you'll be working with:
<code>
{{CODE}}
</code>
Your goal is to create a comprehensive yet concise README.md file that effectively communicates the project's purpose, functionality, and usage. Follow these instructions carefully:
1. Analyze the provided code to understand the project's main features, functionality, and purpose.
2. Create a README.md file with the following sections:
   a. Project Title
   b. Brief Description
   c. Installation
   d. Usage
   e. Main Features
   f. Dependencies (if any can be inferred from the code)
3. Use proper markdown formatting throughout the README. This includes:
   - Using # for the main title
   - Using ## for section headers
   - Using ''' for code blocks
   - Using * or - for bullet points
   - Using ** for bold text when emphasizing important points
4. Keep the content informative but concise. Aim for clarity and brevity.
5. For the Project Title, use the name of the main function or class if apparent, or create a descriptive title based on the code's functionality.
6. In the Brief Description, summarize the main purpose and functionality of the code in 2-3 sentences.
7. For the Installation section, provide basic instructions on how to set up the project. If no specific installation steps are evident, you can include a generic instruction like "Clone the repository and install the required dependencies."
8. In the Usage section, provide examples of how to use the main functions or classes in the code. Use code blocks to illustrate these examples.
9. List the Main Features of the project based on the functions and classes you identify in the code.
10. If you can infer any dependencies from the code (e.g., imported libraries), list them in the Dependencies section. If no dependencies are apparent, you can omit this section.
11. If the purpose of certain parts of the code is not clear, it's okay to make reasonable assumptions, but avoid speculating too much.
12. After analyzing the code and preparing your README content, present your complete README.md file within <readme> tags.
Remember, the goal is to create a README that would be helpful for someone encountering this project for the first time. Focus on clarity, conciseness, and providing useful information based on the available code.`,
	}

	CodeReview = Prompt{
		Name:        "codereview",
		Description: "Generate code review comments",
		Template: `You are an experienced software developer tasked with reviewing the following code:

<code_to_review>
{{CODE_TO_REVIEW}}
</code_to_review>

Please provide a thorough code review, considering the following aspects:
1. Code quality and readability
2. Potential bugs or errors
3. Performance considerations
4. Adherence to best practices and design principles
5. Suggestions for improvement

For each issue or suggestion, please:
1. Specify the line number or code snippet in question
2. Explain the issue or suggestion clearly
3. Provide a recommendation for improvement, if applicable

Your review should be constructive and aimed at improving the code. Be specific in your feedback and explain the reasoning behind your suggestions.

Present your code review comments within <review_comments> tags. You may use markdown formatting for better readability.

Remember, the goal is to provide valuable feedback that will help improve the quality of the code and the skills of the developer.`,
	}
)

var AllPrompts = []Prompt{Commit, Summarize, Readme, CodeReview}

func GetPrompt(name string) (Prompt, bool) {
	for _, p := range AllPrompts {
		if p.Name == name {
			return p, true
		}
	}
	return Prompt{}, false
}

// Update the Format method to include the new prompt
func (p Prompt) Format(input string) string {
	switch p.Name {
	case "commit":
		return strings.Replace(p.Template, "{{CHANGES}}", input, 1)
	case "summarize":
		return strings.Replace(p.Template, "{{TEXT_TO_SUMMARIZE}}", input, 1)
	case "readme":
		return strings.Replace(p.Template, "{{CODE}}", input, 1)
	case "codereview":
		return strings.Replace(p.Template, "{{CODE_TO_REVIEW}}", input, 1)
	default:
		return p.Template
	}
}
