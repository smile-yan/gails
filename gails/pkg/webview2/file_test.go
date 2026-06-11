//go:build windows

package webview2

import "testing"

func TestFile_Construction(t *testing.T) {
	// ICoreWebView2File wraps a file the user dropped into the webview.
	// The COM interface has 4 vtable slots: QueryInterface, AddRef,
	// Release, GetPath. Verify the field shape.
	f := &File{Raw: 0x9abc}
	if f.Raw != 0x9abc {
		t.Errorf("Raw = 0x%x", f.Raw)
	}
}
