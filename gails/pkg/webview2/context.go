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
