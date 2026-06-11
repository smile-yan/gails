//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Settings is a Go wrapper over the COM ICoreWebViewSettings interface. It
// exposes configuration knobs for the WebView2 instance (dev tools, user
// agent, zoom controls, status bar, accelerator keys, swipe navigation,
// autofill, etc.).
type Settings struct {
	Raw  uintptr
	vtbl *iCoreWebViewSettingsVtable
}

// iCoreWebViewSettingsVtable is the COM ICoreWebViewSettings vtable.
// Slot order is invariant — it matches upstream
// pkg/edge/ICoreWebViewSettings.go. The base interface merges
// ICoreWebView2Settings{1..6}, giving 21 method slots (10 Get/Put pairs
// plus PutUserAgent and PutIsGeneralAutofillEnabled, which are not paired
// with a Get in the upstream).
type iCoreWebViewSettingsVtable struct {
	QueryInterface                       uintptr
	AddRef                               uintptr
	Release                              uintptr
	GetIsScriptEnabled                   uintptr
	PutIsScriptEnabled                   uintptr
	GetIsWebMessageEnabled               uintptr
	PutIsWebMessageEnabled               uintptr
	GetAreDefaultScriptDialogsEnabled    uintptr
	PutAreDefaultScriptDialogsEnabled    uintptr
	GetIsStatusBarEnabled                uintptr
	PutIsStatusBarEnabled                uintptr
	GetAreDevToolsEnabled                uintptr
	PutAreDevToolsEnabled                uintptr
	GetAreDefaultContextMenusEnabled     uintptr
	PutAreDefaultContextMenusEnabled     uintptr
	GetAreHostObjectsAllowed             uintptr
	PutAreHostObjectsAllowed             uintptr
	GetIsZoomControlEnabled              uintptr
	PutIsZoomControlEnabled              uintptr
	GetIsBuiltInErrorPageEnabled         uintptr
	PutIsBuiltInErrorPageEnabled         uintptr
	GetUserAgent                         uintptr
	PutUserAgent                         uintptr
	GetAreBrowserAcceleratorKeysEnabled  uintptr
	PutAreBrowserAcceleratorKeysEnabled  uintptr
	GetIsPasswordAutosaveEnabled         uintptr
	PutIsPasswordAutosaveEnabled         uintptr
	GetIsGeneralAutofillEnabled          uintptr
	PutIsGeneralAutofillEnabled          uintptr
	GetIsPinchZoomEnabled                uintptr
	PutIsPinchZoomEnabled                uintptr
	GetIsSwipeNavigationEnabled          uintptr
	PutIsSwipeNavigationEnabled          uintptr
}

// vtable resolves and caches the vtable pointer from Raw.
func (s *Settings) vtable() (*iCoreWebViewSettingsVtable, error) {
	if s.vtbl != nil {
		return s.vtbl, nil
	}
	if s.Raw == 0 {
		return nil, fmt.Errorf("ICoreWebViewSettings: nil COM pointer")
	}
	// Standard COM vtable-pointer dereference. The two uintptr conversions
	// silence govet's unsafe.Pointer check (the value cannot be a pointer
	// to a Go object — it is a foreign COM vtable).
	vtblPtr := *(*uintptr)(unsafe.Pointer(s.Raw))
	s.vtbl = (*iCoreWebViewSettingsVtable)(unsafe.Pointer(vtblPtr))
	return s.vtbl, nil
}

// PutIsScriptEnabled toggles whether JavaScript is enabled in the WebView.
func (s *Settings) PutIsScriptEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutIsScriptEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutIsScriptEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutIsWebMessageEnabled toggles whether the WebView can post messages to
// the host via window.chrome.webview.postMessage.
func (s *Settings) PutIsWebMessageEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutIsWebMessageEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutIsWebMessageEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutAreDefaultScriptDialogsEnabled toggles whether the WebView displays
// default dialogs (alert, confirm, prompt) raised by JavaScript.
func (s *Settings) PutAreDefaultScriptDialogsEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutAreDefaultScriptDialogsEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutAreDefaultScriptDialogsEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutIsStatusBarEnabled toggles whether the status bar is shown when a
// link is hovered.
func (s *Settings) PutIsStatusBarEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutIsStatusBarEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutIsStatusBarEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutAreDevToolsEnabled toggles whether the user can open the F12 dev
// tools. Gails flips this between devtools and production builds.
func (s *Settings) PutAreDevToolsEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutAreDevToolsEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutAreDevToolsEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutAreDefaultContextMenusEnabled toggles whether the WebView shows the
// default right-click context menu.
func (s *Settings) PutAreDefaultContextMenusEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutAreDefaultContextMenusEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutAreDefaultContextMenusEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutAreHostObjectsAllowed toggles whether the WebView is allowed to
// access host objects exposed to JavaScript.
func (s *Settings) PutAreHostObjectsAllowed(allowed bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutAreHostObjectsAllowed,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(allowed)),
	)
	if hr != 0 {
		return fmt.Errorf("PutAreHostObjectsAllowed failed: 0x%08x", hr)
	}
	return nil
}

// PutIsZoomControlEnabled toggles whether the user can change the zoom
// level via Ctrl+/- keyboard shortcuts.
func (s *Settings) PutIsZoomControlEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutIsZoomControlEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutIsZoomControlEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutIsBuiltInErrorPageEnabled toggles whether the WebView shows its
// built-in error page when navigation fails.
func (s *Settings) PutIsBuiltInErrorPageEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutIsBuiltInErrorPageEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutIsBuiltInErrorPageEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutUserAgent sets the User-Agent string sent with every request. Pass
// an empty string to restore the default.
func (s *Settings) PutUserAgent(ua string) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	p, err := windows.UTF16PtrFromString(ua)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutUserAgent,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(p)),
	)
	if hr != 0 {
		return fmt.Errorf("PutUserAgent failed: 0x%08x", hr)
	}
	return nil
}

// PutAreBrowserAcceleratorKeysEnabled toggles whether browser-level
// accelerator keys (Ctrl+P, Ctrl+F, etc.) are handled by the WebView.
func (s *Settings) PutAreBrowserAcceleratorKeysEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutAreBrowserAcceleratorKeysEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutAreBrowserAcceleratorKeysEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutIsPasswordAutosaveEnabled toggles whether the WebView prompts to
// save passwords.
func (s *Settings) PutIsPasswordAutosaveEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutIsPasswordAutosaveEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutIsPasswordAutosaveEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutIsGeneralAutofillEnabled toggles whether the WebView's general
// autofill (addresses, payment info) is enabled.
func (s *Settings) PutIsGeneralAutofillEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutIsGeneralAutofillEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutIsGeneralAutofillEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutIsPinchZoomEnabled toggles whether pinch-zoom is enabled on
// touch-capable devices.
func (s *Settings) PutIsPinchZoomEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutIsPinchZoomEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutIsPinchZoomEnabled failed: 0x%08x", hr)
	}
	return nil
}

// PutIsSwipeNavigationEnabled toggles whether the WebView supports
// swipe-back/swipe-forward navigation on touch devices.
func (s *Settings) PutIsSwipeNavigationEnabled(enabled bool) error {
	vtbl, err := s.vtable()
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		vtbl.PutIsSwipeNavigationEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutIsSwipeNavigationEnabled failed: 0x%08x", hr)
	}
	return nil
}

// toBool32 converts a Go bool into the BOOL (int32) representation that
// the WebView2 COM setters expect.
func toBool32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
