package fileexplorer_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gailsapp/gails/internal/fileexplorer"
)

// Credit: https://stackoverflow.com/a/50631395
func skipCI(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
}

func TestFileExplorer(t *testing.T) {
	skipCI(t)
	// TestFileExplorer verifies that the OpenFileManager function correctly handles:
	// - Opening files in the native file manager across different platforms
	// - Selecting files when the selectFile parameter is true
	// - Various error conditions like non-existent paths
	tempDir := t.TempDir() // Create a temporary directory for tests

	tests := []struct {
		name        string
		path        string
		selectFile  bool
		expectedErr string // substring expected in the error message; "" means no error expected
	}{
		// Success cases — OpenFileManager should return nil.
		{"Open Existing File", tempDir, false, ""},
		{"Select Existing File", tempDir, true, ""},
		{"Path with Special Characters", filepath.Join(tempDir, "test space.txt"), true, ""},
		// Error cases — OpenFileManager should return an error containing expectedErr.
		// The check is performed by fileexplorer.go before invoking the OS file
		// manager, so these cases don't depend on a GUI session.
		{"Non-Existent Path", "/path/does/not/exist", false, "failed to access the specified path"},
		{"No Permission Path", "/root/test.txt", false, "failed to access the specified path"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Run("Windows", func(t *testing.T) {
				runPlatformTest(t, "windows", test.path, test.selectFile, test.expectedErr)
			})
			t.Run("Linux", func(t *testing.T) {
				runPlatformTest(t, "linux", test.path, test.selectFile, test.expectedErr)
			})
			t.Run("Darwin", func(t *testing.T) {
				runPlatformTest(t, "darwin", test.path, test.selectFile, test.expectedErr)
			})
		})
	}
}

func runPlatformTest(t *testing.T, platform, path string, selectFile bool, expectedErr string) {
	if runtime.GOOS != platform {
		t.Skipf("Skipping test on non-%s platform", strings.ToTitle(platform))
	}

	// For success cases, ensure the target file actually exists before invoking
	// the file manager. Error cases deliberately point at non-existent paths and
	// must be left alone so the early os.Stat check in OpenFileManager can fire.
	if expectedErr == "" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte("Test file contents"), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	err := fileexplorer.OpenFileManager(path, selectFile)

	switch {
	case expectedErr == "" && err != nil:
		t.Errorf("OpenFileManager(%q, %v) unexpected error: %v", path, selectFile, err)
	case expectedErr != "" && err == nil:
		t.Errorf("OpenFileManager(%q, %v) expected error containing %q, got nil",
			path, selectFile, expectedErr)
	case expectedErr != "" && !strings.Contains(err.Error(), expectedErr):
		t.Errorf("OpenFileManager(%q, %v) error = %q, want substring %q",
			path, selectFile, err.Error(), expectedErr)
	}
}
