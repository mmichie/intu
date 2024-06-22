package cmd

import (
	"fmt"
	"log"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate a commit message",
	Run:   runCommitCommand,
}

func init() {
	rootCmd.AddCommand(commitCmd)
}

func runCommitCommand(cmd *cobra.Command, args []string) {
	client := intu.NewIntuClient(&intu.OpenAIProvider{APIKey: viper.GetString("OPENAI_API_KEY")})
	message, err := client.GenerateCommitMessage()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Println(message)
}
