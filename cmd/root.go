package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	provider string
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.intu.yaml)")
	rootCmd.PersistentFlags().StringVar(&provider, "provider", "", "AI provider to use (openai or claude)")

	viper.BindPFlag("provider", rootCmd.PersistentFlags().Lookup("provider"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".intu")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
