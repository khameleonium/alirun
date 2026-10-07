package cmd

import (
	"alirun/internal/tui"
	"alirun/pkg/initsys"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var xdgCmd = &cobra.Command{
	Use:   "xdg",
	Short: "Inspect and manage XDG desktop autostart applications (~/.config/autostart)",
	Long: `Manage desktop applications that launch automatically on graphical login.
Supports viewing, enabling/disabling via Hidden=true, manual launching, and adding new .desktop files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			mgr, err := initsys.Get("xdg")
			if err != nil {
				return err
			}
			sType := getServiceType()
			p := tea.NewProgram(tui.NewModel(mgr, sType), tea.WithAltScreen(), tea.WithMouseCellMotion())
			_, err = p.Run()
			return err
		}
		return cmd.Help()
	},
}

var xdgListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all desktop autostart applications",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := initsys.Get("xdg")
		if err != nil {
			return err
		}

		sType := getServiceType()
		services, err := mgr.ListServices(cmd.Context(), sType)
		if err != nil {
			return err
		}

		if len(services) == 0 {
			fmt.Printf("No desktop autostart entries found (scope: %s).\n", sType)
			return nil
		}

		fmt.Printf("\n  Alirun XDG Desktop Autostart — Scope: %s (%d apps)\n\n", strings.ToUpper(string(sType)), len(services))
		fmt.Printf("%-10s %-25s %-30s %s\n", "STATUS", "APPLICATION", "DESCRIPTION", "COMMAND")
		fmt.Println(strings.Repeat("─", 100))

		for _, s := range services {
			st := "● ACTIVE"
			if !s.Enabled {
				st = "○ HIDDEN"
			}

			name := truncateRunes(s.Name, 23)
			desc := truncateRunes(s.Description, 28)
			execStr := truncateRunes(s.ExecPath, 40)

			fmt.Printf("%-10s %-25s %-30s %s\n", st, name, desc, execStr)
		}
		fmt.Println()
		return nil
	},
}

var (
	xdgAddName    string
	xdgAddCommand string
	xdgAddComment string
)

var xdgAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an application or script to desktop autostart",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := initsys.Get("xdg")
		if err != nil {
			return err
		}

		if xdgAddName == "" {
			return fmt.Errorf("application name is required (--name or -n)")
		}
		if xdgAddCommand == "" {
			return fmt.Errorf("command is required (--command or -c)")
		}

		cfg := initsys.ServiceConfig{
			Name:        xdgAddName,
			Description: xdgAddComment,
			ExecStart:   xdgAddCommand,
			Type:        getServiceType(),
		}

		content, err := mgr.GenerateConfig(cfg)
		if err != nil {
			return err
		}

		path, err := mgr.InstallService(cmd.Context(), cfg, content, false)
		if err != nil {
			return err
		}

		fmt.Printf("✔ Successfully added %s to desktop autostart:\n", xdgAddName)
		fmt.Printf("  File:    %s\n", path)
		fmt.Printf("  Command: %s\n", xdgAddCommand)
		return nil
	},
}

var xdgToggleCmd = &cobra.Command{
	Use:   "toggle <name>",
	Short: "Toggle desktop autostart between active and hidden",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr, err := initsys.Get("xdg")
		if err != nil {
			return err
		}

		sType := getServiceType()
		info, err := mgr.GetStatus(cmd.Context(), name, sType)
		if err != nil {
			return err
		}

		if info.Enabled {
			if err := mgr.Disable(cmd.Context(), name, sType); err != nil {
				return err
			}
			fmt.Printf("✔ Disabled (Hidden=true) autostart for %s\n", name)
		} else {
			if err := mgr.Enable(cmd.Context(), name, sType); err != nil {
				return err
			}
			fmt.Printf("✔ Enabled (Hidden=false) autostart for %s\n", name)
		}
		return nil
	},
}

var xdgRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Launch a desktop autostart application now",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := initsys.Get("xdg")
		if err != nil {
			return err
		}
		sType := getServiceType()
		if err := mgr.Start(cmd.Context(), args[0], sType); err != nil {
			return err
		}
		fmt.Printf("✔ Launched %s in desktop session\n", args[0])
		return nil
	},
}

var xdgRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Delete a .desktop autostart file",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := initsys.Get("xdg")
		if err != nil {
			return err
		}
		sType := getServiceType()
		if err := mgr.DeleteService(cmd.Context(), args[0], sType); err != nil {
			return err
		}
		fmt.Printf("✔ Removed desktop autostart entry: %s\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(xdgCmd)

	xdgCmd.AddCommand(xdgListCmd)
	xdgCmd.AddCommand(xdgAddCmd)
	xdgCmd.AddCommand(xdgToggleCmd)
	xdgCmd.AddCommand(xdgRunCmd)
	xdgCmd.AddCommand(xdgRemoveCmd)

	xdgAddCmd.Flags().StringVarP(&xdgAddName, "name", "n", "", "Application name (e.g. 'Telegram')")
	xdgAddCmd.Flags().StringVarP(&xdgAddCommand, "command", "c", "", "Command to execute")
	xdgAddCmd.Flags().StringVarP(&xdgAddComment, "comment", "m", "", "Comment or description")
}
