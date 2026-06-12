package s

// Tests for the unexported helpers and the exported functions that are
// either pure, filesystem-readable, or use the stubbed checkError (via
// exitFunc / stderrWriter hooks) to surface errors instead of calling
// os.Exit(1).
//
// Intentionally NOT covered (see s.go):
//   - EXEC (subprocess), DOWNLOAD (network).
//
// Known latent bug NOT encoded here: COPYDIR2 recurses to COPYDIR instead
// of COPYDIR2 (s.go:189). Pinned by TestCopyDir2_RecursesToCopyDir.

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

// ----- checkError hook-driven wrappers -----
// These functions used to be untestable because checkError called
// os.Exit(1) directly. Now that exitFunc is injectable, we can stub
// it to capture the exit code and stderrWriter to capture the message.

func stubExit(t *testing.T) (captured *int, restore func()) {
	t.Helper()
	origExit := exitFunc
	origStderr := stderrWriter
	var code int
	exitFunc = func(c int) { code = c }
	var buf bytes.Buffer
	stderrWriter = &buf
	return &code, func() {
		exitFunc = origExit
		stderrWriter = origStderr
	}
}

func TestMKDIR_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "newdir")
	MKDIR(target)
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("MKDIR did not create dir: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("MKDIR target is not a directory")
	}
}

func TestMKDIR_Nested(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c")
	MKDIR(target)
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("nested MKDIR failed: %v", err)
	}
}

func TestMKDIR_CustomMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "perms")
	MKDIR(target, 0o700)
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("MKDIR perms = %o, want 0o700", got)
	}
}

func TestRENAME(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	RENAME(src, dst)
	if _, err := os.Stat(dst); err != nil {
		t.Errorf("RENAME did not move file: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("RENAME did not remove source: %v", err)
	}
}

func TestRENAME_SourceNotExist_Exits(t *testing.T) {
	dir := t.TempDir()
	code, restore := stubExit(t)
	defer restore()
	RENAME(filepath.Join(dir, "nope"), filepath.Join(dir, "dst"))
	if *code != 1 {
		t.Errorf("expected exit code 1, got %d", *code)
	}
}

func TestDELETE_RemovesFile(t *testing.T) {
	// DELETE uses filepath.Join(CWD(), filename) — chdir to a temp dir
	// and use a relative name.
	dir := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	f := "victim.txt"
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	DELETE(f)
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Errorf("DELETE did not remove file: %v", err)
	}
}

func TestDELETE_NonExistent_DoesNotExit(t *testing.T) {
	// DELETE swallows the error (unlike MUSTDELETE). No exit expected.
	dir := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	code, restore := stubExit(t)
	defer restore()
	DELETE("nope")
	if *code != 0 {
		t.Errorf("DELETE exited on non-existent file (code %d)", *code)
	}
}

func TestMUSTDELETE_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	f := "victim.txt"
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	MUSTDELETE(f)
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Errorf("MUSTDELETE did not remove file: %v", err)
	}
}

func TestMUSTDELETE_NonExistent_Exits(t *testing.T) {
	dir := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	code, restore := stubExit(t)
	defer restore()
	MUSTDELETE("nope")
	if *code != 1 {
		t.Errorf("expected exit code 1, got %d", *code)
	}
}

func TestTOUCH_CreatesEmptyFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "new.txt")
	TOUCH(target)
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Errorf("TOUCH created non-empty file: %d bytes", info.Size())
	}
}

func TestCHMOD(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	CHMOD(target, 0o600)
	info, _ := os.Stat(target)
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("CHMOD perms = %o, want 0o600", got)
	}
}

func TestRMDIR_RemovesDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "victim")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	RMDIR(sub)
	if _, err := os.Stat(sub); !os.IsNotExist(err) {
		t.Errorf("RMDIR did not remove dir: %v", err)
	}
}

func TestRM_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	RM(f)
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Errorf("RM did not remove file: %v", err)
	}
}

func TestCOPY_DirTarget(t *testing.T) {
	// When target is a directory, COPY appends basename(source) to it.
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(dir, "outdir")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	COPY(src, targetDir)
	out := filepath.Join(targetDir, "src.txt")
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("COPY did not place file in target dir: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("copied content = %q, want %q", got, "hello")
	}
}

func TestMOVE_DirTarget(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(dir, "outdir")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	MOVE(src, targetDir)
	out := filepath.Join(targetDir, "src.txt")
	if _, err := os.Stat(out); err != nil {
		t.Errorf("MOVE did not place file in target dir: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("MOVE did not remove source: %v", err)
	}
}

func TestCD(t *testing.T) {
	// CD is hard to test without polluting other tests' working dir.
	// Just verify it doesn't error on a real dir (we restore cwd).
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	CD(dir)
	got, _ := os.Getwd()
	// macOS resolves /var/folders to /private/var/folders (symlink);
	// compare with EvalSymlinks on both sides to be portable.
	resolvedGot, _ := filepath.EvalSymlinks(got)
	resolvedDir, _ := filepath.EvalSymlinks(dir)
	if resolvedGot != resolvedDir {
		t.Errorf("CD did not change cwd: got %q (resolved %q), want %q (resolved %q)",
			got, resolvedGot, dir, resolvedDir)
	}
}

func TestSYMLINK(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	SYMLINK(target, link)
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("SYMLINK did not create link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("SYMLINK target is not a symlink")
	}
}

func TestENDIR_Idempotent(t *testing.T) {
	// ENDIR is a no-error variant of MKDIR; calling it on an existing
	// dir is a no-op.
	dir := t.TempDir()
	ENDIR(dir) // should not exit
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Errorf("ENDIR dir not present: %v", err)
	}
}

func TestSAVEBYTES_And_LOADBYTES(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "data.bin")
	SAVEBYTES(f, []byte{0x01, 0x02, 0x03})
	got := LOADBYTES(f)
	if !bytes.Equal(got, []byte{0x01, 0x02, 0x03}) {
		t.Errorf("LOADBYTES = %v, want [1 2 3]", got)
	}
}

func TestSAVESTRING_And_LOADSTRING(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "text.txt")
	SAVESTRING(f, "hello world")
	got := LOADSTRING(f)
	if got != "hello world" {
		t.Errorf("LOADSTRING = %q, want %q", got, "hello world")
	}
}

func TestREPLACEALL(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "tpl.txt")
	if err := os.WriteFile(f, []byte("Hello, NAME!"), 0o644); err != nil {
		t.Fatal(err)
	}
	REPLACEALL(f, Sub{"NAME": "World", "Hello": "Hi"})
	got, _ := os.ReadFile(f)
	if string(got) != "Hi, World!" {
		t.Errorf("REPLACEALL = %q, want %q", got, "Hi, World!")
	}
}

func TestCOPYDIR_Success(t *testing.T) {
	// Build a small tree under src, COPYDIR to dst, verify mirror.
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("B"), 0o644); err != nil {
		t.Fatal(err)
	}
	dstParent := t.TempDir()
	dst := filepath.Join(dstParent, "dst")
	COPYDIR(src, dst)
	if _, err := os.Stat(filepath.Join(dst, "a.txt")); err != nil {
		t.Errorf("COPYDIR missing a.txt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "sub", "b.txt")); err != nil {
		t.Errorf("COPYDIR missing sub/b.txt: %v", err)
	}
}

func TestCOPYDIR_DstExists_Exits(t *testing.T) {
	src := t.TempDir()
	dstParent := t.TempDir()
	dst := filepath.Join(dstParent, "dst")
	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	code, restore := stubExit(t)
	defer restore()
	COPYDIR(src, dst)
	if *code != 1 {
		t.Errorf("expected exit 1 when dst exists, got %d", *code)
	}
}

func TestCopyDir2_RecursesToCopyDir(t *testing.T) {
	// Documents the latent bug at s.go:189: COPYDIR2 delegates to COPYDIR
	// (not COPYDIR2) when recursing into subdirectories. The bug only
	// manifests when dst/subdir already exists — the recursive COPYDIR
	// call sees the existing dst/subdir and exits with
	// "destination already exists".
	//
	// A correct COPYDIR2 implementation would call COPYDIR2 recursively,
	// which would idempotently MKDIR(dst/subdir) and succeed.
	//
	// This test pins the current (buggy) behaviour: with the bug, this
	// test asserts exit code 1. A future fix to s.go:189 (replace
	// COPYDIR with COPYDIR2) will need this test updated.
	src := t.TempDir()
	subSrc := filepath.Join(src, "sub")
	if err := os.Mkdir(subSrc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subSrc, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pre-create dst and dst/sub — this is the configuration that
	// triggers the recursive COPYDIR's "destination already exists" check.
	dstParent := t.TempDir()
	dst := filepath.Join(dstParent, "dst")
	if err := os.MkdirAll(filepath.Join(dst, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	code, restore := stubExit(t)
	defer restore()
	COPYDIR2(src, dst)
	if *code != 1 {
		t.Errorf("COPYDIR2 did not exit (code %d); the latent COPYDIR-delegation bug at s.go:189 may have been fixed; update this test", *code)
	}
}
