package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var flagForceDelete bool

var startCmd = &cobra.Command{
	Use:   "start <service_name>",
	Short: "Start a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr, err := getManager()
		if err != nil {
			return err
		}
		sType := getServiceType()
		if err := mgr.Start(context.Background(), name, sType); err != nil {
			return err
		}
		fmt.Printf("✔ Service %s started successfully.\n", name)
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <service_name>",
	Short: "Stop a running service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr, err := getManager()
		if err != nil {
			return err
		}
		sType := getServiceType()
		if err := mgr.Stop(context.Background(), name, sType); err != nil {
			return err
		}
		fmt.Printf("✔ Service %s stopped.\n", name)
		return nil
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart <service_name>",
	Short: "Restart a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr, err := getManager()
		if err != nil {
			return err
		}
		sType := getServiceType()
		if err := mgr.Restart(context.Background(), name, sType); err != nil {
			return err
		}
		fmt.Printf("✔ Service %s restarted.\n", name)
		return nil
	},
}

var enableCmd = &cobra.Command{
	Use:   "enable <service_name>",
	Short: "Enable a service to start on boot/login",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr, err := getManager()
		if err != nil {
			return err
		}
		sType := getServiceType()
		if err := mgr.Enable(context.Background(), name, sType); err != nil {
			return err
		}
		fmt.Printf("✔ Service %s enabled for autostart.\n", name)
		return nil
	},
}

var disableCmd = &cobra.Command{
	Use:   "disable <service_name>",
	Short: "Disable autostart for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr, err := getManager()
		if err != nil {
			return err
		}
		sType := getServiceType()
		if err := mgr.Disable(context.Background(), name, sType); err != nil {
			return err
		}
		fmt.Printf("✔ Service %s disabled from autostart.\n", name)
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:     "delete <service_name>",
	Aliases: []string{"remove", "rm"},
	Short:   "Stop, disable, and delete a service configuration file",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		mgr, err := getManager()
		if err != nil {
			return err
		}
		sType := getServiceType()

		if !flagForceDelete {
			var confirm bool
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title(fmt.Sprintf("Are you sure you want to stop, disable, and DELETE service %q?", name)).
						Description(fmt.Sprintf("This will remove the unit file: %s", mgr.GetConfigPath(name, sType))).
						Value(&confirm),
				),
			)
			if err := form.Run(); err != nil {
				return err
			}
			if !confirm {
				fmt.Println("Aborted. Service was not deleted.")
				return nil
			}
		}

		if err := mgr.DeleteService(context.Background(), name, sType); err != nil {
			return fmt.Errorf("failed to delete service: %w", err)
		}

		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00E676")).
			Bold(true)
		fmt.Println(successStyle.Render(fmt.Sprintf("✔ Service %s deleted successfully (stopped, disabled, unit file removed).", name)))
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&flagForceDelete, "yes", "y", false, "Do not prompt for confirmation")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(enableCmd)
	rootCmd.AddCommand(disableCmd)
	rootCmd.AddCommand(deleteCmd)
}
