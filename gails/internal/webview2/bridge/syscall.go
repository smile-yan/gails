//go:build windows

// Port of upstream
// github.com/wailsapp/wails/webview2/pkg/combridge/syscall.go.
//
// Upstream divergence: the upstream file wraps kernel32's
// `GlobalAlloc` / `GlobalFree` and exposes `allocUintptrObject`,
// which the upstream `combridge.new` uses to allocate a native
// uintptr slot to hold a vtable pointer before handing it to COM.
// The Gails port never needs that path: `bridge.New` (see
// bridge.go) takes a vtable pointer that the caller has already
// placed in native memory, and IUnknownImpl (see iunknown_impl.go)
// is constructed in Go without a separate native allocation step.
// The wrappers are ported verbatim anyway, behind the same
// package-private names, so any future code that needs to allocate
// a single native uintptr slot can use them without re-deriving
// the kernel32 proc handles.

package bridge

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32     = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalAlloc = modkernel32.NewProc("GlobalAlloc")
	procGlobalFree  = modkernel32.NewProc("GlobalFree")

	uintptrSize = unsafe.Sizeof(uintptr(0))
)

// AllocUintptrObject allocates a native block of `size` uintptrs and
// returns the raw uintptr handle plus a Go slice aliasing the
// memory. The caller is responsible for releasing the block with
// GlobalFree once no more views into it remain.
func AllocUintptrObject(size int) (uintptr, []uintptr) {
	v := globalAlloc(uintptr(size) * uintptrSize)
	slice := unsafe.Slice((*uintptr)(unsafe.Pointer(v)), size)
	return v, slice
}

func globalAlloc(dwBytes uintptr) uintptr {
	ret, _, _ := procGlobalAlloc.Call(uintptr(0), dwBytes)
	if ret == 0 {
		panic("globalAlloc failed")
	}

	return ret
}

// GlobalFree releases a block of native memory previously returned
// by AllocUintptrObject (or by globalAlloc directly). The pointer
// must not be used after the call.
func GlobalFree(data uintptr) {
	ret, _, _ := procGlobalFree.Call(data)
	if ret != 0 {
		panic("globalFree failed")
	}
}
