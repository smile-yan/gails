package commands

import "testing"

func TestToCamelCasePlugin(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// The "Plugin" suffix is unconditionally appended. The function treats
		// each non-alphanumeric rune as a CamelCase boundary.
		{"my-plugin", "MyPluginPlugin"},
		{"foo_bar", "FooBarPlugin"},
		{"alreadyCamel", "AlreadyCamelPlugin"},
		{"ALLCAPS", "ALLCAPSPlugin"},
		{"", "Plugin"},
		{"42", "42Plugin"},
		{"a-b_c d", "ABCDPlugin"},
		{"hello world", "HelloWorldPlugin"},
		{"foo.bar", "FooBarPlugin"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			if got := toCamelCasePlugin(tc.in); got != tc.want {
				t.Errorf("toCamelCasePlugin(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
