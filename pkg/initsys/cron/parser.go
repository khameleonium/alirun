package cron

import (
	"bufio"
	"fmt"
	"strings"
	"time"
)

// ParseCrontab parses crontab text into a slice of CronJob pointers
func ParseCrontab(content string, fileName string, isSystem bool) ([]*CronJob, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var jobs []*CronJob
	var commentBuf []string
	lineIdx := 0

	for scanner.Scan() {
		rawLine := scanner.Text()
		trimmed := strings.TrimSpace(rawLine)

		// Empty line
		if trimmed == "" {
			commentBuf = nil
			lineIdx++
			continue
		}

		// Comment or Disabled Job
		if strings.HasPrefix(trimmed, "#") {
			// Check if this commented line is a disabled cron job: e.g. "# 0 3 * * * /backup.sh"
			uncommented := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			sched, usr, cmd, ok := extractCronFields(uncommented, isSystem)
			if ok {
				// It is a disabled cron job!
				comment := strings.Join(commentBuf, " ")
				job := &CronJob{
					ID:            fmt.Sprintf("cron-%d", len(jobs)+1),
					LineIndex:     lineIdx,
					File:          fileName,
					Schedule:      sched,
					HumanSchedule: HumanizeSchedule(sched),
					NextRun:       NextRun(sched, time.Now()),
					User:          usr,
					Command:       cmd,
					Comment:       comment,
					Enabled:       false,
					RawLine:       rawLine,
				}
				jobs = append(jobs, job)
				commentBuf = nil
			} else {
				// Regular comment
				cText := strings.TrimSpace(uncommented)
				if !strings.HasPrefix(cText, "/etc") &&
					!strings.HasPrefix(cText, "Example of job") &&
					!strings.HasPrefix(cText, "|") &&
					!strings.HasPrefix(cText, ".") &&
					!strings.HasPrefix(cText, "*") {
					commentBuf = append(commentBuf, cText)
				}
			}
			lineIdx++
			continue
		}

		// Environment variable: KEY=VAL
		if strings.Contains(trimmed, "=") && !strings.Contains(strings.Split(trimmed, "=")[0], " ") {
			commentBuf = nil
			lineIdx++
			continue
		}

		// Active Cron Job
		sched, usr, cmd, ok := extractCronFields(trimmed, isSystem)
		if ok {
			comment := strings.Join(commentBuf, " ")
			job := &CronJob{
				ID:            fmt.Sprintf("cron-%d", len(jobs)+1),
				LineIndex:     lineIdx,
				File:          fileName,
				Schedule:      sched,
				HumanSchedule: HumanizeSchedule(sched),
				NextRun:       NextRun(sched, time.Now()),
				User:          usr,
				Command:       cmd,
				Comment:       comment,
				Enabled:       true,
				RawLine:       rawLine,
			}
			jobs = append(jobs, job)
			commentBuf = nil
		} else {
			commentBuf = nil
		}

		lineIdx++
	}

	return jobs, scanner.Err()
}

// extractCronFields tries to parse a line into (schedule, user, command, ok)
func extractCronFields(line string, isSystem bool) (schedule, user, command string, ok bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", "", false
	}

	// 1. Check macro format: @hourly, @daily, etc.
	if strings.HasPrefix(fields[0], "@") {
		macro := strings.ToLower(fields[0])
		validMacros := map[string]bool{
			"@reboot": true, "@yearly": true, "@annually": true,
			"@monthly": true, "@weekly": true, "@daily": true,
			"@midnight": true, "@hourly": true,
		}
		if validMacros[macro] {
			if isSystem && len(fields) >= 3 {
				return fields[0], fields[1], strings.Join(fields[2:], " "), true
			} else if !isSystem && len(fields) >= 2 {
				return fields[0], "", strings.Join(fields[1:], " "), true
			}
		}
	}

	// 2. Check 5-field standard cron: min hour dom mon dow
	reqFields := 6
	if isSystem {
		reqFields = 7
	}
	if len(fields) < reqFields {
		return "", "", "", false
	}

	// Validate that the first 3 tokens are valid minute, hour, day-of-month
	if _, err := parseCronField(fields[0], 0, 59); err != nil {
		return "", "", "", false
	}
	if _, err := parseCronField(fields[1], 0, 23); err != nil {
		return "", "", "", false
	}
	if _, err := parseCronField(fields[2], 1, 31); err != nil {
		return "", "", "", false
	}

	sched := strings.Join(fields[:5], " ")
	if isSystem {
		usr := fields[5]
		if strings.EqualFold(usr, "user-name") || strings.EqualFold(usr, "username") {
			return "", "", "", false
		}
		cmd := strings.Join(fields[6:], " ")
		return sched, usr, cmd, true
	}

	cmd := strings.Join(fields[5:], " ")
	return sched, "", cmd, true
}
