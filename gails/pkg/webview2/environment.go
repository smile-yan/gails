//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Environment is a Go wrapper over the COM ICoreWebView2Environment interface.
//
// The Raw field is the COM object pointer; vtbl is resolved lazily by
// the methods that need it. Methods Gails actually calls are exposed
// directly on *Environment; the rest of the vtable is laid out (in
// upstream slot order) so the pointer arithmetic for any future port
// lands on the right slot.
type Environment struct {
	Raw  uintptr
	vtbl *iCoreWebView2EnvironmentVtable
}

// iCoreWebView2EnvironmentVtable is the COM ICoreWebView2Environment
// vtable. 3 IUnknown slots followed by 5 environment methods in
// upstream order, matching iCoreWebView2EnvironmentVtbl in
// github.com/wailsapp/wails/webview2/pkg/edge/corewebview2.go.
//
//	[0]  QueryInterface
//	[1]  AddRef
//	[2]  Release
//	[3]  CreateCoreWebView2Controller
//	[4]  CreateWebResourceResponse
//	[5]  GetBrowserVersionString
//	[6]  AddNewBrowserVersionAvailable
//	[7]  RemoveNewBrowserVersionAvailable
//
// Gails only invokes CreateCoreWebView2Controller from this vtable,
// but the full layout is declared so any future port lands on the
// correct slot.
type iCoreWebView2EnvironmentVtable struct {
	QueryInterface                   uintptr
	AddRef                           uintptr
	Release                          uintptr
	CreateCoreWebView2Controller     uintptr
	CreateWebResourceResponse        uintptr
	GetBrowserVersionString          uintptr
	AddNewBrowserVersionAvailable    uintptr
	RemoveNewBrowserVersionAvailable uintptr
}

// vtable resolves and caches the vtable pointer from Raw. The first
// dereference of a COM object always goes through the vtable, so we
// read it once per Environment lifetime.
func (e *Environment) vtable() (*iCoreWebView2EnvironmentVtable, error) {
	if e.vtbl != nil {
		return e.vtbl, nil
	}
	if e.Raw == 0 {
		return nil, fmt.Errorf("ICoreWebView2Environment: nil COM pointer")
	}
	// Standard COM vtable-pointer dereference. The two uintptr
	// conversions silence govet's unsafe.Pointer check (the value
	// cannot be a pointer to a Go object — it is a foreign COM
	// vtable).
	vtblPtr := *(*uintptr)(unsafe.Pointer(e.Raw))
	e.vtbl = (*iCoreWebView2EnvironmentVtable)(unsafe.Pointer(vtblPtr))
	return e.vtbl, nil
}

// CreateCoreWebView2Controller creates a new WebView2 attached to
// the given parent HWND. The call is asynchronous; the actual
// ICoreWebView2Controller pointer is delivered to the completion
// handler. Mirrors ICoreWebView2Environment::CreateCoreWebView2Controller.
//
// The handler holds a reference to a native COM object; the caller
// must call Close on the handler when finished.
func (e *Environment) CreateCoreWebView2Controller(parentHWND uintptr, handler *CreateControllerCompletedHandler) error {
	vtbl, err := e.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.CreateCoreWebView2Controller,
		uintptr(unsafe.Pointer(e)),
		parentHWND,
		uintptr(unsafe.Pointer(handler.impl.COMObject())),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2Environment::CreateCoreWebView2Controller failed: 0x%08x", hr)
	}
	return nil
}

// Controller is a forward declaration for the ICoreWebView2Controller
// wrapper. Task 21 (Plan Task 21) replaces this stub with the real
// implementation. The callback signature for
// CreateControllerCompletedHandler references *Controller, so a stub
// must exist before that handler can be constructed.
type Controller struct {
	Raw  uintptr
	vtbl *iCoreWebView2ControllerVtable
}

// iCoreWebView2ControllerVtable is the forward-declared ICoreWebView2Controller
// vtable. Real layout (with all slot positions) is defined in
// controller.go (Task 21); only the type is needed here so the
// CreateControllerCompletedHandler callback can take a *Controller.
type iCoreWebView2ControllerVtable struct {
	QueryInterface                     uintptr
	AddRef                             uintptr
	Release                            uintptr
	GetICoreWebView2                   uintptr
	SetBounds                          uintptr
	GetBounds                          uintptr
	GetVisible                         uintptr
	SetVisible                         uintptr
	GetIsVisible                       uintptr
	GetParentWindow                    uintptr
	put_ParentWindow                   uintptr
	NotifyParentWindowPositionChanged  uintptr
	Close                              uintptr
	GetCoreWebView2Controller          uintptr
	AddAcceleratorKeyPressed           uintptr
	RemoveAcceleratorKeyPressed        uintptr
	MoveFocus                          uintptr
	AddGotFocus                        uintptr
	RemoveGotFocus                     uintptr
	AddLostFocus                       uintptr
	RemoveLostFocus                    uintptr
	AddZoomFactorChanged               uintptr
	RemoveZoomFactorChanged            uintptr
	GetZoomFactor                      uintptr
	put_ZoomFactor                     uintptr
	SetBoundsAndZoomFactor             uintptr
	MoveFocusReason                    uintptr
	NotifyParentWindowPositionChanged2 uintptr
}

// CreateControllerCompletedHandler is the Go-side implementation of
// ICoreWebView2CreateCoreWebView2ControllerCompletedHandler.
// Construct one with NewCreateControllerCompletedHandler and pass to
// Environment.CreateCoreWebView2Controller; call Close when done.
type CreateControllerCompletedHandler struct {
	impl *comHandlerImpl
}

// NewCreateControllerCompletedHandler wires a Go callback to the
// ICoreWebView2CreateCoreWebView2ControllerCompletedHandler.Invoke
// vtable slot. The returned handler holds a reference to a native
// COM object; the caller must call Close when finished.
func NewCreateControllerCompletedHandler(callback func(result int32, controller *Controller)) *CreateControllerCompletedHandler {
	trampoline := windows.NewCallback(createControllerCompletedInvokeTrampoline)
	h := NewComHandler(trampoline, callback)
	return &CreateControllerCompletedHandler{impl: h}
}

// Close releases the underlying COM object. Calling Close twice is
// a no-op.
func (h *CreateControllerCompletedHandler) Close() {
	if h.impl == nil {
		return
	}
	h.impl.Release()
	h.impl = nil
}

// createControllerCompletedInvokeTrampoline is the per-instance
// Invoke slot for the
// ICoreWebView2CreateCoreWebView2ControllerCompletedHandler vtable.
// It is registered as a C callback via windows.NewCallback and is
// invoked by WebView2 when controller creation finishes.
//
// The signature is fixed by COM stdcall: the first argument is the
// COM `this` pointer, followed by the Invoke method's typed
// arguments (HRESULT and the new ICoreWebView2Controller*), and the
// return value is an HRESULT uintptr.
func createControllerCompletedInvokeTrampoline(this uintptr, errorCode uintptr, createdController uintptr) uintptr {
	impl := comHandlerFromThis(this)
	if impl == nil {
		return 0x80004003 // E_POINTER
	}
	cb, ok := impl.Callback().(func(result int32, controller *Controller))
	if !ok || cb == nil {
		return 0 // S_OK; nothing to do
	}
	var controller *Controller
	if createdController != 0 {
		// Task 21 will populate the ICoreWebView2Controller vtable
		// slots on Controller. For now, only Raw is set so the
		// callback can at least inspect whether the pointer is
		// non-nil.
		controller = &Controller{Raw: createdController}
	}
	cb(int32(errorCode), controller)
	return 0
}
