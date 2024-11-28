package commands

import (
	"github.com/spf13/cobra"
)

var (
	aiCmd = &cobra.Command{
		Use:   "ai [prompt-name]",
		Short: "AI-powered commands",
		Long:  `AI-powered commands for various tasks.`,
		RunE:  runAICommand,
	}

	askCmd = &cobra.Command{
		Use:   "ask <prompt> [input]",
		Short: "Ask the AI a free-form question",
		Long:  `Ask the AI a free-form question or provide a custom prompt. Input can be provided via stdin or as an argument.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runAskCommand,
	}

	tuiCmd = &cobra.Command{
		Use:   "tui",
		Short: "Start the Text User Interface",
		Long:  `Start an interactive Text User Interface for communicating with the AI assistant.`,
		RunE:  runTUICommand,
	}

	codeReviewCmd = &cobra.Command{
		Use:   "codereview [file]",
		Short: "Generate a code review for a given file or stdin input",
		Long:  `Analyze the provided code file or stdin input and generate constructive code review comments.`,
		RunE:  runCodeReviewCommand,
	}

	commitCmd = &cobra.Command{
		Use:   "commit",
		Short: "Generate a commit message",
		Long:  `Generate a commit message based on the git diff output.`,
		RunE:  runCommitCommand,
	}

	securityReviewCmd = &cobra.Command{
		Use:   "securityreview [file]",
		Short: "Perform a security review for a given file or stdin input",
		Long:  `Analyze the provided code file or stdin input and generate a comprehensive security review.`,
		RunE:  runSecurityReviewCommand,
	}
)
