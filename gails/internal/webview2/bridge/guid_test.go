//go:build windows

package bridge

import "testing"

func TestGUID_IIDUnknownString(t *testing.T) {
	g, err := GUIDFromString("{00000000-0000-0000-C000-000000000046}")
	if err != nil {
		t.Fatalf("GUIDFromString: %v", err)
	}
	want := GUID{Data1: 0x00000000, Data2: 0x0000, Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	if *g != want {
		t.Errorf("got %+v, want %+v", *g, want)
	}
}

func TestGUID_StringRoundTrip(t *testing.T) {
	original := "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"
	g, err := GUIDFromString(original)
	if err != nil {
		t.Fatalf("GUIDFromString: %v", err)
	}
	if g.String() != original {
		t.Errorf("String() = %q, want %q", g.String(), original)
	}
}
