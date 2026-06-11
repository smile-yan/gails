//go:build windows

// Port of upstream
// github.com/wailsapp/wails/webview2/pkg/combridge/vtables.go.
//
// Upstream divergence: the upstream file implements a generic
// GUID-tracked, parent-chained vtable registry (RegisterVTable /
// registerVTableInternal / vTableOf / guidOf / ifceImpl / ifceDef /
// typeInterfaceToString) that backs its higher-level New/Resolve
// machinery in bridge.go. The Gails port has already diverged from
// upstream by redefining IUnknown as a struct (see iunknown.go) with
// an explicit iunknownVtable struct that uses named uintptr fields for
// dispatch. This port therefore keeps RegisterVTable as the smallest
// useful primitive — a slot-order-preserving packer — that downstream
// code can use to build vtable pointers without dragging in the
// generic registry machinery. The GUID/parent-chain logic is
// intentionally not ported; it has no consumer in the Gails port.

package bridge

// VTable is an array of function pointers laid out in the order
// Microsoft IDL defines for a given COM interface. Slot order is a
// hard contract; swapping two slots is silent UB.
type VTable struct {
	Slots []uintptr
}

// RegisterVTable packs the supplied function pointers into a VTable in
// the order given. The caller is responsible for matching slot order
// to the IDL of the target interface.
func RegisterVTable(slots ...uintptr) *VTable {
	return &VTable{Slots: slots}
}
