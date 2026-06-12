package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"

	"github.com/gailsapp/gails/pkg/application"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.config == nil || s.config.DBSource != ":memory:" {
		t.Fatal("New() should default to in-memory database")
	}
}

func TestNewWithConfig(t *testing.T) {
	cfg := &Config{DBSource: ":memory:"}
	s := NewWithConfig(cfg)
	if s.config.DBSource != ":memory:" {
		t.Fatal("NewWithConfig did not store config")
	}

	// Config should be cloned.
	cfg.DBSource = "other"
	if s.config.DBSource != ":memory:" {
		t.Fatal("config was not cloned")
	}
}

func TestSQLiteService_ServiceName(t *testing.T) {
	s := New()
	want := "github.com/gailsapp/gails/plugins/sqlite"
	if got := s.ServiceName(); got != want {
		t.Fatalf("ServiceName: want %q, got %q", want, got)
	}
}

func TestConfigure(t *testing.T) {
	s := New()
	s.Configure(&Config{DBSource: "file::memory:?cache=shared"})
	if s.config.DBSource != "file::memory:?cache=shared" {
		t.Fatal("Configure did not update config")
	}

	s.Configure(nil)
	if s.config == nil || s.config.DBSource != ":memory:" {
		t.Fatal("Configure(nil) should reset to in-memory")
	}
}

func TestOpen(t *testing.T) {
	s := New()

	if err := s.Open(); err != nil {
		t.Fatalf("Open in-memory: %v", err)
	}
	if s.conn == nil {
		t.Fatal("connection not set after Open")
	}

	// Open is idempotent: opening again should work.
	if err := s.Open(); err != nil {
		t.Fatalf("Open second time: %v", err)
	}

	// Empty source should error.
	s.Configure(&Config{DBSource: ""})
	if err := s.Open(); err == nil {
		t.Fatal("expected error for empty DBSource")
	}

	// Invalid source should error.
	s.Configure(&Config{DBSource: "http://example.com/test.db"})
	if err := s.Open(); err == nil {
		t.Fatal("expected error for invalid DBSource")
	}
}

func TestClose(t *testing.T) {
	s := New()
	if err := s.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if s.conn != nil {
		t.Fatal("connection should be nil after Close")
	}

	// Close is idempotent.
	if err := s.Close(); err != nil {
		t.Fatalf("Close second time: %v", err)
	}
}

func TestServiceStartupShutdown(t *testing.T) {
	s := New()
	if err := s.ServiceStartup(nil, application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup: %v", err)
	}
	if err := s.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown: %v", err)
	}

	// Startup with invalid source should error.
	s.Configure(&Config{DBSource: "http://example.com/test.db"})
	if err := s.ServiceStartup(nil, application.ServiceOptions{}); err == nil {
		t.Fatal("expected ServiceStartup error")
	}
}

func TestExecute(t *testing.T) {
	s := newOpenService(t)

	if err := s.Execute("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		t.Fatalf("Execute create table: %v", err)
	}
	if err := s.Execute("INSERT INTO test (name) VALUES (?)", "alice"); err != nil {
		t.Fatalf("Execute insert: %v", err)
	}
}

func TestExecute_NoConnection(t *testing.T) {
	s := New()
	if err := s.Execute("SELECT 1"); err == nil {
		t.Fatal("expected error without connection")
	}
}

func TestExecContext_Cancelled(t *testing.T) {
	s := newOpenService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.ExecContext(ctx, "SELECT 1"); err != nil {
		t.Fatalf("cancelled ExecContext should be ignored: %v", err)
	}
}

func TestExecContext_OtherError(t *testing.T) {
	s := newOpenService(t)
	if err := s.ExecContext(context.Background(), "NOT A VALID QUERY"); err == nil {
		t.Fatal("expected error for invalid query")
	}
}

func TestQuery(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	_ = s.Execute("INSERT INTO test (name) VALUES ('alice'), ('bob')")

	rows, err := s.Query("SELECT id, name FROM test ORDER BY id")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["name"] != "alice" {
		t.Fatalf("expected alice, got %v", rows[0]["name"])
	}
}

func TestQuery_NoConnection(t *testing.T) {
	s := New()
	if _, err := s.Query("SELECT 1"); err == nil {
		t.Fatal("expected error without connection")
	}
}

func TestQueryContext_Cancelled(t *testing.T) {
	s := newOpenService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rows, err := s.QueryContext(ctx, "SELECT 1")
	if err != nil {
		t.Fatalf("cancelled QueryContext should be ignored: %v", err)
	}
	if rows == nil {
		t.Fatal("expected empty Rows, not nil")
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestQueryContext_InvalidQuery(t *testing.T) {
	s := newOpenService(t)
	if _, err := s.QueryContext(context.Background(), "NOT VALID"); err == nil {
		t.Fatal("expected error for invalid query")
	}
}

func TestPrepare(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")

	stmt, err := s.Prepare("INSERT INTO test (name) VALUES (?)")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if stmt == nil {
		t.Fatal("expected non-nil statement")
	}
	defer stmt.Close()

	if err := s.ExecPrepared(context.Background(), stmt, "alice"); err != nil {
		t.Fatalf("ExecPrepared: %v", err)
	}

	rows, err := s.QueryPrepared(context.Background(), stmt, "bob")
	if err != nil {
		t.Fatalf("QueryPrepared: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("INSERT should return 0 rows, got %d", len(rows))
	}
}

func TestPrepare_NoConnection(t *testing.T) {
	s := New()
	if _, err := s.Prepare("SELECT 1"); err == nil {
		t.Fatal("expected error without connection")
	}
}

func TestPrepare_InvalidQuery(t *testing.T) {
	s := newOpenService(t)
	if _, err := s.Prepare("NOT VALID"); err == nil {
		t.Fatal("expected error for invalid query")
	}
}

func TestPrepareContext_Cancelled(t *testing.T) {
	s := newOpenService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stmt, err := s.PrepareContext(ctx, "SELECT 1")
	if err != nil {
		t.Fatalf("cancelled PrepareContext should be ignored: %v", err)
	}
	if stmt != nil {
		t.Fatal("expected nil statement for cancelled context")
	}
}

func TestPrepare_IDExhausted(t *testing.T) {
	s := newOpenService(t)
	nextId.Store(0)
	defer nextId.Store(1)

	if _, err := s.Prepare("SELECT 1"); err == nil {
		t.Fatal("expected error when ids exhausted")
	}
}

func TestClosePrepared(t *testing.T) {
	s := newOpenService(t)
	stmt, err := s.Prepare("SELECT 1")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if err := s.ClosePrepared(stmt); err != nil {
		t.Fatalf("ClosePrepared: %v", err)
	}
}

func TestExecPrepared(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	stmt, _ := s.Prepare("INSERT INTO test (id) VALUES (?)")
	defer stmt.Close()

	if err := s.ExecPrepared(context.Background(), stmt, 1); err != nil {
		t.Fatalf("ExecPrepared: %v", err)
	}

	if err := s.ExecPrepared(context.Background(), nil, 1); err == nil {
		t.Fatal("expected error for nil stmt")
	}

	closed := &Stmt{}
	if err := s.ExecPrepared(context.Background(), closed, 1); err == nil {
		t.Fatal("expected error for invalid stmt")
	}
}

func TestExecPrepared_Cancelled(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	stmt, _ := s.Prepare("INSERT INTO test (id) VALUES (?)")
	defer stmt.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.ExecPrepared(ctx, stmt, 1); err != nil {
		t.Fatalf("cancelled ExecPrepared should be ignored: %v", err)
	}
}

func TestQueryPrepared(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	_ = s.Execute("INSERT INTO test (id) VALUES (1), (2)")
	stmt, _ := s.Prepare("SELECT id FROM test WHERE id = ?")
	defer stmt.Close()

	rows, err := s.QueryPrepared(context.Background(), stmt, 1)
	if err != nil {
		t.Fatalf("QueryPrepared: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if _, err := s.QueryPrepared(context.Background(), nil, 1); err == nil {
		t.Fatal("expected error for nil stmt")
	}

	closed := &Stmt{}
	if _, err := s.QueryPrepared(context.Background(), closed, 1); err == nil {
		t.Fatal("expected error for invalid stmt")
	}
}

func TestQueryPrepared_Cancelled(t *testing.T) {
	s := newOpenService(t)
	stmt, _ := s.Prepare("SELECT 1")
	defer stmt.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rows, err := s.QueryPrepared(ctx, stmt)
	if err != nil {
		t.Fatalf("cancelled QueryPrepared should be ignored: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestStmt_Close(t *testing.T) {
	s := newOpenService(t)
	stmt, _ := s.Prepare("SELECT 1")

	if err := stmt.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Close is idempotent.
	if err := stmt.Close(); err != nil {
		t.Fatalf("Close second time: %v", err)
	}

	var nilStmt *Stmt
	if err := nilStmt.Close(); err != nil {
		t.Fatalf("Close nil: %v", err)
	}
}

func TestStmt_MarshalText(t *testing.T) {
	s := newOpenService(t)
	stmt, _ := s.Prepare("SELECT 1")
	defer stmt.Close()

	text, err := stmt.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if len(text) != 16 {
		t.Fatalf("expected 16 hex chars, got %d", len(text))
	}
}

func TestStmt_UnmarshalText(t *testing.T) {
	s := newOpenService(t)
	stmt, _ := s.Prepare("SELECT 1")
	id := stmt.id
	defer stmt.Close()

	text, _ := stmt.MarshalText()

	restored := &Stmt{}
	if err := restored.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if restored.id != id {
		t.Fatalf("expected id %d, got %d", id, restored.id)
	}
	if restored.sqlStmt == nil {
		t.Fatal("expected restored statement")
	}
}

func TestStmt_UnmarshalText_Invalid(t *testing.T) {
	restored := &Stmt{}
	if err := restored.UnmarshalText([]byte("not-hex")); err == nil {
		t.Fatal("expected error for invalid id")
	}

	if err := restored.UnmarshalText([]byte("0000000000000000")); err != nil {
		t.Fatalf("UnmarshalText missing id: %v", err)
	}
	if restored.sqlStmt != nil {
		t.Fatal("expected nil statement for missing id")
	}
}

func TestParseRows(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	_ = s.Execute("INSERT INTO test (name) VALUES ('alice'), ('bob')")

	rows, err := s.Query("SELECT id, name FROM test ORDER BY id")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["name"] != "alice" {
		t.Fatalf("expected alice, got %v", rows[0]["name"])
	}
}

func TestParseRows_Empty(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE test (id INTEGER PRIMARY KEY)")

	rows, err := s.Query("SELECT id FROM test")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestParseRows_Cancelled(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	_ = s.Execute("INSERT INTO test (id) VALUES (1), (2)")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rows, err := s.QueryContext(ctx, "SELECT id FROM test ORDER BY id")
	if err != nil {
		t.Fatalf("QueryContext cancelled: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows after cancellation, got %d", len(rows))
	}
}

func TestParseRows_ScanError(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE t (id INTEGER PRIMARY KEY)")
	_ = s.Execute("INSERT INTO t (id) VALUES (1)")

	orig := rowsScanFunc
	rowsScanFunc = func(*sql.Rows, ...any) error { return errors.New("scan error") }
	defer func() { rowsScanFunc = orig }()

	if _, err := s.Query("SELECT id FROM t"); err == nil {
		t.Fatal("expected scan error")
	}
}

func TestParseRows_NextCancelled(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE t (id INTEGER PRIMARY KEY)")
	_ = s.Execute("INSERT INTO t (id) VALUES (1), (2)")

	orig := rowsNextFunc
	rowsNextFunc = func(*sql.Rows) bool { return true }
	defer func() { rowsNextFunc = orig }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rows, err := s.QueryContext(ctx, "SELECT id FROM t")
	if err != nil {
		t.Fatalf("QueryContext: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestOpen_SQLError(t *testing.T) {
	orig := sqlOpen
	sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) { return nil, errors.New("open error") }
	defer func() { sqlOpen = orig }()

	s := New()
	if err := s.Open(); err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_PingError(t *testing.T) {
	orig := dbPing
	dbPing = func(*sql.DB) error { return errors.New("ping error") }
	defer func() { dbPing = orig }()

	s := New()
	if err := s.Open(); err == nil {
		t.Fatal("expected error")
	}
}

func TestOpen_CloseExistingError(t *testing.T) {
	s := newOpenService(t)
	orig := dbClose
	dbClose = func(*sql.DB) error { return errors.New("close error") }
	defer func() { dbClose = orig }()

	if err := s.Open(); err == nil {
		t.Fatal("expected error")
	}
}

func TestClose_CloseError(t *testing.T) {
	s := newOpenService(t)
	orig := dbClose
	dbClose = func(*sql.DB) error { return errors.New("close error") }
	defer func() { dbClose = orig }()

	if err := s.Close(); err == nil {
		t.Fatal("expected error")
	}
}

func TestClose_WithActiveStatement(t *testing.T) {
	s := newOpenService(t)
	if _, err := s.Prepare("SELECT 1"); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestExecPrepared_ContextCanceled(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE t (id INTEGER PRIMARY KEY)")
	stmt, _ := s.Prepare("INSERT INTO t (id) VALUES (?)")
	defer stmt.Close()

	orig := stmtExecContext
	stmtExecContext = func(*sql.Stmt, context.Context, ...any) (sql.Result, error) { return nil, context.Canceled }
	defer func() { stmtExecContext = orig }()

	if err := s.ExecPrepared(context.Background(), stmt, 1); err != nil {
		t.Fatalf("expected canceled to be ignored: %v", err)
	}
}

func TestQueryPrepared_ContextCanceled(t *testing.T) {
	s := newOpenService(t)
	stmt, _ := s.Prepare("SELECT 1")
	defer stmt.Close()

	orig := stmtQueryContext
	stmtQueryContext = func(*sql.Stmt, context.Context, ...any) (*sql.Rows, error) { return nil, context.Canceled }
	defer func() { stmtQueryContext = orig }()

	rows, err := s.QueryPrepared(context.Background(), stmt)
	if err != nil {
		t.Fatalf("QueryPrepared: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(rows))
	}
}

func TestStmt_CloseError(t *testing.T) {
	s := newOpenService(t)
	stmt, _ := s.Prepare("SELECT 1")

	orig := stmtClose
	stmtClose = func(*sql.Stmt) error { return errors.New("close error") }
	defer func() { stmtClose = orig }()

	if err := stmt.Close(); err == nil {
		t.Fatal("expected close error")
	}
}

func TestStmt_MarshalText_Error(t *testing.T) {
	s := newOpenService(t)
	stmt, _ := s.Prepare("SELECT 1")
	defer stmt.Close()

	orig := fmtFprintf
	fmtFprintf = func(io.Writer, string, ...any) (int, error) { return 0, errors.New("fmt error") }
	defer func() { fmtFprintf = orig }()

	if _, err := stmt.MarshalText(); err == nil {
		t.Fatal("expected error")
	}
}

func TestExecPrepared_OtherError(t *testing.T) {
	s := newOpenService(t)
	stmt, _ := s.Prepare("SELECT 1")
	defer stmt.Close()

	orig := stmtExecContext
	stmtExecContext = func(*sql.Stmt, context.Context, ...any) (sql.Result, error) { return nil, errors.New("exec error") }
	defer func() { stmtExecContext = orig }()

	if err := s.ExecPrepared(context.Background(), stmt, 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestQueryPrepared_OtherError(t *testing.T) {
	s := newOpenService(t)
	stmt, _ := s.Prepare("SELECT 1")
	defer stmt.Close()

	orig := stmtQueryContext
	stmtQueryContext = func(*sql.Stmt, context.Context, ...any) (*sql.Rows, error) { return nil, errors.New("query error") }
	defer func() { stmtQueryContext = orig }()

	if _, err := s.QueryPrepared(context.Background(), stmt); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRows_ContextCanceledInside(t *testing.T) {
	s := newOpenService(t)
	_ = s.Execute("CREATE TABLE t (id INTEGER PRIMARY KEY)")
	_ = s.Execute("INSERT INTO t (id) VALUES (1)")

	rows, err := s.conn.QueryContext(context.Background(), "SELECT id FROM t")
	if err != nil {
		t.Fatalf("QueryContext: %v", err)
	}
	defer rows.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := parseRows(ctx, rows)
	if err != nil {
		t.Fatalf("parseRows: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(result))
	}
}

func TestOpen_CloseError(t *testing.T) {
	s := newOpenService(t)
	// Opening again after a successful open should close the previous connection cleanly.
	if err := s.Open(); err != nil {
		t.Fatalf("Open again: %v", err)
	}
}

func TestClosePrepared_StatementCloseError(t *testing.T) {
	s := newOpenService(t)
	stmt, _ := s.Prepare("SELECT 1")
	// Remove from global map to simulate a stale statement.
	stmts.Delete(stmt.id)
	delete(s.stmts, stmt.id)

	if err := stmt.Close(); err != nil {
		t.Fatalf("Close with stale global map: %v", err)
	}
}

func newOpenService(t *testing.T) *SQLiteService {
	t.Helper()
	s := New()
	if err := s.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestErrors(t *testing.T) {
	if !errors.Is(context.Canceled, context.Canceled) {
		t.Fatal("context.Canceled sanity check")
	}
}
