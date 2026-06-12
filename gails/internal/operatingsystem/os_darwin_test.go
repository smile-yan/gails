//go:build darwin

package operatingsystem

import "testing"

func TestGetOSName(t *testing.T) {
	cases := []struct {
		version string
		want    string
	}{
		// Known 10.x names
		{"10.10", "Yosemite"},
		{"10.13", "High Sierra"},
		{"10.15", "Catalina"},
		// 11+ single-segment
		{"11", "Big Sur"},
		{"12", "Monterey"},
		{"13", "Ventura"},
		{"14", "Sonoma"},
		{"15", "Sequoia"},
		// Patch versions: the `!strings.HasPrefix(version, "10.")` branch decides
		// whether the version is trimmed. 10.x is NOT trimmed (so 10.15.7 misses
		// the lookup), while 11+ IS trimmed (so 14.5 maps via "14" → Sonoma).
		{"10.15.7", "MacOS 10.15.7"},
		{"14.5", "Sonoma"},
		{"99.0", "MacOS 99.0"},
		{"", "MacOS "},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.version, func(t *testing.T) {
			if got := getOSName(tc.version); got != tc.want {
				t.Errorf("getOSName(%q) = %q, want %q", tc.version, got, tc.want)
			}
		})
	}
}

// TestGetSysctlValue_KernOstype is a smoke test using the most stable sysctl
// key. We do not assert on the exact value (it varies by macOS version); we
// only assert no error and a non-empty trimmed result.
func TestGetSysctlValue_KernOstype(t *testing.T) {
	got, err := getSysctlValue("kern.ostype")
	if err != nil {
		t.Fatalf("getSysctlValue(kern.ostype) error: %v", err)
	}
	if got == "" {
		t.Error("getSysctlValue(kern.ostype) returned empty string")
	}
}
