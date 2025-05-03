package commands

import "github.com/spf13/cobra"

func InitAICommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(tuiCmd)
	aiCmd.AddCommand(askCmd)
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

func InitLSCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(lsCmd)
}
