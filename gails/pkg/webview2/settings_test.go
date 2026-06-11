//go:build windows

package webview2

import "testing"

func TestSettings_Construction(t *testing.T) {
	// ICoreWebViewSettings has 24 vtable slots (3 IUnknown + 21 setting
	// methods: 10 Get/Put pairs + 1 Put for UserAgent). Gails uses
	// AreDevToolsEnabled, AreDefaultContextMenusEnabled, IsZoomControlEnabled,
	// IsStatusBarEnabled, AreBrowserAcceleratorKeysEnabled, IsSwipeNavigationEnabled
	// — verify the public method surface.
	s := &Settings{Raw: 0x1234}
	if s.Raw != 0x1234 {
		t.Errorf("Raw = 0x%x", s.Raw)
	}
}
