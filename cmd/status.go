package cmd

import (
	"alirun/pkg/initsys"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <service_name>",
	Short: "Show detailed status and recent logs of a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr, err := getManager()
		if err != nil {
			return err
		}

		sType := getServiceType()
		ctx := context.Background()

		info, err := mgr.GetStatus(ctx, name, sType)
		if err != nil {
			return fmt.Errorf("failed to retrieve service status: %w", err)
		}

		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#5A56E0")).
			Padding(0, 1)

		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5A56E0")).
			Padding(0, 1)

		var statusStr string
		switch info.Status {
		case initsys.StatusActive:
			statusStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true).Render("● ACTIVE (" + info.SubState + ")")
		case initsys.StatusFailed:
			statusStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF1744")).Bold(true).Render("✖ FAILED (" + info.SubState + ")")
		default:
			statusStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#9E9E9E")).Bold(true).Render("○ INACTIVE (" + info.SubState + ")")
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("Service:     %s", info.Name))
		lines = append(lines, fmt.Sprintf("Description: %s", info.Description))
		lines = append(lines, fmt.Sprintf("Init System: %s", info.InitSystem))
		lines = append(lines, fmt.Sprintf("Scope:       %s", info.Type))
		lines = append(lines, fmt.Sprintf("Status:      %s", statusStr))
		lines = append(lines, fmt.Sprintf("Enabled:     %t", info.Enabled))

		if info.IsTimer {
			timerDesc := "Active"
			if info.TimerNext != "" {
				timerDesc = fmt.Sprintf("Active (Next: %s)", info.TimerNext)
			}
			lines = append(lines, fmt.Sprintf("Timer:       %s", timerDesc))
		}
		if info.PID > 0 {
			lines = append(lines, fmt.Sprintf("Main PID:    %d", info.PID))
		}
		if info.TasksCurrent > 0 {
			lines = append(lines, fmt.Sprintf("Tasks:       %d", info.TasksCurrent))
		}
		if info.MemoryBytes > 0 {
			mb := float64(info.MemoryBytes) / (1024 * 1024)
			lines = append(lines, fmt.Sprintf("Memory:      %.1f MB", mb))
		}
		if info.CPUUsageNSec > 0 {
			cpuSec := float64(info.CPUUsageNSec) / 1e9
			lines = append(lines, fmt.Sprintf("CPU Time:    %.2f s", cpuSec))
		}
		if !info.ActiveSince.IsZero() {
			lines = append(lines, fmt.Sprintf("Active Since:%s (%s ago)",
				info.ActiveSince.Format("2006-01-02 15:04:05"),
				time.Since(info.ActiveSince).Round(time.Second)))
		}
		if info.ConfigPath != "" {
			lines = append(lines, fmt.Sprintf("Unit File:   %s", info.ConfigPath))
		}
		if info.ExecPath != "" {
			lines = append(lines, fmt.Sprintf("Command:     %s", info.ExecPath))
		}

		fmt.Println(headerStyle.Render(fmt.Sprintf(" Service Status: %s ", info.Name)))
		fmt.Println(boxStyle.Render(strings.Join(lines, "\n")))

		// Fetch recent logs
		fmt.Println("\n" + lipgloss.NewStyle().Bold(true).Render("Recent Logs (last 10 lines):"))
		logCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		logChan, err := mgr.StreamLogs(logCtx, name, sType, 10, false)
		if err == nil {
			hasLogs := false
			for line := range logChan {
				hasLogs = true
				fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#B0BEC5")).Render(line))
			}
			if !hasLogs {
				fmt.Println(lipgloss.NewStyle().Faint(true).Render("  (no journal entries recorded yet)"))
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
