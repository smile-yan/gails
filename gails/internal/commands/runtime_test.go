package commands

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")
	content := []byte("hello, gails")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("dst content = %q, want %q", got, content)
	}
}

func TestCopyFile_SourceNotExist(t *testing.T) {
	tmp := t.TempDir()
	err := CopyFile(filepath.Join(tmp, "no-such.txt"), filepath.Join(tmp, "dst.txt"))
	if err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("error is not os.IsNotExist: %v", err)
	}
}

func TestCopyFile_TargetDirNotExist(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := CopyFile(src, filepath.Join(tmp, "nope", "dst.txt"))
	if err == nil {
		t.Fatal("expected error for missing target dir, got nil")
	}
}

func TestCopyFile_StreamCopy(t *testing.T) {
	// A bit redundant with TestCopyFile, but exercises a large payload to
	// confirm io.Copy actually iterates (rather than reading 0 bytes).
	tmp := t.TempDir()
	src := filepath.Join(tmp, "big.bin")
	dst := filepath.Join(tmp, "big-copy.bin")
	payload := make([]byte, 64*1024)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(payload) {
		t.Fatalf("len mismatch: got %d want %d", len(got), len(payload))
	}
	for i := range got {
		if got[i] != payload[i] {
			t.Fatalf("byte mismatch at %d: got %d want %d", i, got[i], payload[i])
		}
	}
}

// Compile-time guard that the package still imports io (so a future cleanup
// doesn't break the test that uses io.Discard-style assertions above).
var _ = io.Discard
var _ = errors.New
