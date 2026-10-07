package editor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandSupportsEditorArguments(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "args")
	target := filepath.Join(dir, "my file.service")
	// Fake editor that records its arguments, configured with an extra flag
	t.Setenv("VISUAL", `printf '%s|' --wait >`+out+`; printf '%s' `)
	cmd := Command(context.Background(), target)
	res, err := cmd.Output()
	if err != nil {
		t.Fatalf("editor command failed: %v", err)
	}
	if string(res) != target {
		t.Errorf("file path must be passed as one argument, got %q", res)
	}
	if data, _ := os.ReadFile(out); !strings.Contains(string(data), "--wait") {
		t.Errorf("editor arguments were not honoured: %q", data)
	}
}
