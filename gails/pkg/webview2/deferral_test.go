//go:build windows

package webview2

import "testing"

func TestDeferral_Construction(t *testing.T) {
	// Deferral wraps ICoreWebView2Deferral. The COM interface has 3 vtable
	// slots: QueryInterface, AddRef, Release, Complete. Verify the field
	// shape.
	d := &Deferral{Raw: 0x5678}
	if d.Raw != 0x5678 {
		t.Errorf("Raw = 0x%x", d.Raw)
	}
}
