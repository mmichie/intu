package cmd

import (
	"fmt"
	"os"

	"github.com/mmichie/intu/internal/ai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	provider string
	verbose  bool
)

var rootCmd = &cobra.Command{
	Use:   "intu",
	Short: "intu is an AI-powered command-line tool",
	Long: `intu is a CLI tool that leverages AI language models to assist with various tasks,
including file content analysis and generating git commit messages.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize AI provider configuration
		ai.InitProviderConfig()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.intu.yaml)")
	rootCmd.PersistentFlags().StringVar(&provider, "provider", "", "AI provider to use (openai or claude)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	viper.BindPFlag("provider", rootCmd.PersistentFlags().Lookup("provider"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".intu" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".intu")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of intu",
	Long:  `All software has versions. This is intu's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("intu v0.0.1")
	},
}
