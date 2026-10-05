package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeServiceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-bot.py", "my-bot"},
		{"My Script! 123.sh", "my-script-123"},
		{"app.bin", "app"},
		{"_test_svc_", "test-svc"},
	}

	for _, tt := range tests {
		got := sanitizeServiceName(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeServiceName(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestInspectScript(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_job.sh")

	content := []byte("#!/bin/bash\necho 'hello world'\n")
	if err := os.WriteFile(scriptPath, content, 0755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	insp, err := Inspect(scriptPath)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if !insp.Exists {
		t.Errorf("expected Exists = true")
	}
	if !insp.IsExecutable {
		t.Errorf("expected IsExecutable = true")
	}
	if insp.DetectedType != FileTypeBash {
		t.Errorf("expected DetectedType = FileTypeBash, got %v", insp.DetectedType)
	}
	if insp.SuggestedName != "test-job" {
		t.Errorf("expected SuggestedName = 'test-job', got %q", insp.SuggestedName)
	}
}
