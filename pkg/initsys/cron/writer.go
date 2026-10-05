package cron

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ReadUserCrontab reads the current user's crontab using `crontab -l`
func ReadUserCrontab() (string, error) {
	cmd := exec.Command("crontab", "-l")
	out, err := cmd.CombinedOutput()
	outStr := string(out)

	if err != nil {
		if strings.Contains(strings.ToLower(outStr), "no crontab") {
			return "", nil
		}
		return "", fmt.Errorf("crontab -l failed: %s (%w)", strings.TrimSpace(outStr), err)
	}

	return outStr, nil
}

// SaveUserCrontab saves the given content as the current user's crontab via `crontab -`
func SaveUserCrontab(content string) error {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		// Removing crontab if empty
		cmd := exec.Command("crontab", "-r")
		_ = cmd.Run()
		return nil
	}

	// Ensure trailing newline for standard crontab
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to save crontab: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// ToggleJobLine modifies the line at lineIndex: comments if enabled, uncomments if commented
func ToggleJobLine(rawContent string, lineIndex int) (string, error) {
	lines := strings.Split(rawContent, "\n")
	if lineIndex < 0 || lineIndex >= len(lines) {
		return rawContent, fmt.Errorf("invalid line index %d (total lines: %d)", lineIndex, len(lines))
	}

	target := lines[lineIndex]
	trimmed := strings.TrimSpace(target)

	if strings.HasPrefix(trimmed, "#") {
		// Uncomment
		lines[lineIndex] = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
	} else {
		// Comment out
		lines[lineIndex] = "# " + target
	}

	return strings.Join(lines, "\n"), nil
}

// DeleteJobLine removes the line at lineIndex from rawContent
func DeleteJobLine(rawContent string, lineIndex int) (string, error) {
	lines := strings.Split(rawContent, "\n")
	if lineIndex < 0 || lineIndex >= len(lines) {
		return rawContent, fmt.Errorf("invalid line index %d", lineIndex)
	}

	// Check if line above was a single comment describing this job
	removeStart := lineIndex
	if lineIndex > 0 && strings.HasPrefix(strings.TrimSpace(lines[lineIndex-1]), "#") {
		prevTrim := strings.TrimSpace(lines[lineIndex-1])
		// Don't remove if it's a section header or disabled job
		if !strings.HasPrefix(prevTrim, "# /etc") && !strings.Contains(prevTrim, "=") {
			_, _, _, isJob := extractCronFields(strings.TrimSpace(strings.TrimPrefix(prevTrim, "#")), false)
			if !isJob {
				removeStart = lineIndex - 1
			}
		}
	}

	newLines := append(lines[:removeStart], lines[lineIndex+1:]...)
	return strings.Join(newLines, "\n"), nil
}

// AppendJob adds a new cron job with optional comment to rawContent
func AppendJob(rawContent, schedule, command, comment string) string {
	var sb strings.Builder
	trimmed := strings.TrimSpace(rawContent)
	if trimmed != "" {
		sb.WriteString(trimmed)
		sb.WriteString("\n\n")
	}

	if comment != "" {
		sb.WriteString("# ")
		sb.WriteString(comment)
		sb.WriteString("\n")
	}

	sb.WriteString(strings.TrimSpace(schedule))
	sb.WriteString(" ")
	sb.WriteString(strings.TrimSpace(command))
	sb.WriteString("\n")

	return sb.String()
}

// RunJobCommand runs a cron job command on-demand, capturing combined output
func RunJobCommand(ctx context.Context, command string) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "/bin/sh", "-c", command)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	output := strings.TrimSpace(buf.String())

	if err != nil {
		if output != "" {
			return output, fmt.Errorf("exit status %v: %s", err, output)
		}
		return "", err
	}

	if output == "" {
		output = "(command completed successfully with no output)"
	}
	return output, nil
}
