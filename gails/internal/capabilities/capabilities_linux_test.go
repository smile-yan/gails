//go:build linux && !gtk3

package capabilities

import "testing"

// TestLinuxNewCapabilities locks in the GTK4/WebKit 6 contract for the
// default Linux build (no -tags gtk3).
func TestLinuxNewCapabilities(t *testing.T) {
	c := NewCapabilities()
	if !c.HasNativeDrag {
		t.Errorf("HasNativeDrag = false, want true (Linux always supports native drag)")
	}
	if c.GTKVersion != 4 {
		t.Errorf("GTKVersion = %d, want 4", c.GTKVersion)
	}
	if c.WebKitVersion != "6.0" {
		t.Errorf("WebKitVersion = %q, want %q", c.WebKitVersion, "6.0")
	}
}
