//go:build windows

// Package bridge is a port of upstream
// github.com/wailsapp/wails/webview2/pkg/combridge. It exposes the raw
// COM bridge primitives (IUnknown, vtable dispatch, syscall helpers) used
// by pkg/w32 and the higher-level webview2 package.
package bridge

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// IUnknown is the root COM interface. The vtbl field is the vtable pointer;
// method dispatch reads function pointers out of the vtable.
type IUnknown struct {
	vtbl *iunknownVtable
}

// iunknownVtable is the layout of the IUnknown vtable as defined by
// Microsoft COM IDL. The slot ORDER is invariant — it must not be changed.
type iunknownVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

// QueryInterface looks up a child interface by IID and returns it as
// a new IUnknown with its own vtable.
func (i *IUnknown) QueryInterface(iid *windows.GUID) (*IUnknown, error) {
	var ppObj uintptr
	hr, _, _ := syscall.SyscallN(
		i.vtbl.QueryInterface,
		uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&ppObj)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("IUnknown::QueryInterface failed: 0x%08x", hr)
	}
	return &IUnknown{vtbl: (*iunknownVtable)(unsafe.Pointer(ppObj))}, nil
}

// AddRef increments the reference count and returns the new value.
func (i *IUnknown) AddRef() int32 {
	ret, _, _ := syscall.SyscallN(
		i.vtbl.AddRef,
		uintptr(unsafe.Pointer(i)),
	)
	return int32(ret)
}

// Release decrements the reference count; the COM object is destroyed
// when the count hits zero.
func (i *IUnknown) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		i.vtbl.Release,
		uintptr(unsafe.Pointer(i)),
	)
	return int32(ret)
}
