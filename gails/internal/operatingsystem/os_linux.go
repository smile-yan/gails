//go:build linux && !android

package operatingsystem

import (
	"fmt"
	"os"
)

// readOsReleaseFile is a package-level indirection over os.ReadFile so tests
// can inject a fixture without touching the real /etc/os-release. Follows the
// hook-override pattern used in internal/gosod/gosod_test.go.
var readOsReleaseFile = func() ([]byte, error) {
	return os.ReadFile("/etc/os-release")
}

// platformInfo is the platform specific method to get system information
func platformInfo() (*OS, error) {
	_, err := os.Stat("/etc/os-release")
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to read system information")
	}

	osRelease, _ := readOsReleaseFile()
	return parseOsRelease(string(osRelease)), nil
}
