package cmd

import (
	"testing"
	"unicode/utf8"
)

func TestTruncateRunes(t *testing.T) {
	s := "Каждые 15 минут запускать резервное копирование"
	got := truncateRunes(s, 23)
	if !utf8.ValidString(got) || utf8.RuneCountInString(got) != 23 {
		t.Errorf("bad truncation: %q", got)
	}
	if truncateRunes("short", 23) != "short" {
		t.Errorf("short strings must be unchanged")
	}
}
