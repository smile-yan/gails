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
