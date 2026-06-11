//go:build windows

package webview2

import "testing"

func TestWebResourceContext_AllIsZero(t *testing.T) {
	// Upstream defines COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL = 0. Our
	// rename must preserve the value 0 because Gails passes it directly
	// to Chromium.AddWebResourceRequestedFilter("*", ctx).
	if int(WebResourceContextAll) != 0 {
		t.Errorf("WebResourceContextAll = %d, want 0", int(WebResourceContextAll))
	}
}

func TestCapability_Constants(t *testing.T) {
	// SwipeNavigation's value must match upstream's edge.SwipeNavigation
	// constant — which itself is an arbitrary integer the WebView2
	// runtime understands. We don't assert a specific number; we assert
	// the named constant is non-zero and stable.
	if int(CapabilitySwipeNavigation) == 0 {
		t.Error("CapabilitySwipeNavigation should be non-zero")
	}
}

func TestRect_ZeroValue(t *testing.T) {
	var r Rect
	if r != (Rect{}) {
		t.Errorf("Rect zero value: got %+v", r)
	}
	if r.Width() != 0 || r.Height() != 0 {
		t.Error("zero Rect must have zero width/height")
	}
}

func TestRect_WidthHeight(t *testing.T) {
	r := Rect{Left: 10, Top: 20, Right: 110, Bottom: 220}
	if r.Width() != 100 {
		t.Errorf("Width = %d, want 100", r.Width())
	}
	if r.Height() != 200 {
		t.Errorf("Height = %d, want 200", r.Height())
	}
}
