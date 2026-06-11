//go:build windows

package webview2

import "testing"

func TestNewController_ReturnsNonNil(t *testing.T) {
	c := NewController()
	if c == nil {
		t.Fatal("NewController returned nil")
	}
}

func TestController_HasExpectedFields(t *testing.T) {
	c := NewController()
	// Environment may be nil at construction; only check the field is
	// addressable (it would be a bug to have it removed by accident).
	_ = c.Environment
}

func TestController_AddWebResourceRequestedFilter_NilSafe(t *testing.T) {
	c := NewController()
	// Should not panic even when View is nil; the call is queued for
	// when the controller is attached.
	c.AddWebResourceRequestedFilter("*", WebResourceContextAll)
}

func TestController_HasCapability_FalseBeforeAttach(t *testing.T) {
	c := NewController()
	// Without an attached View, HasCapability must return false (no
	// capabilities to query) without panicking.
	if c.HasCapability(CapabilitySwipeNavigation) {
		t.Error("expected false before attach")
	}
}

func TestController_OpenDevToolsWindow_NilSafe(t *testing.T) {
	c := NewController()
	// Should not panic when View is nil.
	c.OpenDevToolsWindow()
}
