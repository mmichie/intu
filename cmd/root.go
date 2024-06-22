package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "intu",
	Short: "intu is an AI-powered command-line tool",
	Long: `intu is a CLI tool that leverages AI language models to assist with various tasks,
including file content analysis and generating git commit messages.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.SetEnvPrefix("INTU")
	viper.AutomaticEnv()
}
