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

var showAllNet bool

var portsCmd = &cobra.Command{
	Use:     "ports",
	Aliases: []string{"net", "sockets"},
	Short:   "Show services with listening network ports or active connections",
	Long: `Inspect all active services and find which processes are listening on network ports
or maintaining active outbound connections (using -a / --all).`,
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
			Network string
			IsOut   bool
			Command string
		}

		var activeWithPorts []portEntry
		for _, s := range services {
			if s.PID > 0 {
				netSum := netinfo.GetNetSummaryForPID(s.PID)
				if len(netSum.Listening) > 0 {
					cmdStr := s.ExecPath
					if len(cmdStr) > 36 {
						cmdStr = cmdStr[:33] + "..."
					}
					activeWithPorts = append(activeWithPorts, portEntry{
						Name:    s.Name,
						PID:     s.PID,
						Network: strings.Join(netSum.Listening, ", "),
						IsOut:   false,
						Command: cmdStr,
					})
				} else if showAllNet && netSum.OutboundCount > 0 {
					cmdStr := s.ExecPath
					if len(cmdStr) > 36 {
						cmdStr = cmdStr[:33] + "..."
					}
					activeWithPorts = append(activeWithPorts, portEntry{
						Name:    s.Name,
						PID:     s.PID,
						Network: netSum.FormatSummary(),
						IsOut:   true,
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
			scopeTitle := fmt.Sprintf(" Listening Network Ports — Scope: %s ", strings.ToUpper(string(sType)))
			if showAllNet {
				scopeTitle = fmt.Sprintf(" Active Network Sockets — Scope: %s ", strings.ToUpper(string(sType)))
			}
			fmt.Printf("\n%s\n\n", titleStyle.Render(scopeTitle))
			fmt.Printf("○ No active services with network activity detected (scope: %s).\n", sType)
			if sType == initsys.TypeUser {
				if !showAllNet {
					fmt.Println("  Tip: View active outbound/client connections using: alirun ports -a")
				}
				fmt.Println("  Tip: Inspect system-wide daemons using: alirun ports --system")
				fmt.Println("  Tip: Or create a test web daemon: alirun create \"python3 -m http.server 8080\" --name my-web --now")
			}
			return nil
		}

		headerScope := "Listening Network Ports"
		if showAllNet {
			headerScope = "Network Activity (Listening & Outbound)"
		}
		fmt.Printf("\n%s\n\n", titleStyle.Render(fmt.Sprintf(" %s — Scope: %s (%d services) ", headerScope, strings.ToUpper(string(sType)), len(activeWithPorts))))

		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7986CB"))
		fmt.Printf("%-28s %-8s %-38s %s\n",
			headerStyle.Render("SERVICE"),
			headerStyle.Render("PID"),
			headerStyle.Render("PORTS / NETWORK"),
			headerStyle.Render("COMMAND"),
		)
		fmt.Println(strings.Repeat("─", 96))

		portStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true)
		outStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#29B6F6"))

		for _, e := range activeWithPorts {
			var netStr string
			if e.IsOut {
				netStr = outStyle.Render(e.Network)
			} else {
				netStr = portStyle.Render(e.Network)
			}
			fmt.Printf("%-28s %-8d %-38s %s\n",
				e.Name,
				e.PID,
				netStr,
				e.Command,
			)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	portsCmd.Flags().BoolVarP(&showAllNet, "all", "a", false, "Show services with active outbound connections in addition to listening ports")
	rootCmd.AddCommand(portsCmd)
}
