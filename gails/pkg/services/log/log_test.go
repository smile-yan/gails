package log

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func newTestLogger(t *testing.T) (*LogService, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	l := NewWithConfig(&Config{Logger: logger, LogLevel: slog.LevelDebug})
	return l, buf
}

func TestNew(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("New() returned nil")
	}
	if l.config.Load() == nil {
		t.Fatal("config not initialized")
	}
}

func TestNewWithConfig(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := &Config{Logger: logger, LogLevel: slog.LevelWarn}

	l := NewWithConfig(cfg)
	if l.config.Load().Logger != logger {
		t.Fatal("logger not stored")
	}

	// Config should be cloned: mutating the original should not affect the service.
	cfg.LogLevel = slog.LevelError
	if l.config.Load().LogLevel != slog.LevelWarn {
		t.Fatal("config clone was not deep enough")
	}
}

func TestLogService_ServiceName(t *testing.T) {
	l := New()
	want := "github.com/gailsapp/gails/plugins/log"
	if got := l.ServiceName(); got != want {
		t.Fatalf("ServiceName: want %q, got %q", want, got)
	}
}

func TestConfigure(t *testing.T) {
	l := New()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	l.Configure(&Config{Logger: logger, LogLevel: slog.LevelInfo})

	if l.config.Load().Logger != logger {
		t.Fatal("Configure did not set logger")
	}
	if l.Level() != slog.LevelInfo {
		t.Fatalf("Level: want Info, got %v", l.Level())
	}

	// Configure(nil) should fall back to defaults.
	l.Configure(nil)
	if l.config.Load() == nil {
		t.Fatal("Configure(nil) should produce a default config")
	}
}

func TestLevel(t *testing.T) {
	l, _ := newTestLogger(t)
	if got := l.Level(); got != slog.LevelDebug {
		t.Fatalf("Level: want Debug, got %v", got)
	}
	if got := l.LogLevel(); got != Debug {
		t.Fatalf("LogLevel: want Debug, got %v", got)
	}
}

func TestSetLogLevel(t *testing.T) {
	l, _ := newTestLogger(t)
	l.SetLogLevel(Level(slog.LevelError))
	if l.Level() != slog.LevelError {
		t.Fatalf("SetLogLevel did not update level")
	}
}

func TestLog(t *testing.T) {
	l, buf := newTestLogger(t)
	ctx := context.WithValue(context.Background(), struct{}{}, "v")
	l.Log(ctx, Info, "hello", "key", "value")

	if !strings.Contains(buf.String(), "hello") {
		t.Fatalf("expected log output to contain 'hello', got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "key=value") {
		t.Fatalf("expected log output to contain 'key=value', got %q", buf.String())
	}
}

func TestDebug(t *testing.T) {
	l, buf := newTestLogger(t)
	l.Debug("debug msg", "k", "v")
	if !strings.Contains(buf.String(), "debug msg") {
		t.Fatalf("expected 'debug msg', got %q", buf.String())
	}
}

func TestInfo(t *testing.T) {
	l, buf := newTestLogger(t)
	l.Info("info msg", "k", "v")
	if !strings.Contains(buf.String(), "info msg") {
		t.Fatalf("expected 'info msg', got %q", buf.String())
	}
}

func TestWarning(t *testing.T) {
	l, buf := newTestLogger(t)
	l.Warning("warn msg", "k", "v")
	if !strings.Contains(buf.String(), "warn msg") {
		t.Fatalf("expected 'warn msg', got %q", buf.String())
	}
}

func TestError(t *testing.T) {
	l, buf := newTestLogger(t)
	l.Error("error msg", "k", "v")
	if !strings.Contains(buf.String(), "error msg") {
		t.Fatalf("expected 'error msg', got %q", buf.String())
	}
}

func TestDebugContext(t *testing.T) {
	l, buf := newTestLogger(t)
	ctx := context.WithValue(context.Background(), struct{}{}, "v")
	l.DebugContext(ctx, "debug ctx", "k", "v")
	if !strings.Contains(buf.String(), "debug ctx") {
		t.Fatalf("expected 'debug ctx', got %q", buf.String())
	}
}

func TestInfoContext(t *testing.T) {
	l, buf := newTestLogger(t)
	ctx := context.WithValue(context.Background(), struct{}{}, "v")
	l.InfoContext(ctx, "info ctx", "k", "v")
	if !strings.Contains(buf.String(), "info ctx") {
		t.Fatalf("expected 'info ctx', got %q", buf.String())
	}
}

func TestWarningContext(t *testing.T) {
	l, buf := newTestLogger(t)
	ctx := context.WithValue(context.Background(), struct{}{}, "v")
	l.WarningContext(ctx, "warn ctx", "k", "v")
	if !strings.Contains(buf.String(), "warn ctx") {
		t.Fatalf("expected 'warn ctx', got %q", buf.String())
	}
}

func TestErrorContext(t *testing.T) {
	l, buf := newTestLogger(t)
	ctx := context.WithValue(context.Background(), struct{}{}, "v")
	l.ErrorContext(ctx, "error ctx", "k", "v")
	if !strings.Contains(buf.String(), "error ctx") {
		t.Fatalf("expected 'error ctx', got %q", buf.String())
	}
}

func TestLevelerInterface(t *testing.T) {
	l := New()
	handler := slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: l})
	logger := slog.New(handler)
	l.Configure(&Config{Logger: logger, LogLevel: slog.LevelInfo})

	l.SetLogLevel(Level(slog.LevelError))
	if l.Level() != slog.LevelError {
		t.Fatal("leveler did not reflect updated level")
	}
}
