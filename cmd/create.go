package cmd

import (
	"alirun/pkg/detector"
	"alirun/pkg/highlighter"
	"alirun/pkg/initsys"
	"alirun/pkg/initsys/systemd"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	flagCreateName     string
	flagCreateDesc     string
	flagCreateExec     string
	flagCreateWorkDir  string
	flagCreatePreset   string
	flagCreateSchedule string
	flagCreateSystem   bool
	flagCreateNow      bool
	flagCreateNonInter bool
)

var createCmd = &cobra.Command{
	Use:   "create [path_to_executable_or_script]",
	Short: "Create and configure a new service with an interactive wizard",
	Example: `  alirun create ./myscript.sh
  alirun create /home/user/myproject/main.py
  alirun create /usr/local/bin/mybot --preset daemon`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetPath string
		if len(args) > 0 {
			targetPath = args[0]
		}

		mgr, err := getManager()
		if err != nil {
			return err
		}

		// If path is not provided and interactive, ask for it
		if targetPath == "" && !flagCreateNonInter {
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Path to executable or script").
						Description("Enter the relative or absolute path of the program to run").
						Placeholder("./myscript.sh or /usr/local/bin/app").
						Value(&targetPath).
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("path cannot be empty")
							}
							return nil
						}),
				),
			)
			if err := form.Run(); err != nil {
				return err
			}
		}

		if targetPath == "" {
			return fmt.Errorf("target path is required")
		}

		// Inspect target using detector
		insp, err := detector.Inspect(targetPath)
		if err != nil {
			return fmt.Errorf("inspection failed: %w", err)
		}

		// Print inspection insights
		bannerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#6200EE")).
			Padding(0, 1)

		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C4DFF")).
			Padding(0, 1)

		var infoLines []string
		infoLines = append(infoLines, fmt.Sprintf("Target File:  %s", insp.AbsolutePath))
		infoLines = append(infoLines, fmt.Sprintf("Detected:     %s", insp.DetectedType))
		if insp.VirtualEnvPath != "" {
			infoLines = append(infoLines, fmt.Sprintf("Virtualenv:   %s", insp.VirtualEnvPath))
		}
		if len(insp.Warnings) > 0 {
			for _, w := range insp.Warnings {
				infoLines = append(infoLines, lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAB00")).Render("⚠ "+w))
			}
		}

		fmt.Println(bannerStyle.Render(" Autolirun Service Inspection "))
		fmt.Println(boxStyle.Render(strings.Join(infoLines, "\n")))

		// If file exists and not executable, ask to chmod +x
		if insp.Exists && !insp.IsExecutable && !flagCreateNonInter {
			var fixChmod bool
			confirm := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("File is not executable. Add execute permissions (chmod +x)?").
						Value(&fixChmod),
				),
			)
			if err := confirm.Run(); err == nil && fixChmod {
				if err := detector.MakeExecutable(insp.AbsolutePath); err == nil {
					fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#00E676")).Render("✔ Added chmod +x successfully!"))
				}
			}
		}

		// Pre-populate fields
		name := flagCreateName
		if name == "" {
			name = insp.SuggestedName
		}
		desc := flagCreateDesc
		if desc == "" {
			desc = fmt.Sprintf("%s service", name)
		}
		execCmd := flagCreateExec
		if execCmd == "" {
			execCmd = insp.ExecStartCommand
		}
		workDir := flagCreateWorkDir
		if workDir == "" {
			workDir = insp.SuggestedWorkDir
		}
		presetChoice := flagCreatePreset
		if presetChoice == "" {
			presetChoice = "daemon"
		}

		isSystem := flagCreateSystem || flagSystem
		sType := initsys.TypeUser
		if isSystem {
			sType = initsys.TypeSystem
		}

		// Interactive wizard form
		if !flagCreateNonInter {
			targetLevel := "user"
			if isSystem {
				targetLevel = "system"
			}

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Service Name").
						Description("Identifier used for systemd (without .service)").
						Value(&name).
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("name cannot be empty")
							}
							return nil
						}),

					huh.NewInput().
						Title("Description").
						Description("Human-readable title for the service").
						Value(&desc),

					huh.NewInput().
						Title("ExecStart Command").
						Description("Exact command executed when service starts").
						Value(&execCmd).
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("command cannot be empty")
							}
							return nil
						}),

					huh.NewInput().
						Title("Working Directory").
						Description("Directory from which the command will run").
						Value(&workDir),

					huh.NewSelect[string]().
						Title("Execution Preset").
						Description("How should this service behave?").
						Options(
							huh.NewOption("Background Daemon (Restart=always on failure)", "daemon"),
							huh.NewOption("Web Service / API (Waits for network, Restart=always)", "web"),
							huh.NewOption("One-shot Task (Runs once and terminates cleanly)", "oneshot"),
							huh.NewOption("Scheduled Timer (Systemd .timer cron replacement)", "timer"),
						).
						Value(&presetChoice),

					huh.NewSelect[string]().
						Title("Service Scope").
						Description("User (safe, no sudo required) vs System-wide (requires root)").
						Options(
							huh.NewOption("User Service (~/.config/systemd/user/, no root required)", "user"),
							huh.NewOption("System Service (/etc/systemd/system/, requires root)", "system"),
						).
						Value(&targetLevel),
				),
			)

			if err := form.Run(); err != nil {
				return err
			}

			if targetLevel == "system" {
				sType = initsys.TypeSystem
			} else {
				sType = initsys.TypeUser
			}
		}

		scheduleVal := flagCreateSchedule
		if presetChoice == "timer" {
			if scheduleVal == "" && !flagCreateNonInter {
				scheduleVal = "hourly"
				timerForm := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Timer Schedule").
							Description("Interval (e.g. 15m, 1h) or Systemd Calendar (hourly, daily, '*-*-* 03:00:00')").
							Value(&scheduleVal).
							Validate(func(s string) error {
								if strings.TrimSpace(s) == "" {
									return fmt.Errorf("schedule cannot be empty")
								}
								return nil
							}),
					),
				)
				if err := timerForm.Run(); err != nil {
					return err
				}
			}
			if scheduleVal == "" {
				scheduleVal = "hourly"
			}
		}

		// Prepare ServiceConfig
		cfg := initsys.ServiceConfig{
			Name:             name,
			Description:      desc,
			ExecStart:        execCmd,
			WorkingDirectory: workDir,
			Type:             sType,
			RestartSec:       5,
		}

		switch presetChoice {
		case "web":
			cfg.Preset = initsys.PresetWeb
			cfg.WantsNetwork = true
		case "oneshot":
			cfg.Preset = initsys.PresetOneshot
		case "timer":
			cfg.Preset = initsys.PresetTimer
			cfg.TimerSchedule = scheduleVal
		default:
			cfg.Preset = initsys.PresetDaemon
		}

		// Generate configuration text
		content, err := mgr.GenerateConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to generate configuration: %w", err)
		}

		destPath := mgr.GetConfigPath(cfg.Name, cfg.Type)

		// Transparency Presentation
		fmt.Println("\n" + bannerStyle.Render(" Transparency Preview: Generated Service Unit "))
		fmt.Printf("Destination File: %s\n\n", lipgloss.NewStyle().Bold(true).Render(destPath))

		// Highlight configuration
		fmt.Println(highlighter.HighlightUnit(content))

		if cfg.Preset == initsys.PresetTimer {
			timerContent, err := systemd.GenerateTimerFile(cfg)
			if err == nil {
				timerDestPath := strings.TrimSuffix(destPath, ".service") + ".timer"
				fmt.Println("\n" + bannerStyle.Render(" Transparency Preview: Generated Timer Unit (.timer) "))
				fmt.Printf("Destination File: %s\n\n", lipgloss.NewStyle().Bold(true).Render(timerDestPath))
				fmt.Println(highlighter.HighlightUnit(timerContent))
			}
		}

		var actionChoice = "start"
		if !flagCreateNonInter {
			actionForm := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("What would you like to do?").
						Options(
							huh.NewOption("✔ Save and Start Service Now (enable --now)", "start"),
							huh.NewOption("💾 Save only (without starting)", "save"),
							huh.NewOption("✏ Open in $EDITOR to customize before saving", "edit"),
							huh.NewOption("✖ Cancel", "cancel"),
						).
						Value(&actionChoice),
				),
			)

			if err := actionForm.Run(); err != nil {
				return err
			}
		} else {
			if flagCreateNow {
				actionChoice = "start"
			} else {
				actionChoice = "save"
			}
		}

		if actionChoice == "cancel" {
			fmt.Println("Aborted. No changes were made.")
			return nil
		}

		if actionChoice == "edit" {
			// Write to temp file and open in $EDITOR
			tmpFile, err := os.CreateTemp("", "alirun-*.service")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			tmpPath := tmpFile.Name()
			_ = os.WriteFile(tmpPath, []byte(content), 0644)
			tmpFile.Close()

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "nano"
			}

			cmd := exec.Command(editor, tmpPath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("editor failed: %w", err)
			}

			// Read edited content back
			editedBytes, err := os.ReadFile(tmpPath)
			_ = os.Remove(tmpPath)
			if err != nil {
				return fmt.Errorf("failed to read back edited file: %w", err)
			}
			content = string(editedBytes)

			fmt.Println("\nUpdated configuration after manual edit:")
			fmt.Println(highlighter.HighlightUnit(content))
		}

		enableNow := (actionChoice == "start")
		fmt.Printf("\nInstalling service %s...\n", cfg.Name)

		savedPath, err := mgr.InstallService(context.Background(), cfg, content, enableNow)
		if err != nil {
			return fmt.Errorf("failed to install service: %w", err)
		}

		successStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00E676")).
			Render("✔ Service created and installed successfully!")

		fmt.Println("\n" + successStyle)
		fmt.Printf("File written to: %s\n", savedPath)
		if enableNow {
			fmt.Println("Status: Service enabled and started.")
		} else {
			fmt.Println("Status: Service file created (not yet started).")
		}

		fmt.Println("\nUseful commands:")
		fmt.Printf("  alirun status %s\n", cfg.Name)
		fmt.Printf("  alirun logs %s -f\n", cfg.Name)
		fmt.Printf("  alirun restart %s\n", cfg.Name)

		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&flagCreateName, "name", "", "Service name (default: auto-detected)")
	createCmd.Flags().StringVar(&flagCreateDesc, "desc", "", "Description")
	createCmd.Flags().StringVar(&flagCreateExec, "exec", "", "ExecStart command line")
	createCmd.Flags().StringVar(&flagCreateWorkDir, "workdir", "", "Working directory")
	createCmd.Flags().StringVar(&flagCreatePreset, "preset", "daemon", "Preset: daemon, web, oneshot, timer")
	createCmd.Flags().StringVar(&flagCreateSchedule, "schedule", "", "Schedule for timer preset (e.g. 'hourly', 'daily', '15m', '*-*-* 03:00:00')")
	createCmd.Flags().BoolVar(&flagCreateSystem, "system", false, "Install as system service (default: user service)")
	createCmd.Flags().BoolVar(&flagCreateNow, "now", true, "Enable and start immediately")
	createCmd.Flags().BoolVar(&flagCreateNonInter, "non-interactive", false, "Do not prompt interactively")
	rootCmd.AddCommand(createCmd)
}

func parseIntOrDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
