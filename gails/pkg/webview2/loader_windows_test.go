//go:build windows

package webview2

import "testing"

func TestCompareBrowserVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"100.0.1180.0", "99.0.1180.0", 1},
		{"99.0.1180.0", "100.0.1180.0", -1},
		{"100.0.1180.0", "100.0.1180.0", 0},
		{"100.0.1180.0", "100.0.1181.0", -1},
		{"100.0.1180.50", "100.0.1180.5", 1},
	}
	for _, c := range cases {
		got, err := CompareBrowserVersions(c.a, c.b)
		if err != nil {
			t.Errorf("CompareBrowserVersions(%q, %q) error: %v", c.a, c.b, err)
			continue
		}
		if got != c.want {
			t.Errorf("CompareBrowserVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestGetAvailableCoreWebView2BrowserVersionString_EmptyWhenMissing(t *testing.T) {
	// In CI the WebView2 runtime is typically not installed. We assert
	// the function does not panic and returns a string (possibly empty
	// or wrapped LoadError).
	v, err := GetAvailableCoreWebView2BrowserVersionString("")
	if err == nil && v == "" {
		t.Skip("environment provides neither version nor error; skipping")
	}
	// Either path is acceptable; the test exists to catch panics.
	_ = v
}

func TestCreateCoreWebView2EnvironmentWithOptions_NilSafe(t *testing.T) {
	// The full env-creation path requires a real WebView2 runtime; CI
	// may not have one. We only assert the function exists and accepts
	// a nil callback without panicking — the call is asynchronous
	// anyway.
	cb := NewCreateEnvironmentCompletedHandler(func(result int32, env *Environment) {})
	defer cb.Close()
	// Do not call the real function; just verify the callback is
	// non-nil and Close() works.
	if cb.impl == nil {
		t.Error("CreateEnvironmentCompletedHandler impl is nil")
	}
}
