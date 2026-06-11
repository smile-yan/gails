//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// MessageReceivedEventArgs is the COM ICoreWebView2WebMessageReceivedEventArgs
// wrapper. Call TryGetWebMessageAsString to read the message body.
type MessageReceivedEventArgs struct {
	Raw  uintptr
	vtbl *iCoreWebView2WebMessageReceivedEventArgsVtable
}

type iCoreWebView2WebMessageReceivedEventArgsVtable struct {
	QueryInterface           uintptr
	AddRef                   uintptr
	Release                  uintptr
	TryGetWebMessageAsString uintptr
	GetAdditionalObjects     uintptr
}

// TryGetWebMessageAsString returns the WebMessage posted by the
// webview as a UTF-8 string. The COM method allocates the string
// with CoTaskMemAlloc; Gails is not responsible for freeing it
// (the WebView2 runtime owns the buffer).
func (a *MessageReceivedEventArgs) TryGetWebMessageAsString() (string, error) {
	if a.vtbl == nil {
		vtblPtr := *(*uintptr)(unsafe.Pointer(a.Raw))
		a.vtbl = (*iCoreWebView2WebMessageReceivedEventArgsVtable)(unsafe.Pointer(vtblPtr))
	}
	var p *uint16
	hr, _, _ := syscall.SyscallN(
		a.vtbl.TryGetWebMessageAsString,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(&p)),
	)
	if hr != 0 {
		return "", fmt.Errorf("TryGetWebMessageAsString failed: 0x%08x", hr)
	}
	return windows.UTF16PtrToString(p), nil
}

// Source returns the URI of the document that posted the message.
// Mirrors upstream ICoreWebView2WebMessageReceivedEventArgs::Source.
//
// TODO(port): the GetSource slot is at vtable index [4] (after
// QueryInterface/AddRef/Release/TryGetWebMessageAsString). Add
// the vtable slot to the struct and wire this method.
func (a *MessageReceivedEventArgs) Source() (string, error) {
	return "", nil
}

// AdditionalObjects returns the additional objects posted with
// the message (used by the file-drop flow to ship dropped
// ICoreWebView2File objects). Mirrors upstream
// ICoreWebView2WebMessageReceivedEventArgs::GetAdditionalObjects.
//
// TODO(port): proper COM implementation; the file-drop plumbing
// lives in pkg/assetserver/webview. For now the call site uses
// args.GetAdditionalObjects() and dereferences the result, so
// a stub returning a non-nil placeholder lets the compile pass
// while the real implementation is a Plan Task 28 follow-up.
func (a *MessageReceivedEventArgs) GetAdditionalObjects() (*AdditionalObjects, error) {
	return &AdditionalObjects{}, nil
}

// AdditionalObjects is a placeholder for the
// ICoreWebView2WebMessageReceivedEventArgsCollection returned by
// GetAdditionalObjects. The full implementation (Count, ValueAtIndex,
// Release) is ported in a follow-up task; today only Release is
// invoked by the application layer, so an empty stub suffices.
type AdditionalObjects struct {
	Raw uintptr
}

// GetCount returns the number of additional objects in the
// collection.
func (o *AdditionalObjects) GetCount() (uint32, error) {
	return 0, nil
}

// GetValueAtIndex returns the ICoreWebView2File at the given
// index in the collection.
func (o *AdditionalObjects) GetValueAtIndex(_ uint32) (*File, error) {
	return nil, nil
}

// Release releases the underlying COM object.
func (o *AdditionalObjects) Release() error {
	return nil
}

// MessageReceivedEventHandler is the Go-side
// ICoreWebView2WebMessageReceivedEventHandler implementation.
// Construct one with NewMessageReceivedEventHandler and pass to
// View.AddWebMessageReceived; call Close when done.
type MessageReceivedEventHandler struct {
	impl *comHandlerImpl
}

// NewMessageReceivedEventHandler wires a Go callback to the
// ICoreWebView2WebMessageReceivedEventHandler.Invoke vtable slot.
// The returned handler holds a reference to a native COM object;
// the caller must call Close when finished.
func NewMessageReceivedEventHandler(callback func(view *View, args *MessageReceivedEventArgs)) *MessageReceivedEventHandler {
	trampoline := windows.NewCallback(messageReceivedInvokeTrampoline)
	h := NewComHandler(trampoline, callback)
	return &MessageReceivedEventHandler{impl: h}
}

// Close releases the underlying COM object. Calling Close twice is
// a no-op.
func (h *MessageReceivedEventHandler) Close() {
	if h.impl == nil {
		return
	}
	h.impl.Release()
	h.impl = nil
}

// messageReceivedInvokeTrampoline is the per-instance Invoke slot
// for the ICoreWebView2WebMessageReceivedEventHandler vtable. It
// is registered as a C callback via windows.NewCallback and is
// invoked by WebView2 when the frontend posts a message.
//
// The signature is fixed by COM stdcall: the first argument is the
// COM `this` pointer, followed by the Invoke method's typed
// arguments, and the return value is an HRESULT uintptr.
func messageReceivedInvokeTrampoline(this uintptr, sender uintptr, args uintptr) uintptr {
	impl := comHandlerFromThis(this)
	if impl == nil {
		return 0x80004003 // E_POINTER
	}
	cb, ok := impl.Callback().(func(view *View, args *MessageReceivedEventArgs))
	if !ok || cb == nil {
		return 0 // S_OK; nothing to do
	}
	// The "sender" argument is an ICoreWebView2*. Wrap it in a
	// Gails View so the callback can issue further webview calls
	// (e.g. PostWebMessageAsString) if it needs to.
	var view *View
	if sender != 0 {
		view = &View{Raw: sender}
	}
	cb(view, &MessageReceivedEventArgs{Raw: args})
	return 0
}
