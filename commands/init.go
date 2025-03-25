package commands

import "github.com/spf13/cobra"

func InitAICommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(askCmd)
	aiCmd.AddCommand(tuiCmd)
	aiCmd.AddCommand(modelsCmd)
	InitDaemonCommand(rootCmd)
	aiCmd.PersistentFlags().BoolP("list", "l", false, "List available prompts")
}

func InitCodeReviewCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(codeReviewCmd)
}

func InitCommitCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(commitCmd)
}

func InitSecurityReviewCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(securityReviewCmd)
}
