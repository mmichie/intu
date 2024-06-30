package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mmichie/intu/pkg/intu"
	"github.com/spf13/cobra"
)

const pipeName = "/tmp/intu_pipe"

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the AI daemon with a named pipe",
	Long:  `Start the AI daemon that creates a named pipe for communication.`,
	RunE:  runDaemonCommand,
}

func InitDaemonCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(daemonCmd)
}

func runDaemonCommand(cmd *cobra.Command, args []string) error {
	// Create the named pipe
	err := syscall.Mkfifo(pipeName, 0666)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create named pipe: %w", err)
	}

	// Create AI client
	provider, err := selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select AI provider: %w", err)
	}
	client := intu.NewClient(provider)

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create a done channel to signal when the cleanup is complete
	done := make(chan struct{})

	go func() {
		<-sigChan
		fmt.Println("Shutting down...")
		cancel()
		close(done)
	}()

	fmt.Printf("AI daemon started. Named pipe: %s\n", pipeName)
	fmt.Println("You can now interact with the daemon using this pipe.")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := processPipeInput(ctx, client); err != nil {
					fmt.Fprintf(os.Stderr, "Error processing input: %v\n", err)
				}
			}
		}
	}()

	// Wait for the done signal or context cancellation
	select {
	case <-done:
	case <-ctx.Done():
	}

	// Cleanup
	os.Remove(pipeName)
	fmt.Println("Daemon stopped.")
	return nil
}

func processPipeInput(ctx context.Context, client *intu.Client) error {
	file, err := os.OpenFile(pipeName, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return fmt.Errorf("failed to open pipe: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read from pipe: %w", err)
	}

	response, err := client.ProcessWithAI(ctx, input, "")
	if err != nil {
		return fmt.Errorf("failed to process input: %w", err)
	}

	_, err = file.WriteString(response + "\n")
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}
