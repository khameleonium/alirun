package cron

import (
	"fmt"
	"time"
)

// CronJob represents a single cron entry
type CronJob struct {
	ID            string    `json:"id"`
	LineIndex     int       `json:"line_index"`     // line number in file (0-based)
	File          string    `json:"file"`           // "crontab (user)" or "/etc/crontab" or "/etc/cron.d/..."
	Schedule      string    `json:"schedule"`       // e.g. "*/15 * * * *" or "@daily"
	HumanSchedule string    `json:"human_schedule"` // Human-readable representation, e.g. "Every 15 min"
	NextRun       time.Time `json:"next_run"`
	User          string    `json:"user,omitempty"` // User name for system crontabs
	Command       string    `json:"command"`
	Comment       string    `json:"comment"` // Description from preceding comments
	Enabled       bool      `json:"enabled"` // true if active, false if commented out (#)
	RawLine       string    `json:"raw_line"`
}

// DisplayName returns a friendly unique title for the job
func (j *CronJob) DisplayName() string {
	if j.Comment != "" {
		return j.Comment
	}
	if r := []rune(j.Command); len(r) > 35 {
		return string(r[:32]) + "..."
	}
	if j.Command != "" {
		return j.Command
	}
	return fmt.Sprintf("job-%s-%d", j.File, j.LineIndex+1)
}
