package operatingsystem

import "testing"

func TestParseOsRelease(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  OS
	}{
		{
			name: "all four fields present",
			input: `ID="ubuntu"
NAME="Ubuntu"
VERSION_ID="22.04"
VERSION="22.04.3 LTS (Jammy Jellyfish)"`,
			want: OS{ID: "ubuntu", Name: "Ubuntu", Version: "22.04", Branding: "22.04.3 LTS (Jammy Jellyfish)"},
		},
		{
			name: "ID is lowercased even when uppercase",
			input: `ID="UBUNTU"
NAME="Ubuntu"
VERSION_ID="22.04"
VERSION="22.04"`,
			want: OS{ID: "ubuntu", Name: "Ubuntu", Version: "22.04", Branding: "22.04"},
		},
		{
			name:  "empty input -> all Unknown (except Branding which stays empty)",
			input: "",
			want:  OS{ID: "Unknown", Name: "Unknown", Version: "Unknown", Branding: ""},
		},
		{
			name:  "only garbage -> all Unknown",
			input: "not a key=value pair\n!!!",
			want:  OS{ID: "Unknown", Name: "Unknown", Version: "Unknown", Branding: ""},
		},
		{
			name: "blank lines ignored",
			input: `

ID="debian"
NAME="Debian GNU/Linux"
VERSION_ID="12"
VERSION="12 (bookworm)"
`,
			want: OS{ID: "debian", Name: "Debian GNU/Linux", Version: "12", Branding: "12 (bookworm)"},
		},
		{
			name: "VERSION_ID missing -> Version stays Unknown (current code does not fall back to VERSION)",
			input: `ID="alpine"
NAME="Alpine Linux"
VERSION="3.19"`,
			want: OS{ID: "alpine", Name: "Alpine Linux", Version: "Unknown", Branding: "3.19"},
		},
		{
			// Empty VERSION_ID value leaves Version as "" not "Unknown"; the
			// code only writes the field if the key was matched. This pins the
			// current (slightly surprising) behaviour.
			name:  "VERSION_ID empty value leaves Version empty",
			input: "ID=arch\nNAME=Arch\nVERSION_ID=\nVERSION=rolling",
			want:  OS{ID: "arch", Name: "Arch", Version: "", Branding: "rolling"},
		},
		{
			name: "extra unknown keys are ignored",
			input: `ID="fedora"
NAME="Fedora Linux"
VERSION_ID="39"
VERSION="39 (Workstation Edition)"
PRETTY_NAME="Fedora 39"
CPE_NAME="cpe:/o:fedoraproject:fedora:39"`,
			want: OS{ID: "fedora", Name: "Fedora Linux", Version: "39", Branding: "39 (Workstation Edition)"},
		},
		{
			name:  "key with embedded = splits on first = only",
			input: `ID="centos"` + "\n" + `NAME="CentOS Linux"` + "\n" + `VERSION_ID="7"` + "\n" + `VERSION="7 (Core)"`,
			want:  OS{ID: "centos", Name: "CentOS Linux", Version: "7", Branding: "7 (Core)"},
		},
		{
			// Current behaviour: with CRLF input, the CR sits between the value
			// and the closing `"`. strings.Trim(..., `"`) stops at the CR, so
			// the closing quote is *not* trimmed and CR remains in the value.
			// This pins the current (somewhat surprising) behaviour.
			name:  "CRLF: CR blocks closing-quote trim",
			input: "ID=\"rhel\"\r\nNAME=\"Red Hat\"\r\nVERSION_ID=\"9\"\r\nVERSION=\"9\"",
			want:  OS{ID: "rhel\"\r", Name: "Red Hat\"\r", Version: "9\"\r", Branding: "9"},
		},
		{
			// The Trim cutset is only `"`; single-quoted values pass through
			// unchanged. This pins that behaviour.
			name:  "single-quoted values are not trimmed",
			input: `ID='arch'` + "\n" + `NAME='Arch Linux'` + "\n" + `VERSION_ID='rolling'` + "\n" + `VERSION='rolling'`,
			want:  OS{ID: "'arch'", Name: "'Arch Linux'", Version: "'rolling'", Branding: "'rolling'"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := parseOsRelease(tc.input)
			if got == nil {
				t.Fatalf("parseOsRelease returned nil")
			}
			if *got != tc.want {
				t.Errorf("parseOsRelease(%q)\n  got  %+v\n  want %+v", tc.input, *got, tc.want)
			}
		})
	}
}
