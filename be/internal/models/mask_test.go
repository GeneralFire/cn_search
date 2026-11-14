package models

import "testing"

func TestCleanMask(t *testing.T) {
	for in, want := range map[string]struct {
		clean string
		known int
	}{
		"А*23ВС 77*": {"А*23ВС 77*", 7},
		"%а_*-1·":    {"а*1", 2},
		" * * ":      {" * * ", 0},
		"":           {"", 0},
	} {
		if c, k := CleanMask(in); c != want.clean || k != want.known {
			t.Errorf("%q: %q %d, want %q %d", in, c, k, want.clean, want.known)
		}
	}
}
