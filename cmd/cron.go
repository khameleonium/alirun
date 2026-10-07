package cmd

import (
	"alirun/internal/tui"
	"alirun/pkg/initsys"
	cronpkg "alirun/pkg/initsys/cron"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Inspect, manage, and schedule classic Linux cron jobs",
	Long: `Manage user and system crontabs with human-readable schedule explanations,
toggle enable/disable without deleting lines, on-demand execution, and real-time logs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If running in terminal and no subcommands specified, launch TUI in cron mode!
		if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			mgr, err := initsys.Get("cron")
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

var cronListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all cron jobs with human-readable schedules and next run times",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := initsys.Get("cron")
		if err != nil {
			return err
		}

		sType := getServiceType()
		services, err := mgr.ListServices(cmd.Context(), sType)
		if err != nil {
			return err
		}

		if len(services) == 0 {
			fmt.Printf("No cron jobs found for scope %q.\n", sType)
			return nil
		}

		fmt.Printf("\n  Alirun Cron Jobs — Scope: %s (%d jobs)\n\n", strings.ToUpper(string(sType)), len(services))
		fmt.Printf("%-10s %-25s %-25s %-18s %s\n", "STATUS", "ID / NAME", "SCHEDULE", "NEXT RUN", "COMMAND")
		fmt.Println(strings.Repeat("─", 110))

		for _, s := range services {
			stBadge := "● ACTIVE"
			if s.Status != initsys.StatusActive {
				stBadge = "○ DISABLED"
			}

			dispName := truncateRunes(s.Name, 23)
			sched := truncateRunes(s.SubState, 23)

			next := s.TimerNext
			if next == "" {
				next = "-"
			}

			cmdStr := truncateRunes(s.ExecPath, 40)

			fmt.Printf("%-10s %-25s %-25s %-18s %s\n", stBadge, dispName, sched, next, cmdStr)
		}
		fmt.Println()
		return nil
	},
}

var (
	cronAddSchedule string
	cronAddCommand  string
	cronAddComment  string
	cronAddNow      bool
)

var cronAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new scheduled cron job",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := initsys.Get("cron")
		if err != nil {
			return err
		}

		if strings.TrimSpace(cronAddCommand) == "" {
			return fmt.Errorf("command is required (--command or -c)")
		}

		if strings.TrimSpace(cronAddSchedule) == "" {
			cronAddSchedule = "0 * * * *" // Hourly default
		}

		cfg := initsys.ServiceConfig{
			Description:   cronAddComment,
			ExecStart:     cronAddCommand,
			TimerSchedule: cronAddSchedule,
			Type:          getServiceType(),
		}

		// The immediate run is performed synchronously below: a background run started
		// inside InstallService would be killed when the CLI process exits.
		path, err := mgr.InstallService(cmd.Context(), cfg, "", false)
		if err != nil {
			return fmt.Errorf("failed to add cron job: %w", err)
		}

		human := cronpkg.HumanizeSchedule(cronAddSchedule)
		fmt.Printf("✔ Successfully added cron job to %s:\n", path)
		fmt.Printf("  Schedule: %s (%s)\n", cronAddSchedule, human)
		fmt.Printf("  Command:  %s\n", cronAddCommand)
		if cronAddComment != "" {
			fmt.Printf("  Comment:  %s\n", cronAddComment)
		}
		if cronAddNow {
			fmt.Printf("\n▶ Running command once now:\n  $ %s\n\n", cronAddCommand)
			output, runErr := cronpkg.RunJobCommand(cmd.Context(), cronAddCommand)
			if output != "" {
				fmt.Println(output)
			}
			if runErr != nil {
				return fmt.Errorf("job was added, but the immediate run failed: %w", runErr)
			}
		}

		return nil
	},
}

var cronToggleCmd = &cobra.Command{
	Use:   "toggle <id|name>",
	Short: "Toggle a cron job between enabled and disabled (comments/uncomments)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		mgr, err := initsys.Get("cron")
		if err != nil {
			return err
		}

		sType := getServiceType()
		status, err := mgr.GetStatus(cmd.Context(), target, sType)
		if err != nil {
			return err
		}

		if status.Enabled {
			err = mgr.Disable(cmd.Context(), target, sType)
			if err != nil {
				return err
			}
			fmt.Printf("✔ Disabled cron job: %s\n", status.Name)
		} else {
			err = mgr.Enable(cmd.Context(), target, sType)
			if err != nil {
				return err
			}
			fmt.Printf("✔ Enabled cron job: %s\n", status.Name)
		}
		return nil
	},
}

var cronEnableCmd = &cobra.Command{
	Use:   "enable <id|name>",
	Short: "Enable a disabled cron job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := initsys.Get("cron")
		if err != nil {
			return err
		}
		sType := getServiceType()
		if err := mgr.Enable(cmd.Context(), args[0], sType); err != nil {
			return err
		}
		fmt.Printf("✔ Enabled cron job: %s\n", args[0])
		return nil
	},
}

var cronDisableCmd = &cobra.Command{
	Use:   "disable <id|name>",
	Short: "Disable a cron job without deleting it (comments with #)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := initsys.Get("cron")
		if err != nil {
			return err
		}
		sType := getServiceType()
		if err := mgr.Disable(cmd.Context(), args[0], sType); err != nil {
			return err
		}
		fmt.Printf("✔ Disabled cron job: %s\n", args[0])
		return nil
	},
}

var cronRunCmd = &cobra.Command{
	Use:   "run <id|name>",
	Short: "Execute a cron job command immediately on demand",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := initsys.Get("cron")
		if err != nil {
			return err
		}
		sType := getServiceType()
		status, err := mgr.GetStatus(cmd.Context(), args[0], sType)
		if err != nil {
			return err
		}

		fmt.Printf("▶ Executing command for %s:\n  $ %s\n\n", status.Name, status.ExecPath)
		output, err := cronpkg.RunJobCommand(cmd.Context(), status.ExecPath)
		if output != "" {
			fmt.Println(output)
		}
		if err != nil {
			return fmt.Errorf("execution error: %w", err)
		}
		fmt.Println("\n✔ Command completed successfully.")
		return nil
	},
}

var cronRemoveCmd = &cobra.Command{
	Use:     "remove <id|name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a cron job from crontab",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := initsys.Get("cron")
		if err != nil {
			return err
		}
		sType := getServiceType()
		if err := mgr.DeleteService(cmd.Context(), args[0], sType); err != nil {
			return err
		}
		fmt.Printf("✔ Removed cron job: %s\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cronCmd)

	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronAddCmd)
	cronCmd.AddCommand(cronToggleCmd)
	cronCmd.AddCommand(cronEnableCmd)
	cronCmd.AddCommand(cronDisableCmd)
	cronCmd.AddCommand(cronRunCmd)
	cronCmd.AddCommand(cronRemoveCmd)

	cronAddCmd.Flags().StringVarP(&cronAddSchedule, "schedule", "s", "0 * * * *", "Cron schedule expression (e.g. '*/15 * * * *', '0 3 * * *')")
	cronAddCmd.Flags().StringVarP(&cronAddCommand, "command", "c", "", "Command or script to execute (required)")
	cronAddCmd.Flags().StringVarP(&cronAddComment, "comment", "m", "", "Human description or comment")
	cronAddCmd.Flags().BoolVar(&cronAddNow, "now", false, "Execute the command immediately once after adding")
}
