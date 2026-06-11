//go:build windows

package webview2

import "testing"

func TestContainsFullScreenElementEventArgs_Construction(t *testing.T) {
	a := &ContainsFullScreenElementEventArgs{Raw: 0xf001}
	if a.Raw != 0xf001 {
		t.Errorf("Raw = 0x%x", a.Raw)
	}
}

func TestContainsFullScreenElementChangedEventHandler_HasClose(t *testing.T) {
	h := &ContainsFullScreenElementChangedEventHandler{}
	h.Close()
}
