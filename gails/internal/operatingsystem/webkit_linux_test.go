//go:build linux && cgo

package operatingsystem

import "testing"

func TestWebkitVersion_String(t *testing.T) {
	cases := []struct {
		v    WebkitVersion
		want string
	}{
		{WebkitVersion{2, 4, 11}, "v2.4.11"},
		{WebkitVersion{0, 0, 0}, "v0.0.0"},
		{WebkitVersion{1, 0, 0}, "v1.0.0"},
		{WebkitVersion{255, 255, 255}, "v255.255.255"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.v.String(); got != tc.want {
				t.Errorf("%+v.String() = %q, want %q", tc.v, got, tc.want)
			}
		})
	}
}

func TestWebkitVersion_IsAtLeast(t *testing.T) {
	cases := []struct {
		name             string
		v                WebkitVersion
		maj, min, mic    int
		want             bool
	}{
		{"equal", WebkitVersion{2, 4, 11}, 2, 4, 11, true},
		{"greater major", WebkitVersion{3, 0, 0}, 2, 4, 11, true},
		{"lesser major", WebkitVersion{2, 4, 11}, 3, 0, 0, false},
		{"equal major, greater minor", WebkitVersion{2, 5, 0}, 2, 4, 11, true},
		{"equal major+minor, greater micro", WebkitVersion{2, 4, 12}, 2, 4, 11, true},
		{"equal major+minor, lesser micro", WebkitVersion{2, 4, 10}, 2, 4, 11, false},
		{"all zero vs all zero", WebkitVersion{}, 0, 0, 0, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.v.IsAtLeast(tc.maj, tc.min, tc.mic); got != tc.want {
				t.Errorf("%+v.IsAtLeast(%d,%d,%d) = %v, want %v",
					tc.v, tc.maj, tc.min, tc.mic, got, tc.want)
			}
		})
	}
}
