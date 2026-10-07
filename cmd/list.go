package cmd

import (
	"alirun/internal/tui"
	"alirun/pkg/initsys"
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var (
	flagListAll      bool
	flagListSearch   string
	flagListSort     string
	flagListReverse  bool
	flagListDetailed bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed services and autostart entries",
	Example: `  alirun list
  alirun list -d
  alirun list --sort cpu
  alirun list --sort ram -r
  alirun list --system --search bot`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getManager()
		if err != nil {
			return err
		}

		sType := getServiceType()
		ctx := context.Background()

		services, err := mgr.ListServices(ctx, sType)
		if err != nil {
			return fmt.Errorf("failed to list services: %w", err)
		}

		// Filter
		var filtered []initsys.ServiceInfo
		searchLower := strings.ToLower(flagListSearch)
		for _, s := range services {
			if searchLower != "" {
				nameMatch := strings.Contains(strings.ToLower(s.Name), searchLower)
				descMatch := strings.Contains(strings.ToLower(s.Description), searchLower)
				if !nameMatch && !descMatch {
					continue
				}
			}
			// Skip stopped units that are neither enabled for autostart nor driven by a timer
			// (unless --all). "static" units are skipped too: they can't be enabled.
			if !flagListAll && s.Status == initsys.StatusInactive && s.SubState == "dead" &&
				!s.IsTimer && !strings.HasPrefix(s.UnitFileState, "enabled") {
				continue
			}
			filtered = append(filtered, s)
		}

		if len(filtered) == 0 {
			fmt.Printf("No matching services found (provider: %s, mode: %s)\n", mgr.Name(), sType)
			return nil
		}

		// Determine sort criteria
		var sortField tui.SortField
		switch strings.ToLower(flagListSort) {
		case "status":
			sortField = tui.SortByStatus
		case "start", "started", "time":
			sortField = tui.SortByStartTime
		case "uptime":
			sortField = tui.SortByUptime
		case "cpu":
			sortField = tui.SortByCPU
		case "ram", "memory", "mem":
			sortField = tui.SortByMemory
		default:
			sortField = tui.SortByName
		}

		sortDir := tui.SortAsc
		if flagListReverse {
			sortDir = tui.SortDesc
		} else if cmd.Flags().Changed("sort") && sortField != tui.SortByName && sortField != tui.SortByStatus {
			sortDir = tui.SortDesc
		}

		tui.SortServices(filtered, sortField, sortDir)

		// Table rendering with Lip Gloss
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#5A56E0")).
			Padding(0, 1)

		t := table.New().
			Border(lipgloss.NormalBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")))

		if flagListDetailed {
			t.Headers("STATUS", "NAME", "STATE", "CPU", "RAM", "UPTIME", "PID", "DESCRIPTION")
		} else {
			t.Headers("STATUS", "NAME", "INIT", "TYPE", "STATE", "DESCRIPTION")
		}

		activeCount := 0
		failedCount := 0

		for _, s := range filtered {
			var statusBadge string
			timerPrefix := ""
			if s.IsTimer {
				timerPrefix = "⏱ "
			}

			switch s.Status {
			case initsys.StatusActive:
				statusBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Render(timerPrefix + "● ACTIVE")
				activeCount++
			case initsys.StatusFailed:
				statusBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF1744")).Render(timerPrefix + "✖ FAILED")
				failedCount++
			default:
				statusBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#9E9E9E")).Render(timerPrefix + "○ INACTIVE")
			}

			desc := s.Description
			if flagListDetailed {
				desc = truncateRunes(desc, 30)
				t.Row(
					statusBadge,
					s.Name,
					s.SubState,
					tui.FormatCPU(s.CPUUsageNSec),
					tui.FormatRAM(s.MemoryBytes),
					tui.FormatUptime(s.ActiveSince, s.Status),
					tui.FormatPID(s.PID),
					desc,
				)
			} else {
				desc = truncateRunes(desc, 40)
				t.Row(statusBadge, s.Name, s.InitSystem, string(s.Type), s.SubState, desc)
			}
		}

		modeTitle := fmt.Sprintf(" Alirun Services (%s) — Mode: %s | Sort: %s %s ",
			mgr.Name(), sType, sortField, sortDir.Arrow())
		if flagListDetailed {
			modeTitle += "| View: Detailed "
		}

		fmt.Println(headerStyle.Render(modeTitle))
		fmt.Println(t.Render())
		fmt.Printf("\nTotal: %d | Active: %d | Failed: %d | Inactive: %d\n",
			len(filtered), activeCount, failedCount, len(filtered)-activeCount-failedCount)

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVarP(&flagListAll, "all", "a", false, "Show all services including inactive/dead")
	listCmd.Flags().StringVarP(&flagListSearch, "search", "s", "", "Filter services by name or description")
	listCmd.Flags().StringVar(&flagListSort, "sort", "name", "Sort by: name, status, start, uptime, cpu, ram")
	listCmd.Flags().BoolVarP(&flagListReverse, "reverse", "r", false, "Reverse sort order")
	listCmd.Flags().BoolVarP(&flagListDetailed, "detailed", "d", false, "Show detailed view with CPU, RAM, Uptime, and PID")
	rootCmd.AddCommand(listCmd)
}
