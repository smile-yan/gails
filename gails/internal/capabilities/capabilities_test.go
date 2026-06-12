package capabilities

import (
	"encoding/json"
	"testing"
)

func TestCapabilities_ZeroValueJSON(t *testing.T) {
	got := Capabilities{}.AsBytes()

	var roundTrip Capabilities
	if err := json.Unmarshal(got, &roundTrip); err != nil {
		t.Fatalf("Unmarshal of %q failed: %v", got, err)
	}
	if roundTrip.HasNativeDrag != false {
		t.Errorf("HasNativeDrag = %v, want false", roundTrip.HasNativeDrag)
	}
	if roundTrip.GTKVersion != 0 {
		t.Errorf("GTKVersion = %d, want 0", roundTrip.GTKVersion)
	}
	if roundTrip.WebKitVersion != "" {
		t.Errorf("WebKitVersion = %q, want empty", roundTrip.WebKitVersion)
	}
}

func TestCapabilities_PopulatedJSON(t *testing.T) {
	in := Capabilities{
		HasNativeDrag: true,
		GTKVersion:    3,
		WebKitVersion: "4.1",
	}
	bytes := in.AsBytes()

	var out Capabilities
	if err := json.Unmarshal(bytes, &out); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch: got %+v, want %+v", out, in)
	}
}

func TestCapabilities_AsBytes_ValidJSON(t *testing.T) {
	for _, c := range []Capabilities{
		{},
		{HasNativeDrag: true},
		{GTKVersion: 4, WebKitVersion: "6.0"},
		{HasNativeDrag: true, GTKVersion: 3, WebKitVersion: "4.1"},
	} {
		raw := c.AsBytes()
		if !json.Valid(raw) {
			t.Errorf("AsBytes(%+v) = %q is not valid JSON", c, raw)
		}
	}
}

func TestCapabilities_JSONFieldNames(t *testing.T) {
	// The exact JSON key names are part of the public contract consumed by
	// the runtime. An accidental rename or `omitempty` would silently break it.
	wantKeys := map[string]bool{
		"HasNativeDrag":  true,
		"GTKVersion":     true,
		"WebKitVersion":  true,
	}
	var generic map[string]interface{}
	if err := json.Unmarshal(Capabilities{}.AsBytes(), &generic); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(generic) != len(wantKeys) {
		t.Errorf("got %d keys, want %d (keys=%v)", len(generic), len(wantKeys), generic)
	}
	for k := range wantKeys {
		if _, ok := generic[k]; !ok {
			t.Errorf("missing expected key %q in %v", k, generic)
		}
	}
}
