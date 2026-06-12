package operatingsystem

import (
	"runtime"
	"testing"
)

func TestOS_AsLogSlice(t *testing.T) {
	o := &OS{
		ID:       "darwin",
		Name:     "MacOS",
		Version:  "14.5",
		Branding: "Sonoma",
	}
	got := o.AsLogSlice()
	want := []any{"ID", "darwin", "Name", "MacOS", "Version", "14.5", "Branding", "Sonoma"}
	if len(got) != len(want) {
		t.Fatalf("AsLogSlice len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AsLogSlice[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// TestInfo_DarwinSmoke is a host-gated smoke test: on darwin we shell out to
// `sysctl` for kern.osproductversion / kern.osversion and assert the result
// is non-nil and has the expected field shape. This is *not* a behaviour test
// for the parsed values, just that the wiring on the host platform works.
func TestInfo_DarwinSmoke(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skipf("smoke test only runs on darwin (have %s)", runtime.GOOS)
	}
	os, err := Info()
	if err != nil {
		t.Fatalf("Info() error: %v", err)
	}
	if os == nil {
		t.Fatal("Info() returned nil OS with no error")
	}
	if os.Name == "" {
		t.Error("OS.Name is empty on darwin host (expected \"MacOS\")")
	}
	if os.Version == "" {
		t.Error("OS.Version is empty on darwin host (sysctl returned nothing?)")
	}
}

// TestPlatformInfo_ReadOsReleaseHook sanity-checks the readOsReleaseFile
// indirection on any host. The hook is only invoked by the linux platformInfo
// path, but the variable itself is always declared in os_linux.go with a
// build tag, so we only exercise the contract on linux.
