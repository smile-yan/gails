//go:build windows

package webview2

import "fmt"

// Controller is the top-level facade for a WebView2 instance. It bundles
// the environment, the controller, the webview, and the host window, plus
// the registered event handlers. This is the Gails-style replacement for
// upstream edge.Chromium.
//
// The struct shape mirrors upstream edge.Chromium
// (github.com/wailsapp/wails/webview2/pkg/edge/chromium.go) field-for-field
// per the Porting Protocol. The Gails port only wires a subset of the
// handlers today (MessageReceived, WebResourceRequested,
// NavigationCompleted, ContainsFullScreenElementChanged); the remaining
// handler fields (PermissionRequested, ProcessFailed,
// AcceleratorKeyPressed, envCompleted, controllerCompleted) are kept as
// nil slots so future tasks can attach them without changing the layout.
//
// Controller is a struct that wraps pointers to other Gails types
// (Environment, View, Settings) and to *EventHandler COM objects; it is
// not itself a COM wrapper. Field access is therefore plain Go field
// access — no vtable indirection.
type Controller struct {
	hwnd    uintptr
	padding struct {
		Left   int32
		Top    int32
		Right  int32
		Bottom int32
	}

	Environment *Environment
	View        *View
	Settings    *Settings

	// host is the unexported handle to the raw ICoreWebView2Controller
	// COM pointer. It is set by newControllerFromCOMPointer when
	// WebView2 delivers the controller via the create-completed
	// callback. Future tasks (controller wrapper type) will resolve
	// this into a typed COM-wrapper struct.
	host uintptr

	// Event handler COM object slots. All are constructed via the
	// *EventHandler types in events_*.go. PermissionRequested,
	// ProcessFailed, AcceleratorKeyPressed, envCompleted, and
	// controllerCompleted are forward slots — they are declared so
	// the struct mirrors upstream, but no Gails code binds them yet.
	envCompleted                     *CreateEnvironmentCompletedHandler
	controllerCompleted              *CreateControllerCompletedHandler
	webMessageReceived               *MessageReceivedEventHandler
	containsFullScreenElementChanged *ContainsFullScreenElementChangedEventHandler
	permissionRequested              *PermissionRequestedEventHandler
	webResourceRequested             *WebResourceRequestedEventHandler
	acceleratorKeyPressed            *AcceleratorKeyPressedEventHandler
	navigationCompleted              *NavigationCompletedEventHandler
	processFailed                    *ProcessFailedEventHandler

	webview2RuntimeVersion string

	// Permissions: per-kind map + optional global default. Mirrors
	// upstream edge.Chromium.permissions and edge.Chromium.globalPermission.
	permissions      map[PermissionKind]PermissionState
	globalPermission *PermissionState

	shuttingDown bool
}

// NewController constructs a Controller. The actual WebView2 environment
// is created asynchronously via Loader; use Controller.Attach() to bind
// the controller to a host window.
//
// Unused handler fields (PermissionRequested, ProcessFailed,
// AcceleratorKeyPressed, envCompleted, controllerCompleted) are left nil
// and bound in later tasks.
func NewController() *Controller {
	return &Controller{
		permissions: make(map[PermissionKind]PermissionState),
	}
}

// Attach binds the controller's webview to a host HWND. The actual
// environment/controller creation is asynchronous (the WebView2 loader
// delivers the ICoreWebView2Controller on a background thread); the
// hwnd is stored for later use by resize/visibility calls. The full
// async wiring lives in the environment-creation task.
//
// Port reference: Chromium.Embed from upstream pkg/edge/chromium.go.
func (c *Controller) Attach(hwnd uintptr) error {
	if hwnd == 0 {
		return fmt.Errorf("Controller.Attach: hwnd is zero")
	}
	c.hwnd = hwnd
	return nil
}

// SetGlobalPermission sets the default permission state for all
// permission requests. If set, this overrides per-kind permissions
// from SetPermission. Mirrors upstream
// edge.Chromium.SetGlobalPermission.
func (c *Controller) SetGlobalPermission(state PermissionState) {
	c.globalPermission = &state
}

// SetPermission sets a per-kind permission. Mirrors upstream
// edge.Chromium.SetPermission.
func (c *Controller) SetPermission(kind PermissionKind, state PermissionState) {
	if c.permissions == nil {
		c.permissions = make(map[PermissionKind]PermissionState)
	}
	c.permissions[kind] = state
}

// newControllerFromCOMPointer allocates a Controller whose host field
// holds the raw ICoreWebView2Controller pointer. The Controller is the
// Gails-side facade — the raw COM pointer is intentionally unexported
// so callers do not depend on the COM ABI.
//
// Used by the create-completed trampoline in environment.go.
func newControllerFromCOMPointer(raw uintptr) *Controller {
	return &Controller{
		host:        raw,
		permissions: make(map[PermissionKind]PermissionState),
	}
}

// --- Forward-declared handler types ---------------------------------
//
// The fields above reference types that other tasks own. The forward
// declarations are kept here (with empty bodies) so this file can
// compile standalone before those tasks land. The real types will
// replace these declarations as the owning tasks complete; Go's
// type-system forbids two declarations of the same type, so the
// owning task must remove the matching declaration here when it
// lands its real one.
//
// Each of these is a placeholder struct with the right name so the
// struct field types compile. They are not used for anything beyond
// type identity and are guarded by the unused-field rule (the
// controller_test only checks NewController + c.Environment).
type (
	// CreateEnvironmentCompletedHandler is owned by a later task
	// (Loader environment creation). Declared here as a placeholder
	// so the envCompleted field has a real type.
	CreateEnvironmentCompletedHandler struct{}

	// PermissionRequestedEventHandler is owned by a later task
	// (events PermissionRequested). Declared here as a placeholder.
	PermissionRequestedEventHandler struct{}

	// AcceleratorKeyPressedEventHandler is owned by a later task
	// (events AcceleratorKeyPressed). Declared here as a placeholder.
	AcceleratorKeyPressedEventHandler struct{}

	// ProcessFailedEventHandler is owned by a later task
	// (events ProcessFailed). Declared here as a placeholder.
	ProcessFailedEventHandler struct{}
)

// hostWindowHandle is a placeholder for a future host-window struct
// (the platform's HWND wrapper). The Controller currently uses a raw
// uintptr for the HWND; future tasks may replace it with a richer
// type without changing the public Controller surface.
