package cron

import (
	"alirun/pkg/initsys"
	"context"
	"testing"
)

func TestCronManagerRegistered(t *testing.T) {
	mgr, err := initsys.Get("cron")
	if err != nil {
		t.Fatalf("cron manager should be registered: %v", err)
	}

	if mgr.Name() != "cron" {
		t.Errorf("expected name 'cron', got %s", mgr.Name())
	}
}

func TestCronManagerGenerateConfig(t *testing.T) {
	mgr := NewCronManager()
	cfg := initsys.ServiceConfig{
		Description:   "Database backup",
		ExecStart:     "/opt/backup.sh",
		TimerSchedule: "30 2 * * *",
	}

	content, err := mgr.GenerateConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "# Database backup\n30 2 * * * /opt/backup.sh\n"
	if content != expected {
		t.Errorf("expected %q, got %q", expected, content)
	}
}

func TestCronManagerListServicesSystem(t *testing.T) {
	mgr := NewCronManager()
	// Test listing system cron jobs from /etc/crontab and /etc/cron.d
	services, err := mgr.ListServices(context.Background(), initsys.TypeSystem)
	if err != nil {
		t.Fatalf("unexpected error listing system cron jobs: %v", err)
	}

	// /etc/crontab usually has at least 1-4 standard jobs on Linux systems
	t.Logf("Found %d system cron jobs", len(services))
	for _, s := range services {
		if !s.IsTimer {
			t.Errorf("cron job should be marked as timer")
		}
	}
}
