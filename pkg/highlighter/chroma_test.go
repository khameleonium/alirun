package highlighter

import (
	"strings"
	"testing"
)

func TestHighlightUnit(t *testing.T) {
	input := "[Unit]\nDescription=Test\n[Service]\nExecStart=/usr/bin/test\n"
	highlighted := HighlightUnit(input)
	if !strings.Contains(highlighted, "Description") {
		t.Errorf("expected highlighted output to contain Description")
	}
}

func TestSimpleDiff(t *testing.T) {
	oldContent := "Line 1\nLine 2\n"
	newContent := "Line 1\nLine 2 Modified\n"
	diff := SimpleDiff(oldContent, newContent)
	if !strings.Contains(diff, "--- Original") {
		t.Errorf("expected diff header --- Original")
	}
	if !strings.Contains(diff, "+++ Generated") {
		t.Errorf("expected diff header +++ Generated")
	}
}
