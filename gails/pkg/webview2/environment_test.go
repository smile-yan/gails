//go:build windows

package webview2

import "testing"

func TestEnvironment_Construction(t *testing.T) {
	// ICoreWebView2Environment has 8 vtable slots (3 IUnknown + 5
	// environment methods). Gails uses CreateCoreWebView2Controller.
	e := &Environment{Raw: 0xabcd}
	if e.Raw != 0xabcd {
		t.Errorf("Raw = 0x%x", e.Raw)
	}
}
