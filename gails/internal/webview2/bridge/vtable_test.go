//go:build windows

package bridge

import "testing"

func TestRegisterVTableRaw_SlotOrderPreserved(t *testing.T) {
	// The whole point of RegisterVTableRaw is that slot N is the
	// function we passed as the Nth argument. We test with 4 slots.
	markers := []uintptr{0x1000, 0x2000, 0x3000, 0x4000}
	vt := RegisterVTableRaw(markers...)
	if len(vt.Slots) != 4 {
		t.Fatalf("slot count = %d, want 4", len(vt.Slots))
	}
	for i, want := range markers {
		if vt.Slots[i] != want {
			t.Errorf("slot %d: got 0x%x, want 0x%x", i, vt.Slots[i], want)
		}
	}
}
