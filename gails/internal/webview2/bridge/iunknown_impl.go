//go:build windows

// Port of upstream
// github.com/wailsapp/wails/webview2/pkg/combridge/iunknown_impl.go.
//
// The upstream file defines:
//   - IUnknownFromPointer / IUnknownFromUintptr cast helpers
//   - IUnknownVtbl: a vtable struct with lowercase (private) function
//     pointer fields and uppercase (public) syscall-dispatch methods
//     that take an explicit `this unsafe.Pointer`
//   - IUnknownImpl: the Go-side COM IUnknown object
//
// In this port:
//   - The vtable struct is named iunknownVtable and lives in iunknown.go
//     (Plan Task 3). Its fields are uppercase uintptrs; the syscall
//     dispatch is exposed via methods on *IUnknown.
//   - IUnknownImpl wraps the same vtable pointer. IUnknownImpl and
//     IUnknown are layout-compatible, so IUnknownImpl.QueryInterface
//     can reuse (*IUnknown).QueryInterface directly.

package bridge

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// IUnknownFromPointer casts a generic pointer into an IUnknownImpl pointer.
func IUnknownFromPointer(ref unsafe.Pointer) *IUnknownImpl {
	return (*IUnknownImpl)(ref)
}

// IUnknownFromUintptr casts a native pointer into an IUnknownImpl pointer.
func IUnknownFromUintptr(ref uintptr) *IUnknownImpl {
	return IUnknownFromPointer(unsafe.Pointer(ref))
}

// IUnknownImpl is the Go-side COM IUnknown object. The vtable pointer
// points to an iunknownVtable (see iunknown.go) populated by
// RegisterVTable in the IUnknown init() (Plan Task 5/6).
//
// IUnknownImpl and IUnknown are layout-compatible: both wrap a single
// vtable pointer. Callers that hold one can reinterpret the pointer
// as the other via IUnknownFromPointer / IUnknownFromUintptr.
type IUnknownImpl struct {
	vtbl *iunknownVtable
}

// QueryInterface looks up a child interface by IID and returns it as
// a new IUnknown with its own vtable. The returned IUnknown is a fresh
// refcounted view; callers must call Release on it.
func (i *IUnknownImpl) QueryInterface(iid *windows.GUID) (*IUnknown, error) {
	// IUnknown and IUnknownImpl are layout-compatible, so we can
	// reuse the IUnknown dispatch which lives in iunknown.go.
	return (*IUnknown)(unsafe.Pointer(i)).QueryInterface(iid)
}

// AddRef increments the reference count and returns the new value.
func (i *IUnknownImpl) AddRef() int32 {
	return (*IUnknown)(unsafe.Pointer(i)).AddRef()
}

// Release decrements the reference count; the COM object is destroyed
// when the count hits zero.
func (i *IUnknownImpl) Release() int32 {
	return (*IUnknown)(unsafe.Pointer(i)).Release()
}
