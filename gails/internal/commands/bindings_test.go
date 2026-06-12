package commands

import "testing"

func TestPluralise(t *testing.T) {
	cases := []struct {
		n    int
		word string
		want string
	}{
		{0, "file", "0 files"},
		{1, "file", "1 file"},
		{2, "file", "2 files"},
		{42, "file", "42 files"},
		{-1, "file", "-1 files"}, // negative counts use the plural form (no special-case)
		{1, "category", "1 category"},
		{0, "category", "0 categorys"}, // pins current behaviour: not a real English pluraliser
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			if got := pluralise(tc.n, tc.word); got != tc.want {
				t.Errorf("pluralise(%d, %q) = %q, want %q", tc.n, tc.word, got, tc.want)
			}
		})
	}
}
