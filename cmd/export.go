package cmd

import (
	"alirun/pkg/backup"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export [filename.yaml]",
	Short: "Export services, crontab, and desktop autostart into portable YAML backup",
	Long: `Export systemd service units, user/system crontab, and XDG autostart desktop entries
into a single portable, human-readable YAML backup file.

If no file path is specified, a timestamped file (alirun-backup-YYYYMMDD-HHMMSS.yaml)
is created in the current working directory.
Specify "-" to write YAML to standard output.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sType := getServiceType()
		ctx := context.Background()

		manifest, err := backup.Export(ctx, sType)
		if err != nil {
			return fmt.Errorf("export failed: %w", err)
		}

		yamlData, err := manifest.ToYAML()
		if err != nil {
			return fmt.Errorf("failed to format YAML: %w", err)
		}

		if len(args) > 0 && args[0] == "-" {
			fmt.Print(string(yamlData))
			return nil
		}

		filename := fmt.Sprintf("alirun-backup-%s.yaml", time.Now().Format("20060102-150405"))
		if len(args) > 0 {
			filename = args[0]
		}

		if err := os.WriteFile(filename, yamlData, 0600); err != nil {
			return fmt.Errorf("failed to write export file %s: %w", filename, err)
		}

		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5A56E0")).
			Padding(0, 1)

		var lines []string
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00E676")).Render("✓ Export Successful"))
		lines = append(lines, fmt.Sprintf("File:         %s", filename))
		lines = append(lines, fmt.Sprintf("Scope:        %s", sType))
		lines = append(lines, fmt.Sprintf("Systemd:      %d unit(s)", len(manifest.Systemd)))
		cronDesc := "None"
		if manifest.Crontab != nil {
			if manifest.Crontab.UserCrontab != "" {
				cronDesc = "User crontab included"
			} else if len(manifest.Crontab.SystemFiles) > 0 {
				cronDesc = fmt.Sprintf("%d system cron file(s)", len(manifest.Crontab.SystemFiles))
			}
		}
		lines = append(lines, fmt.Sprintf("Crontab:      %s", cronDesc))
		lines = append(lines, fmt.Sprintf("XDG Desktop:  %d autostart entry(s)", len(manifest.XDG)))

		fmt.Println(boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
