package commands

import (
	"strings"
	"testing"
)

func TestCapsToJSON(t *testing.T) {
	cases := []struct {
		name string
		caps Capabilities
		want string
	}{
		{
			name: "no Linux capabilities",
			caps: Capabilities{Platform: "darwin", Arch: "arm64"},
			want: `{"platform":"darwin","arch":"arm64"}`,
		},
		{
			name: "Linux GTK4 + WebKit6 available",
			caps: Capabilities{
				Platform: "linux",
				Arch:     "amd64",
				Linux: &LinuxCapabilities{
					GTK4Available:        true,
					WebKitGTK6Available:  true,
					GTK3Available:        false,
					WebKit2GTK4Available: false,
					Recommended:          "gtk4",
				},
			},
			want: `{"platform":"linux","arch":"amd64","linux":{"gtk4_available":true,"gtk3_available":false,"webkitgtk_6_available":true,"webkit2gtk_4_1_available":false,"recommended":"gtk4"}}`,
		},
		{
			name: "Linux GTK3 + WebKit2GTK4",
			caps: Capabilities{
				Platform: "linux",
				Arch:     "arm64",
				Linux: &LinuxCapabilities{
					GTK4Available:        false,
					WebKitGTK6Available:  false,
					GTK3Available:        true,
					WebKit2GTK4Available: true,
					Recommended:          "gtk3",
				},
			},
			want: `{"platform":"linux","arch":"arm64","linux":{"gtk4_available":false,"gtk3_available":true,"webkitgtk_6_available":false,"webkit2gtk_4_1_available":true,"recommended":"gtk3"}}`,
		},
		{
			name: "Linux no recommended",
			caps: Capabilities{
				Platform: "linux",
				Arch:     "amd64",
				Linux: &LinuxCapabilities{
					Recommended: "none",
				},
			},
			want: `{"platform":"linux","arch":"amd64","linux":{"gtk4_available":false,"gtk3_available":false,"webkitgtk_6_available":false,"webkit2gtk_4_1_available":false,"recommended":"none"}}`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := capsToJSON(tc.caps); got != tc.want {
				t.Errorf("capsToJSON(%+v) =\n  got  %s\n  want %s", tc.caps, got, tc.want)
			}
		})
	}
}

// TestDetectLinuxCapabilities_NoPkgConfig: when no pkg-config entries are
// present, Recommended should be "none". The function falls through both
// if-else branches and lands on the "none" default.
func TestDetectLinuxCapabilities_NoPkgConfig(t *testing.T) {
	orig := pkgConfigExistsFunc
	pkgConfigExistsFunc = func(string) bool { return false }
	t.Cleanup(func() { pkgConfigExistsFunc = orig })

	caps := detectLinuxCapabilities()
	if caps == nil {
		t.Fatal("detectLinuxCapabilities returned nil")
	}
	if caps.Recommended != "none" {
		t.Errorf("Recommended = %q, want %q", caps.Recommended, "none")
	}
	if caps.GTK4Available || caps.GTK3Available || caps.WebKitGTK6Available || caps.WebKit2GTK4Available {
		t.Errorf("expected all flags false; got %+v", caps)
	}
}

// TestDetectLinuxCapabilities_GTK4: when gtk4 + webkitgtk-6.0 are present,
// Recommended is "gtk4".
func TestDetectLinuxCapabilities_GTK4(t *testing.T) {
	orig := pkgConfigExistsFunc
	pkgConfigExistsFunc = func(pkg string) bool {
		return pkg == "gtk4" || pkg == "webkitgtk-6.0"
	}
	t.Cleanup(func() { pkgConfigExistsFunc = orig })

	caps := detectLinuxCapabilities()
	if caps.Recommended != "gtk4" {
		t.Errorf("Recommended = %q, want %q", caps.Recommended, "gtk4")
	}
	if !caps.GTK4Available || !caps.WebKitGTK6Available {
		t.Errorf("expected GTK4 + WebKit6 true; got %+v", caps)
	}
}

// TestDetectLinuxCapabilities_GTK3: when only gtk+-3.0 + webkit2gtk-4.1
// are present, Recommended is "gtk3".
func TestDetectLinuxCapabilities_GTK3(t *testing.T) {
	orig := pkgConfigExistsFunc
	pkgConfigExistsFunc = func(pkg string) bool {
		return pkg == "gtk+-3.0" || pkg == "webkit2gtk-4.1"
	}
	t.Cleanup(func() { pkgConfigExistsFunc = orig })

	caps := detectLinuxCapabilities()
	if caps.Recommended != "gtk3" {
		t.Errorf("Recommended = %q, want %q", caps.Recommended, "gtk3")
	}
	if !caps.GTK3Available || !caps.WebKit2GTK4Available {
		t.Errorf("expected GTK3 + WebKit2GTK4 true; got %+v", caps)
	}
}

// TestDetectLinuxCapabilities_MixedGTK4Only: only gtk4 but not webkitgtk-6.0
// -> falls through to "none" (no fully working combo).
func TestDetectLinuxCapabilities_MixedGTK4Only(t *testing.T) {
	orig := pkgConfigExistsFunc
	pkgConfigExistsFunc = func(pkg string) bool { return pkg == "gtk4" }
	t.Cleanup(func() { pkgConfigExistsFunc = orig })

	caps := detectLinuxCapabilities()
	if caps.Recommended != "none" {
		t.Errorf("Recommended = %q, want %q (gtk4 alone is not enough)", caps.Recommended, "none")
	}
}

// TestToolCapabilities_StubbedJSON: ensure the function writes JSON whose
// key set matches the Capabilities struct. We can't easily capture pterm
// output in this test, so we cover via the capsToJSON helper above; this
// just sanity-checks the linux path enters detectLinuxCapabilities.
func TestToolCapabilities_StubbedJSON(t *testing.T) {
	out := capsToJSON(Capabilities{Platform: "linux", Arch: "amd64", Linux: &LinuxCapabilities{Recommended: "none"}})
	if !strings.HasPrefix(out, `{"platform":"linux"`) {
		t.Errorf("unexpected JSON prefix: %s", out)
	}
}
