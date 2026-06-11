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

// GetPath is a COM-style alias for Path, kept for application
// code that uses the upstream method name.
func (f *File) GetPath() (string, error) {
	return f.Path()
}

// Release decrements the COM refcount for this File. The
// application layer's drop handler calls this after consuming
// the file's path.
//
// TODO(port): the vtable slot for Release is at index [2] in
// iCoreWebView2FileVtable (modeled but not yet invoked). The
// stub returns nil so the call site compiles; real refcount
// management is a follow-up task.
func (f *File) Release() error {
	return nil
}
