//go:build windows

// Package bridge — see ../README or the plan file for design notes.
//
// Two layers of COM helpers coexist here:
//
//   - The generic layer in comobject.go (New[T], New2[T, T2],
//     Resolve[T], RegisterVTable[T1, T2], ComObject[T]) is a port of
//     upstream pkg/combridge/bridge.go and vtables.go. It backs the
//     refcounted, GUID-dispatched COM objects used by pkg/w32 (e.g.
//     IDropTarget). The package-private `IUnknownInterface` type
//     alias is the empty-interface constraint these generics are
//     parameterised on, mirroring upstream's `type IUnknown
//     interface{}`.
//
//   - The simple layer in this file (NewIUnknown, ResolveIID) wraps
//     a raw vtable pointer as an IUnknown and looks up a child
//     interface via QueryInterface. It is the minimum the WebView2
//     surface (pkg/webview2) needs for direct COM calls; the
//     `events_handler.go` file still uses this layer for its
//     hand-rolled event-handler COM objects.
//
// Upstream divergence notes:
//
//  1. The upstream IUnknown is an empty interface. The Gails port
//     keeps IUnknown as a concrete struct (see iunknown.go) because
//     Tasks 3-7 already rely on the vtable-pointer shape. The
//     generic layer therefore needs a separate name for the
//     empty-interface constraint; that is IUnknownInterface.
//
//  2. The upstream IUnknownImpl is the embedded field that user
//     structs (e.g. DropTarget) embed to inherit IUnknown.
//     IUnknownImpl is ported verbatim (see iunknown_impl.go) and
//     can be embedded in user structs as before.
package bridge

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// NewIUnknown wraps a raw COM vtable pointer in an IUnknown. The
// caller is responsible for the underlying COM object's lifetime —
// NewIUnknown does NOT call AddRef.
//
// The argument is the vtable pointer (i.e. a pointer to a struct of
// function pointers in IDL order). It is reinterpreted as
// *iunknownVtable; downstream casts handle the larger vtable types
// because every COM interface vtable starts with the IUnknown
// slots.
//
// (The plain `New` name is reserved for the generic ComObject
// constructor added in comobject.go; use NewIUnknown for the
// raw-pointer convenience wrapper.)
func NewIUnknown(raw uintptr) *IUnknown {
	return &IUnknown{vtbl: (*iunknownVtable)(unsafe.Pointer(raw))}
}

// ResolveIID looks up a child interface on the given IUnknown. The
// returned pointer is a new IUnknown with its own vtable; the
// caller is responsible for releasing it.
//
// ResolveIID is a thin wrapper around (*IUnknown).QueryInterface.
// It exists so that concrete interface files (e.g. ICoreWebView2)
// can call `bridge.ResolveIID(this, &IID_ICoreWebView2Settings)`
// without each file re-importing the windows package for the
// QueryInterface signature.
//
// (The plain `Resolve` name is reserved for the generic
// ifceP-uintptr lookup added in comobject.go; use ResolveIID for
// the *IUnknown + GUID convenience wrapper.)
func ResolveIID(p *IUnknown, iid *windows.GUID) (*IUnknown, error) {
	return p.QueryInterface(iid)
}
