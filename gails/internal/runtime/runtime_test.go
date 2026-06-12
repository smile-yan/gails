package runtime

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
)

func TestRuntimeInit_Shape(t *testing.T) {
	if !strings.HasPrefix(runtimeInit, "window._gails=") {
		t.Errorf("runtimeInit does not start with %q: %q", "window._gails=", runtimeInit)
	}
	if !strings.Contains(runtimeInit, "window.gails=") {
		t.Errorf("runtimeInit missing window.gails= assignment: %q", runtimeInit)
	}
}

func TestCore_NilFlags(t *testing.T) {
	got := Core(nil)
	want := runtimeInit + invoke + environment
	if got != want {
		t.Errorf("Core(nil) =\n  got  %q\n  want %q", got, want)
	}
	// runtimeInit already contains `window._gails.flags=window._gails.flags||{};`
	// (the JS-side fallback), so the more precise marker is the JSON object
	// opener `{` immediately after the assignment, which only appears when
	// flags were actually marshalled.
	if strings.Contains(got, "window._gails.flags={") {
		t.Errorf("Core(nil) should not set a flags JSON object; got %q", got)
	}
}

func TestCore_EmptyMap(t *testing.T) {
	if got := Core(map[string]any{}); got != Core(nil) {
		t.Errorf("Core(empty) != Core(nil):\n  empty %q\n  nil    %q", got, Core(nil))
	}
}

func TestCore_ValidFlags(t *testing.T) {
	in := map[string]any{
		"theme": "dark",
		"count": 3,
	}
	got := Core(in)
	// Look for the assignment immediately followed by a JSON object, which
	// only happens when flags were actually marshalled.
	const marker = "window._gails.flags={"
	idx := strings.Index(got, marker)
	if idx < 0 {
		t.Fatalf("Core(valid) missing %q: %q", marker, got)
	}
	// Extract the JSON fragment between "{" and the next ";" and assert it
	// round-trips to the same map. Go json.Marshal sorts string keys
	// deterministically, so substring parsing is safe.
	rest := got[idx+len("window._gails.flags="):]
	end := strings.Index(rest, ";")
	if end < 0 {
		t.Fatalf("flags fragment not terminated: %q", rest)
	}
	fragment := rest[:end]

	var out map[string]any
	if err := json.Unmarshal([]byte(fragment), &out); err != nil {
		t.Fatalf("Unmarshal(%q) failed: %v", fragment, err)
	}
	if out["theme"] != "dark" {
		t.Errorf("out[theme] = %v, want %q", out["theme"], "dark")
	}
	// JSON numbers decode to float64
	if out["count"] != float64(3) {
		t.Errorf("out[count] = %v, want 3", out["count"])
	}
}

// TestCore_UnmarshalableValue pins the silent-error-skip behaviour:
// json.Marshal returns an error for unsupported values (channels, funcs),
// and Core() drops the flags fragment entirely. No panic, no partial output.
func TestCore_UnmarshalableValue(t *testing.T) {
	got := Core(map[string]any{"k": func() {}})
	if strings.Contains(got, "window._gails.flags={") {
		t.Errorf("Core(unmarshalable) should not set a flags JSON object; got %q", got)
	}
	if !strings.HasPrefix(got, runtimeInit) {
		t.Errorf("Core(unmarshalable) should still start with runtimeInit; got %q", got)
	}
}

// TestCore_UnionOnHost asserts that on the current host OS the produced
// bootstrap contains the host's invoke and a "Debug":true environment entry.
func TestCore_UnionOnHost(t *testing.T) {
	got := Core(nil)
	if !strings.Contains(got, invoke) {
		t.Errorf("Core() does not contain host invoke:\n  got  %q\n  want to contain %q", got, invoke)
	}
	if !strings.Contains(got, `"OS":"`+runtime.GOOS+`"`) {
		t.Errorf("Core() missing host OS %q in environment: %q", runtime.GOOS, got)
	}
	if !strings.Contains(got, `"Arch":"`+runtime.GOARCH+`"`) {
		t.Errorf("Core() missing host Arch %q in environment: %q", runtime.GOARCH, got)
	}
}
