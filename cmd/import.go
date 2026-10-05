package cmd

import (
	"alirun/pkg/backup"
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var flagDryRun bool

var importCmd = &cobra.Command{
	Use:   "import <backup_file.yaml>",
	Short: "Restore services, crontab, and desktop autostart from YAML backup",
	Long: `Import and restore systemd units, crontab entries, and XDG desktop autostart
from a backup YAML file previously created with "alirun export".

Use --dry-run to simulate and verify file paths without modifying your system.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read backup file %s: %w", filePath, err)
		}

		manifest, err := backup.FromYAML(data)
		if err != nil {
			return err
		}

		ctx := context.Background()
		result, err := backup.Import(ctx, manifest, flagDryRun)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}

		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5A56E0")).
			Padding(0, 1)

		title := "✓ Import Successful"
		if flagDryRun {
			title = "🔍 Import Simulation (Dry Run)"
		}

		var lines []string
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00E676")).Render(title))
		lines = append(lines, fmt.Sprintf("Source File:  %s", filePath))
		lines = append(lines, fmt.Sprintf("Systemd:      %d unit(s) restored", len(result.SystemdRestored)))
		for _, u := range result.SystemdRestored {
			lines = append(lines, fmt.Sprintf("  • %s", u))
		}

		cronStatus := "Not present in backup"
		if result.CrontabRestored {
			cronStatus = "Restored successfully"
		}
		lines = append(lines, fmt.Sprintf("Crontab:      %s", cronStatus))

		lines = append(lines, fmt.Sprintf("XDG Desktop:  %d autostart entry(s) restored", len(result.XDGRestored)))
		for _, x := range result.XDGRestored {
			lines = append(lines, fmt.Sprintf("  • %s", x))
		}

		if len(result.Errors) > 0 {
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF1744")).Render("Warnings / Errors:"))
			for _, e := range result.Errors {
				lines = append(lines, fmt.Sprintf("  ⚠ %s", e))
			}
		}

		fmt.Println(boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
		return nil
	},
}

func init() {
	importCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Simulate restoration without writing files")
	rootCmd.AddCommand(importCmd)
}
