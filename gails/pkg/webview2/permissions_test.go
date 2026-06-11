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
