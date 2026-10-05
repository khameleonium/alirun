package editor

import (
	"alirun/pkg/initsys"
	"alirun/pkg/initsys/cron"
	"alirun/pkg/initsys/systemd"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindEditor determines the user's preferred text editor
func FindEditor() string {
	if ed := os.Getenv("VISUAL"); ed != "" {
		return ed
	}
	if ed := os.Getenv("EDITOR"); ed != "" {
		return ed
	}
	for _, candidate := range []string{"nano", "vim", "nvim", "vi", "micro", "emacs"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return "nano"
}

// EditSession manages the lifecycle of editing a service configuration
type EditSession struct {
	FilePath string
	IsTemp   bool
	OnSaved  func() error
	Cleanup  func()
}

// PrepareEdit configures an edit session for a service
func PrepareEdit(ctx context.Context, info *initsys.ServiceInfo, mgr initsys.Manager, sType initsys.ServiceType) (*EditSession, error) {
	mgrName := ""
	if mgr != nil {
		mgrName = mgr.Name()
	} else if info != nil && info.InitSystem != "" {
		mgrName = info.InitSystem
	}

	// 1. Cron Manager
	if mgrName == "cron" {
		if sType == initsys.TypeUser || strings.Contains(info.ConfigPath, "crontab (user)") || info.ConfigPath == "" {
			// User crontab: export to a temporary file
			content, err := cron.ReadUserCrontab()
			if err != nil {
				return nil, fmt.Errorf("failed to read user crontab: %w", err)
			}

			tmpFile, err := os.CreateTemp("", "alirun-crontab-*.cron")
			if err != nil {
				return nil, fmt.Errorf("failed to create temp crontab file: %w", err)
			}
			tmpPath := tmpFile.Name()
			if _, err := tmpFile.WriteString(content); err != nil {
				tmpFile.Close()
				_ = os.Remove(tmpPath)
				return nil, err
			}
			tmpFile.Close()

			return &EditSession{
				FilePath: tmpPath,
				IsTemp:   true,
				OnSaved: func() error {
					newContent, err := os.ReadFile(tmpPath)
					if err != nil {
						return fmt.Errorf("failed to read modified crontab: %w", err)
					}
					return cron.SaveUserCrontab(string(newContent))
				},
				Cleanup: func() {
					_ = os.Remove(tmpPath)
				},
			}, nil
		}

		// System crontab file (/etc/crontab or /etc/cron.d/...)
		path := info.ConfigPath
		if path == "" {
			path = "/etc/crontab"
		}
		return &EditSession{
			FilePath: path,
			IsTemp:   false,
			OnSaved:  func() error { return nil },
			Cleanup:  func() {},
		}, nil
	}

	// 2. XDG Autostart Manager
	if mgrName == "xdg" {
		path := info.ConfigPath
		if path == "" && mgr != nil {
			path = mgr.GetConfigPath(info.Name, sType)
		}
		if path == "" {
			return nil, fmt.Errorf("desktop config path not found for %s", info.Name)
		}
		return &EditSession{
			FilePath: path,
			IsTemp:   false,
			OnSaved:  func() error { return nil },
			Cleanup:  func() {},
		}, nil
	}

	// 3. Systemd (Default)
	configPath := ""
	if info != nil && info.ConfigPath != "" {
		configPath = info.ConfigPath
	} else if mgr != nil && info != nil {
		configPath = mgr.GetConfigPath(info.Name, sType)
	}

	if configPath == "" && info != nil {
		// Fallback check standard locations
		if sType == initsys.TypeUser {
			home, _ := os.UserHomeDir()
			configPath = filepath.Join(home, ".config", "systemd", "user", ensureServiceSuffix(info.Name))
		} else {
			configPath = filepath.Join("/etc/systemd/system", ensureServiceSuffix(info.Name))
		}
	}

	if configPath == "" {
		return nil, fmt.Errorf("could not determine configuration path for service %q", info.Name)
	}

	// If the file does not exist, check if directory exists or create parent
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to ensure config directory %s: %w", dir, err)
	}

	return &EditSession{
		FilePath: configPath,
		IsTemp:   false,
		OnSaved: func() error {
			// Trigger systemctl daemon-reload so changes are immediately active
			if sMgr, ok := mgr.(*systemd.Manager); ok {
				return sMgr.DaemonReload(ctx, sType)
			}
			args := []string{"daemon-reload"}
			if sType == initsys.TypeUser {
				args = append([]string{"--user"}, args...)
			}
			_ = exec.CommandContext(ctx, "systemctl", args...).Run()
			return nil
		},
		Cleanup: func() {},
	}, nil
}

// RunInteractive launches the editor in terminal mode
func RunInteractive(ctx context.Context, session *EditSession) error {
	defer session.Cleanup()

	editor := FindEditor()
	cmd := exec.CommandContext(ctx, editor, session.FilePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	if session.OnSaved != nil {
		if err := session.OnSaved(); err != nil {
			return fmt.Errorf("failed to apply changes after edit: %w", err)
		}
	}

	return nil
}

func ensureServiceSuffix(name string) string {
	if !strings.HasSuffix(name, ".service") && !strings.HasSuffix(name, ".timer") {
		return name + ".service"
	}
	return name
}
