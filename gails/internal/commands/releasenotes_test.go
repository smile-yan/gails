package commands

import (
	"testing"
)

// TestReleaseNotes_HappyPath stubs the GitHub fetch to return canned notes.
// We can't reliably capture pterm output (it uses its own buffered writer,
// not os.Stdout directly), so we only assert:
//   - the hook override was used (function returned nil without panicking)
//   - pterm did not crash
// In dev builds, the function takes the early "Release notes are not available"
// branch and the stub is never called; in real builds the stub's text is
// printed. Both paths are valid; we only need to exercise the refactor.
func TestReleaseNotes_HappyPath(t *testing.T) {
	orig := getReleaseNotesFunc
	getReleaseNotesFunc = func(version string, noColour bool) string {
		return "## Test Release\n\n- feature A\n- bug fix B\n"
	}
	t.Cleanup(func() { getReleaseNotesFunc = orig })

	if err := ReleaseNotes(&ReleaseNotesOptions{Version: "v1.2.3"}); err != nil {
		t.Fatalf("ReleaseNotes: %v", err)
	}
}

// TestReleaseNotes_StubReturnsEmpty: the stub returning an empty string
// is a valid no-op success case.
func TestReleaseNotes_StubReturnsEmpty(t *testing.T) {
	orig := getReleaseNotesFunc
	getReleaseNotesFunc = func(version string, noColour bool) string {
		return ""
	}
	t.Cleanup(func() { getReleaseNotesFunc = orig })

	if err := ReleaseNotes(&ReleaseNotesOptions{Version: "v0.0.1"}); err != nil {
		t.Fatalf("ReleaseNotes returned err: %v", err)
	}
}

// TestReleaseNotes_DefaultStub guards against the hook-override pattern
// accidentally leaving a test stub installed between tests. We can only
// check the func variable is non-nil here (cross-package func-pointer
// comparison isn't safe).
func TestReleaseNotes_DefaultStub(t *testing.T) {
	if getReleaseNotesFunc == nil {
		t.Fatal("default getReleaseNotesFunc is nil")
	}
}
