package editor

import (
	"alirun/pkg/initsys"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindEditor(t *testing.T) {
	os.Setenv("EDITOR", "my-custom-editor")
	defer os.Unsetenv("EDITOR")

	ed := FindEditor()
	if ed != "my-custom-editor" {
		t.Fatalf("expected my-custom-editor, got %s", ed)
	}
}

func TestPrepareEditSystemd(t *testing.T) {
	tmpDir := t.TempDir()
	unitFile := filepath.Join(tmpDir, "test.service")
	_ = os.WriteFile(unitFile, []byte("[Unit]\nDescription=Test\n"), 0644)

	info := &initsys.ServiceInfo{
		Name:       "test.service",
		ConfigPath: unitFile,
		InitSystem: "systemd",
	}

	session, err := PrepareEdit(context.Background(), info, nil, initsys.TypeUser)
	if err != nil {
		t.Fatalf("PrepareEdit failed: %v", err)
	}
	defer session.Cleanup()

	if session.FilePath != unitFile {
		t.Fatalf("expected filePath %s, got %s", unitFile, session.FilePath)
	}
	if session.IsTemp {
		t.Fatalf("expected non-temp session for existing unit file")
	}
}

func TestPrepareEditSystemdEphemeralCopy(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create an ephemeral unit in /run/user/<uid> if available
	uid := os.Getuid()
	runUserDir := fmt.Sprintf("/run/user/%d", uid)
	if _, err := os.Stat(runUserDir); err != nil {
		t.Skip("skipping /run test: runUserDir not available")
	}

	testRunDir := filepath.Join(runUserDir, "alirun-test-run")
	_ = os.MkdirAll(testRunDir, 0755)
	defer os.RemoveAll(testRunDir)

	ephemeralUnit := filepath.Join(testRunDir, "my-app.service")
	origContent := "[Unit]\nDescription=Autostart App\nExecStart=/usr/bin/app\n"
	_ = os.WriteFile(ephemeralUnit, []byte(origContent), 0644)

	info := &initsys.ServiceInfo{
		Name:       "my-app",
		ConfigPath: ephemeralUnit,
		InitSystem: "systemd",
	}

	session, err := PrepareEdit(context.Background(), info, nil, initsys.TypeUser)
	if err != nil {
		t.Fatalf("PrepareEdit failed: %v", err)
	}
	defer session.Cleanup()

	expectedOverridePath := filepath.Join(tmpHome, ".config", "systemd", "user", "my-app.service")
	if session.FilePath != expectedOverridePath {
		t.Fatalf("expected filePath %s, got %s", expectedOverridePath, session.FilePath)
	}

	// Verify content was copied into override path
	copiedBytes, err := os.ReadFile(expectedOverridePath)
	if err != nil {
		t.Fatalf("failed to read created override file: %v", err)
	}
	if !strings.Contains(string(copiedBytes), "Autostart App") {
		t.Fatalf("expected copied content to contain 'Autostart App', got:\n%s", string(copiedBytes))
	}
}
