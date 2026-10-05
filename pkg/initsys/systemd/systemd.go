package systemd

import (
	"alirun/pkg/initsys"
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

func init() {
	initsys.Register("systemd", New())
}

// Manager implements initsys.Manager for systemd
type Manager struct{}

// New creates a new systemd Manager
func New() *Manager {
	return &Manager{}
}

func (m *Manager) Name() string {
	return "systemd"
}

func (m *Manager) IsAvailable() bool {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return true
	}
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func (m *Manager) GetConfigPath(name string, sType initsys.ServiceType) string {
	unitName := ensureServiceSuffix(name)
	if sType == initsys.TypeSystem {
		return filepath.Join("/etc/systemd/system", unitName)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".config", "systemd", "user", unitName)
}

func (m *Manager) ListServices(ctx context.Context, sType initsys.ServiceType) ([]initsys.ServiceInfo, error) {
	// Try D-Bus first
	units, err := m.listViaDBus(ctx, sType)
	if err == nil && len(units) > 0 {
		return units, nil
	}

	// Fallback to systemctl CLI
	return m.listViaCLI(ctx, sType)
}

func (m *Manager) listViaDBus(ctx context.Context, sType initsys.ServiceType) ([]initsys.ServiceInfo, error) {
	var conn *dbus.Conn
	var err error

	if sType == initsys.TypeUser {
		conn, err = dbus.NewUserConnectionContext(ctx)
	} else {
		conn, err = dbus.NewSystemConnectionContext(ctx)
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	unitStatuses, err := conn.ListUnitsContext(ctx)
	if err != nil {
		return nil, err
	}

	var results []initsys.ServiceInfo
	for _, u := range unitStatuses {
		if !strings.HasSuffix(u.Name, ".service") {
			continue
		}

		status := mapActiveState(u.ActiveState)
		results = append(results, initsys.ServiceInfo{
			Name:        strings.TrimSuffix(u.Name, ".service"),
			Description: u.Description,
			Type:        sType,
			Status:      status,
			SubState:    u.SubState,
			InitSystem:  "systemd",
			ConfigPath:  m.GetConfigPath(u.Name, sType),
		})
	}

	return results, nil
}

func (m *Manager) listViaCLI(ctx context.Context, sType initsys.ServiceType) ([]initsys.ServiceInfo, error) {
	args := []string{"list-units", "--type=service", "--all", "--no-pager", "--plain", "--no-legend"}
	if sType == initsys.TypeUser {
		args = append([]string{"--user"}, args...)
	}

	cmd := exec.CommandContext(ctx, "systemctl", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("systemctl list-units failed: %w", err)
	}

	var results []initsys.ServiceInfo
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		unitFile := fields[0]
		if !strings.HasSuffix(unitFile, ".service") {
			continue
		}

		activeState := fields[2]
		subState := fields[3]
		desc := ""
		if len(fields) > 4 {
			desc = strings.Join(fields[4:], " ")
		}

		results = append(results, initsys.ServiceInfo{
			Name:        strings.TrimSuffix(unitFile, ".service"),
			Description: desc,
			Type:        sType,
			Status:      mapActiveState(activeState),
			SubState:    subState,
			InitSystem:  "systemd",
			ConfigPath:  m.GetConfigPath(unitFile, sType),
		})
	}

	return results, nil
}

func (m *Manager) GetStatus(ctx context.Context, name string, sType initsys.ServiceType) (*initsys.ServiceInfo, error) {
	unitName := ensureServiceSuffix(name)

	args := []string{"show", unitName, "--no-pager",
		"-p", "Id",
		"-p", "Description",
		"-p", "ActiveState",
		"-p", "SubState",
		"-p", "UnitFileState",
		"-p", "MainPID",
		"-p", "MemoryCurrent",
		"-p", "FragmentPath",
		"-p", "ExecStart",
		"-p", "ActiveEnterTimestamp",
	}
	if sType == initsys.TypeUser {
		args = append([]string{"--user"}, args...)
	}

	cmd := exec.CommandContext(ctx, "systemctl", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect service %s: %w", name, err)
	}

	info := &initsys.ServiceInfo{
		Name:       strings.TrimSuffix(unitName, ".service"),
		Type:       sType,
		InitSystem: "systemd",
		ConfigPath: m.GetConfigPath(unitName, sType),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], parts[1]

		switch key {
		case "Description":
			info.Description = val
		case "ActiveState":
			info.Status = mapActiveState(val)
		case "SubState":
			info.SubState = val
		case "UnitFileState":
			info.Enabled = (val == "enabled" || val == "static")
		case "MainPID":
			if pid, err := strconv.Atoi(val); err == nil {
				info.PID = pid
			}
		case "MemoryCurrent":
			if mem, err := strconv.ParseUint(val, 10, 64); err == nil && mem < 18446744073709551615 {
				info.MemoryBytes = mem
			}
		case "FragmentPath":
			if val != "" {
				info.ConfigPath = val
			}
		case "ExecStart":
			info.ExecPath = cleanExecStart(val)
		case "ActiveEnterTimestamp":
			if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", val); err == nil {
				info.ActiveSince = t
			}
		}
	}

	return info, nil
}

func (m *Manager) Start(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.runSystemctl(ctx, sType, "start", ensureServiceSuffix(name))
}

func (m *Manager) Stop(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.runSystemctl(ctx, sType, "stop", ensureServiceSuffix(name))
}

func (m *Manager) Restart(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.runSystemctl(ctx, sType, "restart", ensureServiceSuffix(name))
}

func (m *Manager) Enable(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.runSystemctl(ctx, sType, "enable", ensureServiceSuffix(name))
}

func (m *Manager) Disable(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.runSystemctl(ctx, sType, "disable", ensureServiceSuffix(name))
}

func (m *Manager) GenerateConfig(cfg initsys.ServiceConfig) (string, error) {
	return GenerateUnitFile(cfg)
}

func (m *Manager) InstallService(ctx context.Context, cfg initsys.ServiceConfig, content string, enableNow bool) (string, error) {
	unitPath := m.GetConfigPath(cfg.Name, cfg.Type)

	// Ensure destination directory exists
	dir := filepath.Dir(unitPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file atomically or directly
	if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write unit file %s: %w", unitPath, err)
	}

	// Daemon-reload
	if err := m.runSystemctl(ctx, cfg.Type, "daemon-reload"); err != nil {
		return unitPath, fmt.Errorf("service written, but daemon-reload failed: %w", err)
	}

	if enableNow {
		unitName := ensureServiceSuffix(cfg.Name)
		if err := m.runSystemctl(ctx, cfg.Type, "enable", "--now", unitName); err != nil {
			return unitPath, fmt.Errorf("service created, but failed to enable --now: %w", err)
		}
	}

	return unitPath, nil
}

func (m *Manager) DeleteService(ctx context.Context, name string, sType initsys.ServiceType) error {
	unitName := ensureServiceSuffix(name)
	unitPath := m.GetConfigPath(name, sType)

	// Stop and disable (ignore error if already stopped/disabled)
	_ = m.runSystemctl(ctx, sType, "stop", unitName)
	_ = m.runSystemctl(ctx, sType, "disable", unitName)

	// Remove file if exists
	if _, err := os.Stat(unitPath); err == nil {
		if err := os.Remove(unitPath); err != nil {
			return fmt.Errorf("failed to delete unit file %s: %w", unitPath, err)
		}
	}

	// Reload daemon
	_ = m.runSystemctl(ctx, sType, "daemon-reload")
	_ = m.runSystemctl(ctx, sType, "reset-failed")

	return nil
}

func (m *Manager) StreamLogs(ctx context.Context, name string, sType initsys.ServiceType, lines int, follow bool) (<-chan string, error) {
	unitName := ensureServiceSuffix(name)
	args := []string{"-u", unitName, "--no-pager"}

	if lines > 0 {
		args = append(args, "-n", strconv.Itoa(lines))
	} else {
		args = append(args, "-n", "50")
	}

	if follow {
		args = append(args, "-f")
	}

	if sType == initsys.TypeUser {
		args = append([]string{"--user"}, args...)
	}

	cmd := exec.CommandContext(ctx, "journalctl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open stdout pipe for journalctl: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start journalctl: %w", err)
	}

	logChan := make(chan string, 100)

	go func() {
		defer close(logChan)
		defer cmd.Wait()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case logChan <- scanner.Text():
			}
		}
	}()

	return logChan, nil
}

func (m *Manager) runSystemctl(ctx context.Context, sType initsys.ServiceType, subcmd string, extraArgs ...string) error {
	args := []string{subcmd}
	if sType == initsys.TypeUser {
		args = append([]string{"--user"}, args...)
	}
	args = append(args, extraArgs...)

	cmd := exec.CommandContext(ctx, "systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s failed: %w (output: %s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func ensureServiceSuffix(name string) string {
	if !strings.HasSuffix(name, ".service") {
		return name + ".service"
	}
	return name
}

func mapActiveState(state string) initsys.ServiceStatus {
	switch strings.ToLower(state) {
	case "active", "running":
		return initsys.StatusActive
	case "inactive", "dead":
		return initsys.StatusInactive
	case "failed":
		return initsys.StatusFailed
	default:
		return initsys.StatusUnknown
	}
}

func cleanExecStart(val string) string {
	// systemctl show format for ExecStart: { path=/usr/bin/foo ; argv[]=/usr/bin/foo --arg ; ... }
	if strings.Contains(val, "path=") {
		parts := strings.Split(val, ";")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if strings.HasPrefix(trimmed, "argv[]=") {
				return strings.TrimPrefix(trimmed, "argv[]=")
			}
			if strings.HasPrefix(trimmed, "path=") {
				return strings.TrimPrefix(trimmed, "path=")
			}
		}
	}
	return val
}
