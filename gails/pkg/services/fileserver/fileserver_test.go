package fileserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 before configure, got %d", rec.Code)
	}
}

func TestNewWithConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("world"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	s := NewWithConfig(&Config{RootPath: dir})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hello.txt", nil)
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "world" {
		t.Fatalf("expected body 'world', got %q", got)
	}
}

func TestNewWithConfig_Nil(t *testing.T) {
	s := NewWithConfig(nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatal("expected error body")
	}
}

func TestConfigure(t *testing.T) {
	s := New()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("A"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	s.Configure(&Config{RootPath: dir})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/a.txt", nil)
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Reconfigure to nil should restore 503 behaviour.
	s.Configure(nil)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 after reconfigure, got %d", rec.Code)
	}
}

func TestFileserverService_ServiceName(t *testing.T) {
	s := New()
	want := "github.com/gailsapp/gails/services/fileserver"
	if got := s.ServiceName(); got != want {
		t.Fatalf("ServiceName: want %q, got %q", want, got)
	}
}

func TestFileserverService_ServeHTTP_HandlesNilRequest(t *testing.T) {
	s := New()
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for nil request, got %d", rec.Code)
	}
}

func TestFileserverService_ServeHTTP_CustomHandler(t *testing.T) {
	s := New()
	s.Configure(&Config{RootPath: t.TempDir()})

	// Overwrite the internal handler with a custom one to exercise the pointer path.
	custom := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "custom")
	}))
	s.fs.Store(&custom)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	s.ServeHTTP(rec, req)

	if got := rec.Body.String(); got != "custom" {
		t.Fatalf("expected custom handler response, got %q", got)
	}
}
