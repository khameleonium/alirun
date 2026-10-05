package cmd

import (
	"alirun/pkg/initsys"
	"alirun/pkg/netinfo"
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var portsCmd = &cobra.Command{
	Use:     "ports",
	Aliases: []string{"net", "sockets"},
	Short:   "Show all running services with listening network ports (TCP/UDP)",
	Long: `Inspect all active services and find which processes are actively listening
on TCP and UDP network ports. Useful for discovering web servers, APIs, and network daemons.`,
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

		type portEntry struct {
			Name    string
			PID     int
			Ports   []string
			Command string
		}

		var activeWithPorts []portEntry
		for _, s := range services {
			if s.PID > 0 {
				ports := netinfo.GetListeningPortsForPID(s.PID)
				if len(ports) > 0 {
					cmdStr := s.ExecPath
					if len(cmdStr) > 40 {
						cmdStr = cmdStr[:37] + "..."
					}
					activeWithPorts = append(activeWithPorts, portEntry{
						Name:    s.Name,
						PID:     s.PID,
						Ports:   ports,
						Command: cmdStr,
					})
				}
			}
		}

		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#5A56E0")).
			Padding(0, 1)

		if len(activeWithPorts) == 0 {
			fmt.Printf("\n%s\n\n", titleStyle.Render(fmt.Sprintf(" Listening Network Ports — Scope: %s ", strings.ToUpper(string(sType)))))
			fmt.Printf("○ No active services with listening ports detected (scope: %s).\n", sType)
			if sType == initsys.TypeUser {
				fmt.Println("  Tip: Inspect system-wide daemons using: alirun ports --system")
				fmt.Println("  Tip: Or create a test web daemon: alirun create \"python3 -m http.server 8080\" --name my-web --now")
			}
			return nil
		}

		fmt.Printf("\n%s\n\n", titleStyle.Render(fmt.Sprintf(" Listening Network Ports — Scope: %s (%d services) ", strings.ToUpper(string(sType)), len(activeWithPorts))))

		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7986CB"))
		fmt.Printf("%-28s %-8s %-32s %s\n",
			headerStyle.Render("SERVICE"),
			headerStyle.Render("PID"),
			headerStyle.Render("PORTS"),
			headerStyle.Render("COMMAND"),
		)
		fmt.Println(strings.Repeat("─", 88))

		portStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true)
		for _, e := range activeWithPorts {
			portsStr := portStyle.Render(strings.Join(e.Ports, ", "))
			fmt.Printf("%-28s %-8d %-32s %s\n",
				e.Name,
				e.PID,
				portsStr,
				e.Command,
			)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(portsCmd)
}
