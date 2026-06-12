//go:build windows

package operatingsystem

import "testing"

func TestWindowsVersionInfo_IsAtLeast(t *testing.T) {
	cases := []struct {
		name                       string
		have                       WindowsVersionInfo
		major, minor, buildNumber  int
		want                       bool
	}{
		{"equal", WindowsVersionInfo{10, 0, 19045, ""}, 10, 0, 19045, true},
		{"greater major", WindowsVersionInfo{11, 0, 22000, ""}, 10, 0, 99999, true},
		{"lesser major", WindowsVersionInfo{7, 1, 7601, ""}, 10, 0, 0, false},
		{"greater minor", WindowsVersionInfo{10, 1, 19045, ""}, 10, 0, 99999, true},
		{"lesser minor", WindowsVersionInfo{10, 0, 19045, ""}, 10, 1, 0, false},
		{"greater build", WindowsVersionInfo{10, 0, 20000, ""}, 10, 0, 19045, true},
		{"lesser build", WindowsVersionInfo{10, 0, 19044, ""}, 10, 0, 19045, false},
		{"all zeros vs all zeros", WindowsVersionInfo{}, 0, 0, 0, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.have.IsWindowsVersionAtLeast(tc.major, tc.minor, tc.buildNumber); got != tc.want {
				t.Errorf("%+v.IsAtLeast(%d,%d,%d) = %v, want %v",
					tc.have, tc.major, tc.minor, tc.buildNumber, got, tc.want)
			}
		})
	}
}

// TestRegHelpers_Skip: the regDWORDKeyAsInt / regStringKeyAsInt / regKeyAsString
// helpers take a real golang.org/x/sys/windows/registry.Key. The registry package
// has no interface seam to inject a fake, so we cannot unit-test these helpers
// on a non-Windows host. The Windows host runs GetWindowsVersionInfo in
// integration-style manual testing.
//
// CI: these helpers are exercised by `GetWindowsVersionInfo` smoke tests in a
// follow-up PR that adds a real-Windows CI runner.
func TestRegHelpers_Skip(t *testing.T) {
	t.Skip("reg* helpers require a real registry.Key; golang.org/x/sys/windows/registry has no fake. Add Windows-host CI to exercise them.")
}
