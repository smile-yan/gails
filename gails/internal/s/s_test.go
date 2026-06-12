package s

// Tests for the unexported helpers and the small set of exported functions
// that are pure or filesystem-readable enough to exercise without invoking
// os.Exit via checkError.
//
// Intentionally NOT covered (see s.go):
//   - All wrappers that call checkError (MKDIR, COPY, RMDIR, TOUCH, CHMOD, ...):
//     they terminate the test binary on any error, so they cannot be unit-tested
//     without first refactoring checkError to be injectable. Tracked as future work.
//   - EXEC (subprocess), DOWNLOAD (network).
//
// Known latent bug NOT encoded here: COPYDIR2 recurses to COPYDIR instead of
// COPYDIR2 (s.go:189). Out of scope until the os.Exit refactor lands.

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ----- Sub type alias -----

func TestSub_TypeAlias(t *testing.T) {
	s := Sub{"a": "1", "b": "2"}
	if len(s) != 2 {
		t.Fatalf("len = %d, want 2", len(s))
	}
	if s["a"] != "1" || s["b"] != "2" {
		t.Errorf("contents = %v, want {a:1 b:2}", s)
	}
	// Mutating the alias must be observable through map semantics.
	s["c"] = "3"
	if s["c"] != "3" {
		t.Errorf("after insert, s[c] = %q, want %q", s["c"], "3")
	}
}

// ----- CONTAINS -----

func TestCONTAINS(t *testing.T) {
	cases := []struct {
		name  string
		list  string
		item  string
		want  bool
	}{
		{"present mid-string", "hello world", "lo w", true},
		{"absent", "hello world", "xyz", false},
		{"exact match", "abc", "abc", true},
		{"empty haystack", "", "x", false},
		{"empty needle", "anything", "", true},
		{"both empty", "", "", true},
		{"long-list truncation prefix", "abcdefghijklmnopqrstuvwxyz0123456789XYZ", "abc", true},
		{"case sensitive", "Hello", "hello", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := CONTAINS(tc.list, tc.item); got != tc.want {
				t.Errorf("CONTAINS(%q, %q) = %v, want %v", tc.list, tc.item, got, tc.want)
			}
		})
	}
}

// ----- splitShell -----

func TestSplitShell(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{"plain tokens", "a b c", []string{"a", "b", "c"}, false},
		{"single quotes", "'a b' c", []string{"a b", "c"}, false},
		{"double quotes", `"a b" c`, []string{"a b", "c"}, false},
		{"backslash inside double quote", `"a\"b"`, []string{`a"b`}, false},
		{"backslash outside quote escapes next char", `a\b c`, []string{"ab", "c"}, false},
		{"empty quoted arg", `'' a`, []string{"", "a"}, false},
		{"leading space", ` a`, []string{"a"}, false},
		{"trailing space", `a `, []string{"a"}, false},
		{"multiple spaces between", `a   b`, []string{"a", "b"}, false},
		{"empty string", "", []string{}, false},
		{"whitespace only", "   \t  ", []string{}, false},
		{"mixed single and double", `'foo' "bar baz"`, []string{"foo", "bar baz"}, false},
		{"unterminated single quote", `'foo`, nil, true},
		{"unterminated double quote", `"foo`, nil, true},
		{"escaped space outside quote", `a\ b`, []string{"a b"}, false},
		{"escaped quote inside double", `"he said \"hi\""`, []string{`he said "hi"`}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := splitShell(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("splitShell(%q) err = %v, wantErr = %v", tc.input, err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if !equalStrings(got, tc.want) {
				t.Errorf("splitShell(%q) = %#v, want %#v", tc.input, got, tc.want)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ----- log / mute / unmute / indent / unindent -----
// These all touch package-level mutable state (Output, IndentSize, originalOutput,
// currentIndent). We save and restore in defer so we never leak into another test.

func TestLog_FormatsAndIndents(t *testing.T) {
	prevOutput := Output
	prevIndent := IndentSize
	prevCur := currentIndent
	t.Cleanup(func() {
		Output = prevOutput
		IndentSize = prevIndent
		currentIndent = prevCur
	})

	var buf bytes.Buffer
	Output = &buf
	IndentSize = 2
	currentIndent = 0
	indent()
	log("hi %d", 1)
	want := "  hi 1\n"
	if got := buf.String(); got != want {
		t.Errorf("log output = %q, want %q", got, want)
	}
}

func TestMuteUnmute(t *testing.T) {
	prevOutput := Output
	prevOriginal := originalOutput
	t.Cleanup(func() {
		Output = prevOutput
		originalOutput = prevOriginal
	})

	var buf bytes.Buffer
	Output = &buf
	originalOutput = nil

	mute()
	log("first")
	if buf.Len() != 0 {
		t.Errorf("after mute, buffer = %q, want empty", buf.String())
	}
	unmute()
	log("second")
	if !strings.Contains(buf.String(), "second") {
		t.Errorf("after unmute, buffer = %q, want it to contain %q", buf.String(), "second")
	}
}

func TestIndentUnindent(t *testing.T) {
	prevOutput := Output
	prevIndent := IndentSize
	prevCur := currentIndent
	t.Cleanup(func() {
		Output = prevOutput
		IndentSize = prevIndent
		currentIndent = prevCur
	})

	var buf bytes.Buffer
	Output = &buf
	IndentSize = 4
	currentIndent = 0

	indent()
	indent()
	log("deep")
	deepWant := "        deep\n" // 8 spaces
	if got := buf.String(); got != deepWant {
		t.Errorf("after 2x indent, got %q want %q", got, deepWant)
	}
	buf.Reset()
	unindent()
	log("shallow")
	shallowWant := "    shallow\n" // 4 spaces
	if got := buf.String(); got != shallowWant {
		t.Errorf("after unindent, got %q want %q", got, shallowWant)
	}
}

// ----- DEFER / CALLDEFER -----

func TestDeferCallDefer(t *testing.T) {
	prevDeferred := deferred
	prevOutput := Output
	t.Cleanup(func() {
		deferred = prevDeferred
		Output = prevOutput
	})
	// Avoid log output polluting the test runner.
	Output = io.Discard
	deferred = nil

	var order []int
	DEFER(func() { order = append(order, 1) })
	DEFER(func() { order = append(order, 2) })
	DEFER(func() { order = append(order, 3) })
	CALLDEFER()

	if !equalInts(order, []int{1, 2, 3}) {
		t.Errorf("deferred call order = %v, want [1 2 3]", order)
	}
	// CALLDEFER runs them in order; it does NOT reset the slice (s.go:505-510).
	// This pins the current behaviour; if a future change resets the slice,
	// the existing test will catch it.
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ----- FS booleans + MD5 -----

func TestFSBooleans(t *testing.T) {
	root := t.TempDir()
	// Fixture:
	//   root/
	//     file1.txt     ("hello")
	//     file2.txt     ("world")
	//     subdir/
	//       nested.txt  ("nested")
	mustWrite(t, filepath.Join(root, "file1.txt"), "hello")
	mustWrite(t, filepath.Join(root, "file2.txt"), "world")
	subdir := filepath.Join(root, "subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(subdir, "nested.txt"), "nested")
	emptyDir := filepath.Join(root, "empty")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("EXISTS_present", func(t *testing.T) {
		if !EXISTS(filepath.Join(root, "file1.txt")) {
			t.Error("EXISTS(file1.txt) = false, want true")
		}
	})
	t.Run("EXISTS_absent", func(t *testing.T) {
		if EXISTS(filepath.Join(root, "nope.txt")) {
			t.Error("EXISTS(nope.txt) = true, want false")
		}
	})
	t.Run("ISDIR_file", func(t *testing.T) {
		if ISDIR(filepath.Join(root, "file1.txt")) {
			t.Error("ISDIR(file1.txt) = true, want false")
		}
	})
	t.Run("ISDIR_dir", func(t *testing.T) {
		if !ISDIR(subdir) {
			t.Error("ISDIR(subdir) = false, want true")
		}
	})
	t.Run("ISFILE_file", func(t *testing.T) {
		if !ISFILE(filepath.Join(root, "file1.txt")) {
			t.Error("ISFILE(file1.txt) = false, want true")
		}
	})
	t.Run("ISFILE_dir", func(t *testing.T) {
		if ISFILE(subdir) {
			t.Error("ISFILE(subdir) = true, want false")
		}
	})
	t.Run("ISDIREMPTY_empty", func(t *testing.T) {
		if !ISDIREMPTY(emptyDir) {
			t.Error("ISDIREMPTY(emptyDir) = false, want true")
		}
	})
	t.Run("ISDIREMPTY_nonempty", func(t *testing.T) {
		if ISDIREMPTY(subdir) {
			t.Error("ISDIREMPTY(subdir) = true, want false")
		}
	})
	t.Run("SUBDIRS", func(t *testing.T) {
		got := SUBDIRS(root)
		sort.Strings(got)
		// After sort.Strings the root path comes first because '/' (0x2F) sorts
		// before any letter, so: [root, root+"/empty", root+"/subdir"].
		want := []string{root, emptyDir, subdir}
		if !equalStrings(got, want) {
			t.Errorf("SUBDIRS(root) = %v, want %v", got, want)
		}
	})
	t.Run("FINDFILES_match", func(t *testing.T) {
		got := FINDFILES(root, "file1.txt", "nested.txt")
		sort.Strings(got)
		want := []string{filepath.Join(root, "file1.txt"), filepath.Join(subdir, "nested.txt")}
		if !equalStrings(got, want) {
			t.Errorf("FINDFILES = %v, want %v", got, want)
		}
	})
	t.Run("FINDFILES_nomatch", func(t *testing.T) {
		got := FINDFILES(root, "absent.xyz")
		if len(got) != 0 {
			t.Errorf("FINDFILES(nomatch) = %v, want empty", got)
		}
	})
	t.Run("MD5FILE_known_vector", func(t *testing.T) {
		// md5("hello") = 5d41402abc4b2a76b9719d911017c592
		got := MD5FILE(filepath.Join(root, "file1.txt"))
		if want := "5d41402abc4b2a76b9719d911017c592"; got != want {
			t.Errorf("MD5FILE(hello) = %q, want %q", got, want)
		}
	})
	t.Run("MD5FILE_format", func(t *testing.T) {
		got := MD5FILE(filepath.Join(root, "file2.txt"))
		if matched := regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(got); !matched {
			t.Errorf("MD5FILE(file2.txt) = %q, want 32-char lowercase hex", got)
		}
	})
}

func mustWrite(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
