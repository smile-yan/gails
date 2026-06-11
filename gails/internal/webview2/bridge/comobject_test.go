//go:build windows

// Tests for the generic ComObject / New / Resolve / RegisterVTable
// primitives added to unblock pkg/w32 (Plan Task 25).

package bridge

import "testing"

func TestComObject_RefAndClose(t *testing.T) {
	// ComObject holds a raw COM pointer and a Go-side T.
	// Constructing one and calling Ref() should return the raw pointer;
	// Close() should not panic. A zero-value ComObject is a "closed"
	// object — Ref() will panic, but Close() must be a no-op.
	type myImpl struct{ name string }
	var c ComObject[myImpl]

	// Close on a zero ComObject must not panic.
	if err := c.Close(); err != nil {
		t.Errorf("zero ComObject close error: %v", err)
	}
}

func TestResolve_ReturnsZeroOnUnknownPointer(t *testing.T) {
	// Resolve[T](uintptr) should return a zero-value T for an unknown pointer.
	type myImpl struct{ name string }
	got := Resolve[myImpl](0xDEADBEEF)
	if got.name != "" {
		t.Errorf("Resolve returned %+v, want zero", got)
	}
}

func TestIUnknownInterface_IsEmptyInterface(t *testing.T) {
	// Upstream's combridge.IUnknown is defined as `type IUnknown interface{}`.
	// The Gails port uses IUnknown as a struct (see iunknown.go), so the
	// generic machinery needs a separate name for the "any interface"
	// constraint. This test documents the constraint name.
	// (Compile-only — no runtime assertion.)
	var _ IUnknownInterface = (*struct{})(nil)
}

func TestNew_GenericCompiles(t *testing.T) {
	// Compile-only: New[T IUnknownInterface] must be callable.
	type fakeImpl struct{ x int }
	// We don't actually call it (that would allocate native memory);
	// the assignment proves the signature compiles.
	var _ func(fakeImpl) *ComObject[fakeImpl] = New[fakeImpl]
}

func TestNew2_GenericCompiles(t *testing.T) {
	// Compile-only: New2[T1, T2 IUnknownInterface] must be callable.
	type fakeImplA struct{ x int }
	type fakeImplB struct{ y int }
	var _ func(fakeImplA, fakeImplB) *ComObject[fakeImplA] = New2[fakeImplA, fakeImplB]
}

func TestRegisterVTable_GenericCompiles(t *testing.T) {
	// Compile-only: RegisterVTable[T1, T2 IUnknownInterface] must exist.
	// The actual call needs Windows NewCallback (only works on
	// GOOS=windows), but the type assertion proves the signature.
	var _ func(string, ...interface{}) = RegisterVTable[IUnknownInterface, IUnknownInterface]
}
