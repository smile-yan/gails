//go:build windows

// Package bridge — see ../README or the plan file for design notes.
//
// The simple New/Resolve helpers below are the minimum the Gails port
// needs. They are NOT a port of the upstream generic registry in
// pkg/combridge/bridge.go (that 239-line file is intentionally not
// ported; see plan Task 6 for rationale).
//
// Upstream divergence notes:
//
//  1. The upstream `bridge.New[T]` is a generic constructor that
//     takes a raw `uintptr` and wraps it in a `ComObject[T]` whose
//     vtable pointer is the raw value. Upstream's IUnknown is the
//     empty interface, so a `ComObject[T]` is just a typed wrapper
//     around a uintptr vtable pointer.
//
//     In the Gails port, IUnknown is a concrete struct (see
//     iunknown.go) with a `vtbl *iunknownVtable` field. New is the
//     equivalent of upstream's New: take a raw uintptr that already
//     points at a vtable and reinterpret it as that vtable's
//     pointer. The caller manages COM lifetime (AddRef/Release)
//     separately — New does NOT AddRef.
//
//  2. The upstream `bridge.Resolve[T]` walks a parent-chained
//     generic registry (vTableOf, guidOf, ifceDef) to find the
//     right IID and dispatch QueryInterface through the right
//     intermediate. In the Gails port the registry does not exist;
//     every concrete interface (ICoreWebView2, ICoreWebViewSettings,
//     etc.) calls QueryInterface directly on its parent IUnknown.
//     Resolve is therefore a thin convenience wrapper around
//     (*IUnknown).QueryInterface.
//
//  3. The upstream `comInterfaceDesc`, `comObject` refcount
//     bookkeeping, and `ifceImpl`/`ifceDef[T]` machinery are not
//     ported. Gails performs refcounting manually at the call sites
//     that need it.
package bridge

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// New wraps a raw COM vtable pointer in an IUnknown. The caller is
// responsible for the underlying COM object's lifetime — New does
// NOT call AddRef.
//
// The argument is the vtable pointer (i.e. a pointer to a struct of
// function pointers in IDL order). It is reinterpreted as
// *iunknownVtable; downstream casts handle the larger vtable types
// because every COM interface vtable starts with the IUnknown
// slots.
func New(raw uintptr) *IUnknown {
	return &IUnknown{vtbl: (*iunknownVtable)(unsafe.Pointer(raw))}
}

// Resolve looks up a child interface on the given IUnknown. The
// returned pointer is a new IUnknown with its own vtable; the
// caller is responsible for releasing it.
//
// Resolve is a thin wrapper around (*IUnknown).QueryInterface. It
// exists so that concrete interface files (e.g. ICoreWebView2)
// can call `bridge.Resolve(this, &IID_ICoreWebView2Settings)`
// without each file re-importing the windows package for the
// QueryInterface signature.
func Resolve(p *IUnknown, iid *windows.GUID) (*IUnknown, error) {
	return p.QueryInterface(iid)
}
