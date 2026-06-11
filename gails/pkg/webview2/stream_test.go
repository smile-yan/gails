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
