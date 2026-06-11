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
