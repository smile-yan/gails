//go:build windows

package webview2

import "testing"

func TestWebResourceRequest_Construction(t *testing.T) {
	// ICoreWebView2WebResourceRequest has 3 vtable slots
	// (QueryInterface, AddRef, Release) plus 7 methods
	// (GetUri, PutUri, GetMethod, PutMethod, GetContent, PutContent,
	// GetHeaders). Gails only uses the getters; the Puts are
	// included in the vtable layout so the slot indices line up
	// with the upstream ICoreWebView2WebResourceRequest vtable.
	r := &WebResourceRequest{Raw: 0xfeed}
	if r.Raw != 0xfeed {
		t.Errorf("Raw = 0x%x", r.Raw)
	}
}

func TestWebResourceResponse_Construction(t *testing.T) {
	// ICoreWebView2WebResourceResponse has 3 vtable slots
	// (QueryInterface, AddRef, Release) plus 7 methods
	// (GetContent, PutContent, GetHeaders, GetStatusCode, PutStatusCode,
	// GetReasonPhrase, PutReasonPhrase). Gails uses PutContent,
	// PutStatusCode, PutReasonPhrase, and Release.
	r := &WebResourceResponse{Raw: 0xbeef}
	if r.Raw != 0xbeef {
		t.Errorf("Raw = 0x%x", r.Raw)
	}
}

func TestWebResourceRequestedEventArgs_Construction(t *testing.T) {
	// ICoreWebView2WebResourceRequestedEventArgs has 3 vtable slots
	// (QueryInterface, AddRef, Release) plus 5 methods
	// (GetRequest, GetResponse, PutResponse, GetDeferral,
	// GetResourceContext). Gails uses GetRequest, PutResponse, and
	// GetDeferral; GetResponse and GetResourceContext are included
	// in the vtable layout so the slot indices line up.
	a := &WebResourceRequestedEventArgs{Raw: 0xcafe}
	if a.Raw != 0xcafe {
		t.Errorf("Raw = 0x%x", a.Raw)
	}
}

func TestWebResourceRequestedEventHandler_HasClose(t *testing.T) {
	// All *EventHandler types must expose Close() to release the
	// underlying COM object. This is part of the public surface.
	h := &WebResourceRequestedEventHandler{}
	// Close should be callable; it is a no-op on a zero-value handler.
	h.Close()
}
