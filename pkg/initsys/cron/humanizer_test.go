package cron

import (
	"testing"
	"time"
)

func TestHumanizeSchedule(t *testing.T) {
	tests := []struct {
		expr     string
		contains string
	}{
		{"@reboot", "При загрузке системы"},
		{"@daily", "Ежедневно в 00:00"},
		{"* * * * *", "Каждую минуту"},
		{"*/15 * * * *", "Каждые 15 мин."},
		{"30 2 * * *", "Ежедневно в 02:30"},
		{"0 12 * * 1-5", "По будням в 12:00"},
		{"0 9 * * 0", "По воскресеньям в 09:00"},
		{"15 3 1 * *", "Каждое 1-е число в 03:15"},
	}

	for _, tc := range tests {
		res := HumanizeSchedule(tc.expr)
		if !testing.Short() && len(res) == 0 {
			t.Errorf("empty result for %s", tc.expr)
		}
	}
}

func TestNextRun(t *testing.T) {
	// Starting at 2026-10-05 14:00:00
	from := time.Date(2026, 10, 5, 14, 0, 0, 0, time.UTC)

	// 1. Every 15 minutes: next should be 14:15
	next := NextRun("*/15 * * * *", from)
	expected := time.Date(2026, 10, 5, 14, 15, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}

	// 2. Daily at 15:30: next should be 15:30 today
	nextDaily := NextRun("30 15 * * *", from)
	expectedDaily := time.Date(2026, 10, 5, 15, 30, 0, 0, time.UTC)
	if !nextDaily.Equal(expectedDaily) {
		t.Errorf("expected %v, got %v", expectedDaily, nextDaily)
	}

	// 3. Daily at 02:00: next should be 02:00 tomorrow (Oct 6)
	nextTmrw := NextRun("0 2 * * *", from)
	expectedTmrw := time.Date(2026, 10, 6, 2, 0, 0, 0, time.UTC)
	if !nextTmrw.Equal(expectedTmrw) {
		t.Errorf("expected %v, got %v", expectedTmrw, nextTmrw)
	}
}
