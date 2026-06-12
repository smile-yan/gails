package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateEntitlementsPlist(t *testing.T) {
	cases := []struct {
		name        string
		entitlements []string
		wantContains []string
		wantExact   bool
		want        string
	}{
		{
			name:         "empty",
			entitlements: nil,
			wantExact:   true,
			want: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
</dict>
</plist>
`,
		},
		{
			name:         "single",
			entitlements: []string{"com.apple.security.app-sandbox"},
			wantContains: []string{
				"<key>com.apple.security.app-sandbox</key>",
				"<true/>",
			},
		},
		{
			name:         "multiple",
			entitlements: []string{"a.b.c", "d.e.f", "g.h.i"},
			wantContains: []string{
				"<key>a.b.c</key>",
				"<key>d.e.f</key>",
				"<key>g.h.i</key>",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := generateEntitlementsPlist(tc.entitlements)
			if tc.wantExact {
				if got != tc.want {
					t.Errorf("generateEntitlementsPlist(%v) =\n%q\nwant\n%q", tc.entitlements, got, tc.want)
				}
				return
			}
			for _, sub := range tc.wantContains {
				if !strings.Contains(got, sub) {
					t.Errorf("output missing %q\n-- got --\n%s", sub, got)
				}
			}
		})
	}
}

func TestParseExistingEntitlements(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    map[string]bool
		wantErr bool
	}{
		{
			name: "valid plist with two trues",
			input: `<?xml version="1.0" encoding="UTF-8"?>
<plist>
<dict>
	<key>com.apple.security.app-sandbox</key>
	<true/>
	<key>com.apple.security.network.client</key>
	<true/>
</dict>
</plist>
`,
			want: map[string]bool{
				"com.apple.security.app-sandbox": true,
				"com.apple.security.network.client": true,
			},
		},
		{
			name:  "empty",
			input: "",
			want:  map[string]bool{},
		},
		{
			name:  "key without <true/> on next line is not added",
			input: "<key>com.apple.security.cs.allow-jit</key>\n<false/>\n",
			want:  map[string]bool{},
		},
		{
			name:  "key at EOF without a following <true/> line",
			input: "<key>lone</key>",
			want:  map[string]bool{},
		},
		{
			name:  "non-key lines are ignored",
			input: "garbage\n<dict>\n<key>x</key>\n<true/>\n</dict>\n",
			want:  map[string]bool{"x": true},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "in.plist")
			if err := os.WriteFile(p, []byte(tc.input), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := parseExistingEntitlements(p)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if len(got) != len(tc.want) {
				t.Errorf("len = %d, want %d (got %v)", len(got), len(tc.want), got)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("got[%q] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestParseExistingEntitlements_FileNotExist(t *testing.T) {
	_, err := parseExistingEntitlements(filepath.Join(t.TempDir(), "nope.plist"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("error is not os.IsNotExist: %v", err)
	}
}

func TestWriteEntitlementsFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "build", "darwin", "entitlements.plist")
	keys := []string{"com.apple.security.network.client", "com.apple.security.app-sandbox"}

	if err := writeEntitlementsFile(target, keys); err != nil {
		t.Fatalf("writeEntitlementsFile: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range keys {
		if !strings.Contains(string(data), k) {
			t.Errorf("written file missing %q\n-- got --\n%s", k, data)
		}
	}
}

func TestWriteEntitlementsFile_CreatesDirs(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "deeply", "nested", "path", "out.plist")
	if err := writeEntitlementsFile(target, nil); err != nil {
		t.Fatalf("writeEntitlementsFile: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("file not created: %v", err)
	}
}
