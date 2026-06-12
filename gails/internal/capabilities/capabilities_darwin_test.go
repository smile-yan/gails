//go:build darwin

package capabilities

import "testing"

// TestDarwinNewCapabilities locks in the darwin-specific contract:
// on macOS, native drag is not supported and the version fields are empty.
func TestDarwinNewCapabilities(t *testing.T) {
	c := newCapabilities("ignored-arg")
	if c.HasNativeDrag != false {
		t.Errorf("HasNativeDrag = %v, want false", c.HasNativeDrag)
	}
	if c.GTKVersion != 0 {
		t.Errorf("GTKVersion = %d, want 0", c.GTKVersion)
	}
	if c.WebKitVersion != "" {
		t.Errorf("WebKitVersion = %q, want empty", c.WebKitVersion)
	}
}
