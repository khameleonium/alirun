package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupSerialization(t *testing.T) {
	manifest := &BackupManifest{
		Version:   "1.0",
		CreatedBy: "Alirun",
		Author:    "Создатель: Илья Ульянов | khameleonium",
		Systemd: []UnitBackup{
			{
				Name:    "demo.service",
				Scope:   "user",
				Path:    "/tmp/demo.service",
				Content: "[Unit]\nDescription=Demo\n",
				Enabled: true,
			},
		},
		Crontab: &CronBackup{
			UserCrontab: "0 * * * * /bin/true\n",
		},
		XDG: []XDGBackup{
			{
				Name:    "demo.desktop",
				Scope:   "user",
				Path:    "/tmp/demo.desktop",
				Content: "[Desktop Entry]\nName=Demo\n",
				Enabled: true,
			},
		},
	}

	data, err := manifest.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	parsed, err := FromYAML(data)
	if err != nil {
		t.Fatalf("FromYAML failed: %v", err)
	}

	if parsed.Author != "Создатель: Илья Ульянов | khameleonium" {
		t.Fatalf("expected author %q, got %q", "Создатель: Илья Ульянов | khameleonium", parsed.Author)
	}
	if len(parsed.Systemd) != 1 || parsed.Systemd[0].Name != "demo.service" {
		t.Fatalf("unexpected systemd units: %+v", parsed.Systemd)
	}
	if parsed.Crontab == nil || parsed.Crontab.UserCrontab != "0 * * * * /bin/true\n" {
		t.Fatalf("unexpected crontab: %+v", parsed.Crontab)
	}
	if len(parsed.XDG) != 1 || parsed.XDG[0].Name != "demo.desktop" {
		t.Fatalf("unexpected xdg: %+v", parsed.XDG)
	}
}

func TestImportDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	unitPath := filepath.Join(tmpDir, "test.service")

	manifest := &BackupManifest{
		Version: "1.0",
		Systemd: []UnitBackup{
			{
				Name:    "test.service",
				Scope:   "user",
				Path:    unitPath,
				Content: "[Unit]\nDescription=Dry Run\n",
			},
		},
	}

	res, err := Import(context.Background(), manifest, true)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if len(res.SystemdRestored) != 1 {
		t.Fatalf("expected 1 dry run restoration, got %d", len(res.SystemdRestored))
	}

	if _, err := os.Stat(unitPath); !os.IsNotExist(err) {
		t.Fatalf("file should not exist in dry run")
	}
}

func TestImportExecution(t *testing.T) {
	// Restore into a fresh $HOME: paths stored in the backup belong to another
	// user/machine and must not be used as destinations.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	unitPath := filepath.Join(tmpDir, ".config", "systemd", "user", "installed.service")
	xdgPath := filepath.Join(tmpDir, ".config", "autostart", "installed.desktop")

	manifest := &BackupManifest{
		Version: "1.0",
		Systemd: []UnitBackup{
			{
				Name:    "installed.service",
				Scope:   "user",
				Path:    "/home/olduser/.config/systemd/user/installed.service",
				Content: "[Unit]\nDescription=Installed\n",
			},
		},
		XDG: []XDGBackup{
			{
				Name:    "installed.desktop",
				Scope:   "user",
				Path:    "/home/olduser/.config/autostart/installed.desktop",
				Content: "[Desktop Entry]\nName=Installed\n",
			},
		},
	}

	res, err := Import(context.Background(), manifest, false)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if len(res.Errors) > 0 {
		t.Fatalf("unexpected errors during import: %v", res.Errors)
	}

	data, err := os.ReadFile(unitPath)
	if err != nil || string(data) != "[Unit]\nDescription=Installed\n" {
		t.Fatalf("unit file not written correctly: %v, content: %s", err, string(data))
	}

	xData, err := os.ReadFile(xdgPath)
	if err != nil || string(xData) != "[Desktop Entry]\nName=Installed\n" {
		t.Fatalf("desktop file not written correctly: %v, content: %s", err, string(xData))
	}
}

func TestImportRejectsUnsafeNames(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	manifest := &BackupManifest{
		Systemd: []UnitBackup{{Name: "../../evil.service", Scope: "user", Content: "x"}},
		XDG:     []XDGBackup{{Name: "../evil.desktop", Scope: "user", Content: "x"}},
		Crontab: &CronBackup{SystemFiles: map[string]string{"/etc/passwd": "x"}},
	}
	res, err := Import(context.Background(), manifest, false)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if len(res.SystemdRestored) != 0 || len(res.XDGRestored) != 0 || len(res.Errors) != 3 {
		t.Fatalf("unsafe entries must be rejected, got %+v", res)
	}
}

func TestIsPortableUnitLink(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]bool{
		"/dev/null":                         false,
		"/usr/lib/systemd/system/a.service": false,
		"b.service":                         false,
		"/home/user/project/app.service":    true,
	}
	i := 0
	for target, want := range cases {
		i++
		link := filepath.Join(dir, fmt.Sprintf("l%d.service", i))
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if got := isPortableUnitLink(link); got != want {
			t.Errorf("isPortableUnitLink(-> %s) = %v, want %v", target, got, want)
		}
	}
}
