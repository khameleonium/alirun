package cmd

import (
	"alirun/internal/tui"
	"alirun/pkg/initsys"
	_ "alirun/pkg/initsys/cron"
	_ "alirun/pkg/initsys/openrc"
	_ "alirun/pkg/initsys/systemd"
	_ "alirun/pkg/initsys/xdg"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	flagInit   string
	flagSystem bool
	flagUser   bool
)

var rootCmd = &cobra.Command{
	Use:   "alirun",
	Short: "Alirun — Visual, transparent & easy Linux service and autostart manager",
	Long: `Alirun is a modern, transparent CLI and TUI tool to manage system services,
daemons, and autostart in Linux.

It enables creating systemd units from any executable or script with an
interactive wizard, live syntax-highlighted previews, real-time logs,
and modular support for multiple init systems.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If running in a terminal, launch TUI by default
		if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			mgr, err := getManager()
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagInit, "init", "auto", "Init system provider ('auto', 'systemd', 'cron', 'xdg', 'openrc')")
	rootCmd.PersistentFlags().BoolVar(&flagSystem, "system", false, "Operate on system-wide services (/etc/systemd/system, requires root)")
	rootCmd.PersistentFlags().BoolVar(&flagUser, "user", true, "Operate on user services (~/.config/systemd/user, default)")
}

func getManager() (initsys.Manager, error) {
	if flagInit != "" && flagInit != "auto" {
		return initsys.Get(flagInit)
	}
	return initsys.Detect()
}

func getServiceType() initsys.ServiceType {
	if flagSystem {
		return initsys.TypeSystem
	}
	return initsys.TypeUser
}
