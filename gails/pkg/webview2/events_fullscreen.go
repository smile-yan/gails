//go:build windows

package webview2

import (
	"golang.org/x/sys/windows"
)

// iUnknownVtable is the 3-slot IUnknown vtable layout (QueryInterface,
// AddRef, Release) used as the vtable type for COM interfaces that do
// not extend IUnknown with extra methods. It is a copy of the layout
// defined in internal/webview2/bridge/iunknown.go; it is duplicated
// here (rather than imported) because the bridge package's
// iunknownVtable is unexported and lives in a different module path
// (internal/ vs pkg/) where a public alias would force an awkward
// import. Keeping a private struct in pkg/webview2 keeps the Args type
// usable from external Gails callers without leaking the bridge
// implementation detail.
type iUnknownVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

// ContainsFullScreenElementEventArgs is the COM
// ICoreWebView2ContainsFullScreenElementChangedEventArgs wrapper. It
// is fired when the page's fullscreen-element state changes (e.g.
// when a <video> element enters or exits fullscreen). The interface
// adds no methods beyond IUnknown, so the vtable is the bare
// QueryInterface/AddRef/Release trio. Gails currently only carries
// the raw pointer; a future task that needs to inspect the page
// reference (GetContainsFullScreenElement) can extend this struct
// with the extra vtable slot.
type ContainsFullScreenElementEventArgs struct {
	Raw  uintptr
	vtbl *iUnknownVtable
}

// ContainsFullScreenElementChangedEventHandler is the Go-side
// ICoreWebView2ContainsFullScreenElementChangedEventHandler
// implementation. Construct one with
// NewContainsFullScreenElementChangedEventHandler and pass to the
// ICoreWebView2.AddContainsFullScreenElementChanged wiring on the
// view; call Close when done.
type ContainsFullScreenElementChangedEventHandler struct {
	impl *comHandlerImpl
}

// NewContainsFullScreenElementChangedEventHandler wires a Go
// callback to the
// ICoreWebView2ContainsFullScreenElementChangedEventHandler.Invoke
// vtable slot. The returned handler holds a reference to a native
// COM object; the caller must call Close when finished.
//
// The callback receives the COM "sender" (an ICoreWebView2*) wrapped
// as a *View (forward-declared in events_message.go until
// pkg/webview2/view.go lands in Task 19) and the
// ContainsFullScreenElementEventArgs for this change. The sender is
// passed as nil until the View port lands; the callback must tolerate
// that.
func NewContainsFullScreenElementChangedEventHandler(callback func(view *View, args *ContainsFullScreenElementEventArgs)) *ContainsFullScreenElementChangedEventHandler {
	trampoline := windows.NewCallback(containsFullScreenElementChangedInvokeTrampoline)
	h := NewComHandler(trampoline, callback)
	return &ContainsFullScreenElementChangedEventHandler{impl: h}
}

// Close releases the underlying COM object. Calling Close twice is
// a no-op.
func (h *ContainsFullScreenElementChangedEventHandler) Close() {
	if h.impl == nil {
		return
	}
	h.impl.Release()
	h.impl = nil
}

// containsFullScreenElementChangedInvokeTrampoline is the
// per-instance Invoke slot for the
// ICoreWebView2ContainsFullScreenElementChangedEventHandler vtable.
// It is registered as a C callback via windows.NewCallback and is
// invoked by WebView2 when the page's fullscreen-element state
// changes.
//
// The signature is fixed by COM stdcall: the first argument is the
// COM `this` pointer, followed by the Invoke method's typed
// arguments, and the return value is an HRESULT uintptr.
func containsFullScreenElementChangedInvokeTrampoline(this uintptr, sender uintptr, args uintptr) uintptr {
	impl := comHandlerFromThis(this)
	if impl == nil {
		return 0x80004003 // E_POINTER
	}
	cb, ok := impl.Callback().(func(view *View, args *ContainsFullScreenElementEventArgs))
	if !ok || cb == nil {
		return 0 // S_OK; nothing to do
	}
	// The "sender" argument is an ICoreWebView2*. Wrap it in a
	// Gails View so the callback can issue further webview calls
	// (e.g. Navigate) if it needs to.
	var view *View
	if sender != 0 {
		view = &View{Raw: sender}
	}
	cb(view, &ContainsFullScreenElementEventArgs{Raw: args})
	return 0
}
