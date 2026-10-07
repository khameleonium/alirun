package diagnose

import (
	"alirun/pkg/initsys"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func failed() *initsys.ServiceInfo {
	return &initsys.ServiceInfo{Name: "app", Status: initsys.StatusFailed, SubState: "failed", ExecPath: "/opt/app/run"}
}

func TestDiagnoseMissingExecutableIsNotPermission(t *testing.T) {
	logs := []string{
		"systemd[1]: app.service: Failed to locate executable /opt/app/run: No such file or directory",
		"systemd[1]: app.service: Main process exited, code=exited, status=203/EXEC",
	}
	r := Diagnose(failed(), logs)
	if !strings.Contains(r.Summary, "не найден") {
		t.Errorf("missing executable diagnosed as %q", r.Summary)
	}
}

func TestDiagnoseNoFalseOOM(t *testing.T) {
	logs := []string{"joined chat room #general", "zoom level changed", "fatal: config invalid"}
	if r := Diagnose(failed(), logs); strings.Contains(r.Summary, "OOM") {
		t.Errorf("false OOM diagnosis: %q", r.Summary)
	}
	logs = []string{"systemd[1]: app.service: Failed with result 'oom-kill'."}
	if r := Diagnose(failed(), logs); !strings.Contains(r.Summary, "OOM") {
		t.Errorf("oom-kill not detected: %q", r.Summary)
	}
}

func TestDiagnoseHTTPNotFoundIsNotMissingFile(t *testing.T) {
	logs := []string{`GET /favicon.ico HTTP/1.1" 404 Not Found`, "panic: something broke"}
	if r := Diagnose(failed(), logs); strings.Contains(r.Summary, "не найден") {
		t.Errorf("HTTP 404 diagnosed as missing file: %q", r.Summary)
	}
	logs = []string{"sh[42]: /opt/app/run.sh: 3: foo: not found"}
	if r := Diagnose(failed(), logs); !strings.Contains(r.Summary, "не найден") {
		t.Errorf("shell 'not found' not detected: %q", r.Summary)
	}
}

// systemd 255 user managers log only "status=203/EXEC" without the reason
func TestDiagnose203WithoutReasonUsesFileSystem(t *testing.T) {
	dir := t.TempDir()
	logs := []string{"systemd[3461]: app.service: Main process exited, code=exited, status=203/EXEC"}

	info := failed()
	info.ExecPath = filepath.Join(dir, "missing-app") + " --flag"
	if r := Diagnose(info, logs); !strings.Contains(r.Summary, "не найден") {
		t.Errorf("missing binary: %q", r.Summary)
	}

	notExec := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(notExec, []byte("#!/bin/sh\necho hi\n"), 0644); err != nil {
		t.Fatal(err)
	}
	info.ExecPath = notExec
	if r := Diagnose(info, logs); !strings.Contains(r.Summary, "нет прав") {
		t.Errorf("non-executable file: %q", r.Summary)
	}

	badShebang := filepath.Join(dir, "bad.py")
	if err := os.WriteFile(badShebang, []byte("#!/nonexistent/python9\nprint(1)\n"), 0755); err != nil {
		t.Fatal(err)
	}
	info.ExecPath = "-" + badShebang
	if r := Diagnose(info, logs); !strings.Contains(r.Summary, "не найден") {
		t.Errorf("missing interpreter: %q", r.Summary)
	}
}
