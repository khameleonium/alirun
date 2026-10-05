package xdg

import (
	"alirun/pkg/initsys"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanDesktopExec(t *testing.T) {
	cmd := "/usr/bin/my-app --flag %u %F"
	clean := cleanDesktopExec(cmd)
	if clean != "/usr/bin/my-app --flag" {
		t.Fatalf("expected '/usr/bin/my-app --flag', got %q", clean)
	}
}

func TestGenerateConfig(t *testing.T) {
	mgr := New()
	cfg := initsys.ServiceConfig{
		Name:        "myapp",
		Description: "My Application",
		ExecStart:   "/usr/bin/myapp",
	}

	content, err := mgr.GenerateConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	if !contains(content, "[Desktop Entry]") || !contains(content, "Exec=/usr/bin/myapp") {
		t.Fatalf("generated desktop content missing expected fields: %s", content)
	}
}

func TestParseDesktopFile(t *testing.T) {
	mgr := New()
	tmpDir := t.TempDir()
	desktopPath := filepath.Join(tmpDir, "testapp.desktop")

	content := `[Desktop Entry]
Type=Application
Name=Test Application
Exec=/usr/local/bin/testapp %u
Hidden=false
X-GNOME-Autostart-enabled=true
`
	if err := os.WriteFile(desktopPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test desktop file: %v", err)
	}

	info, err := mgr.parseDesktopFile(desktopPath, "testapp.desktop", initsys.TypeUser)
	if err != nil {
		t.Fatalf("parseDesktopFile failed: %v", err)
	}

	if info.Name != "testapp" {
		t.Fatalf("expected name 'testapp', got %q", info.Name)
	}
	if info.Description != "Test Application" {
		t.Fatalf("expected description 'Test Application', got %q", info.Description)
	}
	if info.ExecPath != "/usr/local/bin/testapp %u" {
		t.Fatalf("expected ExecPath '/usr/local/bin/testapp %%u', got %q", info.ExecPath)
	}
	if !info.Enabled {
		t.Fatalf("expected Enabled true, got false")
	}

	// Test disable/enable toggle
	if err := mgr.toggleEnabled(desktopPath, false); err != nil {
		t.Fatalf("toggleEnabled(false) failed: %v", err)
	}

	infoDisabled, err := mgr.parseDesktopFile(desktopPath, "testapp.desktop", initsys.TypeUser)
	if err != nil {
		t.Fatalf("parseDesktopFile failed: %v", err)
	}
	if infoDisabled.Enabled {
		t.Fatalf("expected Enabled false after setDesktopEnabled(false)")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
