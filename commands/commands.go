// Package commands provides all the CLI commands for the application
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

	lsCmd = RegisterLSCommand()

	modelsCmd = &cobra.Command{
		Use:   "models",
		Short: "List available AI models",
		Long:  `List all available AI models by provider.`,
		RunE:  runModelsCommand,
	}

	askCmd = &cobra.Command{
		Use:   "ask <prompt> [input]",
		Short: "Ask the AI a free-form question",
		Long:  `Ask the AI a free-form question or provide a custom prompt. Input can be provided via stdin or as an argument.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runAskCommand,
	}

	juryCmd = &cobra.Command{
		Use:   "jury <prompt> [input]",
		Short: "Use an AI jury to evaluate responses",
		Long:  `Send a prompt to multiple AI providers and use a jury to evaluate and select the best response.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runJuryCommand,
	}

	collabCmd = &cobra.Command{
		Use:   "collab <prompt> [input]",
		Short: "Collaborative discussion between AI providers",
		Long:  `Start a multi-round collaborative discussion between AI providers to solve a problem or answer a question.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runCollabCommand,
	}

	pipelineCmd = &cobra.Command{
		Use:   "pipeline <name> <prompt> [input]",
		Short: "Run a predefined pipeline",
		Long:  `Run a predefined pipeline configuration with the given prompt. Pipelines can combine multiple providers and processing steps.`,
		Args:  cobra.MinimumNArgs(2),
		RunE:  runPipelineCommand,
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

func init() {
	// Initialize ask command flags
	askCmd.Flags().StringSliceP("parallel", "p", nil, "Run providers in parallel (comma-separated)")
	askCmd.Flags().StringSliceP("serial", "s", nil, "Run providers serially (comma-separated)")
	askCmd.Flags().BoolP("best", "b", false, "Use AI to pick best response")
	askCmd.Flags().String("separator", "\n---\n", "Separator for concatenated responses")

	// Initialize jury command flags
	juryCmd.Flags().StringSliceP("providers", "p", nil, "Providers to generate responses (comma-separated)")
	juryCmd.Flags().StringSliceP("jurors", "j", nil, "Jury members to evaluate responses (defaults to providers if not specified)")
	juryCmd.Flags().String("voting", "majority", "Voting method: majority, consensus, or weighted")

	// Initialize collab command flags
	collabCmd.Flags().StringSliceP("providers", "p", nil, "Providers to participate in collaboration (comma-separated)")
	collabCmd.Flags().IntP("rounds", "r", 3, "Number of discussion rounds")

	// Initialize pipeline command flags
	pipelineCmd.Flags().BoolP("list", "l", false, "List available pipelines")
	pipelineCmd.Flags().BoolP("create", "c", false, "Create a new pipeline")
	pipelineCmd.Flags().String("type", "", "Pipeline type (serial, parallel, collaborative, nested)")
	pipelineCmd.Flags().StringSliceP("providers", "p", nil, "Providers to use in the pipeline (comma-separated)")
	pipelineCmd.Flags().String("combiner", "concat", "Combiner type for parallel pipelines (concat, best-picker, jury)")
	pipelineCmd.Flags().String("judge", "", "Judge provider for best-picker combiner")
	pipelineCmd.Flags().StringSliceP("jurors", "j", nil, "Jury members for jury combiner (comma-separated)")
	pipelineCmd.Flags().String("voting", "majority", "Voting method for jury combiner (majority, consensus, weighted)")
	pipelineCmd.Flags().String("separator", "\n\n", "Separator for concat combiner")
	pipelineCmd.Flags().IntP("rounds", "r", 3, "Number of rounds for collaborative pipelines")

	// Register new commands
	aiCmd.AddCommand(juryCmd)
	aiCmd.AddCommand(collabCmd)
	aiCmd.AddCommand(pipelineCmd)
}
