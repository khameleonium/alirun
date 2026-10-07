package xdg

import (
	"alirun/pkg/initsys"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const multiGroup = `[Desktop Entry]
Type=Application
Name=Telegram
Exec=/usr/bin/telegram -startintray
Actions=quit;

[Desktop Action quit]
Name=Quit Telegram
Exec=/usr/bin/telegram -quit
`

func TestMultiGroupDesktopFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "autostart")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tg.desktop")
	if err := os.WriteFile(path, []byte(multiGroup), 0644); err != nil {
		t.Fatal(err)
	}

	m := New()
	ctx := context.Background()
	info, err := m.GetStatus(ctx, "tg", initsys.TypeUser)
	if err != nil {
		t.Fatal(err)
	}
	if info.ExecPath != "/usr/bin/telegram -startintray" || info.Description != "Telegram" {
		t.Errorf("action group leaked into main entry: exec=%q name=%q", info.ExecPath, info.Description)
	}

	if err := m.Disable(ctx, "tg", initsys.TypeUser); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	mainGroup := strings.Split(string(data), "[Desktop Action quit]")[0]
	if !strings.Contains(mainGroup, "Hidden=true") || !strings.Contains(mainGroup, "X-GNOME-Autostart-enabled=false") {
		t.Errorf("keys must be inserted into [Desktop Entry]:\n%s", data)
	}
	if strings.Count(string(data), "Hidden=") != 1 {
		t.Errorf("Hidden= must appear exactly once:\n%s", data)
	}
	if info, _ := m.GetStatus(ctx, "tg", initsys.TypeUser); info.Enabled {
		t.Errorf("entry should be disabled after Disable")
	}

	if err := m.Enable(ctx, "tg", initsys.TypeUser); err != nil {
		t.Fatal(err)
	}
	if info, _ := m.GetStatus(ctx, "tg", initsys.TypeUser); !info.Enabled {
		t.Errorf("entry should be enabled after Enable")
	}
}
