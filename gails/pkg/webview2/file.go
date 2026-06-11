//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// File is a Go wrapper over the COM ICoreWebView2File interface. It is
// used by the WebView2 drag-and-drop API to expose a file the user dropped
// into the webview.
type File struct {
	Raw  uintptr
	vtbl *iCoreWebView2FileVtable
}

type iCoreWebView2FileVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetPath        uintptr
}

// Path returns the absolute path of the dropped file.
func (f *File) Path() (string, error) {
	if f.vtbl == nil {
		// Lazy vtable resolution: standard COM vtable-pointer dereference.
		vtblPtr := *(*uintptr)(unsafe.Pointer(f.Raw))
		f.vtbl = (*iCoreWebView2FileVtable)(unsafe.Pointer(vtblPtr))
	}
	var p *uint16
	hr, _, _ := syscall.SyscallN(
		f.vtbl.GetPath,
		uintptr(unsafe.Pointer(f)),
		uintptr(unsafe.Pointer(&p)),
	)
	if hr != 0 {
		return "", fmt.Errorf("ICoreWebView2File::GetPath failed: 0x%08x", hr)
	}
	return windows.UTF16PtrToString(p), nil
}
