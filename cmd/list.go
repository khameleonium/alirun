package cmd

import (
	"alirun/pkg/initsys"
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var (
	flagListAll    bool
	flagListSearch string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed services and autostart entries",
	Example: `  alirun list
  alirun list --system
  alirun list --search bot`,
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
			if !flagListAll && s.Status == initsys.StatusInactive && s.SubState == "dead" && !s.Enabled {
				// Skip dead and disabled by default unless --all
				continue
			}
			filtered = append(filtered, s)
		}

		if len(filtered) == 0 {
			fmt.Printf("No matching services found (provider: %s, mode: %s)\n", mgr.Name(), sType)
			return nil
		}

		// Table rendering with Lip Gloss
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#5A56E0")).
			Padding(0, 1)

		t := table.New().
			Border(lipgloss.NormalBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))).
			Headers("STATUS", "NAME", "INIT", "TYPE", "STATE", "DESCRIPTION")

		activeCount := 0
		failedCount := 0

		for _, s := range filtered {
			var statusBadge string
			switch s.Status {
			case initsys.StatusActive:
				statusBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Render("● ACTIVE")
				activeCount++
			case initsys.StatusFailed:
				statusBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF1744")).Render("✖ FAILED")
				failedCount++
			default:
				statusBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#9E9E9E")).Render("○ INACTIVE")
			}

			desc := s.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}

			t.Row(statusBadge, s.Name, s.InitSystem, string(s.Type), s.SubState, desc)
		}

		fmt.Println(headerStyle.Render(fmt.Sprintf(" Alirun Services (%s) — Mode: %s ", mgr.Name(), sType)))
		fmt.Println(t.Render())
		fmt.Printf("\nTotal: %d | Active: %d | Failed: %d | Inactive: %d\n",
			len(filtered), activeCount, failedCount, len(filtered)-activeCount-failedCount)

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVarP(&flagListAll, "all", "a", false, "Show all services including inactive/dead")
	listCmd.Flags().StringVarP(&flagListSearch, "search", "s", "", "Filter services by name or description")
	rootCmd.AddCommand(listCmd)
}
