//go:build windows

package bridge

import (
	"reflect"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestNew_ReturnsIUnknownWithVTable(t *testing.T) {
	// New takes a raw COM pointer (vtable pointer) and wraps it in an
	// IUnknown. We use a synthetic pointer here; the test only asserts
	// the wrapper preserves it.
	raw := uintptr(0xDEAD)
	unk := New(raw)
	if unk == nil {
		t.Fatal("New returned nil")
	}
	// The vtable pointer field is `vtbl` (not `Raw`) — see iunknown.go
	// for the authoritative IUnknown shape.
	if uintptr(unsafe.Pointer(unk.vtbl)) != raw {
		t.Errorf("vtbl = 0x%x, want 0x%x", uintptr(unsafe.Pointer(unk.vtbl)), raw)
	}
}

func TestResolve_DelegatesToQueryInterface(t *testing.T) {
	// Resolve is a thin wrapper around (*IUnknown).QueryInterface. We
	// don't do a real roundtrip here (would need Windows COM runtime);
	// we assert the function exists and has the right signature.
	var _ func(*IUnknown, *windows.GUID) (*IUnknown, error) = Resolve
}

func TestNew_PreservesPointerExactly(t *testing.T) {
	// Additional safety: New must not allocate or transform the
	// pointer in any way. The same uintptr must come back out.
	raw := uintptr(0xCAFEBABE)
	unk := New(raw)
	if uintptr(unsafe.Pointer(unk.vtbl)) != raw {
		t.Errorf("New(0x%x) preserved as 0x%x", raw, uintptr(unsafe.Pointer(unk.vtbl)))
	}
}

func TestResolve_Signature(t *testing.T) {
	// Reflective check that Resolve's signature is the documented
	// one. This is a compile-pass safety net: if a future refactor
	// changes the signature, the assertion above stops compiling and
	// this test gives a clearer error.
	fn := reflect.ValueOf(Resolve)
	typ := fn.Type()
	if typ.NumIn() != 2 {
		t.Errorf("Resolve takes %d args, want 2", typ.NumIn())
	}
	if typ.NumOut() != 2 {
		t.Errorf("Resolve returns %d values, want 2", typ.NumOut())
	}
	if typ.In(0) != reflect.TypeOf((*IUnknown)(nil)) {
		t.Errorf("Resolve arg 0 = %v, want *IUnknown", typ.In(0))
	}
	if typ.In(1) != reflect.TypeOf((*windows.GUID)(nil)) {
		t.Errorf("Resolve arg 1 = %v, want *windows.GUID", typ.In(1))
	}
}
