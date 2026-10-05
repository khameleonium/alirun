package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	flagLogFollow bool
	flagLogLines  int
)

var logsCmd = &cobra.Command{
	Use:   "logs <service_name>",
	Short: "View and follow logs of a service (journalctl)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr, err := getManager()
		if err != nil {
			return err
		}

		sType := getServiceType()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Trap SIGINT and SIGTERM
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			cancel()
		}()

		logChan, err := mgr.StreamLogs(ctx, name, sType, flagLogLines, flagLogFollow)
		if err != nil {
			return fmt.Errorf("failed to stream logs: %w", err)
		}

		for line := range logChan {
			fmt.Println(line)
		}

		return nil
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&flagLogFollow, "follow", "f", false, "Follow log output in real time")
	logsCmd.Flags().IntVarP(&flagLogLines, "lines", "n", 50, "Number of recent lines to display")
	rootCmd.AddCommand(logsCmd)
}
