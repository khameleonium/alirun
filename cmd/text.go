package cmd

// truncateRunes shortens s to at most max characters (runes), replacing the tail with "...".
// Slicing by bytes would cut multi-byte (e.g. Cyrillic) characters in half.
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
