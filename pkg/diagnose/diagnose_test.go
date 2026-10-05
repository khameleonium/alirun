package diagnose

import (
	"alirun/pkg/initsys"
	"testing"
)

func TestDiagnosePermissionDenied(t *testing.T) {
	info := &initsys.ServiceInfo{
		Name:     "worker.service",
		Status:   initsys.StatusFailed,
		SubState: "failed",
		ExecPath: "/home/user/script.sh",
	}
	logs := []string{
		"systemd[1]: Starting worker.service...",
		"systemd[32145]: worker.service: Failed at step EXEC spawning /home/user/script.sh: Permission denied",
		"systemd[1]: worker.service: Main process exited, code=exited, status=203/EXEC",
	}

	report := Diagnose(info, logs)
	if report.IsHealthy {
		t.Errorf("expected unhealthy report")
	}
	if report.Confidence != "HIGH" {
		t.Errorf("expected HIGH confidence, got %s", report.Confidence)
	}
	if report.OffendingLine == "" {
		t.Errorf("expected offending line to be captured")
	}
}

func TestDiagnosePortInUse(t *testing.T) {
	info := &initsys.ServiceInfo{
		Name:     "web.service",
		Status:   initsys.StatusFailed,
		SubState: "failed",
		ExecPath: "python3 -m http.server 8080",
	}
	logs := []string{
		"Serving HTTP on 0.0.0.0 port 8080 ...",
		"OSError: [Errno 98] Address already in use",
	}

	report := Diagnose(info, logs)
	if report.IsHealthy {
		t.Errorf("expected unhealthy report")
	}
	if report.Confidence != "HIGH" {
		t.Errorf("expected HIGH confidence")
	}
}
