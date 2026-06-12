package kvstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gailsapp/gails/pkg/application"
)

func TestNew(t *testing.T) {
	kvs := New()
	if kvs == nil {
		t.Fatal("New() returned nil")
	}
	if kvs.config != nil {
		t.Fatal("New() should produce an in-memory store")
	}
}

func TestNewWithConfig(t *testing.T) {
	kvs := NewWithConfig(nil)
	if kvs.config != nil {
		t.Fatal("NewWithConfig(nil) should produce an in-memory store")
	}

	cfg := &Config{Filename: "/tmp/x", AutoSave: true}
	kvs = NewWithConfig(cfg)
	if kvs.config == nil {
		t.Fatal("NewWithConfig(cfg) should store config")
	}

	// Mutating the original config should not affect the service's copy.
	cfg.Filename = "/tmp/y"
	if kvs.config.Filename != "/tmp/x" {
		t.Fatal("config was not cloned")
	}
}

func TestKVStoreService_ServiceName(t *testing.T) {
	kvs := New()
	want := "github.com/gailsapp/gails/plugins/kvstore"
	if got := kvs.ServiceName(); got != want {
		t.Fatalf("ServiceName: want %q, got %q", want, got)
	}
}

func TestConfigure(t *testing.T) {
	kvs := New()

	cfg := &Config{Filename: "/tmp/x"}
	kvs.Configure(cfg)
	if kvs.config.Filename != "/tmp/x" {
		t.Fatal("Configure did not set config")
	}
	if !kvs.unsaved {
		t.Fatal("Configure should mark store unsaved")
	}

	// Configure to nil makes the store in-memory.
	kvs.Configure(nil)
	if kvs.config != nil {
		t.Fatal("Configure(nil) should clear config")
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	// Missing file is not an error.
	kvs := NewWithConfig(&Config{Filename: path})
	if err := kvs.Load(); err != nil {
		t.Fatalf("Load missing file: %v", err)
	}

	// Empty file is not an error.
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}
	if err := kvs.Load(); err != nil {
		t.Fatalf("Load empty file: %v", err)
	}

	// Valid JSON file.
	if err := os.WriteFile(path, []byte(`{"foo":"bar"}`), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := kvs.Load(); err != nil {
		t.Fatalf("Load valid file: %v", err)
	}
	if got := kvs.Get("foo"); got != "bar" {
		t.Fatalf("expected foo=bar, got %v", got)
	}

	// Invalid JSON file.
	if err := os.WriteFile(path, []byte(`{not json`), 0644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}
	if err := kvs.Load(); err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	// In-memory Load is a no-op.
	mem := New()
	if err := mem.Load(); err != nil {
		t.Fatalf("Load in-memory: %v", err)
	}
}

func TestLoad_ReadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	// Create a directory at the path so reading fails with a non-NotExist error.
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	kvs := NewWithConfig(&Config{Filename: path})
	if err := kvs.Load(); err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	kvs := NewWithConfig(&Config{Filename: path})
	if err := kvs.Set("foo", "bar"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := kvs.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) == "" {
		t.Fatal("saved file is empty")
	}

	// In-memory Save is a no-op.
	mem := New()
	if err := mem.Save(); err != nil {
		t.Fatalf("Save in-memory: %v", err)
	}
}

func TestSave_Error(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readonly", "store.json")
	if err := os.Mkdir(filepath.Dir(path), 0555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(path), 0755) })

	kvs := NewWithConfig(&Config{Filename: path})
	if err := kvs.Set("foo", "bar"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := kvs.Save(); err == nil {
		t.Fatal("expected error saving to readonly dir")
	}
}

func TestServiceShutdown_Error(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readonly", "store.json")
	if err := os.Mkdir(filepath.Dir(path), 0555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(path), 0755) })

	kvs := NewWithConfig(&Config{Filename: path})
	if err := kvs.Set("foo", "bar"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := kvs.ServiceShutdown(); err == nil {
		t.Fatal("expected ServiceShutdown error")
	}
}

func TestSave_MarshalError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	kvs := NewWithConfig(&Config{Filename: path})
	if err := kvs.Set("bad", make(chan int)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := kvs.Save(); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestServiceStartupShutdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	if err := os.WriteFile(path, []byte(`{"foo":"bar"}`), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	kvs := NewWithConfig(&Config{Filename: path})
	if err := kvs.ServiceStartup(nil, application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup: %v", err)
	}
	if got := kvs.Get("foo"); got != "bar" {
		t.Fatalf("expected loaded value bar, got %v", got)
	}

	if err := kvs.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown: %v", err)
	}

	// Startup load error propagates.
	kvs = NewWithConfig(&Config{Filename: filepath.Join(dir, "notdir")})
	if err := os.Mkdir(filepath.Join(dir, "notdir"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := kvs.ServiceStartup(nil, application.ServiceOptions{}); err == nil {
		t.Fatal("expected ServiceStartup error")
	}
}

func TestGet(t *testing.T) {
	kvs := New()
	_ = kvs.Set("foo", "bar")
	_ = kvs.Set("num", 42)

	if got := kvs.Get("foo"); got != "bar" {
		t.Fatalf("Get foo: want bar, got %v", got)
	}
	if got := kvs.Get("missing"); got != nil {
		t.Fatalf("Get missing: want nil, got %v", got)
	}

	all := kvs.Get("").(map[string]any)
	if len(all) != 2 {
		t.Fatalf("Get empty key: want 2 entries, got %d", len(all))
	}
}

func TestSet(t *testing.T) {
	kvs := New()
	if err := kvs.Set("foo", "bar"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := kvs.Get("foo"); got != "bar" {
		t.Fatalf("Get foo after Set: want bar, got %v", got)
	}
}

func TestSet_AutoSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	kvs := NewWithConfig(&Config{Filename: path, AutoSave: true})
	if err := kvs.Set("foo", "bar"); err != nil {
		t.Fatalf("Set autosave: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) == "" {
		t.Fatal("autosave did not write file")
	}
}

func TestSet_AutoSaveError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readonly", "store.json")
	if err := os.Mkdir(filepath.Dir(path), 0555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(path), 0755) })

	kvs := NewWithConfig(&Config{Filename: path, AutoSave: true})
	if err := kvs.Set("foo", "bar"); err == nil {
		t.Fatal("expected autosave error")
	}
}

func TestDelete(t *testing.T) {
	kvs := New()
	_ = kvs.Set("foo", "bar")
	if err := kvs.Delete("foo"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := kvs.Get("foo"); got != nil {
		t.Fatalf("expected nil after Delete, got %v", got)
	}
}

func TestDelete_AutoSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	kvs := NewWithConfig(&Config{Filename: path, AutoSave: true})
	_ = kvs.Set("foo", "bar")
	if err := kvs.Delete("foo"); err != nil {
		t.Fatalf("Delete autosave: %v", err)
	}
	if got := kvs.Get("foo"); got != nil {
		t.Fatalf("expected nil after Delete, got %v", got)
	}
}

func TestClear(t *testing.T) {
	kvs := New()
	_ = kvs.Set("foo", "bar")
	_ = kvs.Set("baz", "qux")
	if err := kvs.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if got := kvs.Get("foo"); got != nil {
		t.Fatalf("expected nil after Clear, got %v", got)
	}
	if got := kvs.Get("").(map[string]any); len(got) != 0 {
		t.Fatalf("expected empty map after Clear, got %v", got)
	}
}

func TestClear_AutoSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	kvs := NewWithConfig(&Config{Filename: path, AutoSave: true})
	_ = kvs.Set("foo", "bar")
	if err := kvs.Clear(); err != nil {
		t.Fatalf("Clear autosave: %v", err)
	}
	if got := kvs.Get("").(map[string]any); len(got) != 0 {
		t.Fatalf("expected empty map after Clear, got %v", got)
	}
}
