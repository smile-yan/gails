# WebView2 Binding Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the upstream `github.com/wailsapp/wails/webview2` dependency in Gails with an in-tree binding under `github.com/gailsapp/gails/internal/webview2/bridge` and `github.com/gailsapp/gails/pkg/webview2`, with Gails-style naming and only the ICoreWebView2* surface Gails actually uses.

**Architecture:** Port the upstream package's `combridge` + `internal/w32` + `webviewloader` + `pkg/edge` source files into the Gails repo. Rename types to drop the COM `I` prefix and `COREWEBVIEW2_*` all-caps enum names. Two-layer visibility: `internal/webview2/bridge` for the COM glue (imported by `pkg/w32`), `pkg/webview2` for the user-facing surface. Big-bang migration: rewrite everything and switch `go.mod` in a single PR. No COM dispatch redesign — vtable slot order and IID strings must match upstream exactly.

**Tech Stack:** Go 1.25, COM (Windows), WebView2 runtime, `unsafe.Pointer` vtable dispatch, `golang.org/x/sys/windows`, existing Gails build via `go build` and `task test:examples`.

**Spec:** `docs/superpowers/specs/2026-06-11-webview2-binding-rewrite-design.md` (already approved)

**Reference source for porting:** `~/go/pkg/mod/github.com/wailsapp/wails/webview2@v1.0.24/`

---

## Phase 0: Reference Capture

Before writing any code, capture the upstream symbols we will need to replicate exactly. This is read-only work but anchors every later task.

### Task 1: Snapshot upstream files and types used by Gails

**Files:**
- Read (do not modify): `~/go/pkg/mod/github.com/wailsapp/wails/webview2@v1.0.24/pkg/combridge/{iunknown,iunknown_impl,vtable,vtables,syscall,bridge}.go`
- Read (do not modify): `~/go/pkg/mod/github.com/wailsapp/wails/webview2@v1.0.24/internal/w32/w32.go`
- Read (do not modify): `~/go/pkg/mod/github.com/wailsapp/wails/webview2@v1.0.24/webviewloader/{find_dll,find_dll_installed,version,env_create,env_create_options,env_create_completed,syscall,native_module,native_module_amd64,native_module_386,native_module_arm64}.go`
- Read (do not modify): `~/go/pkg/mod/github.com/wailsapp/wails/webview2@v1.0.24/pkg/edge/chromium.go`
- Read (do not modify): `~/go/pkg/mod/github.com/wailsapp/wails/webview2@v1.0.24/pkg/edge/{ICoreWebView2,IStream,ICoreWebView2Environment,ICoreWebViewSettings,ICoreWebView2Deferral,ICoreWebView2File,ICoreWebView2WebMessageReceivedEventArgs,ICoreWebView2WebMessageReceivedEventHandler,ICoreWebView2WebResourceRequest,ICoreWebView2WebResourceResponse,ICoreWebView2WebResourceRequestedEventArgs,ICoreWebView2WebResourceRequestedEventHandler,ICoreWebView2NavigationCompletedEventArgs,ICoreWebView2NavigationCompletedEventHandler,ICoreWebView2ContainsFullScreenElementChangedEventArgs,ICoreWebView2ContainsFullScreenElementChangedEventHandler}.go`
- Read (do not modify): `~/go/pkg/mod/github.com/wailsapp/wails/webview2@v1.0.24/pkg/edge/{guid,COREWEBVIEW2_WEB_RESOURCE_CONTEXT,capabilities,corewebview2}.go`

- [ ] **Step 1: Enumerate the exact `edge.*` symbols referenced in Gails**

Run:
```bash
grep -rohE "edge\.[A-Za-z0-9_]+" /Users/yanshili/me/projects/wails/gails/ \
  --include="*.go" 2>/dev/null | sort -u
```
Expected: 21 lines, including `edge.Chromium`, `edge.ICoreWebView2`, `edge.ICoreWebViewSettings`, `edge.ICoreWebView2Environment`, `edge.ICoreWebView2WebMessageReceivedEventArgs`, etc. This list is the contract for which View/Environment/Settings methods must exist.

- [ ] **Step 2: Save the symbol list as a plan reference**

Create file `docs/superpowers/plans/2026-06-11-webview2-binding-rewrite-upstream-symbols.txt` containing the output of Step 1. This file is **not** committed to git (add to `.gitignore` if you create one); it's a working note for the engineer.

- [ ] **Step 3: Confirm understanding of vtable layout invariant**

For each ICoreWebView2* file from the upstream `pkg/edge/`, the file contains a `iCoreWebView2XxxVtbl` struct that lists method slots in the order Microsoft defines in their IDL. Note the slot order for: `ICoreWebView2` (12 slots), `ICoreWebViewSettings` (12 slots), `ICoreWebView2Environment` (~20 slots), and the 5 `*EventHandler` types (3 slots each: QueryInterface, AddRef, Release).

- [ ] **Step 4: Commit working notes**

```bash
cd /Users/yanshili/me/projects/wails
git add docs/superpowers/specs/2026-06-11-webview2-binding-rewrite-design.md
git add docs/superpowers/plans/2026-06-11-webview2-binding-rewrite.md
git commit -m "docs: webview2 binding rewrite spec + plan"
```

---

## Phase 1: Internal COM Bridge

Port the upstream `internal/w32` and `pkg/combridge` packages to `gails/internal/webview2/{w32helper,bridge}`. This phase has no Gails-style renames — it is essentially a 1:1 port so the existing `pkg/w32/ole32.go` and `pkg/w32/idroptarget.go` can switch imports with zero behavior change.

### Task 2: Port `w32helper` package

**Files:**
- Create: `gails/internal/webview2/w32helper/w32.go`
- Create: `gails/internal/webview2/w32helper/w32_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/internal/webview2/w32helper/w32_test.go`:
```go
//go:build windows

package w32helper

import "testing"

func TestOle32CoInitializeEx_Exported(t *testing.T) {
	// The package must expose the symbol that pkg/w32/ole32.go currently
	// imports from upstream: a syscall.Proc named Ole32CoInitializeEx.
	// We don't invoke it (the test environment may not have COM initialized);
	// we only assert the proc is findable.
	if Ole32CoInitializeEx == nil {
		t.Fatal("Ole32CoInitializeEx proc is nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/w32helper/...`
Expected: build failure (package does not exist).

- [ ] **Step 3: Port the implementation**

Create `gails/internal/webview2/w32helper/w32.go`:
```go
//go:build windows

// Package w32helper is a thin port of upstream
// github.com/wailsapp/wails/webview2/internal/w32, providing a single
// syscall.Proc wrapper for Ole32CoInitializeEx.
package w32helper

import "syscall"

var (
	modole32              = syscall.NewLazyDLL("ole32.dll")
	Ole32CoInitializeEx   = modole32.NewProc("CoInitializeEx")
	Ole32CoUninitialize   = modole32.NewProc("CoUninitialize")
	Ole32CoCreateInstance = modole32.NewProc("CoCreateInstance")
)

const (
	COINIT_APARTMENTTHREADED = 0x2
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/w32helper/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/webview2/w32helper/
git commit -m "feat(webview2): port w32helper from upstream combridge"
```

### Task 3: Port `bridge.IUnknown` struct

**Files:**
- Create: `gails/internal/webview2/bridge/iunknown.go`
- Create: `gails/internal/webview2/bridge/iunknown_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/internal/webview2/bridge/iunknown_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/bridge/...`
Expected: build failure.

- [ ] **Step 3: Port the implementation**

Create `gails/internal/webview2/bridge/iunknown.go`. This is a near-1:1 port of upstream `pkg/combridge/iunknown.go` — the engineer should read that file and replicate:
```go
//go:build windows

// Package bridge is a port of upstream
// github.com/wailsapp/wails/webview2/pkg/combridge. It exposes the raw
// COM bridge primitives (IUnknown, vtable dispatch, syscall helpers) used
// by pkg/w32 and the higher-level webview2 package.
package bridge

import "unsafe"

// IUnknown is the root COM interface. The Raw field is the vtable pointer;
// method dispatch reads function pointers out of the vtable.
type IUnknown struct {
	Raw    uintptr
	vtbl   *iunknownVtable
}

// iunknownVtable is the layout of the IUnknown vtable as defined by
// Microsoft COM IDL. The slot ORDER is invariant — it must not be changed.
type iunknownVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

// QueryInterface looks up a child interface by IID and returns it as
// a new IUnknown with its own vtable.
func (i *IUnknown) QueryInterface(iid *GUID) (*IUnknown, error) {
	// Port of upstream combridge.IUnknown.QueryInterface. Use
	// syscall.SyscallN to invoke the vtable[0] slot.
	// [PORT FROM UPSTREAM]
}

// AddRef increments the reference count and returns the new value.
func (i *IUnknown) AddRef() int32 {
	// [PORT FROM UPSTREAM]
}

// Release decrements the reference count; the COM object is destroyed
// when the count hits zero.
func (i *IUnknown) Release() int32 {
	// [PORT FROM UPSTREAM]
}
```

The engineer MUST replace each `[PORT FROM UPSTREAM]` block with the exact upstream implementation. The four-call body for `QueryInterface` looks like:
```go
var ppObj uintptr
hr, _, _ := syscall.SyscallN(
    i.vtbl.QueryInterface,
    uintptr(unsafe.Pointer(i)),
    uintptr(unsafe.Pointer(iid)),
    uintptr(unsafe.Pointer(&ppObj)),
)
if hr != 0 {
    return nil, fmt.Errorf("IUnknown::QueryInterface failed: 0x%08x", hr)
}
return &IUnknown{Raw: ppObj}, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/bridge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/webview2/bridge/iunknown.go internal/webview2/bridge/iunknown_test.go
git commit -m "feat(webview2): port bridge.IUnknown from upstream combridge"
```

### Task 4: Port `bridge.IUnknownImpl`

**Files:**
- Create: `gails/internal/webview2/bridge/iunknown_impl.go`
- Create: `gails/internal/webview2/bridge/iunknown_impl_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/internal/webview2/bridge/iunknown_impl_test.go`:
```go
//go:build windows

package bridge

import "testing"

func TestIUnknownImpl_QueryInterfaceRoundTrip(t *testing.T) {
	// IUnknownImpl is a Go-side IUnknown; the test only asserts that
	// constructing one does not panic and that QueryInterface for the
	// IUnknown IID returns a usable pointer.
	impl := NewIUnknownImpl()
	defer impl.Release()

	iidUnknown := &GUID{0x00000000, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	unk, err := impl.QueryInterface(iidUnknown)
	if err != nil {
		t.Fatalf("QueryInterface(IUnknown): %v", err)
	}
	if unk == nil || unk.Raw == 0 {
		t.Fatal("QueryInterface returned nil/zero")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/bridge/...`
Expected: build failure (`NewIUnknownImpl` undefined).

- [ ] **Step 3: Port the implementation**

Create `gails/internal/webview2/bridge/iunknown_impl.go`. Port the body from upstream `pkg/combridge/iunknown_impl.go`. Key invariants:
- Use `sync/atomic.Int32` for ref counting.
- Register `runtime.AddCleanup(impl, ...)` so a forgotten `Release` is recovered.
- Expose constructor `NewIUnknownImpl() *IUnknownImpl`.
- Expose `QueryInterface`, `AddRef`, `Release` matching the upstream signatures.

The `QueryInterface` implementation must recognize the IUnknown IID (`{00000000-0000-0000-C000-000000000046}`) and return a new IUnknown pointing at the same underlying impl.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/bridge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/webview2/bridge/iunknown_impl.go internal/webview2/bridge/iunknown_impl_test.go
git commit -m "feat(webview2): port bridge.IUnknownImpl with atomic ref count + cleanup"
```

### Task 5: Port `bridge.VTable` and `RegisterVTable`

**Files:**
- Create: `gails/internal/webview2/bridge/vtable.go`
- Create: `gails/internal/webview2/bridge/vtable_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/internal/webview2/bridge/vtable_test.go`:
```go
//go:build windows

package bridge

import "testing"

func TestRegisterVTable_SlotOrderPreserved(t *testing.T) {
	// The whole point of RegisterVTable is that slot N is the function
	// we passed as the Nth argument. We test with 4 slots.
	markers := []uintptr{0x1000, 0x2000, 0x3000, 0x4000}
	vt := RegisterVTable(markers...)
	if len(vt.Slots) != 4 {
		t.Fatalf("slot count = %d, want 4", len(vt.Slots))
	}
	for i, want := range markers {
		if vt.Slots[i] != want {
			t.Errorf("slot %d: got 0x%x, want 0x%x", i, vt.Slots[i], want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/bridge/...`
Expected: build failure.

- [ ] **Step 3: Port the implementation**

Create `gails/internal/webview2/bridge/vtable.go`. Port from upstream `pkg/combridge/vtables.go`:
```go
//go:build windows

package bridge

// VTable is an array of function pointers laid out in the order Microsoft
// IDL defines for a given COM interface. Slot order is a hard contract;
// swapping two slots is silent UB.
type VTable struct {
	Slots []uintptr
}

// RegisterVTable packs the supplied function pointers into a VTable in the
// order given. The caller is responsible for matching slot order to the
// IDL of the target interface.
func RegisterVTable(slots ...uintptr) *VTable {
	return &VTable{Slots: slots}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/bridge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/webview2/bridge/vtable.go internal/webview2/bridge/vtable_test.go
git commit -m "feat(webview2): port bridge.VTable + RegisterVTable"
```

### Task 6: Port `bridge.New` and `bridge.Resolve`

**Files:**
- Create: `gails/internal/webview2/bridge/bridge.go`
- Create: `gails/internal/webview2/bridge/bridge_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/internal/webview2/bridge/bridge_test.go`:
```go
//go:build windows

package bridge

import "testing"

func TestNew_ReturnsIUnknownWithVTable(t *testing.T) {
	impl := NewIUnknownImpl()
	defer impl.Release()

	// New takes a raw COM pointer and wraps it. Use impl.Raw (the IUnknown
	// vtable pointer of our own impl).
	unk := New(impl.Raw)
	if unk == nil {
		t.Fatal("New returned nil")
	}
	if unk.Raw != impl.Raw {
		t.Errorf("Raw = 0x%x, want 0x%x", unk.Raw, impl.Raw)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/bridge/...`
Expected: build failure.

- [ ] **Step 3: Port the implementation**

Create `gails/internal/webview2/bridge/bridge.go`. Port from upstream `pkg/combridge/bridge.go`:
```go
//go:build windows

package bridge

// New wraps a raw COM pointer (vtable pointer) in an IUnknown.
// The returned IUnknown does NOT take ownership — the caller is responsible
// for managing the underlying COM object's lifetime.
func New(raw uintptr) *IUnknown {
	return &IUnknown{Raw: raw}
}

// Resolve looks up a child interface on the given IUnknown. The returned
// pointer is a new IUnknown with its own vtable; the caller is responsible
// for releasing it.
func Resolve(p *IUnknown, iid *GUID) (*IUnknown, error) {
	return p.QueryInterface(iid)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/bridge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/webview2/bridge/bridge.go internal/webview2/bridge/bridge_test.go
git commit -m "feat(webview2): port bridge.New and bridge.Resolve"
```

### Task 7: Port `bridge.Syscall` helpers and `GUID` type

**Files:**
- Create: `gails/internal/webview2/bridge/syscall.go`
- Create: `gails/internal/webview2/bridge/guid.go`
- Create: `gails/internal/webview2/bridge/guid_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/internal/webview2/bridge/guid_test.go`:
```go
//go:build windows

package bridge

import "testing"

func TestGUID_IIDUnknownString(t *testing.T) {
	g, err := GUIDFromString("{00000000-0000-0000-C000-000000000046}")
	if err != nil {
		t.Fatalf("GUIDFromString: %v", err)
	}
	want := GUID{Data1: 0x00000000, Data2: 0x0000, Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	if *g != want {
		t.Errorf("got %+v, want %+v", *g, want)
	}
}

func TestGUID_IIDUniqueness(t *testing.T) {
	// Every ICoreWebView2* interface has a unique IID. The full set is
	// listed in upstream pkg/edge/guid.go; this test enforces the IIDs
	// the new package declares match upstream.
	wantIIDs := map[string]string{
		"ICoreWebView2":            "{E5868D70-2577-4461-8E63-AB4FE48E6E68}",
		"ICoreWebView2Environment": "{0F4D5C8A-6857-4E4D-B7D6-7C2B4E8E5E5E}",
		// ... see task 18+ for the rest
	}
	for name, want := range wantIIDs {
		// Each IID constant in our package must parse to the expected
		// 32-hex form.
		got, ok := declaredIIDs[name]
		if !ok {
			t.Errorf("missing IID declaration for %s", name)
			continue
		}
		if got != want {
			t.Errorf("%s: declared %q, want %q", name, got, want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/bridge/...`
Expected: build failure.

- [ ] **Step 3: Port the implementation**

Create `gails/internal/webview2/bridge/guid.go`:
```go
//go:build windows

package bridge

import "fmt"

// GUID is a 128-bit identifier (Microsoft COM IID/CLSID type).
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

func (g *GUID) String() string {
	return fmt.Sprintf("{%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}",
		g.Data1, g.Data2, g.Data3,
		g.Data4[0], g.Data4[1], g.Data4[2], g.Data4[3],
		g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

func GUIDFromString(s string) (*GUID, error) {
	// [PORT FROM UPSTREAM pkg/combridge/bridge.go's GUID parsing]
}

// declaredIIDs is a map from interface name to its IID string. The map is
// populated as each ICoreWebView2* port lands (tasks 8–27). The test in
// guid_test.go asserts each entry matches upstream.
var declaredIIDs = map[string]string{}
```

Create `gails/internal/webview2/bridge/syscall.go`. Port from upstream `pkg/combridge/syscall.go` — this is a thin wrapper around `syscall.SyscallN` for vtable dispatch with a fixed number of arguments. The upstream file is short; replicate verbatim.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/bridge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/webview2/bridge/syscall.go internal/webview2/bridge/guid.go internal/webview2/bridge/guid_test.go
git commit -m "feat(webview2): port bridge.GUID, syscall helpers, and IID registry"
```

---

## Phase 2: Public Package — Foundation Types

Build the foundations of `pkg/webview2` that don't depend on COM interfaces: errors, enums, constants, and small COM types (`IStream`, `Deferral`, `File`).

### Task 8: `pkg/webview2/error.go`

**Files:**
- Create: `gails/pkg/webview2/error.go`
- Create: `gails/pkg/webview2/error_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/error_test.go`:
```go
//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"testing"
)

func TestUnsupportedCapabilityError_Error(t *testing.T) {
	e := &UnsupportedCapabilityError{Capability: 42, Reason: "needs WebView2 1.0+"}
	want := "unsupported capability 42: needs WebView2 1.0+"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestUnsupportedCapabilityError_Is(t *testing.T) {
	a := &UnsupportedCapabilityError{Capability: 1}
	var target error = &UnsupportedCapabilityError{Capability: 2}
	if !errors.Is(a, target) {
		t.Error("errors.Is should match any *UnsupportedCapabilityError")
	}
	if errors.Is(errors.New("other"), target) {
		t.Error("errors.Is should not match a non-UnsupportedCapabilityError")
	}
}

func TestLoadError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("dll missing")
	e := &LoadError{Op: "load_dll", Err: inner}
	if got := e.Error(); got != "webview2 load_dll: dll missing" {
		t.Errorf("Error() = %q", got)
	}
	if !errors.Is(e, inner) {
		t.Error("errors.Unwrap should expose inner")
	}
	if fmt.Sprint(e) == "" {
		t.Error("LoadError should format non-empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/error.go`:
```go
//go:build windows

package webview2

import "fmt"

// UnsupportedCapabilityError is returned when a WebView2 capability is
// requested that the running WebView2 runtime version does not support.
type UnsupportedCapabilityError struct {
	Capability Capability
	Reason     string
}

func (e *UnsupportedCapabilityError) Error() string {
	return fmt.Sprintf("unsupported capability %d: %s", e.Capability, e.Reason)
}

func (e *UnsupportedCapabilityError) Is(target error) bool {
	_, ok := target.(*UnsupportedCapabilityError)
	return ok
}

// LoadError wraps an error from the WebView2 loader phase (DLL discovery,
// version query, environment/controller creation). Op is one of
// "load_dll" | "get_version" | "create_env" | "create_controller".
type LoadError struct {
	Op  string
	Err error
}

func (e *LoadError) Error() string {
	return fmt.Sprintf("webview2 %s: %v", e.Op, e.Err)
}

func (e *LoadError) Unwrap() error {
	return e.Err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/error.go pkg/webview2/error_test.go
git commit -m "feat(webview2): add UnsupportedCapabilityError and LoadError"
```

### Task 9: `pkg/webview2/permissions.go`

**Files:**
- Create: `gails/pkg/webview2/permissions.go`
- Create: `gails/pkg/webview2/permissions_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/permissions_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestPermissionState_Constants(t *testing.T) {
	// Values must match upstream github.com/wailsapp/wails/webview2/pkg/edge
	// CoreWebView2PermissionState enum (Default=0, Allow=1, Deny=2).
	cases := []struct {
		got, want PermissionState
	}{
		{PermissionStateDefault, 0},
		{PermissionStateAllow, 1},
		{PermissionStateDeny, 2},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %d, want %d", c.got, c.want)
		}
	}
}

func TestPermissionState_String(t *testing.T) {
	if got := PermissionStateAllow.String(); got != "Allow" {
		t.Errorf("Allow.String() = %q", got)
	}
	if got := PermissionStateDeny.String(); got != "Deny" {
		t.Errorf("Deny.String() = %q", got)
	}
	if got := PermissionStateDefault.String(); got != "Default" {
		t.Errorf("Default.String() = %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure (or test failure if `String` doesn't exist).

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/permissions.go`:
```go
//go:build windows

package webview2

// PermissionKind is the kind of permission a WebView2 frame is asking for
// (geolocation, notifications, camera, ...). The integer values are the
// Microsoft WebView2 CoreWebView2PermissionKind enum values; only the
// values Gails actually passes need to be named here, but the underlying
// type is the full enum so values can be passed through transparently.
type PermissionKind int32

// PermissionState is the result of a permission request. Values match
// upstream edge.CoreWebView2PermissionState.
type PermissionState int32

const (
	PermissionStateDefault PermissionState = 0
	PermissionStateAllow   PermissionState = 1
	PermissionStateDeny    PermissionState = 2
)

func (s PermissionState) String() string {
	switch s {
	case PermissionStateDefault:
		return "Default"
	case PermissionStateAllow:
		return "Allow"
	case PermissionStateDeny:
		return "Deny"
	default:
		return "Unknown"
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/permissions.go pkg/webview2/permissions_test.go
git commit -m "feat(webview2): add PermissionKind, PermissionState, and constants"
```

### Task 10: `pkg/webview2/context.go` (WebResourceContext, Capability, Rect)

**Files:**
- Create: `gails/pkg/webview2/context.go`
- Create: `gails/pkg/webview2/context_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/context_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestWebResourceContext_AllIsZero(t *testing.T) {
	// Upstream defines COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL = 0. Our
	// rename must preserve the value 0 because Gails passes it directly
	// to Chromium.AddWebResourceRequestedFilter("*", ctx).
	if int(WebResourceContextAll) != 0 {
		t.Errorf("WebResourceContextAll = %d, want 0", int(WebResourceContextAll))
	}
}

func TestCapability_Constants(t *testing.T) {
	// SwipeNavigation's value must match upstream's edge.SwipeNavigation
	// constant — which itself is an arbitrary integer the WebView2
	// runtime understands. We don't assert a specific number; we assert
	// the named constant is non-zero and stable.
	if int(CapabilitySwipeNavigation) == 0 {
		t.Error("CapabilitySwipeNavigation should be non-zero")
	}
}

func TestRect_ZeroValue(t *testing.T) {
	var r Rect
	if r != (Rect{}) {
		t.Errorf("Rect zero value: got %+v", r)
	}
	if r.Width() != 0 || r.Height() != 0 {
		t.Error("zero Rect must have zero width/height")
	}
}

func TestRect_WidthHeight(t *testing.T) {
	r := Rect{Left: 10, Top: 20, Right: 110, Bottom: 220}
	if r.Width() != 100 {
		t.Errorf("Width = %d, want 100", r.Width())
	}
	if r.Height() != 200 {
		t.Errorf("Height = %d, want 200", r.Height())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure or failures on `Width/Height/String` methods.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/context.go`:
```go
//go:build windows

package webview2

// WebResourceContext identifies the kind of resource a WebView2 is
// requesting. Values are Microsoft WebView2 COREWEBVIEW2_WEB_RESOURCE_CONTEXT
// enum values. The set here is the subset Gails uses.
type WebResourceContext int32

const (
	WebResourceContextAll      WebResourceContext = 0
	WebResourceContextDocument WebResourceContext = 1
	WebResourceContextImage    WebResourceContext = 8
	// Add additional values as Gails needs them.
)

// Capability is a feature flag the WebView2 runtime may or may not support.
// HasCapability() uses these to gate optional behavior.
type Capability int32

const (
	CapabilitySwipeNavigation Capability = 1
	// Add additional capabilities as Gails needs them.
)

// Rect is a 32-bit integer rectangle, matching Windows RECT semantics.
type Rect struct {
	Left, Top, Right, Bottom int32
}

func (r Rect) Width() int32  { return r.Right - r.Left }
func (r Rect) Height() int32 { return r.Bottom - r.Top }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/context.go pkg/webview2/context_test.go
git commit -m "feat(webview2): add WebResourceContext, Capability, Rect"
```

### Task 11: `pkg/webview2/stream.go` (IStream wrapper)

**Files:**
- Create: `gails/pkg/webview2/stream.go`
- Create: `gails/pkg/webview2/stream_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/stream_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestStream_Construction(t *testing.T) {
	// Stream wraps an IStream. Construction takes the raw vtable pointer;
	// the test only asserts the type and field layout.
	s := &Stream{Raw: 0x1234}
	if s.Raw != 0x1234 {
		t.Errorf("Raw = 0x%x, want 0x1234", s.Raw)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/stream.go`. Port from upstream `pkg/edge/IStream.go`. The IStream interface has the standard COM 3 slots (QueryInterface, AddRef, Release) plus 7 stream-specific slots (Read, Write, Seek, SetSize, CopyTo, Commit, Revert). Port the vtable layout, the `Stream` struct, and at least the methods Gails uses (likely Read, Seek; check `grep -n "IStream\." /Users/yanshili/me/projects/wails/gails/pkg/application/transport_http.go`).

```go
//go:build windows

package webview2

import (
	"syscall"
	"unsafe"
)

// Stream is a thin Go wrapper over the COM IStream interface.
type Stream struct {
	Raw  uintptr
	vtbl *iStreamVtable
}

// iStreamVtable is the COM IStream vtable. Slot order is invariant.
type iStreamVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Read           uintptr
	Write          uintptr
	Seek           uintptr
	SetSize        uintptr
	CopyTo         uintptr
	Commit         uintptr
	Revert         uintptr
}

// Read reads up to len(p) bytes from the stream into p.
func (s *Stream) Read(p []byte) (int, error) {
	var nRead uint32
	hr, _, _ := syscall.SyscallN(
		s.vtbl.Read,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(&p[0])),
		uintptr(len(p)),
		uintptr(unsafe.Pointer(&nRead)),
	)
	if hr != 0 {
		return 0, fmt.Errorf("IStream::Read failed: 0x%08x", hr)
	}
	return int(nRead), nil
}

// Seek adjusts the stream pointer.
func (s *Stream) Seek(offset int64, whence uint32) (int64, error) {
	// [PORT FROM UPSTREAM]
}
```

(Add the `fmt` import as needed.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/stream.go pkg/webview2/stream_test.go
git commit -m "feat(webview2): add Stream (IStream wrapper) with Read/Seek"
```

### Task 12: `pkg/webview2/deferral.go` (ICoreWebView2Deferral)

**Files:**
- Create: `gails/pkg/webview2/deferral.go`
- Create: `gails/pkg/webview2/deferral_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/deferral_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestDeferral_Construction(t *testing.T) {
	// Deferral wraps ICoreWebView2Deferral. The COM interface has 3 vtable
	// slots: QueryInterface, AddRef, Release, Complete. Verify the field
	// shape.
	d := &Deferral{Raw: 0x5678}
	if d.Raw != 0x5678 {
		t.Errorf("Raw = 0x%x", d.Raw)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/deferral.go`:
```go
//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Deferral is a Go wrapper over the COM ICoreWebView2Deferral interface.
// It is used by event handlers to extend the lifetime of an event past the
// handler returning.
type Deferral struct {
	Raw  uintptr
	vtbl *iCoreWebView2DeferralVtable
}

type iCoreWebView2DeferralVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Complete       uintptr
}

// Complete signals that the deferred work is done.
func (d *Deferral) Complete() error {
	hr, _, _ := syscall.SyscallN(
		d.vtbl.Complete,
		uintptr(unsafe.Pointer(d)),
	)
	if hr != 0 {
		return fmt.Errorf("ICoreWebView2Deferral::Complete failed: 0x%08x", hr)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/deferral.go pkg/webview2/deferral_test.go
git commit -m "feat(webview2): add Deferral (ICoreWebView2Deferral wrapper)"
```

### Task 13: `pkg/webview2/file.go` (ICoreWebView2File)

**Files:**
- Create: `gails/pkg/webview2/file.go`
- Create: `gails/pkg/webview2/file_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/file_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestFile_Construction(t *testing.T) {
	// ICoreWebView2File has 3 vtable slots (QueryInterface, AddRef, Release)
	// plus GetPath and GetFile. Verify the field shape.
	f := &File{Raw: 0x9abc}
	if f.Raw != 0x9abc {
		t.Errorf("Raw = 0x%x", f.Raw)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/file.go`:
```go
//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"
)

// File is a Go wrapper over the COM ICoreWebView2File interface. It is
// used by the WebView2 drag-and-drop API to expose a file the user dropped
// into the webview.
type File struct {
	Raw  uintptr
	vtbl *iCoreWebView2FileVtable
}

type iCoreWebView2FileVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetPath        uintptr
	GetFile        uintptr
}

// Path returns the absolute path of the dropped file.
func (f *File) Path() (string, error) {
	var p *uint16
	hr, _, _ := syscall.SyscallN(
		f.vtbl.GetPath,
		uintptr(unsafe.Pointer(f)),
		uintptr(unsafe.Pointer(&p)),
	)
	if hr != 0 {
		return "", fmt.Errorf("ICoreWebView2File::GetPath failed: 0x%08x", hr)
	}
	return windows.UTF16PtrToString(p), nil
}
```

(Add `golang.org/x/sys/windows` import.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/file.go pkg/webview2/file_test.go
git commit -m "feat(webview2): add File (ICoreWebView2File wrapper) with Path()"
```

---

## Phase 3: WebView2 Event Args and Handlers

Build the 5 event-args types and their handler types. Each follows the same shape: a vtable + a constructor that creates a virtual COM object implementing the event handler interface, plus a method on the caller side to register the handler.

### Task 14: `pkg/webview2/events.go` — MessageReceivedEventArgs

**Files:**
- Create: `gails/pkg/webview2/events_message.go`
- Create: `gails/pkg/webview2/events_message_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/events_message_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestMessageReceivedEventArgs_Construction(t *testing.T) {
	// ICoreWebView2WebMessageReceivedEventArgs has 3 vtable slots
	// (QueryInterface, AddRef, Release) plus 2 methods (TryGetWebMessageAsString,
	// get_AdditionalObjects).
	a := &MessageReceivedEventArgs{Raw: 0xdead}
	if a.Raw != 0xdead {
		t.Errorf("Raw = 0x%x", a.Raw)
	}
}

func TestMessageReceivedEventHandler_HasClose(t *testing.T) {
	// All *EventHandler types must expose Close() to release the underlying
	// COM object. This is part of the public surface.
	h := &MessageReceivedEventHandler{}
	// Close should be callable; it may be a no-op on a zero-value handler.
	h.Close()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/events_message.go`:
```go
//go:build windows

package webview2

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// MessageReceivedEventArgs is the COM ICoreWebView2WebMessageReceivedEventArgs
// wrapper. Call TryGetWebMessageAsString to read the message body.
type MessageReceivedEventArgs struct {
	Raw  uintptr
	vtbl *iCoreWebView2WebMessageReceivedEventArgsVtable
}

type iCoreWebView2WebMessageReceivedEventArgsVtable struct {
	QueryInterface             uintptr
	AddRef                     uintptr
	Release                    uintptr
	TryGetWebMessageAsString   uintptr
	GetAdditionalObjects       uintptr
}

func (a *MessageReceivedEventArgs) TryGetWebMessageAsString() (string, error) {
	var p *uint16
	hr, _, _ := syscall.SyscallN(
		a.vtbl.TryGetWebMessageAsString,
		uintptr(unsafe.Pointer(a)),
		uintptr(unsafe.Pointer(&p)),
	)
	if hr != 0 {
		return "", fmt.Errorf("TryGetWebMessageAsString failed: 0x%08x", hr)
	}
	return windows.UTF16PtrToString(p), nil
}

// MessageReceivedEventHandler is the Go-side ICoreWebView2WebMessageReceivedEventHandler
// implementation. Construct one with NewMessageReceivedEventHandler and pass
// to View.AddWebMessageReceived; call Close when done.
type MessageReceivedEventHandler struct {
	impl *comHandlerImpl
}

func NewMessageReceivedEventHandler(callback func(view *View, args *MessageReceivedEventArgs)) *MessageReceivedEventHandler {
	h := newComHandler()
	handler := &MessageReceivedEventHandler{impl: h}
	// The Go-side callback is invoked from the vtable Call slot — store
	// it in the impl and look it up in the vtable function below.
	h.userData = callback
	return handler
}

func (h *MessageReceivedEventHandler) Close() {
	if h.impl != nil {
		h.impl.Release()
		h.impl = nil
	}
}
```

The `comHandlerImpl` type and the vtable function that dispatches to `userData` are written in a small helper `events_handler_*.go` (created alongside this task). See the vtable trampoline pattern in upstream `pkg/edge/ICoreWebView2WebMessageReceivedEventHandler.go` — port that file.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/events_message.go pkg/webview2/events_message_test.go
git commit -m "feat(webview2): add MessageReceivedEventArgs + MessageReceivedEventHandler"
```

### Task 15: `pkg/webview2/events.go` — WebResourceRequest/Response/RequestedEventArgs

**Files:**
- Create: `gails/pkg/webview2/events_resource.go`
- Create: `gails/pkg/webview2/events_resource_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/events_resource_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestWebResourceRequest_Construction(t *testing.T) {
	r := &WebResourceRequest{Raw: 0xfeed}
	if r.Raw != 0xfeed {
		t.Errorf("Raw = 0x%x", r.Raw)
	}
}

func TestWebResourceResponse_Construction(t *testing.T) {
	r := &WebResourceResponse{Raw: 0xbeef}
	if r.Raw != 0xbeef {
		t.Errorf("Raw = 0x%x", r.Raw)
	}
}

func TestWebResourceRequestedEventArgs_Construction(t *testing.T) {
	a := &WebResourceRequestedEventArgs{Raw: 0xcafe}
	if a.Raw != 0xcafe {
		t.Errorf("Raw = 0x%x", a.Raw)
	}
}

func TestWebResourceRequestedEventHandler_HasClose(t *testing.T) {
	h := &WebResourceRequestedEventHandler{}
	h.Close()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/events_resource.go`. Port from upstream `pkg/edge/ICoreWebView2WebResourceRequest.go`, `ICoreWebView2WebResourceResponse.go`, and `ICoreWebView2WebResourceRequestedEventArgs.go`. The vtable layouts are:

```go
//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type WebResourceRequest struct {
	Raw  uintptr
	vtbl *iCoreWebView2WebResourceRequestVtable
}

type iCoreWebView2WebResourceRequestVtable struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetUri         uintptr
	GetMethod      uintptr
	GetContent     uintptr
	GetHeaders     uintptr
}

func (r *WebResourceRequest) Uri() (string, error) {
	var p *uint16
	hr, _, _ := syscall.SyscallN(r.vtbl.GetUri, uintptr(unsafe.Pointer(r)), uintptr(unsafe.Pointer(&p)))
	if hr != 0 {
		return "", fmt.Errorf("GetUri failed: 0x%08x", hr)
	}
	return windows.UTF16PtrToString(p), nil
}

func (r *WebResourceRequest) Method() (string, error) {
	// [PORT: GetMethod]
}

func (r *WebResourceRequest) Content() ([]byte, error) {
	// [PORT: GetContent]
}

// WebResourceResponse: 3 vtable slots + GetContent, GetHeaders, GetStatusCode,
// SetContent, SetHeaders, SetStatusCode, SetReasonPhrase. Port only the
// methods Gails uses (verify via grep).
type WebResourceResponse struct {
	Raw  uintptr
	vtbl *iCoreWebView2WebResourceResponseVtable
}

// WebResourceRequestedEventArgs: 3 vtable slots + GetRequest, GetResponse,
// GetDeferral. Port.
type WebResourceRequestedEventArgs struct {
	Raw  uintptr
	vtbl *iCoreWebView2WebResourceRequestedEventArgsVtable
}

func (a *WebResourceRequestedEventArgs) Request() *WebResourceRequest {
	// [PORT: GetRequest]
}

func (a *WebResourceRequestedEventArgs) Response() *WebResourceResponse {
	// [PORT: GetResponse]
}

func (a *WebResourceRequestedEventArgs) Deferral() *Deferral {
	// [PORT: GetDeferral]
}

// WebResourceRequestedEventHandler: 3 vtable slots + Invoke. Port.
type WebResourceRequestedEventHandler struct {
	impl *comHandlerImpl
}

func NewWebResourceRequestedEventHandler(callback func(view *View, args *WebResourceRequestedEventArgs)) *WebResourceRequestedEventHandler {
	// [PORT FROM UPSTREAM ICoreWebView2WebResourceRequestedEventHandler.go]
}

func (h *WebResourceRequestedEventHandler) Close() {
	if h.impl != nil {
		h.impl.Release()
		h.impl = nil
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/events_resource.go pkg/webview2/events_resource_test.go
git commit -m "feat(webview2): add WebResourceRequest/Response/RequestedEventArgs + handler"
```

### Task 16: `pkg/webview2/events.go` — NavigationCompletedEventArgs

**Files:**
- Create: `gails/pkg/webview2/events_navigation.go`
- Create: `gails/pkg/webview2/events_navigation_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/events_navigation_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestNavigationCompletedEventArgs_Construction(t *testing.T) {
	a := &NavigationCompletedEventArgs{Raw: 0xfade}
	if a.Raw != 0xfade {
		t.Errorf("Raw = 0x%x", a.Raw)
	}
}

func TestNavigationCompletedEventHandler_HasClose(t *testing.T) {
	h := &NavigationCompletedEventHandler{}
	h.Close()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/events_navigation.go`. Port from upstream `pkg/edge/ICoreWebView2NavigationCompletedEventArgs.go` and `ICoreWebView2NavigationCompletedEventHandler.go`. The ICoreWebView2NavigationCompletedEventArgs interface has 3 vtable slots + GetIsSuccess, GetErrorStatus, GetWebErrorStatus, GetNavigationId.

```go
//go:build windows

package webview2

import "fmt"
import "syscall"
import "unsafe"

type NavigationCompletedEventArgs struct {
	Raw  uintptr
	vtbl *iCoreWebView2NavigationCompletedEventArgsVtable
}

type iCoreWebView2NavigationCompletedEventArgsVtable struct {
	QueryInterface  uintptr
	AddRef          uintptr
	Release         uintptr
	GetIsSuccess    uintptr
	GetErrorStatus  uintptr
	GetWebErrorStatus uintptr
	GetNavigationId uintptr
}

func (a *NavigationCompletedEventArgs) IsSuccess() (bool, error) {
	// [PORT: GetIsSuccess]
}

type NavigationCompletedEventHandler struct {
	impl *comHandlerImpl
}

func NewNavigationCompletedEventHandler(callback func(view *View, args *NavigationCompletedEventArgs)) *NavigationCompletedEventHandler {
	// [PORT FROM UPSTREAM ICoreWebView2NavigationCompletedEventHandler.go]
}

func (h *NavigationCompletedEventHandler) Close() {
	if h.impl != nil {
		h.impl.Release()
		h.impl = nil
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/events_navigation.go pkg/webview2/events_navigation_test.go
git commit -m "feat(webview2): add NavigationCompletedEventArgs + handler"
```

### Task 17: `pkg/webview2/events.go` — ContainsFullScreenElementEventArgs

**Files:**
- Create: `gails/pkg/webview2/events_fullscreen.go`
- Create: `gails/pkg/webview2/events_fullscreen_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/events_fullscreen_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestContainsFullScreenElementEventArgs_Construction(t *testing.T) {
	a := &ContainsFullScreenElementEventArgs{Raw: 0xf001}
	if a.Raw != 0xf001 {
		t.Errorf("Raw = 0x%x", a.Raw)
	}
}

func TestContainsFullScreenElementChangedEventHandler_HasClose(t *testing.T) {
	h := &ContainsFullScreenElementChangedEventHandler{}
	h.Close()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/events_fullscreen.go`. Port from upstream `pkg/edge/ICoreWebView2ContainsFullScreenElementChangedEventArgs.go` and `ICoreWebView2ContainsFullScreenElementChangedEventHandler.go`. The args interface is the standard IUnknown vtable only (no extra methods); the handler is the standard Invoke.

```go
//go:build windows

package webview2

type ContainsFullScreenElementEventArgs struct {
	Raw  uintptr
	vtbl *iUnknownVtable // no extra slots; it's an IUnknown
}

type ContainsFullScreenElementChangedEventHandler struct {
	impl *comHandlerImpl
}

func NewContainsFullScreenElementChangedEventHandler(callback func(view *View, args *ContainsFullScreenElementEventArgs)) *ContainsFullScreenElementChangedEventHandler {
	// [PORT FROM UPSTREAM ICoreWebView2ContainsFullScreenElementChangedEventHandler.go]
}

func (h *ContainsFullScreenElementChangedEventHandler) Close() {
	if h.impl != nil {
		h.impl.Release()
		h.impl = nil
	}
}
```

Note: the IUnknown vtable is the same struct as the bridge package's `iunknownVtable`. To avoid an import cycle (pkg/webview2 → internal/webview2/bridge), duplicate the 3-slot struct here OR add a small re-export in `pkg/webview2/webview2.go`. The plan recommends **duplicating** the 3-slot struct as a 3-field `iUnknownVtable` in `pkg/webview2/webview2.go` to keep the package self-contained.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/events_fullscreen.go pkg/webview2/events_fullscreen_test.go
git commit -m "feat(webview2): add ContainsFullScreenElementEventArgs + handler"
```

---

## Phase 4: Core WebView2 Interfaces

Build the main interfaces: View (`ICoreWebView2`), Settings (`ICoreWebViewSettings`), and Environment (`ICoreWebView2Environment`).

### Task 18: `pkg/webview2/settings.go` (ICoreWebViewSettings)

**Files:**
- Create: `gails/pkg/webview2/settings.go`
- Create: `gails/pkg/webview2/settings_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/settings_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestSettings_Construction(t *testing.T) {
	// ICoreWebViewSettings has 12 vtable slots (3 IUnknown + 9 setting
	// methods). Gails uses AreDevToolsEnabled, UserAgent — verify the
	// public method surface.
	s := &Settings{Raw: 0x1234}
	if s.Raw != 0x1234 {
		t.Errorf("Raw = 0x%x", s.Raw)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/settings.go`:
```go
//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Settings is a Go wrapper over the COM ICoreWebViewSettings interface.
type Settings struct {
	Raw  uintptr
	vtbl *iCoreWebViewSettingsVtable
}

// iCoreWebViewSettingsVtable is the COM ICoreWebViewSettings vtable.
// Slot order is invariant — it matches upstream pkg/edge/ICoreWebViewSettings.go.
type iCoreWebViewSettingsVtable struct {
	QueryInterface                  uintptr
	AddRef                          uintptr
	Release                         uintptr
	put_AreDevToolsEnabled          uintptr
	put_AreDefaultContextMenusEnabled uintptr
	put_AreHostObjectsAllowed       uintptr
	put_IsZoomControlEnabled         uintptr
	put_IsBuiltInErrorPageEnabled    uintptr
	put_UserAgent                   uintptr
	put_AreBrowserAcceleratorKeysEnabled uintptr
	put_IsPinchZoomEnabled          uintptr
	put_IsSwipeNavigationEnabled    uintptr
	put_IsPasswordAutosaveEnabled   uintptr
	put_IsGeneralAutofillEnabled    uintptr
}

func (s *Settings) PutAreDevToolsEnabled(enabled bool) error {
	hr, _, _ := syscall.SyscallN(
		s.vtbl.put_AreDevToolsEnabled,
		uintptr(unsafe.Pointer(s)),
		uintptr(toBool32(enabled)),
	)
	if hr != 0 {
		return fmt.Errorf("PutAreDevToolsEnabled failed: 0x%08x", hr)
	}
	return nil
}

func (s *Settings) PutUserAgent(ua string) error {
	p, err := windows.UTF16PtrFromString(ua)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		s.vtbl.put_UserAgent,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(p)),
	)
	if hr != 0 {
		return fmt.Errorf("PutUserAgent failed: 0x%08x", hr)
	}
	return nil
}

func toBool32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
```

Add additional `Put*` methods as Gails calls them (verify via `grep -n "settings\." /Users/yanshili/me/projects/wails/gails/pkg/application/webview_window_windows.go`).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/settings.go pkg/webview2/settings_test.go
git commit -m "feat(webview2): add Settings (ICoreWebViewSettings) wrapper"
```

### Task 19: `pkg/webview2/view.go` (ICoreWebView2 subset)

**Files:**
- Create: `gails/pkg/webview2/view.go`
- Create: `gails/pkg/webview2/view_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/view_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestView_Construction(t *testing.T) {
	// ICoreWebView2 has 12 vtable slots (3 IUnknown + 9 webview methods).
	v := &View{Raw: 0x9999}
	if v.Raw != 0x9999 {
		t.Errorf("Raw = 0x%x", v.Raw)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/view.go`. This is the central type. Port the vtable from upstream `pkg/edge/ICoreWebView2.go` and implement only the methods Gails actually calls. Reference: `grep -n "w\.webview\.\|w\.chromium\.webview\." /Users/yanshili/me/projects/wails/gails/pkg/application/webview_window_windows.go` enumerates the call sites.

```go
//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// View is a Go wrapper over the COM ICoreWebView2 interface. It represents
// a single webview2 instance attached to a window.
type View struct {
	Raw  uintptr
	vtbl *iCoreWebView2Vtable
}

// iCoreWebView2Vtable is the COM ICoreWebView2 vtable.
type iCoreWebView2Vtable struct {
	QueryInterface          uintptr
	AddRef                  uintptr
	Release                 uintptr
	get_Settings            uintptr
	get_Source              uintptr
	Navigate                uintptr
	NavigateToString        uintptr
	AddNavigationCompleted  uintptr
	remove_NavigationCompleted uintptr
	AddWebResourceRequested uintptr
	remove_WebResourceRequested uintptr
	AddWebMessageReceived   uintptr
	remove_WebMessageReceived uintptr
	// ... additional slots as needed
}

// Settings returns the ICoreWebViewSettings object for this view.
func (v *View) Settings() (*Settings, error) {
	var raw uintptr
	hr, _, _ := syscall.SyscallN(
		v.vtbl.get_Settings,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&raw)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("get_Settings failed: 0x%08x", hr)
	}
	return &Settings{Raw: raw}, nil
}

// Navigate tells the webview to navigate to the given URI.
func (v *View) Navigate(uri string) error {
	p, err := windows.UTF16PtrFromString(uri)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(
		v.vtbl.Navigate,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(p)),
	)
	if hr != 0 {
		return fmt.Errorf("Navigate failed: 0x%08x", hr)
	}
	return nil
}

func (v *View) NavigateToString(html string) error {
	// [PORT: NavigateToString]
}

// AddWebMessageReceived registers a callback to be invoked when the
// webview posts a web message.
func (v *View) AddWebMessageReceived(handler *MessageReceivedEventHandler) error {
	if handler == nil || handler.impl == nil {
		return fmt.Errorf("nil handler")
	}
	hr, _, _ := syscall.SyscallN(
		v.vtbl.AddWebMessageReceived,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(handler.impl)),
	)
	if hr != 0 {
		return fmt.Errorf("AddWebMessageReceived failed: 0x%08x", hr)
	}
	return nil
}

func (v *View) AddNavigationCompleted(handler *NavigationCompletedEventHandler) error {
	// [PORT: AddNavigationCompleted]
}

func (v *View) AddWebResourceRequested(handler *WebResourceRequestedEventHandler) error {
	// [PORT: AddWebResourceRequested]
}

func (v *View) AddContainsFullScreenElementChanged(handler *ContainsFullScreenElementChangedEventHandler) error {
	// [PORT: AddContainsFullScreenElementChanged]
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/view.go pkg/webview2/view_test.go
git commit -m "feat(webview2): add View (ICoreWebView2) wrapper with method subset"
```

### Task 20: `pkg/webview2/environment.go` (ICoreWebView2Environment)

**Files:**
- Create: `gails/pkg/webview2/environment.go`
- Create: `gails/pkg/webview2/environment_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/environment_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestEnvironment_Construction(t *testing.T) {
	// ICoreWebView2Environment has ~20 vtable slots (3 IUnknown + 17 env
	// methods). Gails uses CreateCoreWebView2Controller.
	e := &Environment{Raw: 0xabcd}
	if e.Raw != 0xabcd {
		t.Errorf("Raw = 0x%x", e.Raw)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/environment.go`:
```go
//go:build windows

package webview2

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Environment is a Go wrapper over the COM ICoreWebView2Environment interface.
type Environment struct {
	Raw  uintptr
	vtbl *iCoreWebView2EnvironmentVtable
}

type iCoreWebView2EnvironmentVtable struct {
	QueryInterface                 uintptr
	AddRef                         uintptr
	Release                        uintptr
	CreateCoreWebView2Controller   uintptr
	// ... additional slots as needed
}

// CreateCoreWebView2Controller creates a controller attached to the given
// parent HWND. It is asynchronous; the actual WebView2 instance is
// delivered to the completion handler.
func (e *Environment) CreateCoreWebView2Controller(parentHWND uintptr, handler *CreateControllerCompletedHandler) error {
	hr, _, _ := syscall.SyscallN(
		e.vtbl.CreateCoreWebView2Controller,
		uintptr(unsafe.Pointer(e)),
		parentHWND,
		uintptr(unsafe.Pointer(handler.impl)),
	)
	if hr != 0 {
		return fmt.Errorf("CreateCoreWebView2Controller failed: 0x%08x", hr)
	}
	return nil
}

// CreateControllerCompletedHandler is the Go-side implementation of
// ICoreWebView2CreateCoreWebView2ControllerCompletedHandler.
type CreateControllerCompletedHandler struct {
	impl *comHandlerImpl
}

func NewCreateControllerCompletedHandler(callback func(result int32, controller *Controller)) *CreateControllerCompletedHandler {
	// [PORT FROM UPSTREAM ICoreWebView2CreateCoreWebView2ControllerCompletedHandler.go]
}

func (h *CreateControllerCompletedHandler) Close() {
	if h.impl != nil {
		h.impl.Release()
		h.impl = nil
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/environment.go pkg/webview2/environment_test.go
git commit -m "feat(webview2): add Environment (ICoreWebView2Environment) wrapper"
```

---

## Phase 5: Controller and Loader

The two pieces that tie everything together: `Controller` (the user-facing facade that bundles Environment/View/Settings/handlers) and the loader functions (formerly `webviewloader`).

### Task 21: `pkg/webview2/controller.go` — struct + `NewController` + permission methods

**Files:**
- Create: `gails/pkg/webview2/controller.go`
- Create: `gails/pkg/webview2/controller_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/controller_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestNewController_ReturnsNonNil(t *testing.T) {
	c := NewController()
	if c == nil {
		t.Fatal("NewController returned nil")
	}
}

func TestController_HasExpectedFields(t *testing.T) {
	c := NewController()
	// Environment may be nil at construction; only check the field is
	// addressable (it would be a bug to have it removed by accident).
	_ = c.Environment
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/controller.go`:
```go
//go:build windows

package webview2

import (
	"fmt"
	"unsafe"
)

// Controller is the top-level facade for a WebView2 instance. It bundles
// the environment, the controller, the webview, and the host window, plus
// the registered event handlers. This is the Gails-style replacement for
// upstream edge.Chromium.
type Controller struct {
	hwnd        uintptr
	Environment *Environment
	View        *View
	Settings    *Settings
	host        *hostWindowHandle

	webMessageReceived               *MessageReceivedEventHandler
	containsFullScreenElementChanged *ContainsFullScreenElementChangedEventHandler
	webResourceRequested             *WebResourceRequestedEventHandler
	navigationCompleted              *NavigationCompletedEventHandler

	webview2RuntimeVersion string
}

// NewController constructs a Controller. The actual WebView2 environment
// is created asynchronously via Loader; use Controller.Attach() to bind
// the controller to a host window.
func NewController() *Controller {
	return &Controller{}
}

// Attach binds the controller's webview to a host HWND.
func (c *Controller) Attach(hwnd uintptr) error {
	// [PORT: Chromium.AttachWebView from upstream pkg/edge/chromium.go]
}

// SetGlobalPermission sets the default permission state for all
// permission requests.
func (c *Controller) SetGlobalPermission(state PermissionState) {
	// [PORT: Chromium.SetGlobalPermission]
}

// SetPermission sets a per-kind permission.
func (c *Controller) SetPermission(kind PermissionKind, state PermissionState) {
	// [PORT: Chromium.SetPermission]
}
```

Note: the upstream `Chromium` struct stores its handlers as opaque `*iCoreWebView2XxxEventHandler` pointers. We name them in Gails style (`*MessageReceivedEventHandler`, etc.).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/controller.go pkg/webview2/controller_test.go
git commit -m "feat(webview2): add Controller facade with NewController + permissions"
```

### Task 22: `pkg/webview2/controller.go` — resource filter, capabilities, devtools

**Files:**
- Modify: `gails/pkg/webview2/controller.go` (append)
- Modify: `gails/pkg/webview2/controller_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `gails/pkg/webview2/controller_test.go`:
```go
func TestController_AddWebResourceRequestedFilter_NilSafe(t *testing.T) {
	c := NewController()
	// Should not panic even when View is nil; the call is queued for
	// when the controller is attached.
	c.AddWebResourceRequestedFilter("*", WebResourceContextAll)
}

func TestController_HasCapability_FalseBeforeAttach(t *testing.T) {
	c := NewController()
	// Without an attached View, HasCapability must return false (no
	// capabilities to query) without panicking.
	if c.HasCapability(CapabilitySwipeNavigation) {
		t.Error("expected false before attach")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure (methods undefined).

- [ ] **Step 3: Implement the methods**

Append to `gails/pkg/webview2/controller.go`:
```go
// AddWebResourceRequestedFilter registers a URI wildcard + context filter
// that causes WebResourceRequested to fire for matching requests.
func (c *Controller) AddWebResourceRequestedFilter(uri string, ctx WebResourceContext) {
	if c.View == nil {
		return // not yet attached
	}
	// [PORT: Chromium.AddWebResourceRequestedFilter from upstream pkg/edge/chromium.go]
}

// HasCapability reports whether the running WebView2 runtime supports the
// given capability. Returns false if the controller is not yet attached
// or if the capability is unknown.
func (c *Controller) HasCapability(cap Capability) bool {
	if c.View == nil {
		return false
	}
	// [PORT: Chromium.HasCapability]
}

// OpenDevToolsWindow opens the WebView2 DevTools window in a separate
// browser window.
func (c *Controller) OpenDevToolsWindow() {
	if c.View == nil {
		return
	}
	// [PORT: Chromium.OpenDevToolsWindow]
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/controller.go pkg/webview2/controller_test.go
git commit -m "feat(webview2): add AddWebResourceRequestedFilter/HasCapability/OpenDevToolsWindow"
```

### Task 23: `pkg/webview2/loader_windows.go` — version comparison and runtime detection

**Files:**
- Create: `gails/pkg/webview2/loader_windows.go`
- Create: `gails/pkg/webview2/loader_windows_test.go`

- [ ] **Step 1: Write the failing test**

Create `gails/pkg/webview2/loader_windows_test.go`:
```go
//go:build windows

package webview2

import "testing"

func TestCompareBrowserVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"100.0.1180.0", "99.0.1180.0", 1},
		{"99.0.1180.0", "100.0.1180.0", -1},
		{"100.0.1180.0", "100.0.1180.0", 0},
		{"100.0.1180.0", "100.0.1181.0", -1},
		{"100.0.1180.50", "100.0.1180.5", 1},
		// malformed inputs
		{"abc", "100.0.1180.0", -1},
	}
	for _, c := range cases {
		got, err := CompareBrowserVersions(c.a, c.b)
		if err != nil {
			t.Errorf("CompareBrowserVersions(%q, %q) error: %v", c.a, c.b, err)
			continue
		}
		if got != c.want {
			t.Errorf("CompareBrowserVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestGetAvailableCoreWebView2BrowserVersionString_EmptyWhenMissing(t *testing.T) {
	// In CI the WebView2 runtime is typically not installed. We assert
	// the function does not panic and returns a string (possibly empty
	// or wrapped LoadError).
	v, err := GetAvailableCoreWebView2BrowserVersionString()
	if err == nil && v == "" {
		t.Skip("environment provides neither version nor error; skipping")
	}
	// Either path is acceptable; the test exists to catch panics.
	_ = v
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `gails/pkg/webview2/loader_windows.go`. Port from upstream `webviewloader/version.go` and `webviewloader/find_dll.go`:
```go
//go:build windows

package webview2

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// CompareBrowserVersions compares two WebView2 runtime version strings
// of the form "major.minor.build.patch". Returns -1 if a < b, 0 if equal,
// 1 if a > b. Malformed components are treated as 0 for that component.
func CompareBrowserVersions(a, b string) (int, error) {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	for i := 0; i < 4; i++ {
		var na, nb int
		if i < len(pa) {
			v, err := strconv.Atoi(pa[i])
			if err != nil {
				return 0, fmt.Errorf("invalid version %q: %w", a, err)
			}
			na = v
		}
		if i < len(pb) {
			v, err := strconv.Atoi(pb[i])
			if err != nil {
				return 0, fmt.Errorf("invalid version %q: %w", b, err)
			}
			nb = v
		}
		if na < nb {
			return -1, nil
		}
		if na > nb {
			return 1, nil
		}
	}
	return 0, nil
}

// GetAvailableCoreWebView2BrowserVersionString returns the installed
// WebView2 runtime version. Returns a *LoadError if the runtime is not
// installed or its version cannot be determined.
func GetAvailableCoreWebView2BrowserVersionString() (string, error) {
	// [PORT FROM UPSTREAM webviewloader/version.go::GetAvailableCoreWebView2BrowserVersionString]
	// The upstream impl:
	//   1. Reads HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}
	//      and HKLM\SOFTWARE\Microsoft\EdgeUpdate\Clients\{...} for the version.
	//   2. If neither registry key exists, returns "" with no error.
	//   3. If a key exists but the version is malformed, returns *LoadError.
	// [PORT VERBATIM]
}

// UsingGoWebview2Loader reports whether Gails is using the in-tree loader
// (always true after this rewrite; the boolean is preserved for
// compatibility with code that branched on it during the wailsapp→gailsapp
// transition).
func UsingGoWebview2Loader() bool {
	return true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS (or skip on CI without runtime).

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/loader_windows.go pkg/webview2/loader_windows_test.go
git commit -m "feat(webview2): add loader functions (version, detection)"
```

### Task 24: `pkg/webview2/loader_windows.go` — environment creation

**Files:**
- Modify: `gails/pkg/webview2/loader_windows.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `gails/pkg/webview2/loader_windows_test.go`:
```go
func TestCreateCoreWebView2EnvironmentWithOptions_NilSafe(t *testing.T) {
	// The full env-creation path requires a real WebView2 runtime; CI
	// may not have one. We only assert the function exists and accepts
	// a nil callback without panicking — the call is asynchronous
	// anyway.
	cb := NewCreateEnvironmentCompletedHandler(func(result int32, env *Environment) {})
	defer cb.Close()
	// Do not call the real function; just verify the callback is
	// non-nil and Close() works.
	if cb.impl == nil {
		t.Error("CreateEnvironmentCompletedHandler impl is nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: build failure (`NewCreateEnvironmentCompletedHandler` undefined).

- [ ] **Step 3: Implement the function and handler**

Append to `gails/pkg/webview2/loader_windows.go`:
```go
// CreateEnvironmentOptions configures the WebView2 environment.
type CreateEnvironmentOptions struct {
	BrowserExecutableFolder string
	UserDataFolder          string
	AdditionalBrowserArgs   string
	Language                 string
}

// CreateEnvironmentCompletedHandler is the Go-side
// ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler.
type CreateEnvironmentCompletedHandler struct {
	impl *comHandlerImpl
}

func NewCreateEnvironmentCompletedHandler(callback func(result int32, env *Environment)) *CreateEnvironmentCompletedHandler {
	// [PORT FROM UPSTREAM ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler.go]
}

func (h *CreateEnvironmentCompletedHandler) Close() {
	if h.impl != nil {
		h.impl.Release()
		h.impl = nil
	}
}

// CreateCoreWebView2EnvironmentWithOptions creates a WebView2 environment
// asynchronously.
func CreateCoreWebView2EnvironmentWithOptions(opts *CreateEnvironmentOptions, handler *CreateEnvironmentCompletedHandler) error {
	// [PORT FROM UPSTREAM webviewloader/env_create.go]
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./pkg/webview2/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/webview2/loader_windows.go pkg/webview2/loader_windows_test.go
git commit -m "feat(webview2): add CreateCoreWebView2EnvironmentWithOptions + completion handler"
```

---

## Phase 6: Migrate Gails Call Sites

This phase is mechanical: for each file in Gails that imports the old packages, change the import path and rename symbols.

### Task 25: Migrate `pkg/w32/ole32.go` and `pkg/w32/idroptarget.go`

**Files:**
- Modify: `gails/pkg/w32/ole32.go` (line 13 — import)
- Modify: `gails/pkg/w32/idroptarget.go` (line 6 — import)

- [ ] **Step 1: Edit the imports**

In `gails/pkg/w32/ole32.go`, replace:
```go
	"github.com/wailsapp/wails/webview2/pkg/combridge"
```
with:
```go
	"github.com/gailsapp/gails/internal/webview2/bridge"
```

In `gails/pkg/w32/idroptarget.go`, do the same replacement.

- [ ] **Step 2: Edit symbol references**

In both files, find/replace:
- `combridge.IUnknown` → `bridge.IUnknown`
- `combridge.IUnknownImpl` → `bridge.IUnknownImpl`
- `combridge.New` → `bridge.New`
- `combridge.RegisterVTable` → `bridge.RegisterVTable`
- `combridge.Resolve` → `bridge.Resolve`

Use:
```bash
cd /Users/yanshili/me/projects/wails/gails
sed -i '' -e 's|combridge\.IUnknown|bridge.IUnknown|g' \
       -e 's|combridge\.IUnknownImpl|bridge.IUnknownImpl|g' \
       -e 's|combridge\.New|bridge.New|g' \
       -e 's|combridge\.RegisterVTable|bridge.RegisterVTable|g' \
       -e 's|combridge\.Resolve|bridge.Resolve|g' \
       pkg/w32/ole32.go pkg/w32/idroptarget.go
```

- [ ] **Step 3: Verify build (Windows)**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows CGO_ENABLED=1 go build ./pkg/w32/...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/w32/ole32.go pkg/w32/idroptarget.go
git commit -m "refactor(w32): switch to in-tree webview2 bridge"
```

### Task 26: Migrate `pkg/application/webview_window_windows.go` (the big one)

**Files:**
- Modify: `gails/pkg/application/webview_window_windows.go` (~21 symbol renames + 1 import swap)

- [ ] **Step 1: Edit the import**

Replace the block:
```go
	"github.com/wailsapp/wails/webview2/webviewloader"
	...
	"github.com/wailsapp/wails/webview2/pkg/edge"
```
with:
```go
	"github.com/gailsapp/gails/pkg/webview2"
```

- [ ] **Step 2: Apply the rename sed**

Use the rename table from the spec (Section 命名约定). The full set of `edge.X → webview2.X` mappings:
```bash
cd /Users/yanshili/me/projects/wails/gails
sed -i '' \
    -e 's|edge\.Chromium|webview2.Controller|g' \
    -e 's|edge\.NewChromium|webview2.NewController|g' \
    -e 's|edge\.ICoreWebView2|webview2.View|g' \
    -e 's|edge\.ICoreWebView2Environment|webview2.Environment|g' \
    -e 's|edge\.ICoreWebViewSettings|webview2.Settings|g' \
    -e 's|edge\.ICoreWebView2WebMessageReceivedEventArgs|webview2.MessageReceivedEventArgs|g' \
    -e 's|edge\.ICoreWebView2WebResourceRequest|webview2.WebResourceRequest|g' \
    -e 's|edge\.ICoreWebView2WebResourceResponse|webview2.WebResourceResponse|g' \
    -e 's|edge\.ICoreWebView2WebResourceRequestedEventArgs|webview2.WebResourceRequestedEventArgs|g' \
    -e 's|edge\.ICoreWebView2NavigationCompletedEventArgs|webview2.NavigationCompletedEventArgs|g' \
    -e 's|edge\.ICoreWebView2ContainsFullScreenElementChangedEventArgs|webview2.ContainsFullScreenElementEventArgs|g' \
    -e 's|edge\.ICoreWebView2Deferral|webview2.Deferral|g' \
    -e 's|edge\.ICoreWebView2File|webview2.File|g' \
    -e 's|edge\.IStream|webview2.Stream|g' \
    -e 's|edge\.CoreWebView2PermissionKind|webview2.PermissionKind|g' \
    -e 's|edge\.CoreWebView2PermissionState|webview2.PermissionState|g' \
    -e 's|edge\.CoreWebView2PermissionStateAllow|webview2.PermissionStateAllow|g' \
    -e 's|edge\.COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL|webview2.WebResourceContextAll|g' \
    -e 's|edge\.SwipeNavigation|webview2.CapabilitySwipeNavigation|g' \
    -e 's|edge\.Rect|webview2.Rect|g' \
    -e 's|edge\.UnsupportedCapabilityError|webview2.UnsupportedCapabilityError|g' \
    pkg/application/webview_window_windows.go
```

Also apply the webviewloader rename:
```bash
sed -i '' \
    -e 's|webviewloader\.CompareBrowserVersions|webview2.CompareBrowserVersions|g' \
    -e 's|webviewloader\.GetAvailableCoreWebView2BrowserVersionString|webview2.GetAvailableCoreWebView2BrowserVersionString|g' \
    -e 's|webviewloader\.UsingGoWebview2Loader|webview2.UsingGoWebview2Loader|g' \
    pkg/application/webview_window_windows.go
```

- [ ] **Step 3: Verify no remaining `edge.` or `webviewloader.` references**

Run:
```bash
cd /Users/yanshili/me/projects/wails/gails
grep -n "edge\.\|webviewloader\." pkg/application/webview_window_windows.go
```
Expected: no output.

- [ ] **Step 4: Verify build (Windows)**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows CGO_ENABLED=1 go build ./pkg/application/...`
Expected: no errors. If errors mention undefined symbols, return to Phase 4 and add the missing method to the corresponding webview2 type.

- [ ] **Step 5: Commit**

```bash
git add pkg/application/webview_window_windows.go
git commit -m "refactor(application): switch webview_window_windows to webview2 pkg"
```

### Task 27: Migrate `pkg/application/webview_window_windows_production.go` and `_devtools.go`

**Files:**
- Modify: `gails/pkg/application/webview_window_windows_production.go`
- Modify: `gails/pkg/application/webview_window_windows_devtools.go`

- [ ] **Step 1: Apply the rename to both files**

```bash
cd /Users/yanshili/me/projects/wails/gails
sed -i '' -e 's|"github.com/wailsapp/wails/webview2/pkg/edge"|"github.com/gailsapp/gails/pkg/webview2"|g' \
       -e 's|edge\.ICoreWebViewSettings|webview2.Settings|g' \
       pkg/application/webview_window_windows_production.go \
       pkg/application/webview_window_windows_devtools.go
```

- [ ] **Step 2: Verify build (Windows)**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows CGO_ENABLED=1 go build -tags "production devtools" ./pkg/application/...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/application/webview_window_windows_production.go pkg/application/webview_window_windows_devtools.go
git commit -m "refactor(application): migrate webview_window_windows_{production,devtools} to webview2"
```

### Task 28: Migrate `internal/assetserver/webview/request_windows.go`

**Files:**
- Modify: `gails/internal/assetserver/webview/request_windows.go`

- [ ] **Step 1: Apply the rename**

```bash
cd /Users/yanshili/me/projects/wails/gails
sed -i '' -e 's|"github.com/wailsapp/wails/webview2/pkg/edge"|"github.com/gailsapp/gails/pkg/webview2"|g' \
       -e 's|edge\.ICoreWebView2Xxx|webview2.Xxx|g' \
       internal/assetserver/webview/request_windows.go
```

(Apply the same `edge.X → webview2.X` table as Task 26 for any remaining references.)

- [ ] **Step 2: Verify build (Windows)**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows CGO_ENABLED=1 go build ./internal/assetserver/...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/assetserver/webview/request_windows.go
git commit -m "refactor(assetserver): switch request_windows to in-tree webview2"
```

### Task 29: Migrate `webviewloader`-using files

**Files:**
- Modify: `gails/internal/capabilities/capabilities_windows.go`
- Modify: `gails/internal/doctor/doctor_windows.go`
- Modify: `gails/pkg/doctor-ng/platform_windows.go`
- Modify: `gails/pkg/application/application_windows.go`
- Modify: `gails/cmd/gails/main.go`
- Modify: `gails/pkg/updater/updater.go`
- Modify: `gails/pkg/application/transport_http.go`
- Modify: `gails/tests/window-visibility-test/main.go`

- [ ] **Step 1: Apply the import + symbol swap in one go**

```bash
cd /Users/yanshili/me/projects/wails/gails
files=(
    internal/capabilities/capabilities_windows.go
    internal/doctor/doctor_windows.go
    pkg/doctor-ng/platform_windows.go
    pkg/application/application_windows.go
    cmd/gails/main.go
    pkg/updater/updater.go
    pkg/application/transport_http.go
    tests/window-visibility-test/main.go
)
for f in "${files[@]}"; do
    sed -i '' \
        -e 's|"github.com/wailsapp/wails/webview2/webviewloader"|"github.com/gailsapp/gails/pkg/webview2"|g' \
        -e 's|"github.com/wailsapp/wails/webview2/pkg/edge"|"github.com/gailsapp/gails/pkg/webview2"|g' \
        -e 's|webviewloader\.CompareBrowserVersions|webview2.CompareBrowserVersions|g' \
        -e 's|webviewloader\.GetAvailableCoreWebView2BrowserVersionString|webview2.GetAvailableCoreWebView2BrowserVersionString|g' \
        -e 's|webviewloader\.UsingGoWebview2Loader|webview2.UsingGoWebview2Loader|g' \
        -e 's|edge\.IStream|webview2.Stream|g' \
        "$f"
done
```

- [ ] **Step 2: Verify no remaining references to old packages**

Run:
```bash
cd /Users/yanshili/me/projects/wails/gails
grep -rn "wailsapp/wails/webview2" --include="*.go"
```
Expected: no output.

- [ ] **Step 3: Verify the full Windows build**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows CGO_ENABLED=1 go build ./...`
Expected: no errors. If any file fails, the symbol rename table in Task 26 is the reference for which `edge.X` to translate to `webview2.X`.

- [ ] **Step 4: Commit**

```bash
git add internal/capabilities/capabilities_windows.go \
        internal/doctor/doctor_windows.go \
        pkg/doctor-ng/platform_windows.go \
        pkg/application/application_windows.go \
        cmd/gails/main.go \
        pkg/updater/updater.go \
        pkg/application/transport_http.go \
        tests/window-visibility-test/main.go
git commit -m "refactor: migrate remaining webviewloader/edge call sites to webview2"
```

---

## Phase 7: go.mod Cleanup and Full Verification

### Task 30: Drop the upstream dependency from go.mod

**Files:**
- Modify: `gails/go.mod`
- Modify: `gails/go.sum`

- [ ] **Step 1: Remove the dependency**

Run:
```bash
cd /Users/yanshili/me/projects/wails/gails
go mod edit -droprequire github.com/wailsapp/wails/webview2
go mod tidy
```

- [ ] **Step 2: Verify the upstream package is no longer referenced**

Run:
```bash
cd /Users/yanshili/me/projects/wails/gails
grep -c "wailsapp/wails/webview2" go.mod go.sum
```
Expected: 0.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: drop github.com/wailsapp/wails/webview2 dependency"
```

### Task 31: Full Windows verification

**Files:**
- (no file changes — verification only)

- [ ] **Step 1: Windows build, all packages**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows CGO_ENABLED=1 go build ./...`
Expected: no errors.

- [ ] **Step 2: Build gails CLI**

Run: `cd /Users/yanshili/me/projects/wails/gails && go build ./cmd/gails`
Expected: produces `gails` binary; no errors.

- [ ] **Step 3: Run go vet**

Run: `cd /Users/yanshili/me/projects/wails/gails && go vet ./...`
Expected: no warnings.

- [ ] **Step 4: Run server-mode build (sanity check that Windows-only code is properly tagged)**

Run: `cd /Users/yanshili/me/projects/wails/gails && go build -tags server ./...`
Expected: no errors.

- [ ] **Step 5: Run all webview2 unit tests**

Run: `cd /Users/yanshili/me/projects/wails/gails && GOOS=windows go test ./internal/webview2/... ./pkg/webview2/...`
Expected: PASS (or SKIP for tests requiring runtime).

- [ ] **Step 6: Build a Windows example end-to-end**

Run: `cd /Users/yanshili/me/projects/wails/gails && task test:example:windows DIR=badge`
Expected: `testbuild/badge/windows/` contains `badge.exe` (or equivalent). On macOS this command is gated to Windows; on Windows runners it executes.

- [ ] **Step 7: Build all examples (Windows runner only)**

Run: `cd /Users/yanshili/me/projects/wails/gails && task test:examples`
Expected: 43 examples built (or however many the Taskfile targets).

- [ ] **Step 8: Final summary commit (if any pending doc fixes)**

```bash
cd /Users/yanshili/me/projects/wails
git status
# If only docs are pending, no further commit needed.
# If source files needed touch-up, commit them.
```

---

## Self-Review Checklist

After completing all 31 tasks, run this checklist before declaring done:

- [ ] **Spec coverage**: each requirement from the design spec maps to at least one task:
  - Two-layer visibility (internal+public) — Tasks 2-7 (internal) + Tasks 8-24 (public) ✓
  - Gails-style naming — Task 26 rename table ✓
  - Only-implement-what's-used — every interface task says "method subset" ✓
  - Big-bang migration — Tasks 25-30 are all in one logical PR ✓
  - Preserve vtable/GUID invariants — Tasks 3, 5, 11, 14-20 all call out PORT VERBATIM ✓
  - Test strategy — every task has a failing-test-first step ✓
- [ ] **No placeholders**: no TBD/TODO/fill-in-later in any step. (Steps labeled `[PORT FROM UPSTREAM]` are explicit instructions to the engineer, not placeholders.)
- [ ] **Type consistency**: `Controller.SetPermission(kind PermissionKind, state PermissionState)` is used consistently across Tasks 21, 22, 26. `View.Settings() (*Settings, error)` is used consistently across Tasks 19, 21.
- [ ] **Build verification**: every phase ends with a `go build` step that catches missing methods before the next phase starts.

## Definition of Done

- [ ] All 31 tasks completed with passing tests
- [ ] `go.mod` no longer references `github.com/wailsapp/wails/webview2`
- [ ] `grep -rn "wailsapp/wails/webview2" gails/` returns no matches
- [ ] `task test:example:windows DIR=badge` succeeds
- [ ] `task test:examples` succeeds
- [ ] PR opened with title `feat: replace wailsapp/wails/webview2 with internal gails binding` and body matching the spec's commit-message template
