//go:build windows

package bridge

import (
	"reflect"
	"testing"
	"unsafe"
)

func TestIUnknownVTableSlots(t *testing.T) {
	// Microsoft IDL defines IUnknown with exactly 3 slots:
	// QueryInterface, AddRef, Release. The vtable pointer must be
	// representable as a uintptr.
	var v iunknownVtable
	typ := reflect.TypeOf(&v).Elem()
	if typ.NumField() != 3 {
		t.Fatalf("iunknownVtable has %d fields, want 3", typ.NumField())
	}
	wantNames := []string{"QueryInterface", "AddRef", "Release"}
	for i, want := range wantNames {
		if typ.Field(i).Name != want {
			t.Errorf("slot %d: got %q, want %q", i, typ.Field(i).Name, want)
		}
	}
}

func TestIUnknownSize(t *testing.T) {
	// IUnknown is a single uintptr-sized field (the vtable pointer).
	var u IUnknown
	got := unsafe.Sizeof(u)
	if got != unsafe.Sizeof(uintptr(0)) {
		t.Errorf("IUnknown size = %d, want %d", got, unsafe.Sizeof(uintptr(0)))
	}
}
