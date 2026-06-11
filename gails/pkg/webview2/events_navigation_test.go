//go:build windows

package webview2

import "testing"

func TestNavigationCompletedEventArgs_Construction(t *testing.T) {
	a := &NavigationCompletedEventArgs{Raw: 0xfade}
	if a.Raw != 0xfade {
		t.Errorf("Raw = 0x%x", a.Raw)
	}
}

func TestNavigationCompletedEventHandler_HasClose(t *testing.T) {
	h := &NavigationCompletedEventHandler{}
	h.Close()
}
