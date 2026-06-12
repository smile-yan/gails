package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpdateTaskfileVars_NoVarsSection_AutoInserts: a Taskfile without a
// `vars:` section gets a new `vars:` block auto-inserted before `tasks:`.
// This replaces the previous quirk where the function only printed a
// pterm warning and left the file unchanged.
func TestUpdateTaskfileVars_NoVarsSection_AutoInserts(t *testing.T) {
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
	s := string(got)
	if !strings.Contains(s, "vars:") {
		t.Errorf("vars: header missing:\n%s", s)
	}
	if !strings.Contains(s, `GOOS: "darwin"`) {
		t.Errorf("GOOS var missing:\n%s", s)
	}
	// The vars block must come BEFORE tasks:
	varsIdx := strings.Index(s, "vars:")
	tasksIdx := strings.Index(s, "tasks:")
	if varsIdx < 0 || tasksIdx < 0 || varsIdx > tasksIdx {
		t.Errorf("vars: must appear before tasks: (vars=%d, tasks=%d):\n%s",
			varsIdx, tasksIdx, s)
	}
}

// TestUpdateTaskfileVars_NoVarsNoTasks: when there's neither vars: nor
// tasks: the new vars block is appended at end of file.
func TestUpdateTaskfileVars_NoVarsNoTasks(t *testing.T) {
	dir := t.TempDir()
	tf := filepath.Join(dir, "Taskfile.yml")
	original := "version: '3'\n"
	if err := os.WriteFile(tf, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateTaskfileVars(tf, map[string]string{"FOO": "bar"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(tf)
	s := string(got)
	if !strings.Contains(s, "vars:") {
		t.Errorf("vars: header missing:\n%s", s)
	}
	if !strings.Contains(s, `FOO: "bar"`) {
		t.Errorf("FOO var missing:\n%s", s)
	}
}

// TestUpdateTaskfileVars_NoVarsNoBlankLine: when the line before tasks:
// is non-empty (no blank line separator), the new vars block gets its
// own leading blank line so YAML readers see a clear block boundary.
func TestUpdateTaskfileVars_NoVarsNoBlankLine(t *testing.T) {
	dir := t.TempDir()
	tf := filepath.Join(dir, "Taskfile.yml")
	// No blank line between version and tasks
	original := "version: '3'\ntasks:\n  build: {}\n"
	if err := os.WriteFile(tf, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateTaskfileVars(tf, map[string]string{"X": "y"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(tf)
	s := string(got)
	if !strings.Contains(s, "vars:") || !strings.Contains(s, `X: "y"`) {
		t.Errorf("vars block missing:\n%s", s)
	}
	// Expect blank lines on both sides of the new vars block
	if !strings.Contains(s, "version: '3'\n\nvars:") {
		t.Errorf("expected blank line before vars:\n%s", s)
	}
	if !strings.Contains(s, "vars:\n  X: \"y\"\n\ntasks:") {
		t.Errorf("expected blank line after vars:\n%s", s)
	}
}

// TestUpdateTaskfileVars_MultiVarInsertion: a single call with multiple
// vars adds all of them in order.
func TestUpdateTaskfileVars_MultiVarInsertion(t *testing.T) {
	dir := t.TempDir()
	tf := filepath.Join(dir, "Taskfile.yml")
	original := "version: '3'\n\ntasks:\n  build: {}\n"
	if err := os.WriteFile(tf, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateTaskfileVars(tf, map[string]string{
		"GOOS":   "linux",
		"GOARCH": "amd64",
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(tf)
	s := string(got)
	if !strings.Contains(s, `GOOS: "linux"`) {
		t.Errorf("missing GOOS=linux:\n%s", s)
	}
	if !strings.Contains(s, `GOARCH: "amd64"`) {
		t.Errorf("missing GOARCH=amd64:\n%s", s)
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
