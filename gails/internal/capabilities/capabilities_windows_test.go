//go:build windows

package capabilities

import "testing"

// TestWindowsNewCapabilities_EmptyInput exercises the contract that
// `NewCapabilities("")` returns zero values (no webview2 detected, no
// native drag, no version) without panicking.
func TestWindowsNewCapabilities_EmptyInput(t *testing.T) {
	c := NewCapabilities("")
	if c.HasNativeDrag {
		t.Errorf(`HasNativeDrag = true, want false (no webview2 reported for version "")`)
	}
	if c.GTKVersion != 0 {
		t.Errorf("GTKVersion = %d, want 0 (Windows has no GTK)", c.GTKVersion)
	}
	if c.WebKitVersion != "" {
		t.Errorf(`WebKitVersion = %q, want ""`, c.WebKitVersion)
	}
}
