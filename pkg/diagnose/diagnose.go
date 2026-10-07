package diagnose

import (
	"alirun/pkg/initsys"
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// oomRe matches "oom" as a word ("oom-kill", "oom_kill", "oom killer"), not inside "room"/"zoom"/"bloom"
	oomRe = regexp.MustCompile(`\boom($|[^a-z])`)
	// shellNotFoundRe matches shell errors like "sh: 1: foo: not found" but not "404 Not Found"
	shellNotFoundRe = regexp.MustCompile(`(?m):\s*not found\s*$`)
)

// IssueReport contains findings and suggested resolutions for a service
type IssueReport struct {
	ServiceName   string `json:"service_name"`
	IsHealthy     bool   `json:"is_healthy"`
	Summary       string `json:"summary"`
	RootCause     string `json:"root_cause"`
	SuggestedFix  string `json:"suggested_fix"`
	OffendingLine string `json:"offending_line,omitempty"`
	Confidence    string `json:"confidence"` // "HIGH", "MEDIUM", "LOW"
}

// Diagnose inspects service info and its recent log lines to determine why it failed
func Diagnose(info *initsys.ServiceInfo, logLines []string) IssueReport {
	if info == nil {
		return IssueReport{
			IsHealthy: false,
			Summary:   "No service info available",
		}
	}

	if info.Status == initsys.StatusActive {
		return IssueReport{
			ServiceName:  info.Name,
			IsHealthy:    true,
			Summary:      "Служба работает стабильно (Service is active and running normally)",
			SuggestedFix: "No action required.",
			Confidence:   "HIGH",
		}
	}

	logsCombined := strings.Join(logLines, "\n")
	logsLower := strings.ToLower(logsCombined)

	// systemd uses status=203/EXEC both for a missing and for a non-executable file, and newer
	// versions often don't log the reason at all, so check the file system to tell them apart.
	missingExec := strings.Contains(logsLower, "status=203/exec") && executableMissing(info.ExecPath)

	// 1. File not found or status=127. Must be checked before 203/EXEC: systemd reports a
	// missing executable as "No such file or directory" together with status=203/EXEC.
	if missingExec || strings.Contains(logsLower, "status=127") || strings.Contains(logsLower, "no such file or directory") ||
		strings.Contains(logsLower, "command not found") || strings.Contains(logsLower, "executable file not found") ||
		shellNotFoundRe.MatchString(logsLower) {
		return IssueReport{
			ServiceName:   info.Name,
			IsHealthy:     false,
			Summary:       "Файл или интерпретатор не найден (No such file or directory / 127)",
			RootCause:     fmt.Sprintf("Команда %q указывает на несуществующий путь, либо интерпретатор скрипта не установлен в системе.", info.ExecPath),
			SuggestedFix:  "Укажите абсолютный путь к файлу (например, /usr/bin/python3 или /home/user/app.sh) и убедитесь в его существовании.",
			OffendingLine: findMatchingLine(logLines, "no such file", "command not found", "executable file not found", ": not found", "status=127", "203/exec"),
			Confidence:    "HIGH",
		}
	}

	// 2. Missing execution rights or 203/EXEC
	if strings.Contains(logsLower, "status=203/exec") || strings.Contains(logsLower, "permission denied") {
		return IssueReport{
			ServiceName:   info.Name,
			IsHealthy:     false,
			Summary:       "Ошибка доступа: нет прав на запуск файла (Permission denied / 203 EXEC)",
			RootCause:     fmt.Sprintf("Файл команды %q не имеет флага исполняемости (+x) или скрипт содержит неверный shebang (например, #!/usr/bin/python3).", info.ExecPath),
			SuggestedFix:  fmt.Sprintf("Выполните: chmod +x %s и проверьте первую строчку файла.", extractBinary(info.ExecPath)),
			OffendingLine: findMatchingLine(logLines, "permission denied", "203/exec"),
			Confidence:    "HIGH",
		}
	}

	// 3. Port already in use / bind failure
	if strings.Contains(logsLower, "address already in use") || strings.Contains(logsLower, "bind: address already in use") || strings.Contains(logsLower, "eaddrinuse") {
		return IssueReport{
			ServiceName:   info.Name,
			IsHealthy:     false,
			Summary:       "Сетевой порт занят другим процессом (Port already in use)",
			RootCause:     "Служба пытается занять порт, который уже используется другим работающим сервисом или приложением.",
			SuggestedFix:  "Проверьте занятый порт командой 'alirun list -d' или 'ss -tulpn', остановите конфликтующий процесс либо смените порт в конфигурации.",
			OffendingLine: findMatchingLine(logLines, "address already in use", "eaddrinuse"),
			Confidence:    "HIGH",
		}
	}

	// 4. Out of Memory (OOM Killer)
	if oomRe.MatchString(logsLower) || strings.Contains(logsLower, "out of memory") || strings.Contains(logsLower, "signal=kill") || strings.Contains(logsLower, "status=137") {
		return IssueReport{
			ServiceName:   info.Name,
			IsHealthy:     false,
			Summary:       "Процесс аварийно завершён ядром из-за нехватки памяти (OOM Killed)",
			RootCause:     "Служба превысила доступный лимит оперативной памяти (MemoryMax) либо в системе закончилась память (RAM + Swap).",
			SuggestedFix:  "Увеличьте лимит MemoryMax в юните, добавьте swap-файл или оптимизируйте потребление памяти приложением.",
			OffendingLine: findMatchingLine(logLines, "oom-kill", "oom_kill", "oom killer", "out of memory", "signal=kill", "status=137"),
			Confidence:    "HIGH",
		}
	}

	// 5. Working Directory failure (200/CHDIR)
	if strings.Contains(logsLower, "200/chdir") || strings.Contains(logsLower, "failed at step chdir") {
		return IssueReport{
			ServiceName:   info.Name,
			IsHealthy:     false,
			Summary:       "Рабочая папка не существует (Working directory does not exist)",
			RootCause:     "Параметр WorkingDirectory указывает на несуществующий каталог на диске.",
			SuggestedFix:  "Создайте указанную директорию (mkdir -p <dir>) или исправьте путь в настройках юнита.",
			OffendingLine: findMatchingLine(logLines, "chdir"),
			Confidence:    "HIGH",
		}
	}

	// 6. User step failure (217/USER)
	if strings.Contains(logsLower, "217/user") || strings.Contains(logsLower, "failed at step user") {
		return IssueReport{
			ServiceName:   info.Name,
			IsHealthy:     false,
			Summary:       "Указанный пользователь системы не существует (User does not exist)",
			RootCause:     "Параметр User= задаёт имя пользователя, которого нет в /etc/passwd.",
			SuggestedFix:  "Создайте пользователя через useradd или удалите строчку User= для запуска от текущего пользователя.",
			OffendingLine: findMatchingLine(logLines, "217/user", "failed at step user"),
			Confidence:    "HIGH",
		}
	}

	// 7. General crash with log line
	lastErrLine := ""
	for i := len(logLines) - 1; i >= 0; i-- {
		line := logLines[i]
		if strings.Contains(strings.ToLower(line), "error") ||
			strings.Contains(strings.ToLower(line), "fatal") ||
			strings.Contains(strings.ToLower(line), "panic") ||
			strings.Contains(strings.ToLower(line), "failed") {
			lastErrLine = line
			break
		}
	}

	summary := fmt.Sprintf("Служба завершилась с ошибкой (%s)", info.SubState)
	cause := "Служба аварийно завершилась. Изучите вывод ошибки ниже."
	if lastErrLine != "" {
		cause = fmt.Sprintf("Последняя ошибка в журнале: %s", lastErrLine)
	}

	return IssueReport{
		ServiceName:   info.Name,
		IsHealthy:     false,
		Summary:       summary,
		RootCause:     cause,
		SuggestedFix:  fmt.Sprintf("Попробуйте запустить команду вручную в терминале для проверки: %s", info.ExecPath),
		OffendingLine: lastErrLine,
		Confidence:    "MEDIUM",
	}
}

func extractBinary(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) > 0 {
		// Strip systemd ExecStart= prefixes ("-", "@", "+", "!", "!!", ":")
		return strings.TrimLeft(parts[0], "-@+!:")
	}
	return cmd
}

// executableMissing reports whether the command's binary, or the interpreter from its
// shebang line, does not exist. Only absolute paths can be checked reliably.
func executableMissing(execPath string) bool {
	bin := extractBinary(execPath)
	if !filepath.IsAbs(bin) {
		return false
	}
	f, err := os.Open(bin)
	if err != nil {
		return errors.Is(err, fs.ErrNotExist)
	}
	defer f.Close()

	line, _ := bufio.NewReader(f).ReadString('\n')
	if interp, ok := strings.CutPrefix(line, "#!"); ok {
		if fields := strings.Fields(interp); len(fields) > 0 && filepath.IsAbs(fields[0]) {
			if _, err := os.Stat(fields[0]); errors.Is(err, fs.ErrNotExist) {
				return true
			}
		}
	}
	return false
}

func findMatchingLine(lines []string, patterns ...string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		lower := strings.ToLower(lines[i])
		for _, p := range patterns {
			if strings.Contains(lower, p) {
				return strings.TrimSpace(lines[i])
			}
		}
	}
	return ""
}
