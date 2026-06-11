//go:build windows

package webview2

import "testing"

func TestMessageReceivedEventArgs_Construction(t *testing.T) {
	// ICoreWebView2WebMessageReceivedEventArgs has 3 vtable slots
	// (QueryInterface, AddRef, Release) plus 2 methods (TryGetWebMessageAsString,
	// get_AdditionalObjects).
	a := &MessageReceivedEventArgs{Raw: 0xdead}
	if a.Raw != 0xdead {
		t.Errorf("Raw = 0x%x", a.Raw)
	}
}

func TestMessageReceivedEventHandler_HasClose(t *testing.T) {
	// All *EventHandler types must expose Close() to release the underlying
	// COM object. This is part of the public surface.
	h := &MessageReceivedEventHandler{}
	// Close should be callable; it may be a no-op on a zero-value handler.
	h.Close()
}
