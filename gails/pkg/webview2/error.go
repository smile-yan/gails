//go:build windows

package webview2

import "fmt"

// Capability is a placeholder forward-declaration for the WebView2 capability
// enumeration. The canonical definition of Capability lives in
// pkg/webview2/context.go (Task 10) and is expected to be `type Capability
// int32`. This declaration will be removed when Task 10 lands and the real
// definition is introduced. It exists now so that this file compiles
// independently in the TDD red-green cycle.
type Capability int32

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
