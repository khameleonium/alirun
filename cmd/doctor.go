package cmd

import (
	"alirun/pkg/diagnose"
	"alirun/pkg/initsys"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor [service_name]",
	Aliases: []string{"diagnose", "doc"},
	Short:   "Diagnose failed services with automated root-cause analysis and suggestions",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getManager()
		if err != nil {
			return err
		}

		sType := getServiceType()
		ctx := cmd.Context()

		if len(args) > 0 {
			// Diagnose specific service
			name := args[0]
			info, err := mgr.GetStatus(ctx, name, sType)
			if err != nil {
				return err
			}

			logs := fetchRecentLogs(ctx, mgr, name, sType, 40)
			report := diagnose.Diagnose(info, logs)
			printDiagnosticReport(report, info)
			return nil
		}

		// Scan all services for failures
		services, err := mgr.ListServices(ctx, sType)
		if err != nil {
			return err
		}

		var failedServices []initsys.ServiceInfo
		for _, s := range services {
			if s.Status == initsys.StatusFailed {
				failedServices = append(failedServices, s)
			}
		}

		if len(failedServices) == 0 {
			successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true)
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ All %d services are healthy! No failed units detected (scope: %s).", len(services), sType)))
			return nil
		}

		fmt.Printf("⚠ Found %d failed service(s) in %s scope. Running diagnostic inspection...\n\n", len(failedServices), sType)

		for _, s := range failedServices {
			// ListServices does not include the command line and unit path; fetch full details
			info := &s
			if full, err := mgr.GetStatus(ctx, s.Name, sType); err == nil {
				info = full
			}
			logs := fetchRecentLogs(ctx, mgr, s.Name, sType, 40)
			report := diagnose.Diagnose(info, logs)
			printDiagnosticReport(report, info)
			fmt.Println()
		}

		return nil
	},
}

func fetchRecentLogs(ctx context.Context, mgr initsys.Manager, name string, sType initsys.ServiceType, lines int) []string {
	ch, err := mgr.StreamLogs(ctx, name, sType, lines, false)
	if err != nil {
		return nil
	}
	var res []string
	timeout := time.After(2 * time.Second)
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				return res
			}
			res = append(res, line)
		case <-timeout:
			return res
		}
	}
}

func printDiagnosticReport(report diagnose.IssueReport, info *initsys.ServiceInfo) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#FF1744")).Padding(0, 1)
	if report.IsHealthy {
		titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#00E676")).Padding(0, 1)
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C4DFF")).
		Padding(0, 1)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  Service: %s\n\n", titleStyle.Render("DIAGNOSTIC REPORT"), report.ServiceName))
	sb.WriteString(fmt.Sprintf("📌 Итог (Summary):      %s\n", lipgloss.NewStyle().Bold(true).Render(report.Summary)))
	sb.WriteString(fmt.Sprintf("🔍 Причина (Cause):     %s\n", report.RootCause))

	if report.OffendingLine != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF1744")).Bold(true)
		sb.WriteString(fmt.Sprintf("⚡ Строка ошибки (Log):  %s\n", errStyle.Render(report.OffendingLine)))
	}

	fixStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true)
	sb.WriteString(fmt.Sprintf("\n💡 Решение (Suggested Fix):\n   %s\n", fixStyle.Render(report.SuggestedFix)))

	if info != nil && info.ExecPath != "" {
		sb.WriteString(fmt.Sprintf("\n⚙  Команда запуска:     %s\n", info.ExecPath))
	}
	if info != nil && info.ConfigPath != "" {
		sb.WriteString(fmt.Sprintf("📄 Конфигурация:        %s\n", info.ConfigPath))
	}

	fmt.Println(boxStyle.Render(sb.String()))
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
