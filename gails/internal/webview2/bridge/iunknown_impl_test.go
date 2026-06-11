//go:build windows

package bridge

import (
	"reflect"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// TestIUnknownImpl_PublicSurface asserts the public method surface of
// IUnknownImpl matches the upstream file. This is a compile-only test
// for COM dispatch ports — running AddRef/Release/QueryInterface
// against a real vtable is covered by integration tests in later
// tasks (Plan Task 5: RegisterVTable, Task 6: New + Resolve).
func TestIUnknownImpl_PublicSurface(t *testing.T) {
	// IUnknownImpl must be a single uintptr-sized field (the vtable
	// pointer) so the layout is COM-compatible.
	var impl IUnknownImpl
	if got := unsafe.Sizeof(impl); got != unsafe.Sizeof(uintptr(0)) {
		t.Errorf("IUnknownImpl size = %d, want %d", got, unsafe.Sizeof(uintptr(0)))
	}

	// IUnknownImpl must expose the three IUnknown methods.
	typ := reflect.TypeOf(&impl)
	wantMethods := map[string]bool{
		"QueryInterface": false,
		"AddRef":         false,
		"Release":        false,
	}
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		if _, ok := wantMethods[name]; ok {
			wantMethods[name] = true
		}
	}
	for name, found := range wantMethods {
		if !found {
			t.Errorf("IUnknownImpl missing method %q", name)
		}
	}

	// QueryInterface must accept a *windows.GUID and return (*IUnknown, error).
	q, ok := typ.MethodByName("QueryInterface")
	if !ok {
		t.Fatal("QueryInterface method not found")
	}
	qt := q.Type
	if qt.NumIn() != 2 { // receiver + iid
		t.Errorf("QueryInterface has %d params (incl. receiver), want 2", qt.NumIn())
	}
	if qt.NumOut() != 2 { // *IUnknown + error
		t.Errorf("QueryInterface has %d return values, want 2", qt.NumOut())
	}
	if qt.In(1) != reflect.TypeOf((*windows.GUID)(nil)) {
		t.Errorf("QueryInterface iid arg is %v, want *windows.GUID", qt.In(1))
	}
	if qt.Out(0) != reflect.TypeOf((*IUnknown)(nil)) {
		t.Errorf("QueryInterface first return is %v, want *IUnknown", qt.Out(0))
	}
}
