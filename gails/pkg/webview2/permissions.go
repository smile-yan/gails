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
