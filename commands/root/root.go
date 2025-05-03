package root

import (
	"fmt"
	"os"

	"github.com/mmichie/intu/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	DefaultOpenAIModel = "gpt-4"
	DefaultClaudeModel = "claude-3-5-sonnet-20240620"
	DefaultGeminiModel = "gemini-1.5-pro"
	DefaultGrokModel   = "grok-1"
	DefaultProvider    = "openai"
)

var (
	cfgFile  string
	provider string
	model    string
	verbose  bool
)

// RootCmd is the root command for intu
var RootCmd = &cobra.Command{
	Use:   "intu",
	Short: "intu is an AI-powered command-line tool",
	Long: `intu is a CLI tool that leverages AI language models to assist with various tasks,
including file content analysis and generating git commit messages.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize default values for AI providers
		viper.SetDefault("openai_model", DefaultOpenAIModel)
		viper.SetDefault("claude_model", DefaultClaudeModel)
		viper.SetDefault("gemini_model", DefaultGeminiModel)
		viper.SetDefault("grok_model", DefaultGrokModel)
		viper.SetDefault("default_provider", DefaultProvider)

		// Bind environment variables
		viper.BindEnv("openai_api_key", "OPENAI_API_KEY")
		viper.BindEnv("claude_api_key", "CLAUDE_API_KEY")
		viper.BindEnv("gemini_api_key", "GEMINI_API_KEY")
		viper.BindEnv("xai_api_key", "XAI_API_KEY")

		// Set the provider if specified in the command line
		if provider != "" {
			viper.Set("default_provider", provider)
		}

		// Set the model if specified in the command line
		if model != "" {
			selectedProvider := viper.GetString("default_provider")
			modelKey := selectedProvider + "_model"
			viper.Set(modelKey, model)
		}
	},
}

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.intu.yaml)")
	RootCmd.PersistentFlags().StringVar(&provider, "provider", "", "AI provider to use (openai, claude, gemini, or grok)")
	RootCmd.PersistentFlags().StringVar(&model, "model", "", "AI model to use (specific to the selected provider)")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	viper.BindPFlag("provider", RootCmd.PersistentFlags().Lookup("provider"))
	viper.BindPFlag("model", RootCmd.PersistentFlags().Lookup("model"))
	viper.BindPFlag("verbose", RootCmd.PersistentFlags().Lookup("verbose"))

	// Initialize all commands
	commands.InitAICommand(RootCmd)
	commands.InitCatCommand(RootCmd)
	commands.InitCodeReviewCommand(RootCmd)
	commands.InitCommitCommand(RootCmd)
	commands.InitSecurityReviewCommand(RootCmd)
	commands.InitLSCommand(RootCmd)
	commands.InitGrepCommand(RootCmd)
	commands.InitGlobCommand(RootCmd)
	commands.InitReadCommand(RootCmd)

	RootCmd.AddCommand(versionCmd)
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
