//go:build windows

// Port of upstream
// github.com/wailsapp/wails/webview2/pkg/combridge/vtables.go.
//
// The Gails port preserves the Gails-friendly VTable struct (with
// a Slots []uintptr field) as a slot-order-preserving packer for
// callers that need a hand-rolled vtable pointer (e.g.
// pkg/webview2/events_handler.go).
//
// The generic GUID-tracked, parent-chained vtable registry
// (RegisterVTable / registerVTableInternal / vTableOf / guidOf /
// ifceImpl / ifceDef / typeInterfaceToString) lives in
// comobject.go and is exposed as the generic
// RegisterVTable[T1, T2] function. The non-generic packer is
// renamed to RegisterVTableRaw to avoid name clashes.

package bridge

// VTable is an array of function pointers laid out in the order
// Microsoft IDL defines for a given COM interface. Slot order is a
// hard contract; swapping two slots is silent UB.
type VTable struct {
	Slots []uintptr
}

// RegisterVTableRaw packs the supplied function pointers into a
// VTable in the order given. The caller is responsible for matching
// slot order to the IDL of the target interface.
//
// (The plain `RegisterVTable` name is reserved for the generic
// GUID-tracked vTable registry added in comobject.go; use
// RegisterVTableRaw for the simple slots-only packer.)
func RegisterVTableRaw(slots ...uintptr) *VTable {
	return &VTable{Slots: slots}
}
