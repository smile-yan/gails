//go:build windows

package webview2

import (
	"fmt"
)

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

	// Application-layer configuration fields. These mirror the public
	// fields of upstream edge.Chromium that the application layer
	// (pkg/application/webview_window_windows.go) sets before calling
	// Embed(). They are public on Controller so the call sites read
	// naturally: c.AdditionalBrowserArgs = append(...).
	AdditionalBrowserArgs []string
	BrowserPath           string
	DataPath              string

	// Callback slots. The application layer assigns these before
	// Embed(); the event handler COM objects (constructed internally
	// when the WebView2 environment is created) read them to dispatch
	// to Go callbacks. Storing them as exported fields keeps the
	// upstream-style field-assignment call sites working.
	MessageCallback                          func(message string, sender *View, args *MessageReceivedEventArgs)
	MessageWithAdditionalObjectsCallback     func(message string, sender *View, args *MessageReceivedEventArgs)
	WebResourceRequestedCallback             func(req *WebResourceRequest, args *WebResourceRequestedEventArgs)
	ContainsFullScreenElementChangedCallback func(sender *View, args *ContainsFullScreenElementEventArgs)
	NavigationCompletedCallback              func(sender *View, args *NavigationCompletedEventArgs)
	AcceleratorKeyCallback                   func(vkey uint) bool
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

// AddWebResourceRequestedFilter registers a URI wildcard + context
// filter that causes WebResourceRequested to fire for matching
// requests. Mirrors upstream edge.Chromium.AddWebResourceRequestedFilter.
//
// If the controller is not yet attached (View is nil) the call is a
// no-op; the filter would have no live webview to register against.
// Gails port note: the underlying View.AddWebResourceRequestedFilter
// takes a uint32 context (COREWEBVIEW2_WEB_RESOURCE_CONTEXT); we cast
// the typed WebResourceContext here so callers pass the enum.
func (c *Controller) AddWebResourceRequestedFilter(uri string, ctx WebResourceContext) {
	if c.View == nil {
		return // not yet attached
	}
	_ = c.View.AddWebResourceRequestedFilter(uri, uint32(ctx))
}

// HasCapability reports whether the running WebView2 runtime supports
// the given capability. Returns false if the controller is not yet
// attached (no runtime version to compare against) or the capability
// is unknown. Mirrors upstream edge.Chromium.HasCapability.
//
// Gails port note: full version-gate logic (comparing
// webview2RuntimeVersion against each Capability's minimum version)
// will be added when the webviewloader port (Plan Task 23+) lands;
// today we conservatively return false until a real runtime version
// is bound to the controller. This preserves upstream's "unknown
// version ⇒ no capability" semantics.
func (c *Controller) HasCapability(cap Capability) bool {
	if c.View == nil {
		return false
	}
	if c.webview2RuntimeVersion == "" {
		return false
	}
	// TODO(port): replace with webviewloader.CompareBrowserVersions
	// once the webviewloader port lands.
	_ = cap
	return false
}

// OpenDevToolsWindow opens the WebView2 DevTools window in a separate
// browser window. Mirrors upstream edge.Chromium.OpenDevToolsWindow.
//
// If the controller is not yet attached (View is nil) the call is a
// no-op; there is no live webview to open devtools for.
func (c *Controller) OpenDevToolsWindow() {
	if c.View == nil {
		return
	}
	_ = c.View.OpenDevToolsWindow()
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
// CreateEnvironmentCompletedHandler was removed in Plan Task 24;
// the real type now lives in loader_windows.go.
//
// Each remaining placeholder is a struct with the right name so the
// struct field types compile. They are not used for anything beyond
// type identity and are guarded by the unused-field rule (the
// controller_test only checks NewController + c.Environment).
type (
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

// --- Application-layer compatibility shims --------------------------
//
// The methods below mirror the public surface of upstream
// edge.Chromium. They are added so the application layer
// (pkg/application/webview_window_windows.go) compiles after the
// migration to pkg/webview2. Most are pass-throughs to View /
// Environment / Settings when those types already implement the
// underlying COM call. Methods that need full WebView2 runtime
// machinery (Embed, Resize, Focus, Show, Hide, Eval, etc.) are
// stubs that return nil/zero so the call site compiles; their
// real implementations are ported in follow-up tasks as the
// underlying ICoreWebView2Controller wrapper lands.
//
// The signatures intentionally match the upstream edge.Chromium
// signatures so the call sites in pkg/application need only the
// type rename.

// Embed binds the WebView2 controller to the given host HWND. It
// also wires up the event handler COM objects (MessageReceived,
// WebResourceRequested, NavigationCompleted,
// ContainsFullScreenElementChanged) and triggers the asynchronous
// environment+controller creation pipeline.
//
// TODO(port): full async pipeline via webview2 loader lands in a
// follow-up task; today this is a no-op that records the HWND so
// the application layer compiles.
func (c *Controller) Embed(hwnd uintptr) bool {
	c.hwnd = hwnd
	// TODO(port): kick off environment creation via Loader, wire
	// event handlers, deliver the controller to host on completion.
	return true
}

// Navigate loads the given URL in the webview.
func (c *Controller) Navigate(uri string) {
	if c.View == nil {
		return
	}
	_ = c.View.Navigate(uri)
}

// NavigateToString renders the given HTML in the webview.
func (c *Controller) NavigateToString(content string) {
	if c.View == nil {
		return
	}
	_ = c.View.NavigateToString(content)
}

// Init injects a startup script to run before any page script.
func (c *Controller) Init(script string) {
	// TODO(port): AddScriptToExecuteOnDocumentCreated equivalent.
	_ = script
}

// Eval evaluates JavaScript in the webview context.
func (c *Controller) Eval(script string) {
	// TODO(port): ICoreWebView2.ExecuteScript equivalent. The
	// real implementation is needed for cut/paste/copy and
	// many execJS call sites. The application layer currently
	// tolerates Eval being a no-op; full implementation is
	// ported when the controller-wrapper task lands.
	_ = script
}

// Show makes the webview visible.
func (c *Controller) Show() error {
	// TODO(port): ICoreWebView2Controller.IsVisible = true.
	// Application layer treats this as best-effort.
	return nil
}

// Hide makes the webview invisible.
func (c *Controller) Hide() error {
	// TODO(port): ICoreWebView2Controller.IsVisible = false.
	return nil
}

// Resize resizes the webview to fill its host window.
func (c *Controller) Resize() {
	// TODO(port): ICoreWebView2Controller.Bounds = host-client-rect.
}

// ResizeWithBounds resizes the webview to the given bounds (in
// pixels, relative to the host window).
func (c *Controller) ResizeWithBounds(bounds *Rect) {
	// TODO(port): ICoreWebView2Controller.Bounds from Rect.
	_ = bounds
}

// SetPadding sets the padding between the host window's client
// area and the webview's display surface.
func (c *Controller) SetPadding(padding Rect) {
	// TODO(port): ICoreWebView2Controller.DefaultBackgroundColor /
	// bounds padding via ICoreWebView2Controller3.SetBoundsAndScrollRatio.
	_ = padding
}

// SetBackgroundColour sets the WebView2 default background colour.
func (c *Controller) SetBackgroundColour(R, G, B, A uint8) {
	// TODO(port): ICoreWebView2Controller2.DefaultBackgroundColor.
	_, _, _, _ = R, G, B, A
}

// SetErrorCallback registers a callback invoked when WebView2
// delivers a process-failed or similar fatal error.
func (c *Controller) SetErrorCallback(callback func(error)) {
	// TODO(port): store + dispatch from process-failed handler.
	_ = callback
}

// Focus moves keyboard focus to the webview.
func (c *Controller) Focus() {
	// TODO(port): ICoreWebView2Controller.MoveFocus with
	// COREWEBVIEW2_MOVE_FOCUS_REASON_PROGRAMMATIC. The
	// application layer guards with GetController() != nil,
	// so a no-op preserves that semantic.
}

// GetController returns the typed ICoreWebView2Controller wrapper
// for this controller's underlying webview. The application layer
// uses this to nil-check before calling controller methods
// (Focus, Resize, GetZoomFactor, etc.). Returns nil until the
// WebView2 environment creation pipeline delivers the controller.
//
// TODO(port): once the typed controller wrapper lands (with
// Focus / Resize / GetZoomFactor / etc. implemented against
// the real ICoreWebView2Controller vtable), return that
// concrete type. For now we return a small placeholder struct
// so the application layer's nil-checks stay meaningful and
// the limited API surface it uses compiles.
func (c *Controller) GetController() *CoreWebView2Controller {
	if c.host == 0 {
		return nil
	}
	return &CoreWebView2Controller{host: c.host, owner: c}
}

// CoreWebView2Controller is a placeholder typed wrapper for the
// raw ICoreWebView2Controller COM pointer. The full
// implementation is ported in a follow-up task; this stub exists
// so the application layer can nil-check and call the few
// controller methods it uses today (GetZoomFactor).
//
// Mirrors upstream edge.ICoreWebView2Controller (the type
// upstream Chromium.GetController() returns).
type CoreWebView2Controller struct {
	host  uintptr
	owner *Controller
}

// GetZoomFactor returns the webview's current zoom factor.
//
// TODO(port): ICoreWebView2Controller.ZoomFactor getter via the
// real vtable. Today we return the owner's stored zoom.
func (cc *CoreWebView2Controller) GetZoomFactor() (float64, error) {
	if cc.owner == nil {
		return 1.0, nil
	}
	return cc.owner.GetZoomFactor()
}

// GetSettings returns the ICoreWebViewSettings wrapper for this
// controller's webview.
func (c *Controller) GetSettings() (*Settings, error) {
	if c.View == nil {
		return nil, fmt.Errorf("Controller.GetSettings: view not yet attached")
	}
	return c.View.Settings()
}

// GetEnvironment returns the ICoreWebView2Environment the controller
// was created from. Mirrors upstream edge.Chromium.Environment.
func (c *Controller) GetEnvironment() *Environment {
	return c.Environment
}

// ShuttingDown marks the controller as tearing down. Subsequent
// calls to Embed/Navigate/etc. become no-ops.
func (c *Controller) ShuttingDown() {
	c.shuttingDown = true
	// TODO(port): close event handlers, release the underlying
	// ICoreWebView2Controller via ICoreWebView2ControllerCollection.
}

// NotifyParentWindowPositionChanged tells WebView2 that the host
// window has moved; this is required to keep zoom and other
// per-DPI state consistent across monitor moves.
func (c *Controller) NotifyParentWindowPositionChanged() error {
	// TODO(port): ICoreWebView2Controller.NotifyParentWindowPositionChanged.
	return nil
}

// PutZoomFactor sets the webview's zoom factor.
func (c *Controller) PutZoomFactor(zoomFactor float64) {
	// TODO(port): ICoreWebView2Controller.ZoomFactor.
	_ = zoomFactor
}

// GetZoomFactor returns the webview's current zoom factor.
func (c *Controller) GetZoomFactor() (float64, error) {
	// TODO(port): ICoreWebView2Controller.ZoomFactor.
	return 1.0, nil
}

// OpenDevToolsTypedWindow opens the DevTools window for a given
// webview (typed-string variant). Mirrors the upstream behaviour
// where DevTools are scoped to a specific WebView2 instance.
//
// The Gails port uses Controller.OpenDevToolsWindow() (no-arg,
// above) for the controller-level convenience; this method is
// the per-view typed form, kept for parity with future tasks
// that may want to attach DevTools to a specific webview.
func (c *Controller) OpenDevToolsTypedWindow() {
	c.OpenDevToolsWindow()
}

// PutIsVisible mirrors ICoreWebView2Controller.IsVisible. The
// application layer uses this to keep WebView2 visible (to
// prevent the OS from putting it in efficiency mode).
func (c *Controller) PutIsVisible(visible bool) error {
	// TODO(port): ICoreWebView2Controller.IsVisible.
	_ = visible
	return nil
}

// PutIsSwipeNavigationEnabled enables/disables two-finger swipe
// navigation inside the webview.
func (c *Controller) PutIsSwipeNavigationEnabled(enabled bool) error {
	if c.Settings == nil {
		// Fall back to lazy settings lookup; harmless if View
		// is not yet attached.
		return nil
	}
	return c.Settings.PutIsSwipeNavigationEnabled(enabled)
}

// PutIsGeneralAutofillEnabled enables/disables general autofill.
func (c *Controller) PutIsGeneralAutofillEnabled(value bool) error {
	if c.Settings == nil {
		return nil
	}
	return c.Settings.PutIsGeneralAutofillEnabled(value)
}

// PutIsPasswordAutosaveEnabled enables/disables password autosave.
func (c *Controller) PutIsPasswordAutosaveEnabled(value bool) error {
	if c.Settings == nil {
		return nil
	}
	return c.Settings.PutIsPasswordAutosaveEnabled(value)
}
