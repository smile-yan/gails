package commands

import (
	"testing"

	"github.com/gailsapp/gails/internal/defaults"
	"github.com/gailsapp/gails/internal/flags"
)

// TestApplyGlobalDefaults covers each branch of applyGlobalDefaults (init.go:103-133).
// Each case mutates one field of a baseline flags.Init that already has the
// built-in default value, then asserts the field was overridden when the
// GlobalDefaults has a non-empty value, OR was left alone otherwise.
func TestApplyGlobalDefaults(t *testing.T) {
	cases := []struct {
		name string
		// mutate returns a copy of the baseline options pre-mutation
		mutate  func(*flags.Init)
		globals defaults.GlobalDefaults
		check   func(t *testing.T, got flags.Init)
	}{
		{
			name: "template default applied",
			mutate: func(o *flags.Init) {
				o.TemplateName = "vanilla" // built-in default sentinel
			},
			globals: defaults.GlobalDefaults{
				Project: defaults.ProjectDefaults{DefaultTemplate: "react-ts"},
			},
			check: func(t *testing.T, got flags.Init) {
				if got.TemplateName != "react-ts" {
					t.Errorf("TemplateName = %q, want %q", got.TemplateName, "react-ts")
				}
			},
		},
		{
			name: "template not overridden when user picked a non-default",
			mutate: func(o *flags.Init) {
				o.TemplateName = "svelte" // not the default sentinel
			},
			globals: defaults.GlobalDefaults{
				Project: defaults.ProjectDefaults{DefaultTemplate: "react-ts"},
			},
			check: func(t *testing.T, got flags.Init) {
				if got.TemplateName != "svelte" {
					t.Errorf("TemplateName = %q, want %q (user choice preserved)", got.TemplateName, "svelte")
				}
			},
		},
		{
			name: "company default applied",
			mutate: func(o *flags.Init) {
				o.ProductCompany = "My Company"
			},
			globals: defaults.GlobalDefaults{
				Author: defaults.AuthorDefaults{Company: "Acme Corp"},
			},
			check: func(t *testing.T, got flags.Init) {
				if got.ProductCompany != "Acme Corp" {
					t.Errorf("ProductCompany = %q, want %q", got.ProductCompany, "Acme Corp")
				}
			},
		},
		{
			name:    "copyright generated when at built-in default",
			mutate:  func(o *flags.Init) { o.ProductCopyright = "© now, My Company" },
			globals: defaults.GlobalDefaults{Author: defaults.AuthorDefaults{Company: "Acme"}},
			check: func(t *testing.T, got flags.Init) {
				// GenerateCopyright uses a fallback template if CopyrightTemplate
				// is empty; either way the result must contain "Acme".
				if !contains(got.ProductCopyright, "Acme") {
					t.Errorf("ProductCopyright = %q, want it to contain %q", got.ProductCopyright, "Acme")
				}
			},
		},
		{
			name:   "product identifier generated when empty",
			mutate: func(o *flags.Init) { o.ProductIdentifier = "" },
			globals: defaults.GlobalDefaults{
				Project: defaults.ProjectDefaults{ProductIdentifierPrefix: "io.gails"},
			},
			check: func(t *testing.T, got flags.Init) {
				if got.ProductIdentifier != "io.gails.test" {
					t.Errorf("ProductIdentifier = %q, want %q", got.ProductIdentifier, "io.gails.test")
				}
			},
		},
		{
			name:   "product identifier left when user set it",
			mutate: func(o *flags.Init) { o.ProductIdentifier = "com.custom.app" },
			globals: defaults.GlobalDefaults{
				Project: defaults.ProjectDefaults{ProductIdentifierPrefix: "io.gails"},
			},
			check: func(t *testing.T, got flags.Init) {
				if got.ProductIdentifier != "com.custom.app" {
					t.Errorf("ProductIdentifier = %q, want user value preserved", got.ProductIdentifier)
				}
			},
		},
		{
			name:   "description generated when at default",
			mutate: func(o *flags.Init) { o.ProductDescription = "My Product Description" },
			globals: defaults.GlobalDefaults{
				Project: defaults.ProjectDefaults{DescriptionTemplate: "The {name} app"},
			},
			check: func(t *testing.T, got flags.Init) {
				if !contains(got.ProductDescription, "test") {
					t.Errorf("ProductDescription = %q, want it to contain project name %q", got.ProductDescription, "test")
				}
			},
		},
		{
			name:   "version overridden when at default",
			mutate: func(o *flags.Init) { o.ProductVersion = "0.1.0" },
			globals: defaults.GlobalDefaults{
				Project: defaults.ProjectDefaults{DefaultVersion: "1.2.3"},
			},
			check: func(t *testing.T, got flags.Init) {
				if got.ProductVersion != "1.2.3" {
					t.Errorf("ProductVersion = %q, want %q", got.ProductVersion, "1.2.3")
				}
			},
		},
		{
			name:    "globals empty -> options unchanged",
			mutate:  func(o *flags.Init) {},
			globals: defaults.GlobalDefaults{},
			check: func(t *testing.T, got flags.Init) {
				if got.TemplateName != "vanilla" {
					t.Errorf("TemplateName = %q, want %q", got.TemplateName, "vanilla")
				}
				if got.ProductCompany != "My Company" {
					t.Errorf("ProductCompany = %q, want %q", got.ProductCompany, "My Company")
				}
				if got.ProductVersion != "0.1.0" {
					t.Errorf("ProductVersion = %q, want %q", got.ProductVersion, "0.1.0")
				}
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			opts := &flags.Init{
				TemplateName:       "vanilla",
				ProjectName:        "test",
				ProductName:        "My Product",
				ProductDescription: "My Product Description",
				ProductVersion:     "0.1.0",
				ProductCompany:     "My Company",
				ProductCopyright:   "© now, My Company",
				ProductComments:    "This is a comment",
			}
			tc.mutate(opts)
			applyGlobalDefaults(opts, tc.globals)
			tc.check(t, *opts)
		})
	}
}

func TestSanitizeFileName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"with space", "with_space"},
		{"with/slash", "with_slash"},
		{"with\\bslash", "with_bslash"},
		{"with:colon", "with_colon"},
		{"accénts", "acc_nts"}, // é → _ (not in [a-zA-Z0-9_.-])
		{"path/with.multiple-dots", "path_with.multiple-dots"},
		{"", ""},
		{"a_b-c.d", "a_b-c.d"}, // allowed chars survive
		{"???", "___"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			if got := sanitizeFileName(tc.in); got != tc.want {
				t.Errorf("sanitizeFileName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// contains is a small helper used by TestApplyGlobalDefaults to assert
// substring membership without pulling in strings just for one test.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(substr) == 0 || stringIndex(s, substr) >= 0))
}

func stringIndex(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestGitURLToModulePath(t *testing.T) {
	tests := []struct {
		name   string
		gitURL string
		want   string
	}{
		{
			name:   "Simple GitHub URL",
			gitURL: "github.com/username/project",
			want:   "github.com/username/project",
		},
		{
			name:   "GitHub URL with .git suffix",
			gitURL: "github.com/username/project.git",
			want:   "github.com/username/project",
		},
		{
			name:   "HTTPS GitHub URL",
			gitURL: "https://github.com/username/project",
			want:   "github.com/username/project",
		},
		{
			name:   "HTTPS GitHub URL with .git suffix",
			gitURL: "https://github.com/username/project.git",
			want:   "github.com/username/project",
		},
		{
			name:   "HTTP GitHub URL",
			gitURL: "http://github.com/username/project",
			want:   "github.com/username/project",
		},
		{
			name:   "HTTP GitHub URL with .git suffix",
			gitURL: "http://github.com/username/project.git",
			want:   "github.com/username/project",
		},
		{
			name:   "SSH GitHub URL",
			gitURL: "git@github.com:username/project",
			want:   "github.com/username/project",
		},
		{
			name:   "SSH GitHub URL with .git suffix",
			gitURL: "git@github.com:username/project.git",
			want:   "github.com/username/project",
		},
		{
			name:   "Alternative SSH URL format",
			gitURL: "ssh://git@github.com/username/project.git",
			want:   "github.com/username/project",
		},
		{
			name:   "Git protocol URL",
			gitURL: "git://github.com/username/project.git",
			want:   "github.com/username/project",
		},
		{
			name:   "File system URL",
			gitURL: "file:///path/to/project.git",
			want:   "path/to/project",
		},
		{
			name:   "SSH GitLab URL",
			gitURL: "git@gitlab.com:username/project.git",
			want:   "gitlab.com/username/project",
		},
		{
			name:   "SSH Custom Domain",
			gitURL: "git@git.company.com:username/project.git",
			want:   "git.company.com/username/project",
		},
		{
			name:   "GitLab URL",
			gitURL: "gitlab.com/username/project",
			want:   "gitlab.com/username/project",
		},
		{
			name:   "BitBucket URL",
			gitURL: "bitbucket.org/username/project",
			want:   "bitbucket.org/username/project",
		},
		{
			name:   "Custom domain",
			gitURL: "git.company.com/username/project",
			want:   "git.company.com/username/project",
		},
		{
			name:   "Custom domain with HTTPS and .git",
			gitURL: "https://git.company.com/username/project.git",
			want:   "git.company.com/username/project",
		},
		{
			name:   "Empty string",
			gitURL: "",
			want:   "",
		},
		{
			name:   "Just .git suffix",
			gitURL: ".git",
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gitURLToModulePath(tt.gitURL); got != tt.want {
				t.Errorf("gitURLToModulePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
