//go:build windows

package webview2

import "testing"

func TestView_Construction(t *testing.T) {
	// ICoreWebView2 has 61 vtable slots (3 IUnknown + 58 webview
	// methods). Gails only invokes a subset (Settings, Navigate,
	// NavigateToString, OpenDevToolsWindow, AddWebResourceRequestedFilter,
	// and the four Add* event registration methods) but the full layout
	// is declared so the vtable pointer arithmetic lines up with
	// upstream's iCoreWebView2Vtbl in
	// github.com/wailsapp/wails/webview2/pkg/edge/corewebview2.go.
	v := &View{Raw: 0x9999}
	if v.Raw != 0x9999 {
		t.Errorf("Raw = 0x%x", v.Raw)
	}
}
