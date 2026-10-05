package cmd

import (
	"alirun/pkg/editor"
	"alirun/pkg/initsys"
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [service_name]",
	Short: "Open unit, crontab, or desktop autostart configuration in $EDITOR",
	Long: `Open and edit service configurations directly in your preferred editor ($VISUAL or $EDITOR).
Automatically detects whether the target is a systemd unit, a crontab entry, or an XDG .desktop file.
Upon saving and exiting, changes are automatically reloaded (systemctl daemon-reload or crontab update).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := getManager()
		if err != nil {
			return err
		}
		sType := getServiceType()
		ctx := context.Background()

		var serviceName string
		if len(args) > 0 {
			serviceName = args[0]
		}

		// If manager is cron and no serviceName provided, edit user crontab directly
		if serviceName == "" {
			if mgr.Name() == "cron" {
				session, err := editor.PrepareEdit(ctx, &initsys.ServiceInfo{Name: "crontab (user)", ConfigPath: "crontab (user)"}, mgr, sType)
				if err != nil {
					return err
				}
				if err := editor.RunInteractive(ctx, session); err != nil {
					return err
				}
				fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true).Render("✓ Crontab successfully updated."))
				return nil
			}
			return fmt.Errorf("service name is required: alirun edit <service_name>")
		}

		// Retrieve service info
		info, err := mgr.GetStatus(ctx, serviceName, sType)
		if err != nil {
			// If not found in current manager, check if it's an XDG desktop file or cron
			if strings.HasSuffix(serviceName, ".desktop") {
				if xdgMgr, xErr := initsys.Get("xdg"); xErr == nil {
					if xInfo, xErr2 := xdgMgr.GetStatus(ctx, serviceName, sType); xErr2 == nil {
						info = xInfo
						mgr = xdgMgr
						err = nil
					}
				}
			}
		}

		if err != nil {
			return fmt.Errorf("service %q not found: %w", serviceName, err)
		}

		session, err := editor.PrepareEdit(ctx, info, mgr, sType)
		if err != nil {
			return err
		}

		fmt.Printf("Opening %s in %s...\n", session.FilePath, editor.FindEditor())
		if err := editor.RunInteractive(ctx, session); err != nil {
			return err
		}

		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Bold(true).Render("✓ Configuration saved and changes reloaded."))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
