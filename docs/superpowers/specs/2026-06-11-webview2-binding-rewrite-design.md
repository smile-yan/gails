# WebView2 Binding Rewrite — Design Spec

**Date**: 2026-06-11
**Status**: Draft, awaiting user review
**Author**: brainstorming session

## Context

Gails (`github.com/gailsapp/gails`) is a Go framework for cross-platform desktop
and mobile apps. On Windows it embeds WebView2 (Edge/Chromium) via the
upstream Go package `github.com/wailsapp/wails/webview2 v1.0.24`. The goal of
this spec is to remove that upstream dependency and bring the binding fully
under the `gailsapp` namespace, with Gails-style naming and a refactored
internal layout.

The reason for the rewrite is governance: the Gails project is the spiritual
successor to Wails v3, and depending on the `wailsapp` org's package is at
odds with owning the Windows binding entirely. The current dependency has
been functional but is no longer being actively updated upstream; inlining
the binding lets Gails iterate on Windows support independently.

## Goals

1. Drop `github.com/wailsapp/wails/webview2` from `go.mod` with zero behavior
   change visible to end users.
2. Move all WebView2 Go code into `gailsapp`'s namespace, split into one
   internal package and one public package.
3. Use Gails-style naming (drop the `I` COM prefix, no all-caps enum names).
4. Implement only the ICoreWebView2* surface Gails actually uses — do not
   carry forward the full upstream interface catalogue.
5. Preserve the existing event/callback model and reference-counting
   semantics; do not redesign COM dispatch.

## Non-Goals

- Rewriting or optimizing the COM vtable dispatch itself.
- Supporting WebView2 runtime version negotiation / IID-to-version routing.
- Exposing a generic, reusable WebView2 binding library for third parties.
- Adding new WebView2 features (printing, additional event types, etc.).

## Package Layout

```
gails/
├── internal/
│   └── webview2/
│       ├── bridge/
│       │   ├── iunknown.go         (port of upstream combridge/iunknown.go)
│       │   ├── iunknown_impl.go    (port of upstream combridge/iunknown_impl.go)
│       │   ├── vtable.go           (port of upstream combridge/vtable.go)
│       │   ├── syscall.go          (port of upstream combridge/syscall.go)
│       │   └── bridge.go           (port of upstream combridge/bridge.go)
│       └── w32helper/
│           └── w32.go              (port of upstream internal/w32/w32.go)
└── pkg/
    └── webview2/                   (single public package, //go:build windows)
        ├── webview2.go             (NewController, version helpers)
        ├── controller.go           (Controller struct, the public facade)
        ├── environment.go          (Environment)
        ├── settings.go             (Settings)
        ├── view.go                 (View, the ICoreWebView2 wrapper)
        ├── events.go               (all *EventArgs + handler types, consolidated)
        ├── stream.go               (Stream, the IStream wrapper)
        ├── permissions.go          (PermissionKind, PermissionState, consts)
        ├── context.go              (WebResourceContext, Capability, Rect)
        ├── deferral.go             (Deferral)
        ├── file.go                 (File)
        ├── error.go                (UnsupportedCapabilityError, LoadError)
        └── loader_windows.go       (port of upstream webviewloader/*)
```

Rationale:
- The internal namespace (`internal/webview2/...`) holds the COM glue. It
  has two sub-packages: `bridge` (the COM bridge) and `w32helper` (a single
  `Ole32CoInitializeEx` wrapper ported from upstream `internal/w32`).
  `bridge` is imported by `pkg/w32/ole32.go` and `pkg/w32/idroptarget.go`
  (existing call sites) and by the public package. `w32helper` is imported
  only by `bridge`.
- One public package (`pkg/webview2`) consolidates the user-facing surface
  that was previously split across `webviewloader` and `pkg/edge`.
- `w32helper` is kept separate (one file) to avoid colliding with the
  `pkg/w32` symbol namespace and to keep the bridge package focused on COM.

## Public API Surface

### `Controller` (the main user-facing type)

```go
type Controller struct { /* wraps env + webview + controller + host + settings */ }

func NewController() *Controller

func (c *Controller) SetGlobalPermission(state PermissionState)
func (c *Controller) SetPermission(kind PermissionKind, state PermissionState)
func (c *Controller) AddWebResourceRequestedFilter(uri string, ctx WebResourceContext)
func (c *Controller) HasCapability(cap Capability) bool
func (c *Controller) OpenDevToolsWindow()
```

Models the role of upstream `edge.Chromium`. Methods intentionally narrow
to the operations Gails actually performs; non-callers of unused methods
do not exist on the new type.

### `View`, `Environment`, `Settings`

```go
type View struct { ... }       // wraps ICoreWebView2
type Environment struct { ... } // wraps ICoreWebView2Environment
type Settings struct { ... }   // wraps ICoreWebViewSettings
```

Each is constructed via the parent (`Controller` exposes its `Environment`,
`Controller.CreateWebView(...)` returns a `View`, `View.Settings` returns
`*Settings`). Method subsets are exactly what `pkg/application/webview_window_windows*.go`
call; nothing more. The subset is determined at port time by grepping
`edge\.<TypeName>` usages across `gails/` and listing every method called
on each value; only those methods get implementations.

### Events

```go
type MessageReceivedEventArgs struct { ... }
type WebResourceRequest struct { ... }
type WebResourceResponse struct { ... }
type WebResourceRequestedEventArgs struct { ... }
type NavigationCompletedEventArgs struct { ... }
type ContainsFullScreenElementEventArgs struct { ... }
type WebResourceRequestedEventHandler struct { ... }
type NavigationCompletedEventHandler struct { ... }
type ContainsFullScreenElementChangedEventHandler struct { ... }
```

Event subscription follows the upstream pattern: caller constructs a handler
object (e.g. `&WebResourceRequestedEventHandler{...}`) and registers it via
`View.AddXxx(handler)`. Handlers are released by the caller via `handler.Close()`.
For handlers that take no useful state (e.g. `ContainsFullScreenElementChanged`),
an `OnContainsFullScreenElementChanged(func())` helper is exposed to skip the
boilerplate.

### Enums and constants

```go
type PermissionKind int32
type PermissionState int32
const (
    PermissionStateDefault PermissionState = 0
    PermissionStateAllow   PermissionState = 1
    PermissionStateDeny    PermissionState = 2
)

type WebResourceContext int32
const (
    WebResourceContextAll      WebResourceContext = 0
    WebResourceContextDocument WebResourceContext = 1
    // ... only the values Gails actually passes
)

type Capability int32
const (
    CapabilitySwipeNavigation Capability = ...
)

type Rect struct{ Left, Top, Right, Bottom int32 }
```

### Loader (formerly `webviewloader`)

```go
func CompareBrowserVersions(a, b string) (int, error)
func GetAvailableCoreWebView2BrowserVersionString() (string, error)
func UsingGoWebview2Loader() bool
```

### Errors

```go
type UnsupportedCapabilityError struct {
    Capability Capability
    Reason     string
}
func (e *UnsupportedCapabilityError) Error() string
func (e *UnsupportedCapabilityError) Is(target error) bool

type LoadError struct {
    Op  string // "load_dll" | "get_version" | "create_env" | "create_controller"
    Err error
}
func (e *LoadError) Error() string
func (e *LoadError) Unwrap() error
```

`UnsupportedCapabilityError` is a direct port of the upstream type. `LoadError`
is new and aggregates errors from the DLL-discovery and environment-creation
phase.

## Internal COM Bridge

The internal bridge package is essentially a 1:1 port of upstream `combridge`
plus the upstream `internal/w32` (which contains a single `Ole32CoInitializeEx`
wrapper). Key invariants preserved from upstream:

1. **vtable slot order**: every ported ICoreWebView2* method keeps its
   position in the vtable exactly as upstream defined it. Slot order is a
   Microsoft-IDL contract; any deviation causes silent UB.
2. **GUID / IID strings**: each interface's IID is the exact 32-hex string
   upstream uses. One hex error means `QueryInterface` always fails.
3. **Reference counting**: `runtime.AddCleanup`-based finalizers continue to
   serve as the safety net for unbalances Release calls.
4. **STA COM apartment**: `runtime.LockOSThread` + `Ole32CoInitializeEx` at
   package init must remain; WebView2 rejects non-STA apartments.
5. **Handler vtables**: the `*EventHandler` types expose a virtual COM
   object (QueryInterface / AddRef / Release slots) to WebView2; the port
   preserves this exactly.

## Naming Conventions

| Upstream symbol                            | New symbol                              |
| ------------------------------------------ | --------------------------------------- |
| `edge.Chromium`                            | `webview2.Controller`                   |
| `edge.ICoreWebView2`                       | `webview2.View`                         |
| `edge.ICoreWebView2Environment`            | `webview2.Environment`                  |
| `edge.ICoreWebViewSettings`                | `webview2.Settings`                     |
| `edge.ICoreWebView2WebMessageReceivedEventArgs` | `webview2.MessageReceivedEventArgs` |
| `edge.ICoreWebView2WebResourceRequest`     | `webview2.WebResourceRequest`           |
| `edge.ICoreWebView2WebResourceResponse`    | `webview2.WebResourceResponse`          |
| `edge.ICoreWebView2WebResourceRequestedEventArgs` | `webview2.WebResourceRequestedEventArgs` |
| `edge.ICoreWebView2NavigationCompletedEventArgs` | `webview2.NavigationCompletedEventArgs` |
| `edge.ICoreWebView2ContainsFullScreenElementChangedEventArgs` | `webview2.ContainsFullScreenElementEventArgs` |
| `edge.ICoreWebView2Deferral`               | `webview2.Deferral`                     |
| `edge.ICoreWebView2File`                   | `webview2.File`                         |
| `edge.IStream`                             | `webview2.Stream`                       |
| `edge.CoreWebView2PermissionKind`          | `webview2.PermissionKind`               |
| `edge.CoreWebView2PermissionState`         | `webview2.PermissionState`              |
| `edge.CoreWebView2PermissionStateAllow`    | `PermissionStateAllow` const            |
| `edge.COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL` | `WebResourceContextAll` const         |
| `edge.SwipeNavigation`                     | `webview2.CapabilitySwipeNavigation`    |
| `edge.Rect`                                | `webview2.Rect`                         |
| `edge.UnsupportedCapabilityError`          | `webview2.UnsupportedCapabilityError`   |
| `edge.NewChromium`                         | `webview2.NewController`                |
| `combridge.IUnknown`                       | `bridge.IUnknown`                       |
| `combridge.IUnknownImpl`                   | `bridge.IUnknownImpl`                   |
| `combridge.New`                            | `bridge.New`                            |
| `combridge.RegisterVTable`                 | `bridge.RegisterVTable`                 |
| `combridge.Resolve`                        | `bridge.Resolve`                        |
| `webviewloader.CompareBrowserVersions`     | `webview2.CompareBrowserVersions`       |
| `webviewloader.GetAvailableCoreWebView2BrowserVersionString` | `webview2.GetAvailableCoreWebView2BrowserVersionString` |
| `webviewloader.UsingGoWebview2Loader`      | `webview2.UsingGoWebview2Loader`        |

The `I` prefix (Microsoft's IUnknown-derived interface convention) is dropped
throughout. The `COREWEBVIEW2_*` enum-value naming is replaced with
`WebResourceContext*` and `PermissionState*` (Go-idiomatic mixed caps).

## Error Handling

Errors flow through standard `error` returns. Categories:

| Category                | Origin                                      | Type                                  |
| ----------------------- | ------------------------------------------- | ------------------------------------- |
| COM HRESULT failure     | any ICoreWebView2* method                   | plain `error` wrapping HRESULT        |
| Unsupported capability  | `HasCapability` returned false + caller used it | `*UnsupportedCapabilityError`     |
| DLL not found           | `find_dll` / `find_dll_installed`           | `*LoadError{Op: "load_dll"}`          |
| Runtime version missing | `GetAvailableCoreWebView2BrowserVersionString` | `*LoadError{Op: "get_version"}`   |
| Env creation failed     | `CreateCoreWebView2EnvironmentWithOptions`  | `*LoadError{Op: "create_env"}`        |
| Controller creation     | `CreateCoreWebView2Controller`              | `*LoadError{Op: "create_controller"}` |

Existing Gails call sites continue to use `errors.Is(err, webview2.UnsupportedCapabilityError)`
and `globalApplication.error(...)` / `globalApplication.handleFatalError(...)`.
No new error-middleware or panic-recovery is introduced.

## Testing Strategy

WebView2 is a Windows-only COM API; CI must run tests on `windows-latest`.
Three tiers:

### Tier 1: Pure logic (Windows-only, since the package itself is `//go:build windows`)
- `TestCompareBrowserVersions` — version-string comparison edge cases
- `TestPermissionState_Constants` — enum values match upstream
- `TestUnsupportedCapabilityError_Is` — `errors.Is` behavior
- `TestLoadError_Unwrap` — `errors.Unwrap` chain
- `TestRect_ZeroValue` — zero-value semantics
- `TestWebResourceContext_KnownValues` — enum values match upstream

### Tier 2: COM dispatch invariants (`//go:build windows`)
- `TestIUnknownVTableSize` — assert exactly 3 slots
- `TestGUID_FormatAndIIDUniqueness` — every IID is 32-hex and unique
- `TestComPtr_Layout` — `unsafe.Sizeof` matches upstream
- `TestRegisterVTable_SlotOrder` — registration preserves slot order
- `TestController_PublicSurface` (compile_test.go) — reflection-based
  signature check on every public method
- `TestEvents_AllHandlersHaveClose` — every `*EventHandler` has `Close()`

### Tier 3: Integration smoke (windows-latest CI)
- `tests/webview2-rewrite-smoke/main.go` — boots a minimal gails app, asserts
  `webview2.NewController()` succeeds, `GetAvailableCoreWebView2BrowserVersionString`
  returns non-empty, a navigation-completed callback fires.
- Existing `task test:example:windows DIR=badge` and `task test:examples`
  exercise the full Gails build path on Windows.

## Migration Plan

Big-bang: one (or two closely related) PR.

### File additions (port + rename)

1. `internal/webview2/w32helper/w32.go` ← upstream `internal/w32/w32.go`
2. `internal/webview2/bridge/iunknown.go` ← upstream `pkg/combridge/iunknown.go`
3. `internal/webview2/bridge/iunknown_impl.go` ← upstream `pkg/combridge/iunknown_impl.go`
4. `internal/webview2/bridge/vtable.go` ← upstream `pkg/combridge/vtables.go` (split)
5. `internal/webview2/bridge/syscall.go` ← upstream `pkg/combridge/syscall.go`
6. `internal/webview2/bridge/bridge.go` ← upstream `pkg/combridge/bridge.go`
7. `pkg/webview2/loader_windows.go` ← upstream `webviewloader/{find_dll,find_dll_installed,version,native_module*,syscall}.go`
8. `pkg/webview2/error.go` ← port `UnsupportedCapabilityError` from
   upstream `pkg/edge/capabilities.go`; add `LoadError`
9. `pkg/webview2/permissions.go`, `pkg/webview2/context.go` ← port
   `PermissionKind` / `PermissionState` / `WebResourceContext` / `Capability` /
   `Rect` from upstream `pkg/edge/`
10. `pkg/webview2/environment.go` ← port `ICoreWebView2Environment`
    (method subset only)
11. `pkg/webview2/view.go` ← port `ICoreWebView2` (method subset only)
12. `pkg/webview2/settings.go` ← port `ICoreWebViewSettings` (method subset only)
13. `pkg/webview2/controller.go` ← port `pkg/edge/chromium.go`'s
    `Chromium` struct, renamed to `Controller`
14. `pkg/webview2/events.go` ← port 5 `*EventArgs` files, consolidated
15. `pkg/webview2/stream.go` ← port `IStream`
16. `pkg/webview2/deferral.go` ← port `ICoreWebView2Deferral`
17. `pkg/webview2/file.go` ← port `ICoreWebView2File`

### File modifications (mechanical rename + import swap)

- `pkg/w32/ole32.go` — `combridge` → `gails/internal/webview2/bridge`
- `pkg/w32/idroptarget.go` — same
- `pkg/application/webview_window_windows.go` — `edge.X` → `webview2.X`,
  21 symbol renames; import swap
- `pkg/application/webview_window_windows_production.go` — import swap +
  `edge.ICoreWebViewSettings` → `webview2.Settings`
- `pkg/application/webview_window_windows_devtools.go` — same
- `internal/assetserver/webview/request_windows.go` — import swap + symbol renames
- `internal/capabilities/capabilities_windows.go` — `webviewloader.X` →
  `webview2.X`
- `internal/doctor/doctor_windows.go` — same
- `pkg/doctor-ng/platform_windows.go` — same
- `pkg/application/application_windows.go` — same
- `cmd/gails/main.go` — same
- `pkg/updater/updater.go` — same
- `pkg/application/transport_http.go` — `edge.IStream` → `webview2.Stream`
- `tests/window-visibility-test/main.go` — same

### go.mod

```bash
go mod edit -droprequire github.com/wailsapp/wails/webview2
go mod tidy
```

### Verification

```bash
GOOS=windows CGO_ENABLED=1 go build ./...
go build ./cmd/gails
GOOS=windows go test -tags windows ./internal/webview2/... ./pkg/webview2/...
go vet ./...
go build -tags server ./...     # confirm server mode still works
task test:example:windows DIR=badge
task test:examples
```

### Commit message

```
feat: replace wailsapp/wails/webview2 with internal gails binding

- Add gails/internal/webview2/bridge (port of upstream combridge)
- Add gails/internal/webview2/w32helper (port of upstream internal/w32)
- Add gails/pkg/webview2 (port of upstream pkg/edge + webviewloader,
  Gails-style naming, only the ICoreWebView2* surface Gails uses)
- Update 13 call sites in pkg/, internal/, cmd/, tests/
- Drop github.com/wailsapp/wails/webview2 from go.mod
```

## Risks and Mitigations

| Risk                                                                   | Mitigation                                                                                  |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| vtable slot order or IID typo in port breaks QueryInterface silently   | Tier-2 dispatch invariant tests in CI on every PR                                           |
| Missed method on View / Environment / Settings causes compile error in callers | Tier-2 `TestController_PublicSurface` reflection test enumerates all methods           |
| IUnknown reference leak due to non-port of cleanup hook                 | Tier-2 smoke + Windows integration test (boots real gails app)                              |
| The 21-symbol rename misses a spot                                      | Mechanical `sed` script (in this spec) + `go vet ./...` + `go build ./...` post-edit        |
| Upstream already had latent bugs that we now inherit                    | Behavior is gated on the same vtable layout; smoke test catches regressions on Windows CI   |
| The ICoreWebView2 interface version negotiation breaks in the future   | Out of scope; spec targets current WebView2 runtime. Future PR can add negotiation if needed |

## Open Questions

None at this time. The brainstorming session resolved scope, API shape,
visibility layering, versioning strategy, implementation approach, phasing,
controller shape, and event model.

## Out of Scope (Future Work)

- Generating WebView2 bindings from IDL (would require a code generator).
- Supporting a future WebView2 runtime API version that introduces new
  required methods.
- Publishing `gails/pkg/webview2` as a separately importable module.
- Switching the COM bridge to a `go-ole`-style library.
