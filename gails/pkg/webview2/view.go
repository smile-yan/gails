//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// View is a Go wrapper over the COM ICoreWebView2 interface. It
// represents a single webview2 instance attached to a window.
//
// The Raw field is the COM object pointer; vtbl is resolved lazily
// by the methods that need it. Methods Gails actually calls are
// exposed directly on *View; the rest of the vtable is laid out
// (in upstream slot order) so the pointer arithmetic for any future
// port lands on the right slot.
type View struct {
	Raw  uintptr
	vtbl *iCoreWebView2Vtable
}

// iCoreWebView2Vtable is the COM ICoreWebView2 vtable. 3 IUnknown
// slots followed by 58 methods in upstream order, matching
// iCoreWebView2Vtbl in
// github.com/wailsapp/wails/webview2/pkg/edge/corewebview2.go.
//
//	[0]   QueryInterface
//	[1]   AddRef
//	[2]   Release
//	[3]   GetSettings
//	[4]   GetSource
//	[5]   Navigate
//	[6]   NavigateToString
//	[7]   AddNavigationStarting
//	[8]   RemoveNavigationStarting
//	[9]   AddContentLoading
//	[10]  RemoveContentLoading
//	[11]  AddSourceChanged
//	[12]  RemoveSourceChanged
//	[13]  AddHistoryChanged
//	[14]  RemoveHistoryChanged
//	[15]  AddNavigationCompleted
//	[16]  RemoveNavigationCompleted
//	[17]  AddFrameNavigationStarting
//	[18]  RemoveFrameNavigationStarting
//	[19]  AddFrameNavigationCompleted
//	[20]  RemoveFrameNavigationCompleted
//	[21]  AddScriptDialogOpening
//	[22]  RemoveScriptDialogOpening
//	[23]  AddPermissionRequested
//	[24]  RemovePermissionRequested
//	[25]  AddProcessFailed
//	[26]  RemoveProcessFailed
//	[27]  AddScriptToExecuteOnDocumentCreated
//	[28]  RemoveScriptToExecuteOnDocumentCreated
//	[29]  ExecuteScript
//	[30]  CapturePreview
//	[31]  Reload
//	[32]  PostWebMessageAsJSON
//	[33]  PostWebMessageAsString
//	[34]  AddWebMessageReceived
//	[35]  RemoveWebMessageReceived
//	[36]  CallDevToolsProtocolMethod
//	[37]  GetBrowserProcessID
//	[38]  GetCanGoBack
//	[39]  GetCanGoForward
//	[40]  GoBack
//	[41]  GoForward
//	[42]  GetDevToolsProtocolEventReceiver
//	[43]  Stop
//	[44]  AddNewWindowRequested
//	[45]  RemoveNewWindowRequested
//	[46]  AddDocumentTitleChanged
//	[47]  RemoveDocumentTitleChanged
//	[48]  GetDocumentTitle
//	[49]  AddHostObjectToScript
//	[50]  RemoveHostObjectFromScript
//	[51]  OpenDevToolsWindow
//	[52]  AddContainsFullScreenElementChanged
//	[53]  RemoveContainsFullScreenElementChanged
//	[54]  GetContainsFullScreenElement
//	[55]  AddWebResourceRequested
//	[56]  RemoveWebResourceRequested
//	[57]  AddWebResourceRequestedFilter
//	[58]  RemoveWebResourceRequestedFilter
//	[59]  AddWindowCloseRequested
//	[60]  RemoveWindowCloseRequested
//
// Gails only invokes a small subset of these (Settings, Navigate,
// NavigateToString, OpenDevToolsWindow, AddWebResourceRequestedFilter,
// and the four Add* event registration methods), but the full
// layout is declared so any future port lands on the correct slot.
type iCoreWebView2Vtable struct {
	QueryInterface                        uintptr
	AddRef                                uintptr
	Release                               uintptr
	GetSettings                           uintptr
	GetSource                             uintptr
	Navigate                              uintptr
	NavigateToString                      uintptr
	AddNavigationStarting                 uintptr
	RemoveNavigationStarting              uintptr
	AddContentLoading                     uintptr
	RemoveContentLoading                  uintptr
	AddSourceChanged                      uintptr
	RemoveSourceChanged                   uintptr
	AddHistoryChanged                     uintptr
	RemoveHistoryChanged                  uintptr
	AddNavigationCompleted                uintptr
	RemoveNavigationCompleted             uintptr
	AddFrameNavigationStarting            uintptr
	RemoveFrameNavigationStarting         uintptr
	AddFrameNavigationCompleted           uintptr
	RemoveFrameNavigationCompleted        uintptr
	AddScriptDialogOpening                uintptr
	RemoveScriptDialogOpening             uintptr
	AddPermissionRequested                uintptr
	RemovePermissionRequested             uintptr
	AddProcessFailed                      uintptr
	RemoveProcessFailed                   uintptr
	AddScriptToExecuteOnDocumentCreated   uintptr
	RemoveScriptToExecuteOnDocumentCreated uintptr
	ExecuteScript                         uintptr
	CapturePreview                        uintptr
	Reload                                uintptr
	PostWebMessageAsJSON                  uintptr
	PostWebMessageAsString                uintptr
	AddWebMessageReceived                 uintptr
	RemoveWebMessageReceived              uintptr
	CallDevToolsProtocolMethod            uintptr
	GetBrowserProcessID                   uintptr
	GetCanGoBack                          uintptr
	GetCanGoForward                       uintptr
	GoBack                                uintptr
	GoForward                             uintptr
	GetDevToolsProtocolEventReceiver      uintptr
	Stop                                  uintptr
	AddNewWindowRequested                 uintptr
	RemoveNewWindowRequested              uintptr
	AddDocumentTitleChanged               uintptr
	RemoveDocumentTitleChanged            uintptr
	GetDocumentTitle                      uintptr
	AddHostObjectToScript                 uintptr
	RemoveHostObjectFromScript            uintptr
	OpenDevToolsWindow                    uintptr
	AddContainsFullScreenElementChanged   uintptr
	RemoveContainsFullScreenElementChanged uintptr
	GetContainsFullScreenElement          uintptr
	AddWebResourceRequested               uintptr
	RemoveWebResourceRequested            uintptr
	AddWebResourceRequestedFilter         uintptr
	RemoveWebResourceRequestedFilter      uintptr
	AddWindowCloseRequested               uintptr
	RemoveWindowCloseRequested            uintptr
}

// vtable resolves and caches the vtable pointer from Raw. The first
// dereference of a COM object always goes through the vtable, so we
// read it once per View lifetime.
func (v *View) vtable() (*iCoreWebView2Vtable, error) {
	if v.vtbl != nil {
		return v.vtbl, nil
	}
	if v.Raw == 0 {
		return nil, fmt.Errorf("ICoreWebView2: nil COM pointer")
	}
	// Standard COM vtable-pointer dereference. The two uintptr
	// conversions silence govet's unsafe.Pointer check (the value
	// cannot be a pointer to a Go object — it is a foreign COM
	// vtable).
	vtblPtr := *(*uintptr)(unsafe.Pointer(v.Raw))
	v.vtbl = (*iCoreWebView2Vtable)(unsafe.Pointer(vtblPtr))
	return v.vtbl, nil
}

// Settings returns the ICoreWebViewSettings associated with this
// WebView. Mirrors ICoreWebView2::get_Settings.
func (v *View) Settings() (*Settings, error) {
	vtbl, err := v.vtable()
	if err != nil {
		return nil, err
	}
	var raw uintptr
	hr, _, _ := syscall.SyscallN(
		vtbl.GetSettings,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&raw)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("ICoreWebView2::get_Settings failed: 0x%08x", hr)
	}
	if raw == 0 {
		return nil, nil
	}
	return &Settings{Raw: raw}, nil
}

// Navigate loads the given URI in the WebView. Mirrors
// ICoreWebView2::Navigate.
func (v *View) Navigate(uri string) error {
	vtbl, err := v.vtable()
	if err != nil {
		return err
	}
	p, err := windows.UTF16PtrFromString(uri)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.Navigate,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(p)),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2::Navigate failed: 0x%08x", hr)
	}
	return nil
}

// NavigateToString loads the given HTML string in the WebView,
// with no associated URL. Mirrors ICoreWebView2::NavigateToString.
func (v *View) NavigateToString(html string) error {
	vtbl, err := v.vtable()
	if err != nil {
		return err
	}
	p, err := windows.UTF16PtrFromString(html)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.NavigateToString,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(p)),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2::NavigateToString failed: 0x%08x", hr)
	}
	return nil
}

// OpenDevToolsWindow opens the browser-style dev tools window for
// the WebView. Mirrors ICoreWebView2::OpenDevToolsWindow.
func (v *View) OpenDevToolsWindow() error {
	vtbl, err := v.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.OpenDevToolsWindow,
		uintptr(unsafe.Pointer(v)),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2::OpenDevToolsWindow failed: 0x%08x", hr)
	}
	return nil
}

// AddWebResourceRequestedFilter registers a URI wildcard and
// resource context filter so that AddWebResourceRequested fires
// only for matching requests. Mirrors
// ICoreWebView2::AddWebResourceRequestedFilter.
//
// The context argument is a COREWEBVIEW2_WEB_RESOURCE_CONTEXT value
// (uint32 in the WebView2 IDL). The port task that defines that
// enum will replace this with a typed constant; for now the caller
// passes the underlying uint32 to keep the wrapper decoupled.
func (v *View) AddWebResourceRequestedFilter(uri string, context uint32) error {
	vtbl, err := v.vtable()
	if err != nil {
		return err
	}
	p, err := windows.UTF16PtrFromString(uri)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.AddWebResourceRequestedFilter,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(p)),
		uintptr(context),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2::AddWebResourceRequestedFilter failed: 0x%08x", hr)
	}
	return nil
}

// AddWebMessageReceived registers a Go callback to be invoked
// when the webview posts a message via window.chrome.webview.postMessage.
// Mirrors ICoreWebView2::AddWebMessageReceived.
//
// The handler holds a reference to a native COM object; the caller
// must call Close on the handler when finished.
func (v *View) AddWebMessageReceived(handler *MessageReceivedEventHandler) error {
	vtbl, err := v.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.AddWebMessageReceived,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(handler.impl.COMObject())),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2::AddWebMessageReceived failed: 0x%08x", hr)
	}
	return nil
}

// AddWebResourceRequested registers a Go callback to be invoked
// when the WebView2 issues a request that matches one of the
// previously-registered resource filters. Mirrors
// ICoreWebView2::AddWebResourceRequested.
//
// The handler holds a reference to a native COM object; the caller
// must call Close on the handler when finished.
func (v *View) AddWebResourceRequested(handler *WebResourceRequestedEventHandler) error {
	vtbl, err := v.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.AddWebResourceRequested,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(handler.impl.COMObject())),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2::AddWebResourceRequested failed: 0x%08x", hr)
	}
	return nil
}

// AddNavigationCompleted registers a Go callback to be invoked when
// a navigation completes (success or failure). Mirrors
// ICoreWebView2::AddNavigationCompleted.
//
// The handler holds a reference to a native COM object; the caller
// must call Close on the handler when finished.
func (v *View) AddNavigationCompleted(handler *NavigationCompletedEventHandler) error {
	vtbl, err := v.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.AddNavigationCompleted,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(handler.impl.COMObject())),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2::AddNavigationCompleted failed: 0x%08x", hr)
	}
	return nil
}

// AddContainsFullScreenElementChanged registers a Go callback to be
// invoked when the page's fullscreen-element state changes (e.g.
// when a <video> element enters or exits fullscreen). Mirrors
// ICoreWebView2::AddContainsFullScreenElementChanged.
//
// The handler holds a reference to a native COM object; the caller
// must call Close on the handler when finished.
func (v *View) AddContainsFullScreenElementChanged(handler *ContainsFullScreenElementChangedEventHandler) error {
	vtbl, err := v.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.AddContainsFullScreenElementChanged,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(handler.impl.COMObject())),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2::AddContainsFullScreenElementChanged failed: 0x%08x", hr)
	}
	return nil
}
