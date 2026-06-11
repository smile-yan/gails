//go:build windows

package webview2

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/gailsapp/gails/internal/webview2/bridge"
)

// comHandlerImpl is the Go-side state backing a *EventHandler COM
// object. It is what the vtable trampolines dispatch into.
//
// Layout of the COM object we hand to WebView2 (allocated in native
// memory via bridge.allocUintptrObject):
//
//	[0]  uintptr  vtable pointer (points to comHandlerVtable layout)
//	[1]  uintptr  impl pointer   (points to a comHandlerImpl in Go heap)
//
// The vtable layout (4 slots, IDL order):
//
//	[0]  QueryInterface
//	[1]  AddRef
//	[2]  Release
//	[3]  Invoke
//
// QueryInterface/AddRef/Release are shared between all
// comHandlerImpl instances (they don't depend on per-instance
// state). Invoke is per-instance because it needs to dispatch to
// the Go callback stored in impl.
//
// The Go-side `release` counter on comHandlerImpl tracks the COM
// refcount; when it hits zero the native memory block backing the
// COM object is freed via bridge.globalFree and the vtable is
// freed likewise. The comHandlerImpl itself is released to the GC
// when the last *EventHandler reference goes away.
type comHandlerImpl struct {
	callback interface{} // func(view *View, args *MessageReceivedEventArgs) or similar
	release  int32       // refcount; Release() decrements, hits 0 → free
	comObj   uintptr     // native pointer to the [vtable, impl] block
	vtbl     uintptr     // native pointer to the vtable block
}

var (
	// comHandlerSharedVTable is a vtable whose first three slots
	// (QueryInterface/AddRef/Release) are shared between every
	// comHandlerImpl. The fourth slot is filled in per-instance
	// when the instance is constructed (Invoke depends on the Go
	// callback type).
	comHandlerSharedVTableOnce  sync.Once
	comHandlerSharedVTableSlots [3]uintptr
	comHandlerSharedVTable      uintptr
)

// comHandlerNewVTable allocates a fresh 4-slot vtable in native
// memory. The first three slots are copied from
// comHandlerSharedVTable; the fourth (Invoke) is provided by the
// caller.
func comHandlerNewVTable(invoke uintptr) uintptr {
	v, slice := bridge.AllocUintptrObject(4)
	// The shared slots are stable for the lifetime of the
	// process, so we can copy them by value at allocation time.
	src := comHandlerSharedSlots()
	slice[0] = src[0]
	slice[1] = src[1]
	slice[2] = src[2]
	slice[3] = invoke
	return v
}

func comHandlerSharedSlots() [3]uintptr {
	comHandlerSharedVTableOnce.Do(func() {
		v, slice := bridge.AllocUintptrObject(3)
		slice[0] = windows.NewCallback(comHandlerQueryInterfaceTrampoline)
		slice[1] = windows.NewCallback(comHandlerAddRefTrampoline)
		slice[2] = windows.NewCallback(comHandlerReleaseTrampoline)
		comHandlerSharedVTableSlots = [3]uintptr{slice[0], slice[1], slice[2]}
		comHandlerSharedVTable = v
	})
	return comHandlerSharedVTableSlots
}

// comHandlerNewObject allocates a 2-uintptr native block and
// populates [vtable, impl]. The vtable pointer is the per-instance
// vtable (with Invoke filled in). The impl pointer is the Go
// comHandlerImpl.
func comHandlerNewObject(vtbl, impl uintptr) uintptr {
	p, slice := bridge.AllocUintptrObject(2)
	slice[0] = vtbl
	slice[1] = impl
	return p
}

// comHandlerFromThis extracts the comHandlerImpl pointer from the
// second slot of a COM object block whose first slot is the vtable
// pointer. COM calls give us the COM object pointer (a pointer to
// the start of the block) as the first uintptr argument.
func comHandlerFromThis(this uintptr) *comHandlerImpl {
	if this == 0 {
		return nil
	}
	// The COM object layout is [vtable, impl]. impl is at offset
	// sizeof(uintptr).
	implPtr := *(*uintptr)(unsafe.Pointer(this + unsafe.Sizeof(uintptr(0))))
	if implPtr == 0 {
		return nil
	}
	return (*comHandlerImpl)(unsafe.Pointer(implPtr))
}

// --- vtable trampolines (Go funcs registered as C callbacks) ---

// comHandlerQueryInterfaceTrampoline is the shared QueryInterface
// slot for every comHandlerImpl. It supports only IUnknown (the
// base interface of every *EventHandler) and the IID of whichever
// *EventHandler the caller is registering (we accept any IID and
// return the same object — strict IID checking is out of scope for
// the Gails port).
func comHandlerQueryInterfaceTrampoline(this, refiid, ppvObject uintptr) uintptr {
	if ppvObject == 0 {
		// E_POINTER
		return 0x80004003
	}
	// Just return the same object. COM AddRef is the caller's job.
	*(*uintptr)(unsafe.Pointer(ppvObject)) = this
	impl := comHandlerFromThis(this)
	if impl != nil {
		atomic.AddInt32(&impl.release, 1)
	}
	return 0 // S_OK
}

// comHandlerAddRefTrampoline is the shared AddRef slot.
func comHandlerAddRefTrampoline(this uintptr) uintptr {
	impl := comHandlerFromThis(this)
	if impl == nil {
		return 0
	}
	return uintptr(atomic.AddInt32(&impl.release, 1))
}

// comHandlerReleaseTrampoline is the shared Release slot. When the
// refcount hits zero, it frees both the COM object block and the
// per-instance vtable.
func comHandlerReleaseTrampoline(this uintptr) uintptr {
	impl := comHandlerFromThis(this)
	if impl == nil {
		return 0
	}
	newCount := atomic.AddInt32(&impl.release, -1)
	if newCount != 0 {
		return uintptr(newCount)
	}
	// Free the per-instance vtable and the COM object block.
	if impl.vtbl != 0 {
		bridge.GlobalFree(impl.vtbl)
		impl.vtbl = 0
	}
	if impl.comObj != 0 {
		bridge.GlobalFree(impl.comObj)
		impl.comObj = 0
	}
	return 0
}

// NewComHandler constructs a comHandlerImpl that wraps a Go
// callback. The returned pointer is a freshly-allocated
// comHandlerImpl on the Go heap; the COM object block and vtable
// live in native memory owned by the impl.
//
// invokeTrampoline is the C-callable function pointer (typically
// from windows.NewCallback) for the Invoke slot. Different handler
// types pass different trampolines because each has a different
// Invoke signature.
//
// callback is the Go-side callback invoked by the trampoline.
func NewComHandler(invokeTrampoline uintptr, callback interface{}) *comHandlerImpl {
	impl := &comHandlerImpl{
		callback: callback,
		release:  1, // initial refcount; the COM object is born refcounted
	}
	impl.vtbl = comHandlerNewVTable(invokeTrampoline)
	impl.comObj = comHandlerNewObject(impl.vtbl, uintptr(unsafe.Pointer(impl)))
	return impl
}

// COMObject returns the native COM object pointer (suitable for
// passing to AddWebMessageReceived et al.).
func (h *comHandlerImpl) COMObject() uintptr {
	return h.comObj
}

// AddRef increments the refcount.
func (h *comHandlerImpl) AddRef() int32 {
	return atomic.AddInt32(&h.release, 1)
}

// Release decrements the refcount. Mirrors comHandlerReleaseTrampoline.
func (h *comHandlerImpl) Release() int32 {
	newCount := atomic.AddInt32(&h.release, -1)
	if newCount != 0 {
		return newCount
	}
	if h.vtbl != 0 {
		bridge.GlobalFree(h.vtbl)
		h.vtbl = 0
	}
	if h.comObj != 0 {
		bridge.GlobalFree(h.comObj)
		h.comObj = 0
	}
	return 0
}

// Callback returns the Go callback stored in the impl. The
// per-instance Invoke trampolines cast this to the right func type.
func (h *comHandlerImpl) Callback() interface{} {
	return h.callback
}
