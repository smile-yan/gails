package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpdateTaskfileVars_NoVarsSection: a Taskfile without a `vars:`
// section is left unchanged (the function prints a pterm warning rather
// than auto-inserting the section). This pins the current behaviour.
func TestUpdateTaskfileVars_NoVarsSection(t *testing.T) {
	dir := t.TempDir()
	tf := filepath.Join(dir, "Taskfile.yml")
	original := "version: '3'\n\ntasks:\n  build:\n    cmds:\n      - go build\n"
	if err := os.WriteFile(tf, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateTaskfileVars(tf, map[string]string{"GOOS": "darwin"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(tf)
	if err != nil {
		t.Fatal(err)
	}
	// The file should be unchanged — the function does NOT auto-create a
	// vars section. (This is a known limitation flagged with a pterm warning
	// at signing_setup.go:513.)
	if string(got) != original {
		t.Errorf("file changed unexpectedly:\n--- want ---\n%s--- got ---\n%s", original, got)
	}
}

// TestUpdateTaskfileVars_UncommentsAndSets: a Taskfile with a commented
// `# GOOS:` line gets the comment stripped and the value set.
func TestUpdateTaskfileVars_UncommentsAndSets(t *testing.T) {
	dir := t.TempDir()
	tf := filepath.Join(dir, "Taskfile.yml")
	original := `version: '3'

vars:
  # GOOS: "linux"
  # GOARCH: "amd64"

tasks:
  build: {}
`
	if err := os.WriteFile(tf, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateTaskfileVars(tf, map[string]string{
		"GOOS":   "darwin",
		"GOARCH": "arm64",
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(tf)
	s := string(got)
	if strings.Contains(s, "# GOOS:") {
		t.Errorf("GOOS still commented out:\n%s", s)
	}
	if !strings.Contains(s, `GOOS: "darwin"`) {
		t.Errorf("missing GOOS=darin:\n%s", s)
	}
	if !strings.Contains(s, `GOARCH: "arm64"`) {
		t.Errorf("missing GOARCH=arm64:\n%s", s)
	}
}

// TestUpdateTaskfileVars_EmptyValueKeepsComment: setting a var to ""
// keeps it as a comment (don't overwrite with empty).
func TestUpdateTaskfileVars_EmptyValueKeepsComment(t *testing.T) {
	dir := t.TempDir()
	tf := filepath.Join(dir, "Taskfile.yml")
	original := `version: '3'

vars:
  # GOOS: "linux"

tasks:
  build: {}
`
	if err := os.WriteFile(tf, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateTaskfileVars(tf, map[string]string{"GOOS": ""}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(tf)
	s := string(got)
	if !strings.Contains(s, "# GOOS:") {
		t.Errorf("GOOS should remain commented when value is empty:\n%s", s)
	}
}

// TestUpdateTaskfileVars_FileNotExist: assert clean error for missing Taskfile.
func TestUpdateTaskfileVars_FileNotExist(t *testing.T) {
	err := updateTaskfileVars(filepath.Join(t.TempDir(), "nope.yml"), map[string]string{"X": "y"})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
