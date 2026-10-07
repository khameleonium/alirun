package systemd

import (
	"alirun/pkg/initsys"
	"strings"
	"testing"
)

func TestIsTimeSpan(t *testing.T) {
	spans := []string{"15m", "15min", "30sec", "1d", "2w", "1h 30min", "1M", "90s"}
	cals := []string{"hourly", "daily", "minutely", "*-*-* 03:00:00", "*-*-* 03:00 Europe/Paris", "Mon *-*-* 10:00"}
	for _, s := range spans {
		if !IsTimeSpan(s) {
			t.Errorf("%q should be a time span", s)
		}
	}
	for _, s := range cals {
		if IsTimeSpan(s) {
			t.Errorf("%q should be a calendar expression", s)
		}
	}

	out, err := GenerateTimerFile(initsys.ServiceConfig{Name: "x", TimerSchedule: "15min"})
	if err != nil || !strings.Contains(out, "OnUnitActiveSec=15min") {
		t.Errorf("15min must become OnUnitActiveSec, got:\n%s", out)
	}
}

func TestEnsureServiceSuffixKeepsTimers(t *testing.T) {
	cases := map[string]string{"a": "a.service", "a.service": "a.service", "a.timer": "a.timer", "a.socket": "a.socket"}
	for in, want := range cases {
		if got := ensureServiceSuffix(in); got != want {
			t.Errorf("ensureServiceSuffix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatTimerNext(t *testing.T) {
	cases := map[string]string{
		"Wed 2026-10-07 23:25:18 +10  1h 1min Wed 2026-10-07 22:17:02 +10  7min ago fwupd-refresh.timer fwupd-refresh.service": "Wed 2026-10-07 23:25:18 (1h 1min left)",
		"Wed 2026-10-07 22:34:43 +10    10min Wed 2026-10-07 21:42:16 +10         - anacron.timer anacron.service":             "Wed 2026-10-07 22:34:43 (10min left)",
		"Thu 2026-10-08 06:27:37 +10 8h - - motd-news.timer motd-news.service":                                                 "Thu 2026-10-08 06:27:37 (8h left)",
		"- - Wed 2026-10-07 21:42:16 +10 1h ago x.timer x.service":                                                             "",
	}
	for line, want := range cases {
		if got := formatTimerNext(strings.Fields(line)); got != want {
			t.Errorf("formatTimerNext(%q) = %q, want %q", line, got, want)
		}
	}
}
