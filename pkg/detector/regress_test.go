package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMakeExecutableDoesNotWidenAccess(t *testing.T) {
	cases := map[os.FileMode]os.FileMode{0600: 0700, 0644: 0755, 0640: 0750}
	for in, want := range cases {
		p := filepath.Join(t.TempDir(), "s.sh")
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), in); err != nil {
			t.Fatal(err)
		}
		_ = os.Chmod(p, in)
		if err := MakeExecutable(p); err != nil {
			t.Fatal(err)
		}
		st, _ := os.Stat(p)
		if st.Mode().Perm() != want {
			t.Errorf("%o -> %o, want %o", in, st.Mode().Perm(), want)
		}
	}
}
