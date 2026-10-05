package editor

import (
	"alirun/pkg/initsys"
	"context"
	"os"
	"path/filepath"
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
