package systemd

import (
	"alirun/pkg/initsys"
	"strings"
	"testing"
)

func TestGenerateUnitFileDaemon(t *testing.T) {
	cfg := initsys.ServiceConfig{
		Name:             "test-worker",
		Description:      "Test Background Worker",
		ExecStart:        "/usr/local/bin/worker --flag",
		WorkingDirectory: "/opt/worker",
		Preset:           initsys.PresetDaemon,
		Type:             initsys.TypeUser,
		RestartSec:       3,
		Environment: map[string]string{
			"LOG_LEVEL": "debug",
			"PORT":      "8080",
		},
	}

	content, err := GenerateUnitFile(cfg)
	if err != nil {
		t.Fatalf("GenerateUnitFile failed: %v", err)
	}

	expectedSubstrings := []string{
		"[Unit]",
		"Description=Test Background Worker",
		"[Service]",
		"Type=simple",
		"ExecStart=/usr/local/bin/worker --flag",
		"WorkingDirectory=/opt/worker",
		"Restart=always",
		"RestartSec=3s",
		"Environment=LOG_LEVEL=debug",
		"Environment=PORT=8080",
		"StandardOutput=journal",
		"[Install]",
		"WantedBy=default.target",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(content, sub) {
			t.Errorf("generated unit missing expected substring %q\nFull content:\n%s", sub, content)
		}
	}
}

func TestGenerateUnitFileOneshot(t *testing.T) {
	cfg := initsys.ServiceConfig{
		Name:        "backup-job",
		ExecStart:   "/usr/bin/backup.sh",
		Preset:      initsys.PresetOneshot,
		Type:        initsys.TypeSystem,
		Description: "One-shot Backup",
	}

	content, err := GenerateUnitFile(cfg)
	if err != nil {
		t.Fatalf("GenerateUnitFile failed: %v", err)
	}

	if !strings.Contains(content, "Type=oneshot") {
		t.Errorf("expected Type=oneshot in oneshot service")
	}
	if !strings.Contains(content, "WantedBy=multi-user.target") {
		t.Errorf("expected WantedBy=multi-user.target for system service")
	}
}
