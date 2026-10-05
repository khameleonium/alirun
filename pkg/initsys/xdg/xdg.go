package xdg

import (
	"alirun/pkg/initsys"
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func init() {
	initsys.Register("xdg", New())
}

// Manager manages XDG desktop autostart files (~/.config/autostart/*.desktop)
type Manager struct{}

// New creates a new XDG autostart manager
func New() *Manager {
	return &Manager{}
}

func (m *Manager) Name() string {
	return "xdg"
}

func (m *Manager) IsAvailable() bool {
	// Available if user has a HOME directory
	home, err := os.UserHomeDir()
	return err == nil && home != ""
}

func (m *Manager) GetConfigPath(name string, sType initsys.ServiceType) string {
	desktopName := ensureDesktopSuffix(name)
	if sType == initsys.TypeSystem {
		return filepath.Join("/etc", "xdg", "autostart", desktopName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "autostart", desktopName)
}

func (m *Manager) ListServices(ctx context.Context, sType initsys.ServiceType) ([]initsys.ServiceInfo, error) {
	var dirs []string
	if sType == initsys.TypeSystem {
		dirs = append(dirs, "/etc/xdg/autostart")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, filepath.Join(home, ".config", "autostart"))
	}

	var results []initsys.ServiceInfo
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".desktop") {
				continue
			}
			fullPath := filepath.Join(dir, entry.Name())
			name := strings.TrimSuffix(entry.Name(), ".desktop")

			info, err := m.parseDesktopFile(fullPath, name, sType)
			if err == nil {
				results = append(results, *info)
			}
		}
	}

	return results, nil
}

func (m *Manager) GetStatus(ctx context.Context, name string, sType initsys.ServiceType) (*initsys.ServiceInfo, error) {
	fullPath := m.GetConfigPath(name, sType)
	if _, err := os.Stat(fullPath); err != nil {
		return nil, fmt.Errorf("desktop autostart file not found: %s", fullPath)
	}
	return m.parseDesktopFile(fullPath, name, sType)
}

func (m *Manager) Start(ctx context.Context, name string, sType initsys.ServiceType) error {
	info, err := m.GetStatus(ctx, name, sType)
	if err != nil {
		return err
	}
	if info.ExecPath == "" {
		return fmt.Errorf("no Exec path defined in %s", name)
	}

	// Clean Exec command (strip %u, %f, etc. from XDG spec)
	cmdClean := cleanDesktopExec(info.ExecPath)
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdClean+" &")
	return cmd.Start()
}

func cleanDesktopExec(execStr string) string {
	parts := strings.Fields(execStr)
	var clean []string
	for _, p := range parts {
		if strings.HasPrefix(p, "%") && len(p) == 2 {
			continue // skip %f, %F, %u, %U, %i, %c, %k
		}
		clean = append(clean, p)
	}
	return strings.Join(clean, " ")
}

func (m *Manager) Stop(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.Disable(ctx, name, sType)
}

func (m *Manager) Restart(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.Start(ctx, name, sType)
}

func (m *Manager) Enable(ctx context.Context, name string, sType initsys.ServiceType) error {
	// Set Hidden=false or X-GNOME-Autostart-enabled=true in desktop file
	fullPath := m.GetConfigPath(name, sType)
	return m.toggleEnabled(fullPath, true)
}

func (m *Manager) Disable(ctx context.Context, name string, sType initsys.ServiceType) error {
	fullPath := m.GetConfigPath(name, sType)
	return m.toggleEnabled(fullPath, false)
}

func (m *Manager) GenerateConfig(cfg initsys.ServiceConfig) (string, error) {
	desc := cfg.Description
	if desc == "" {
		desc = cfg.Name
	}

	var sb strings.Builder
	sb.WriteString("[Desktop Entry]\n")
	sb.WriteString("Type=Application\n")
	sb.WriteString(fmt.Sprintf("Name=%s\n", cfg.Name))
	sb.WriteString(fmt.Sprintf("Comment=%s\n", desc))
	sb.WriteString(fmt.Sprintf("Exec=%s\n", cfg.ExecStart))
	if cfg.WorkingDirectory != "" {
		sb.WriteString(fmt.Sprintf("Path=%s\n", cfg.WorkingDirectory))
	}
	sb.WriteString("Terminal=false\n")
	sb.WriteString("Hidden=false\n")
	sb.WriteString("X-GNOME-Autostart-enabled=true\n")

	return sb.String(), nil
}

func (m *Manager) InstallService(ctx context.Context, cfg initsys.ServiceConfig, content string, enableNow bool) (string, error) {
	path := m.GetConfigPath(cfg.Name, cfg.Type)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func (m *Manager) DeleteService(ctx context.Context, name string, sType initsys.ServiceType) error {
	path := m.GetConfigPath(name, sType)
	if _, err := os.Stat(path); err == nil {
		return os.Remove(path)
	}
	return nil
}

func (m *Manager) StreamLogs(ctx context.Context, name string, sType initsys.ServiceType, lines int, follow bool) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "[XDG autostart logs are handled by your desktop display manager / xsession-errors]"
	close(ch)
	return ch, nil
}

func (m *Manager) parseDesktopFile(path, name string, sType initsys.ServiceType) (*initsys.ServiceInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info := &initsys.ServiceInfo{
		Name:       strings.TrimSuffix(name, ".desktop"),
		Type:       sType,
		Enabled:    true,
		ConfigPath: path,
		InitSystem: "xdg",
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch k {
		case "Name":
			info.Description = v
		case "Exec":
			info.ExecPath = v
		case "Hidden":
			if v == "true" {
				info.Enabled = false
			}
		case "X-GNOME-Autostart-enabled":
			if v == "false" {
				info.Enabled = false
			}
		}
	}

	if info.Enabled {
		info.Status = initsys.StatusActive
		info.SubState = "autostart"
	} else {
		info.Status = initsys.StatusInactive
		info.SubState = "hidden"
	}

	return info, nil
}

func (m *Manager) toggleEnabled(path string, enable bool) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	foundHidden := false
	foundGnome := false

	hiddenVal := "false"
	gnomeVal := "true"
	if !enable {
		hiddenVal = "true"
		gnomeVal = "false"
	}

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "Hidden=") {
			newLines = append(newLines, "Hidden="+hiddenVal)
			foundHidden = true
		} else if strings.HasPrefix(trimmed, "X-GNOME-Autostart-enabled=") {
			newLines = append(newLines, "X-GNOME-Autostart-enabled="+gnomeVal)
			foundGnome = true
		} else {
			newLines = append(newLines, l)
		}
	}

	if !foundHidden {
		newLines = append(newLines, "Hidden="+hiddenVal)
	}
	if !foundGnome {
		newLines = append(newLines, "X-GNOME-Autostart-enabled="+gnomeVal)
	}

	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}

func ensureDesktopSuffix(name string) string {
	if !strings.HasSuffix(name, ".desktop") {
		return name + ".desktop"
	}
	return name
}
