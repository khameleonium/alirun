package cron

import (
	"alirun/pkg/initsys"
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// CronManager implements initsys.Manager for classic Linux crontab
type CronManager struct {
	mu sync.Mutex
}

// NewCronManager creates a new CronManager
func NewCronManager() *CronManager {
	return &CronManager{}
}

func init() {
	initsys.Register("cron", NewCronManager())
}

func (m *CronManager) Name() string {
	return "cron"
}

func (m *CronManager) IsAvailable() bool {
	if _, err := exec.LookPath("crontab"); err == nil {
		return true
	}
	if _, err := os.Stat("/etc/crontab"); err == nil {
		return true
	}
	return false
}

// ListServices returns crontab entries mapped into initsys.ServiceInfo
func (m *CronManager) ListServices(ctx context.Context, sType initsys.ServiceType) ([]initsys.ServiceInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var allJobs []*CronJob

	if sType == initsys.TypeUser {
		content, err := ReadUserCrontab()
		if err != nil {
			return nil, err
		}
		jobs, err := ParseCrontab(content, "crontab (user)", false)
		if err != nil {
			return nil, err
		}
		allJobs = append(allJobs, jobs...)
	} else {
		// System mode: read /etc/crontab and /etc/cron.d/*
		if data, err := os.ReadFile("/etc/crontab"); err == nil {
			jobs, _ := ParseCrontab(string(data), "/etc/crontab", true)
			allJobs = append(allJobs, jobs...)
		}

		cronDFiles, _ := filepath.Glob("/etc/cron.d/*")
		for _, f := range cronDFiles {
			base := filepath.Base(f)
			if strings.HasPrefix(base, ".") {
				continue
			}
			if data, err := os.ReadFile(f); err == nil {
				jobs, _ := ParseCrontab(string(data), f, true)
				allJobs = append(allJobs, jobs...)
			}
		}
	}

	var services []initsys.ServiceInfo
	for i, j := range allJobs {
		j.ID = fmt.Sprintf("cron-%d", i+1)
		st := initsys.StatusActive
		if !j.Enabled {
			st = initsys.StatusInactive
		}

		nextStr := "-"
		if !j.NextRun.IsZero() {
			nextStr = j.NextRun.Format("2006-01-02 15:04")
		} else if strings.EqualFold(j.Schedule, "@reboot") {
			nextStr = "At system boot"
		}

		desc := fmt.Sprintf("[%s] %s", j.Schedule, j.HumanSchedule)
		if j.Comment != "" {
			desc = fmt.Sprintf("%s — [%s] %s", j.Comment, j.Schedule, j.HumanSchedule)
		}

		fullName := fmt.Sprintf("[%s] %s", j.ID, j.DisplayName())

		services = append(services, initsys.ServiceInfo{
			Name:        fullName,
			Description: desc,
			Type:        sType,
			Status:      st,
			SubState:    j.HumanSchedule,
			Enabled:     j.Enabled,
			IsTimer:     true,
			TimerNext:   nextStr,
			ConfigPath:  j.File,
			ExecPath:    j.Command,
		})
	}

	return services, nil
}

// GetStatus returns status for a specific cron job
func (m *CronManager) GetStatus(ctx context.Context, name string, sType initsys.ServiceType) (*initsys.ServiceInfo, error) {
	services, err := m.ListServices(ctx, sType)
	if err != nil {
		return nil, err
	}

	target := strings.ToLower(strings.TrimSpace(name))
	for _, s := range services {
		sLower := strings.ToLower(s.Name)
		if sLower == target || strings.Contains(sLower, target) || strings.Contains(target, sLower) {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("cron job %q not found", name)
}

// Start executes the cron job command immediately
func (m *CronManager) Start(ctx context.Context, name string, sType initsys.ServiceType) error {
	info, err := m.GetStatus(ctx, name, sType)
	if err != nil {
		return err
	}
	_, err = RunJobCommand(ctx, info.ExecPath)
	return err
}

// Stop disables the cron job
func (m *CronManager) Stop(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.Disable(ctx, name, sType)
}

// Restart executes the cron job immediately
func (m *CronManager) Restart(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.Start(ctx, name, sType)
}

// Enable uncomments the cron job
func (m *CronManager) Enable(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.setJobEnabled(ctx, name, sType, true)
}

// Disable comments out the cron job
func (m *CronManager) Disable(ctx context.Context, name string, sType initsys.ServiceType) error {
	return m.setJobEnabled(ctx, name, sType, false)
}

func (m *CronManager) setJobEnabled(ctx context.Context, name string, sType initsys.ServiceType, enable bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sType != initsys.TypeUser {
		return fmt.Errorf("modifying system crontab requires root privileges directly on the file")
	}

	content, err := ReadUserCrontab()
	if err != nil {
		return err
	}

	jobs, err := ParseCrontab(content, "crontab (user)", false)
	if err != nil {
		return err
	}

	target := strings.ToLower(strings.TrimSpace(name))
	var matchedJob *CronJob
	for _, j := range jobs {
		fullName := strings.ToLower(fmt.Sprintf("[%s] %s", j.ID, j.DisplayName()))
		if fullName == target || strings.Contains(fullName, target) || strings.Contains(target, strings.ToLower(j.ID)) {
			matchedJob = j
			break
		}
	}

	if matchedJob == nil {
		return fmt.Errorf("cron job %q not found", name)
	}

	if matchedJob.Enabled == enable {
		return nil // Already in desired state
	}

	newContent, err := ToggleJobLine(content, matchedJob.LineIndex)
	if err != nil {
		return err
	}

	return SaveUserCrontab(newContent)
}

// DeleteService removes a cron job from crontab
func (m *CronManager) DeleteService(ctx context.Context, name string, sType initsys.ServiceType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sType != initsys.TypeUser {
		return fmt.Errorf("deleting system crontab entries requires editing system file with root privileges")
	}

	content, err := ReadUserCrontab()
	if err != nil {
		return err
	}

	jobs, err := ParseCrontab(content, "crontab (user)", false)
	if err != nil {
		return err
	}

	target := strings.ToLower(strings.TrimSpace(name))
	var matchedJob *CronJob
	for _, j := range jobs {
		fullName := strings.ToLower(fmt.Sprintf("[%s] %s", j.ID, j.DisplayName()))
		if fullName == target || strings.Contains(fullName, target) || strings.Contains(target, strings.ToLower(j.ID)) {
			matchedJob = j
			break
		}
	}

	if matchedJob == nil {
		return fmt.Errorf("cron job %q not found", name)
	}

	newContent, err := DeleteJobLine(content, matchedJob.LineIndex)
	if err != nil {
		return err
	}

	return SaveUserCrontab(newContent)
}

// GenerateConfig generates a crontab entry text preview
func (m *CronManager) GenerateConfig(cfg initsys.ServiceConfig) (string, error) {
	schedule := cfg.TimerSchedule
	if schedule == "" {
		schedule = "0 * * * *" // Hourly default
	}

	var sb strings.Builder
	if cfg.Description != "" {
		sb.WriteString("# ")
		sb.WriteString(cfg.Description)
		sb.WriteString("\n")
	}
	sb.WriteString(schedule)
	sb.WriteString(" ")
	sb.WriteString(cfg.ExecStart)
	sb.WriteString("\n")

	return sb.String(), nil
}

// GetConfigPath returns destination crontab identifier
func (m *CronManager) GetConfigPath(name string, sType initsys.ServiceType) string {
	if sType == initsys.TypeUser {
		return "crontab (user)"
	}
	return "/etc/crontab"
}

// InstallService adds a new cron job into crontab
func (m *CronManager) InstallService(ctx context.Context, cfg initsys.ServiceConfig, content string, enableNow bool) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.Type != initsys.TypeUser {
		return "", fmt.Errorf("installing system cron job requires editing /etc/crontab with root")
	}

	currentContent, err := ReadUserCrontab()
	if err != nil {
		return "", err
	}

	schedule := cfg.TimerSchedule
	if schedule == "" {
		schedule = "0 * * * *"
	}

	newContent := AppendJob(currentContent, schedule, cfg.ExecStart, cfg.Description)
	if err := SaveUserCrontab(newContent); err != nil {
		return "", err
	}

	if enableNow {
		go func() {
			_, _ = RunJobCommand(context.Background(), cfg.ExecStart)
		}()
	}

	return "crontab (user)", nil
}

// StreamLogs reads cron logs from systemd journal
func (m *CronManager) StreamLogs(ctx context.Context, name string, sType initsys.ServiceType, lines int, follow bool) (<-chan string, error) {
	ch := make(chan string, 100)

	go func() {
		defer close(ch)

		// Try journalctl first
		args := []string{"-u", "cron", "-n", fmt.Sprintf("%d", lines), "--no-pager"}
		if follow {
			args = append(args, "-f")
		}

		cmd := exec.CommandContext(ctx, "journalctl", args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			ch <- fmt.Sprintf("Error streaming logs: %v", err)
			return
		}

		if err := cmd.Start(); err != nil {
			ch <- fmt.Sprintf("Error starting journalctl: %v", err)
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		_ = cmd.Wait()
	}()

	return ch, nil
}
