package cmd

import (
	"alirun/pkg/updater"
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version, commit, and build information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(updater.Info())
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for updates and update Alirun to the latest release",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Current version: %s\n", updater.Version)
		fmt.Println("🔍 Checking for updates from GitHub...")

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		result, err := updater.SelfUpdate(ctx, "")
		if err != nil {
			return fmt.Errorf("update check/application failed: %w", err)
		}

		if !result.Found {
			fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true).Render("✔ You are already using the latest version of Alirun!"))
			return nil
		}

		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true)
		fmt.Println(successStyle.Render(fmt.Sprintf("✔ Successfully updated Alirun to version %s!", result.LatestVersion)))
		if result.ReleaseNotes != "" {
			fmt.Printf("\nRelease Notes:\n%s\n", result.ReleaseNotes)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
}
