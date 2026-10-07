package systemd

import (
	"alirun/pkg/initsys"
	"alirun/pkg/netinfo"
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

	timersMap := m.getActiveTimersMap(ctx, sType)
	bulkMetrics := m.getBulkMetricsMap(ctx, sType)

	var results []initsys.ServiceInfo
	for _, u := range unitStatuses {
		if !strings.HasSuffix(u.Name, ".service") {
			continue
		}

		status := mapActiveState(u.ActiveState)
		svc := initsys.ServiceInfo{
			Name:        strings.TrimSuffix(u.Name, ".service"),
			Description: u.Description,
			Type:        sType,
			Status:      status,
			SubState:    u.SubState,
			InitSystem:  "systemd",
			ConfigPath:  m.GetConfigPath(u.Name, sType),
		}
		if next, ok := timersMap[svc.Name]; ok {
			svc.IsTimer = true
			svc.TimerNext = next
		}
		if bm, ok := bulkMetrics[svc.Name]; ok {
			svc.UnitFileState = bm.UnitFileState
			svc.Enabled = isEnabledState(bm.UnitFileState)
			svc.PID = bm.PID
			svc.MemoryBytes = bm.MemoryBytes
			svc.CPUUsageNSec = bm.CPUUsageNSec
			svc.ActiveSince = bm.ActiveSince
		}
		results = append(results, svc)
	}

	return results, nil
}

type unitBulkMetrics struct {
	UnitFileState string
	PID           int
	MemoryBytes   uint64
	CPUUsageNSec  uint64
	ActiveSince   time.Time
}

func (m *Manager) getBulkMetricsMap(ctx context.Context, sType initsys.ServiceType) map[string]unitBulkMetrics {
	metricsMap := make(map[string]unitBulkMetrics)
	args := []string{"show", "*.service", "--no-pager",
		"-p", "Id",
		"-p", "UnitFileState",
		"-p", "MainPID",
		"-p", "MemoryCurrent",
		"-p", "CPUUsageNSec",
		"-p", "ActiveEnterTimestamp",
	}
	if sType == initsys.TypeUser {
		args = append([]string{"--user"}, args...)
	}

	out, err := exec.CommandContext(ctx, "systemctl", args...).Output()
	if err != nil {
		return metricsMap
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var curId string
	var curMetrics unitBulkMetrics

	commitCurrent := func() {
		if curId != "" {
			name := strings.TrimSuffix(curId, ".service")
			metricsMap[name] = curMetrics
			curId = ""
			curMetrics = unitBulkMetrics{}
		}
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			commitCurrent()
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], parts[1]
		switch key {
		case "Id":
			curId = val
		case "UnitFileState":
			curMetrics.UnitFileState = val
		case "MainPID":
			if pid, err := strconv.Atoi(val); err == nil {
				curMetrics.PID = pid
			}
		case "MemoryCurrent":
			if mem, err := strconv.ParseUint(val, 10, 64); err == nil && mem < 18446744073709551615 {
				curMetrics.MemoryBytes = mem
			}
		case "CPUUsageNSec":
			if cpu, err := strconv.ParseUint(val, 10, 64); err == nil && cpu < 18446744073709551615 {
				curMetrics.CPUUsageNSec = cpu
			}
		case "ActiveEnterTimestamp":
			if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", val); err == nil {
				curMetrics.ActiveSince = t
			}
		}
	}
	commitCurrent()

	return metricsMap
}

func (m *Manager) getActiveTimersMap(ctx context.Context, sType initsys.ServiceType) map[string]string {
	timerMap := make(map[string]string)
	args := []string{"list-timers", "--no-legend", "--full"}
	if sType == initsys.TypeUser {
		args = append([]string{"--user"}, args...)
	}
	out, err := exec.CommandContext(ctx, "systemctl", args...).Output()
	if err != nil {
		return timerMap
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			activatedUnit := fields[len(fields)-1]
			svcName := strings.TrimSuffix(activatedUnit, ".service")
			timerMap[svcName] = formatTimerNext(fields)
		}
	}
	return timerMap
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

	timersMap := m.getActiveTimersMap(ctx, sType)
	bulkMetrics := m.getBulkMetricsMap(ctx, sType)

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

		svc := initsys.ServiceInfo{
			Name:        strings.TrimSuffix(unitFile, ".service"),
			Description: desc,
			Type:        sType,
			Status:      mapActiveState(activeState),
			SubState:    subState,
			InitSystem:  "systemd",
			ConfigPath:  m.GetConfigPath(unitFile, sType),
		}
		if next, ok := timersMap[svc.Name]; ok {
			svc.IsTimer = true
			svc.TimerNext = next
		}
		if bm, ok := bulkMetrics[svc.Name]; ok {
			svc.UnitFileState = bm.UnitFileState
			svc.Enabled = isEnabledState(bm.UnitFileState)
			svc.PID = bm.PID
			svc.MemoryBytes = bm.MemoryBytes
			svc.CPUUsageNSec = bm.CPUUsageNSec
			svc.ActiveSince = bm.ActiveSince
		}
		results = append(results, svc)
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
		"-p", "CPUUsageNSec",
		"-p", "TasksCurrent",
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
			info.UnitFileState = val
			info.Enabled = isEnabledState(val)
		case "MainPID":
			if pid, err := strconv.Atoi(val); err == nil {
				info.PID = pid
			}
		case "MemoryCurrent":
			if mem, err := strconv.ParseUint(val, 10, 64); err == nil && mem < 18446744073709551615 {
				info.MemoryBytes = mem
			}
		case "CPUUsageNSec":
			if cpu, err := strconv.ParseUint(val, 10, 64); err == nil && cpu < 18446744073709551615 {
				info.CPUUsageNSec = cpu
			}
		case "TasksCurrent":
			if tasks, err := strconv.ParseUint(val, 10, 64); err == nil && tasks < 18446744073709551615 {
				info.TasksCurrent = tasks
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

	// Check if a timer exists or is active for this service
	timerPath := m.GetTimerConfigPath(name, sType)
	if _, err := os.Stat(timerPath); err == nil {
		info.IsTimer = true
	}

	timerName := strings.TrimSuffix(unitName, ".service") + ".timer"
	timerArgs := []string{"list-timers", "--no-legend", "--full", timerName}
	if sType == initsys.TypeUser {
		timerArgs = append([]string{"--user"}, timerArgs...)
	}
	if timerOut, err := exec.CommandContext(ctx, "systemctl", timerArgs...).Output(); err == nil {
		timerText := strings.TrimSpace(string(timerOut))
		if timerText != "" {
			info.IsTimer = true
			info.TimerNext = formatTimerNext(strings.Fields(timerText))
		}
	}

	// Inspect network activity if process is running
	if info.PID > 0 {
		netSum := netinfo.GetNetSummaryForPID(info.PID)
		info.Ports = netSum.Listening
		info.NetSummary = netSum.FormatSummary()
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

func (m *Manager) GetTimerConfigPath(name string, sType initsys.ServiceType) string {
	timerName := strings.TrimSuffix(name, ".service")
	if !strings.HasSuffix(timerName, ".timer") {
		timerName += ".timer"
	}
	if sType == initsys.TypeSystem {
		return filepath.Join("/etc/systemd/system", timerName)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".config", "systemd", "user", timerName)
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

	// If timer preset, also generate and write the corresponding .timer unit
	if cfg.Preset == initsys.PresetTimer {
		timerPath := m.GetTimerConfigPath(cfg.Name, cfg.Type)
		timerContent, err := GenerateTimerFile(cfg)
		if err != nil {
			return unitPath, fmt.Errorf("service written, but timer generation failed: %w", err)
		}
		if err := os.WriteFile(timerPath, []byte(timerContent), 0644); err != nil {
			return unitPath, fmt.Errorf("service written, but failed to write timer %s: %w", timerPath, err)
		}
	}

	// Daemon-reload
	if err := m.runSystemctl(ctx, cfg.Type, "daemon-reload"); err != nil {
		return unitPath, fmt.Errorf("service written, but daemon-reload failed: %w", err)
	}

	if enableNow {
		if cfg.Preset == initsys.PresetTimer {
			timerName := strings.TrimSuffix(cfg.Name, ".service") + ".timer"
			if err := m.runSystemctl(ctx, cfg.Type, "enable", "--now", timerName); err != nil {
				return unitPath, fmt.Errorf("timer created, but failed to enable --now: %w", err)
			}
		} else {
			unitName := ensureServiceSuffix(cfg.Name)
			if err := m.runSystemctl(ctx, cfg.Type, "enable", "--now", unitName); err != nil {
				return unitPath, fmt.Errorf("service created, but failed to enable --now: %w", err)
			}
		}
	}

	return unitPath, nil
}

func (m *Manager) DeleteService(ctx context.Context, name string, sType initsys.ServiceType) error {
	unitName := ensureServiceSuffix(name)
	unitPath := m.GetConfigPath(name, sType)
	timerPath := m.GetTimerConfigPath(name, sType)

	// If timer exists, stop and disable it
	if _, err := os.Stat(timerPath); err == nil {
		timerName := strings.TrimSuffix(name, ".service") + ".timer"
		_ = m.runSystemctl(ctx, sType, "stop", timerName)
		_ = m.runSystemctl(ctx, sType, "disable", timerName)
		_ = os.Remove(timerPath)
	}

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

// DaemonReload reloads the systemd daemon configuration
func (m *Manager) DaemonReload(ctx context.Context, sType initsys.ServiceType) error {
	return m.runSystemctl(ctx, sType, "daemon-reload")
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

// unitSuffixes lists unit types that must not get an extra ".service" appended
// (e.g. "backup.timer" must stay "backup.timer", not become "backup.timer.service").
var unitSuffixes = []string{".service", ".timer", ".socket", ".path", ".target", ".mount"}

var weekdayAbbrevs = map[string]bool{
	"Mon": true, "Tue": true, "Wed": true, "Thu": true, "Fri": true, "Sat": true, "Sun": true,
}

// formatTimerNext converts one "systemctl list-timers --no-legend" row
// (NEXT LEFT LAST PASSED UNIT ACTIVATES) into "Thu 2026-10-08 00:00:00 (1h 35min left)".
// NEXT is either "-"/"n/a" or "Day Date Time TZ"; LEFT can consist of several words
// ("1h 35min"), so it is read up to the beginning of the LAST column.
func formatTimerNext(fields []string) string {
	if len(fields) == 0 || fields[0] == "-" || fields[0] == "n/a" {
		return ""
	}
	if len(fields) < 4 {
		return strings.Join(fields, " ")
	}
	next := strings.Join(fields[:3], " ")

	// The last two columns are always UNIT and ACTIVATES
	end := len(fields) - 2
	var left []string
	for i := 4; i < end; i++ {
		if weekdayAbbrevs[fields[i]] || fields[i] == "-" || fields[i] == "n/a" {
			break // start of the LAST column
		}
		if fields[i] == "left" {
			continue // older systemd versions print "10min left" in the LEFT column
		}
		left = append(left, fields[i])
	}
	if len(left) == 0 {
		return next
	}
	return fmt.Sprintf("%s (%s left)", next, strings.Join(left, " "))
}

func ensureServiceSuffix(name string) string {
	for _, suffix := range unitSuffixes {
		if strings.HasSuffix(name, suffix) {
			return name
		}
	}
	return name + ".service"
}

func isEnabledState(state string) bool {
	return state == "enabled" || state == "static"
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
