package cron

import (
	"testing"
)

func TestParseUserCrontab(t *testing.T) {
	crontabData := `
# Environment variables
SHELL=/bin/bash
PATH=/usr/local/bin:/usr/bin:/bin

# Daily backup script
0 3 * * * /home/user/backup.sh > /dev/null 2>&1

# Temporarily disabled crawler
# */10 * * * * /home/user/crawler.py

@reboot /home/user/on_boot.sh
`

	jobs, err := ParseCrontab(crontabData, "crontab (user)", false)
	if err != nil {
		t.Fatalf("unexpected error parsing crontab: %v", err)
	}

	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}

	// 1. Daily backup
	j1 := jobs[0]
	if !j1.Enabled {
		t.Errorf("job 1 should be enabled")
	}
	if j1.Schedule != "0 3 * * *" {
		t.Errorf("expected schedule '0 3 * * *', got %q", j1.Schedule)
	}
	if j1.Comment != "Daily backup script" {
		t.Errorf("expected comment 'Daily backup script', got %q", j1.Comment)
	}
	if j1.Command != "/home/user/backup.sh > /dev/null 2>&1" {
		t.Errorf("expected command, got %q", j1.Command)
	}

	// 2. Disabled crawler
	j2 := jobs[1]
	if j2.Enabled {
		t.Errorf("job 2 should be disabled")
	}
	if j2.Schedule != "*/10 * * * *" {
		t.Errorf("expected schedule '*/10 * * * *', got %q", j2.Schedule)
	}

	// 3. Macro reboot
	j3 := jobs[2]
	if !j3.Enabled || j3.Schedule != "@reboot" {
		t.Errorf("expected enabled @reboot, got %v (%s)", j3.Enabled, j3.Schedule)
	}
}

func TestParseSystemCrontab(t *testing.T) {
	systemCrontab := `
SHELL=/bin/sh
PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin

# m h dom mon dow user  command
17 *	* * *	root    cd / && run-parts --report /etc/cron.hourly
25 6	* * *	root    test -x /usr/sbin/anacron || { cd / && run-parts --report /etc/cron.daily; }
`

	jobs, err := ParseCrontab(systemCrontab, "/etc/crontab", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

	if jobs[0].User != "root" || jobs[0].Schedule != "17 * * * *" {
		t.Errorf("expected root and '17 * * * *', got %s, %s", jobs[0].User, jobs[0].Schedule)
	}
}
