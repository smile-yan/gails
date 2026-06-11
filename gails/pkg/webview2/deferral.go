//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Deferral is a Go wrapper over the COM ICoreWebView2Deferral interface.
// It is used by event handlers to extend the lifetime of an event past the
// handler returning.
type Deferral struct {
	Raw  uintptr
	vtbl *iCoreWebView2DeferralVtable
}

type iCoreWebView2DeferralVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Complete       uintptr
}

// Complete signals that the deferred work is done.
func (d *Deferral) Complete() error {
	if d.vtbl == nil {
		// Lazy vtable resolution: standard COM vtable-pointer dereference.
		vtblPtr := *(*uintptr)(unsafe.Pointer(d.Raw))
		d.vtbl = (*iCoreWebView2DeferralVtable)(unsafe.Pointer(vtblPtr))
	}
	hr, _, _ := syscall.SyscallN(
		d.vtbl.Complete,
		uintptr(unsafe.Pointer(d)),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2Deferral::Complete failed: 0x%08x", hr)
	}
	return nil
}

// Release decrements the COM refcount. Mirrors the IUnknown::Release
// slot inherited by ICoreWebView2Deferral.
func (d *Deferral) Release() error {
	if d.vtbl == nil {
		vtblPtr := *(*uintptr)(unsafe.Pointer(d.Raw))
		d.vtbl = (*iCoreWebView2DeferralVtable)(unsafe.Pointer(vtblPtr))
	}
	_, _, _ = syscall.SyscallN(
		d.vtbl.Release,
		uintptr(unsafe.Pointer(d)),
	)
	return nil
}
