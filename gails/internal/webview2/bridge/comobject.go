//go:build windows

// Port of upstream
// github.com/wailsapp/wails/webview2/pkg/combridge/bridge.go (and
// the C-side QueryInterface / AddRef / Release trampolines from
// iunknown.go).
//
// This file adds the GENERIC ComObject / New / Resolve /
// RegisterVTable machinery that pkg/w32 needs. It coexists with
// the simpler New(uintptr)/Resolve(*IUnknown, *windows.GUID) in
// bridge.go: the older helpers wrap a raw vtable pointer in an
// IUnknown; the new generics build real COM objects with refcount
// and GUID-based vtable dispatch.
//
// Upstream divergence: the Gails port defines IUnknown as a
// concrete struct (iunknown.go) wrapping a vtable pointer. The
// generic machinery here uses the empty interface `interface{}`
// (aliased as IUnknownInterface) as the constraint, matching
// upstream's `type IUnknown interface{}`. This is intentional: it
// keeps the generic constraints orthogonal to the vtable-wrapper
// IUnknown.

package bridge

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/windows"
)

const iUnknownGUID = "{00000000-0000-0000-C000-000000000046}"

// IUnknownInterface is the empty-interface constraint used by the
// generic RegisterVTable / New / Resolve machinery. It is a
// type alias for `interface{}` so it can stand in for upstream's
// `combridge.IUnknown` (which is also an empty interface) without
// conflicting with the Gails IUnknown struct (see iunknown.go).
type IUnknownInterface = interface{}

var (
	comIfcePointersL sync.RWMutex
	comIfcePointers  = map[uintptr]*comObject{} // Map from ComInterfacePointer to the Go comObject
)

// comObject is the internal refcounted, multi-interface wrapper.
// Each interface implementation registered via ifceDef gets its own
// native interface pointer in ifcesImpl; refCount tracks the
// overall lifetime; ifces is the IID -> index map used for
// QueryInterface dispatch.
type comObject struct {
	l sync.Mutex

	refCount  int32
	ifces     map[string]int     // Map of ComInterfaceGUID to Interface Slots
	ifcesImpl []comInterfaceDesc // Slots with InterfaceDescriptors
}

// comInterfaceDesc pairs a native COM interface pointer with the
// Go-side implementation that should be called when the interface
// is invoked.
type comInterfaceDesc struct {
	ref  uintptr // The native Com InterfacePointer
	impl any     // The golang target object
}

// ifceImpl is the contract a generic interface implementation
// must satisfy: it has to expose its Go object and its registered
// vTable so the comObject constructor can wire it up.
type ifceImpl interface {
	impl() any
	ifce() (*vTable, error)
}

// ifceDef is the generic wrapper around a Go object that the
// comObject constructor knows how to consume. The type parameter T
// is the Go-side interface that the user passes to New[T] or
// New2[T1, T2]; ifceDef looks up the vTable registered for T and
// remembers the impl.
type ifceDef[T any] struct {
	objImpl any
}

func (i ifceDef[T]) impl() any { return i.objImpl }

func (i ifceDef[T]) ifce() (*vTable, error) {
	vtable := vTableOf[T]()
	if vtable == nil {
		return nil, fmt.Errorf("unable to find vTable for %s", typeInterfaceToStringOnly[T]())
	}
	return vtable, nil
}

// Resolve returns the Go-side object stored under the given
// native COM interface pointer. Returns a zero value of T if the
// pointer is unknown.
func Resolve[T IUnknownInterface](ifceP uintptr) T {
	comIfcePointersL.RLock()
	comObj := comIfcePointers[ifceP]
	comIfcePointersL.RUnlock()

	var n T
	if comObj != nil {
		if t := comObj.resolve(ifceP); t != nil {
			n = t.(T)
		}
	}
	return n
}

// New returns a new ComObject that implements the single Go
// interface T. COM calls will be redirected to T's methods through
// the vTable registered for T.
func New[T IUnknownInterface](obj T) *ComObject[T] {
	cObj := new(ifceDef[T]{obj})
	return newComObject[T](cObj)
}

// New2 returns a new ComObject that implements two Go interfaces.
// Use this for objects that need multiple-inheritance COM
// interfaces that are not descendants of each other.
func New2[T, T2 IUnknownInterface](obj T, obj2 T2) *ComObject[T] {
	cObj := new(ifceDef[T]{obj}, ifceDef[T2]{obj2})
	return newComObject[T](cObj)
}

// new builds the comObject for one or more ifceImpls. It always
// prepends a default IUnknown ifce so that QueryInterface / AddRef
// / Release are always available; the user's impls are appended.
func new(impls ...ifceImpl) *comObject {
	impls = append([]ifceImpl{ifceDef[IUnknownInterface]{}}, impls...)

	cObj := &comObject{
		refCount:  1,
		ifces:     map[string]int{},
		ifcesImpl: make([]comInterfaceDesc, len(impls)),
	}

	for i, ifceDef := range impls {
		vtable, err := ifceDef.ifce()
		if err != nil {
			panic(err)
		}

		needsImplement := false
		for table := vtable; table != nil; table = table.Parent {
			guid := table.ComGUID
			if idx, found := cObj.ifces[guid]; found {
				// This Interface is already implemented
				if guid == iUnknownGUID {
					// IUnknown is a special interface and never has a user specific implementation
				} else if cObj.ifcesImpl[idx].impl != ifceDef.impl() {
					panic(fmt.Sprintf("Interface '%s' is already implemented by another object", table.Name))
				}
				break
			}

			needsImplement = true
			cObj.ifces[guid] = i
		}

		if !needsImplement {
			continue
		}

		ifceP, ifcePSlice := AllocUintptrObject(1)
		ifcePSlice[0] = vtable.ComVTable
		cObj.ifcesImpl[i] = comInterfaceDesc{ifceP, ifceDef.impl()}
	}

	comIfcePointersL.Lock()
	for _, ifceImpl := range cObj.ifcesImpl {
		comIfcePointers[ifceImpl.ref] = cObj
	}
	comIfcePointersL.Unlock()

	return cObj
}

func newComObject[T IUnknownInterface](comObj *comObject) *ComObject[T] {
	c := &ComObject[T]{obj: comObj}
	// Finalizer is async to avoid blocking the GC goroutine on locks.
	runtime.SetFinalizer(c, func(obj *ComObject[T]) { obj.close(true) })
	return c
}

// ComObject is the typed Go-side handle to a registered COM
// object. The type parameter T is the primary interface returned
// by Ref() — typically the "topmost" user interface implemented
// (e.g. iDropTarget).
type ComObject[T IUnknownInterface] struct {
	obj    *comObject
	closed int32
}

// Ref returns the native uintptr for the interface pointer of T.
// This is the value to pass to native COM APIs. Panics if the
// object has been closed.
func (o *ComObject[T]) Ref() uintptr {
	if atomic.LoadInt32(&o.closed) != 0 {
		panic("ComObject has been released")
	}
	return o.obj.queryInterface(guidOf[T](), false)
}

// Close releases the native COM object. The underlying object is
// only destroyed when the ref count hits zero. After Close, Ref
// will panic.
func (o *ComObject[T]) Close() error {
	o.close(false)
	return nil
}

func (o *ComObject[T]) close(asyncRelease bool) {
	if atomic.CompareAndSwapInt32(&o.closed, 0, 1) {
		runtime.SetFinalizer(o, nil)
		if asyncRelease {
			go o.obj.release()
		} else {
			o.obj.release()
		}
	}
}

// queryInterface looks up an interface by GUID. withAddRef is true
// when called from COM's QueryInterface (which takes a ref).
func (c *comObject) queryInterface(ifceGUID string, withAddRef bool) uintptr {
	c.l.Lock()
	defer c.l.Unlock()
	if c.refCount <= 0 {
		panic("call on released com object")
	}

	i, found := c.ifces[ifceGUID]
	if !found {
		return 0
	}

	if withAddRef {
		c.refCount++
	}
	return c.ifcesImpl[i].ref
}

// resolve finds the Go impl stored under a given native COM
// interface pointer.
func (c *comObject) resolve(ifceP uintptr) any {
	c.l.Lock()
	defer c.l.Unlock()
	if c.refCount <= 0 {
		panic("call on destroyed com object")
	}

	for _, ifce := range c.ifcesImpl {
		if ifce.ref != ifceP {
			continue
		}
		return ifce.impl
	}
	return nil
}

func (c *comObject) addRef() int32 {
	c.l.Lock()
	defer c.l.Unlock()
	if c.refCount <= 0 {
		panic("call on destroyed com object")
	}
	c.refCount++
	return c.refCount
}

func (c *comObject) release() int32 {
	c.l.Lock()
	defer c.l.Unlock()
	if c.refCount <= 0 {
		panic("call on destroyed com object")
	}

	if c.refCount--; c.refCount == 0 {
		comIfcePointersL.Lock()
		for _, ref := range c.ifcesImpl {
			delete(comIfcePointers, ref.ref)
		}
		comIfcePointersL.Unlock()

		for _, impl := range c.ifcesImpl {
			ref := impl.ref
			if ref == 0 {
				continue
			}
			GlobalFree(ref)
		}
	}

	return c.refCount
}

// vTable is the registered descriptor for a COM interface. Parent
// chains a vTable to its base interface (e.g. iDropTarget ->
// IUnknown). ComVTable is the native pointer that COM dispatches
// into; ComProcs is the underlying native uintptr slice that holds
// the function pointers in IDL order.
type vTable struct {
	Parent *vTable

	Name      string
	ComGUID   string
	ComVTable uintptr
	ComProcs  []uintptr
}

var (
	vTablesL sync.Mutex
	vTables  = make(map[string]*vTable)
)

// RegisterVTable registers the vTable trampoline methods for the
// COM interface T. TParent is the base interface; it must be
// IUnknownInterface (or another already-registered interface).
// The first parameter of each fn is the uintptr of the COM
// object; the Go object can be retrieved with Resolve(). After
// resolving, the call must be redirected to the Go object. The
// order of fns must match the IDL of the interface.
func RegisterVTable[TParent, T IUnknownInterface](guid string, fns ...interface{}) {
	registerVTableInternal[TParent, T](guid, false, fns...)
}

func registerVTableInternal[TParent, T IUnknownInterface](guid string, isInternal bool, fns ...interface{}) {
	vTablesL.Lock()
	defer vTablesL.Unlock()

	t, tName := typeInterfaceToString[T]()
	tParent, tParentName := typeInterfaceToString[TParent]()
	if !t.Implements(tParent) {
		panic(fmt.Errorf("RegisterVTable '%s': '%s' must implement '%s'", tName, tName, tParentName))
	}

	if !isInternal {
		if t == reflect.TypeOf((*IUnknownInterface)(nil)).Elem() {
			panic(fmt.Errorf("RegisterVTable '%s' IUnknown can't be registered", tName))
		}
		if t == tParent {
			panic(fmt.Errorf("RegisterVTable '%s': T and TParent can't be the same type", tName))
		}
	}

	var parent *vTable
	var parentProcs []uintptr
	var parentProcsCount int
	if t != tParent {
		parent = vTables[tParentName]
		if parent == nil {
			panic(fmt.Errorf("RegisterVTable '%s': Parent VTable '%s' not registered", tName, tParentName))
		}
		parentProcs = parent.ComProcs
		parentProcsCount = len(parentProcs)
	}

	comGuid, err := windows.GUIDFromString(guid)
	if err != nil {
		panic(fmt.Errorf("RegisterVTable '%s': invalid guid: %s", tName, err))
	}

	vt := &vTable{
		Parent:  parent,
		Name:    tName,
		ComGUID: comGuid.String(),
	}
	vt.ComVTable, vt.ComProcs = AllocUintptrObject(parentProcsCount + len(fns))

	for i, proc := range parentProcs {
		vt.ComProcs[i] = proc
	}
	for i, fn := range fns {
		vt.ComProcs[parentProcsCount+i] = windows.NewCallback(fn)
	}

	vTables[tName] = vt
}

func typeInterfaceToString[T any]() (reflect.Type, string) {
	t := reflect.TypeOf((*T)(nil))
	if t.Kind() != reflect.Pointer {
		panic("must be a (*yourInterfaceType)(nil)")
	}
	t = t.Elem()
	return t, t.PkgPath() + "/" + t.Name()
}

func typeInterfaceToStringOnly[T any]() string {
	_, name := typeInterfaceToString[T]()
	return name
}

func guidOf[T any]() string {
	vtable := vTableOf[T]()
	if vtable == nil {
		return ""
	}
	return vtable.ComGUID
}

func vTableOf[T any]() *vTable {
	name := typeInterfaceToStringOnly[T]()
	vTablesL.Lock()
	defer vTablesL.Unlock()
	return vTables[name]
}

// init wires up the IUnknown vTable — the root of every COM
// interface vTable. The functions live in iunknown_callbacks.go
// because they are the C-callable side of the COM contract and
// need to be grouped with the related trampolines.
func init() {
	// Build the IUnknown vTable by hand instead of via
	// registerVTableInternal, because the latter would reject
	// T == tParent (which it panics on for non-internal callers)
	// and we want to skip the "IUnknown can't be registered"
	// check.
	vTablesL.Lock()
	defer vTablesL.Unlock()

	key := typeInterfaceToStringOnly[IUnknownInterface]()
	if _, exists := vTables[key]; exists {
		return
	}

	comGuid, err := windows.GUIDFromString(iUnknownGUID)
	if err != nil {
		panic(err)
	}

	vt := &vTable{
		Name:    key,
		ComGUID: comGuid.String(),
	}
	vt.ComVTable, vt.ComProcs = AllocUintptrObject(3)
	vt.ComProcs[0] = windows.NewCallback(iUnknownQueryInterface)
	vt.ComProcs[1] = windows.NewCallback(iUnknownAddRef)
	vt.ComProcs[2] = windows.NewCallback(iUnknownRelease)
	vTables[key] = vt
}
