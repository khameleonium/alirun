package updater

import "testing"

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"1.2.0", "v1.1.0", true},
		{"1.1.0", "v1.1.0", false}, // same version must not be re-installed
		{"1.0.0", "v1.1.0", false}, // never downgrade
		{"0.1.0", "v0.1.0-dev", true},
	}
	for _, c := range cases {
		got, err := isNewerVersion(c.latest, c.current)
		if err != nil || got != c.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, %v; want %v", c.latest, c.current, got, err, c.want)
		}
	}
	if _, err := isNewerVersion("1.0.0", "d82469b-dirty"); err == nil {
		t.Errorf("git-describe dev version must be rejected")
	}
}
