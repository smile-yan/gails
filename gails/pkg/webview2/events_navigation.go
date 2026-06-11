//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// NavigationCompletedEventArgs is the COM
// ICoreWebView2NavigationCompletedEventArgs wrapper. Gails uses
// it to detect when a navigation (initial load, redirect, reload,
// fragment, etc.) has completed and to read the success/error
// status. The webview_window_windows.go migration wires the
// NavigationCompleted event to a Go callback that emits the
// WebViewNavigationCompleted window event.
type NavigationCompletedEventArgs struct {
	Raw  uintptr
	vtbl *iCoreWebView2NavigationCompletedEventArgsVtable
}

// iCoreWebView2NavigationCompletedEventArgsVtable is the COM
// ICoreWebView2NavigationCompletedEventArgs vtable. 3 IUnknown
// slots followed by 3 methods in upstream order:
//
//	[0]  QueryInterface
//	[1]  AddRef
//	[2]  Release
//	[3]  GetIsSuccess
//	[4]  GetWebErrorStatus
//	[5]  GetNavigationId
//
// The plan's starting vtable incorrectly listed a separate
// GetErrorStatus slot. Per the upstream WebView2 IDL and the
// reference Go port in
// github.com/wailsapp/wails/webview2/ICoreWebView2NavigationCompletedEventArgs.go
// the interface only has three post-IUnknown methods: IsSuccess,
// WebErrorStatus, and NavigationId. The error status returned by
// GetWebErrorStatus is a COREWEBVIEW2_WEB_ERROR_STATUS enum value
// (uint32), not an HRESULT — if the navigation failed, IsSuccess
// is FALSE and WebErrorStatus describes why.
type iCoreWebView2NavigationCompletedEventArgsVtable struct {
	QueryInterface    uintptr
	AddRef            uintptr
	Release           uintptr
	GetIsSuccess      uintptr
	GetWebErrorStatus uintptr
	GetNavigationId   uintptr
}

// IsSuccess returns whether the navigation succeeded. Returns
// (false, nil) for navigations that failed (network error,
// HTTP error, certificate error, etc.) — callers should consult
// WebErrorStatus to distinguish failure modes.
func (a *NavigationCompletedEventArgs) IsSuccess() (bool, error) {
	vtbl, err := a.vtable()
	if err != nil {
		return false, err
	}
	// COM BOOL is marshalled as a 4-byte int (VARIANT_BOOL).
	var raw int32
	hr, _, _ := syscall.SyscallN(
		vtbl.GetIsSuccess,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(&raw)),
	)
	if hr != 0 {
		return false, fmt.Errorf("ICoreWebView2NavigationCompletedEventArgs::GetIsSuccess failed: 0x%08x", hr)
	}
	return raw != 0, nil
}

// WebErrorStatus returns the COREWEBVIEW2_WEB_ERROR_STATUS value
// describing the failure mode for this navigation. The enum is
// not yet ported to Gails (it lives in a separate task alongside
// the rest of the WebView2 error types); we return the raw uint32
// so callers can compare against the documented enum values
// (e.g. COREWEBVIEW2_WEB_ERROR_STATUS_CONNECTION_RESET = 9).
// Returns 0 when IsSuccess is true.
func (a *NavigationCompletedEventArgs) WebErrorStatus() (uint32, error) {
	vtbl, err := a.vtable()
	if err != nil {
		return 0, err
	}
	var status uint32
	hr, _, _ := syscall.SyscallN(
		vtbl.GetWebErrorStatus,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(&status)),
	)
	if hr != 0 {
		return 0, fmt.Errorf("ICoreWebView2NavigationCompletedEventArgs::GetWebErrorStatus failed: 0x%08x", hr)
	}
	return status, nil
}

// NavigationId returns the unique ID for this navigation. The ID
// increments on every navigation; correlating it with
// NavigationStarting lets callers match starting/completed
// events for the same navigation.
func (a *NavigationCompletedEventArgs) NavigationId() (uint64, error) {
	vtbl, err := a.vtable()
	if err != nil {
		return 0, err
	}
	var id uint64
	hr, _, _ := syscall.SyscallN(
		vtbl.GetNavigationId,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(&id)),
	)
	if hr != 0 {
		return 0, fmt.Errorf("ICoreWebView2NavigationCompletedEventArgs::GetNavigationId failed: 0x%08x", hr)
	}
	return id, nil
}

func (a *NavigationCompletedEventArgs) vtable() (*iCoreWebView2NavigationCompletedEventArgsVtable, error) {
	if a.vtbl != nil {
		return a.vtbl, nil
	}
	if a.Raw == 0 {
		return nil, fmt.Errorf("ICoreWebView2NavigationCompletedEventArgs: nil COM pointer")
	}
	vtblPtr := *(*uintptr)(unsafe.Pointer(a.Raw))
	a.vtbl = (*iCoreWebView2NavigationCompletedEventArgsVtable)(unsafe.Pointer(vtblPtr))
	return a.vtbl, nil
}

// NavigationCompletedEventHandler is the Go-side
// ICoreWebView2NavigationCompletedEventHandler implementation.
// Construct one with NewNavigationCompletedEventHandler and pass
// to View.AddNavigationCompleted; call Close when done.
type NavigationCompletedEventHandler struct {
	impl *comHandlerImpl
}

// NewNavigationCompletedEventHandler wires a Go callback to the
// ICoreWebView2NavigationCompletedEventHandler.Invoke vtable
// slot. The returned handler holds a reference to a native COM
// object; the caller must call Close when finished.
//
// The callback receives the COM "sender" (an ICoreWebView2*)
// wrapped as a *View (forward-declared in events_message.go
// until pkg/webview2/view.go lands in Task 19) and the
// strongly-typed NavigationCompletedEventArgs for this
// navigation. The sender is passed as nil until the View port
// lands; the callback must tolerate that.
func NewNavigationCompletedEventHandler(callback func(view *View, args *NavigationCompletedEventArgs)) *NavigationCompletedEventHandler {
	trampoline := windows.NewCallback(navigationCompletedInvokeTrampoline)
	h := NewComHandler(trampoline, callback)
	return &NavigationCompletedEventHandler{impl: h}
}

// Close releases the underlying COM object. Calling Close twice
// is a no-op.
func (h *NavigationCompletedEventHandler) Close() {
	if h.impl == nil {
		return
	}
	h.impl.Release()
	h.impl = nil
}

// navigationCompletedInvokeTrampoline is the per-instance Invoke
// slot for the ICoreWebView2NavigationCompletedEventHandler
// vtable. It is registered as a C callback via windows.NewCallback
// and is invoked by WebView2 when a navigation completes (success
// or failure).
//
// The signature is fixed by COM stdcall: the first argument is the
// COM `this` pointer, followed by the Invoke method's typed
// arguments, and the return value is an HRESULT uintptr.
func navigationCompletedInvokeTrampoline(this uintptr, sender uintptr, args uintptr) uintptr {
	impl := comHandlerFromThis(this)
	if impl == nil {
		return 0x80004003 // E_POINTER
	}
	cb, ok := impl.Callback().(func(view *View, args *NavigationCompletedEventArgs))
	if !ok || cb == nil {
		return 0 // S_OK; nothing to do
	}
	// The "sender" argument is an ICoreWebView2*. The Gails View
	// wrapper is not yet implemented in this task; we pass a
	// nil view for now and let the callback deal with it. Later
	// tasks (View) will populate this.
	_ = sender
	cb(nil, &NavigationCompletedEventArgs{Raw: args})
	return 0
}
