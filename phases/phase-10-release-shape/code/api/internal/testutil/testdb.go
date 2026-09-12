// Package testutil provides a real PostgreSQL database for the integration
// tests. Nothing here is used by production code.
//
// The tests run against a dedicated database (biletflow_test by default) that
// is created and populated from db/init on first use, so they can never touch
// the development data.
package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultAdminURL = "postgres://biletflow:biletflow_dev_password@localhost:5433/postgres?sslmode=disable"
	testDBName      = "biletflow_test"
)

var (
	setupOnce sync.Once
	pool      *pgxpool.Pool
	setupErr  error
)

// adminURL is the connection string for the maintenance database used to
// CREATE DATABASE. Override it with TEST_ADMIN_DATABASE_URL.
func adminURL() string {
	if v := os.Getenv("TEST_ADMIN_DATABASE_URL"); v != "" {
		return v
	}
	return defaultAdminURL
}

// TestDatabaseURL is the connection string for the test database itself.
func TestDatabaseURL() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return strings.Replace(adminURL(), "/postgres?", "/"+testDBName+"?", 1)
}

// Pool returns a pooled connection to a freshly schema'd test database. The
// database is created and the Phase 1 schema applied once per test binary.
//
// The test fails - it does not skip - when the database is unreachable: a test
// suite that quietly skips its integration coverage is worse than one that
// stops and says the database is not running.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	setupOnce.Do(func() { pool, setupErr = setup() })
	if setupErr != nil {
		t.Fatalf("test database unavailable: %v\n\nStart it with: make up   (from the repository root)", setupErr)
	}
	return pool
}

func setup() (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := ensureDatabase(ctx); err != nil {
		return nil, err
	}

	p, err := pgxpool.New(ctx, TestDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("connect to test database: %w", err)
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("ping test database: %w", err)
	}

	if err := applySchema(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

// ensureDatabase creates the test database when it does not exist yet.
func ensureDatabase(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, adminURL())
	if err != nil {
		return fmt.Errorf("connect to %s: %w", adminURL(), err)
	}
	defer func() { _ = conn.Close(ctx) }()

	var exists bool
	if err := conn.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, testDBName).Scan(&exists); err != nil {
		return fmt.Errorf("check for %s: %w", testDBName, err)
	}
	if exists {
		return nil
	}

	// CREATE DATABASE cannot run inside a transaction or take a placeholder.
	if _, err := conn.Exec(ctx, `CREATE DATABASE `+pgx.Identifier{testDBName}.Sanitize()); err != nil {
		return fmt.Errorf("create %s: %w", testDBName, err)
	}
	return nil
}

// applySchema runs every script in db/init against the test database. The
// scripts are idempotent, so this is safe on an already-populated database.
func applySchema(ctx context.Context) error {
	dir := filepath.Join(RepoRoot(), "db", "init")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read %s: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no .sql files found in %s", dir)
	}

	// The schema files contain multiple statements and dollar-quoted blocks,
	// which need the simple protocol rather than prepared statements.
	cfg, err := pgx.ParseConfig(TestDatabaseURL())
	if err != nil {
		return err
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect for schema load: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	for _, name := range files {
		sqlBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if _, err := conn.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
	}
	return nil
}

// RepoRoot walks up from this source file until it finds the directory holding
// db/init, so tests work no matter which package directory they run from.
func RepoRoot() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("testutil: cannot determine source location")
	}

	dir := filepath.Dir(thisFile)
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "db", "init")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	panic("testutil: could not locate the repository root (no db/init directory found)")
}

// Reset empties every table so each test starts from a known state. It is
// registered as a cleanup, so a failing test still leaves the database clean.
func Reset(t *testing.T, p *pgxpool.Pool) {
	t.Helper()
	truncateAll(t, p)
	t.Cleanup(func() { truncateAll(t, p) })
}

func truncateAll(t *testing.T, p *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := p.Query(ctx,
		`SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatalf("scan table name: %v", err)
		}
		names = append(names, pgx.Identifier{name}.Sanitize())
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("list tables: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("the test database has no tables - was db/init applied?")
	}

	// TRUNCATE bypasses row-level triggers, so it also clears the append-only
	// audit_logs table that rejects DELETE.
	stmt := "TRUNCATE TABLE " + strings.Join(names, ", ") + " RESTART IDENTITY CASCADE"
	if _, err := p.Exec(ctx, stmt); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	restoreDefaults(t, ctx, p)
}

// restoreDefaults puts back the rows that db/init seeds as part of the schema
// rather than as test data.
//
// platform_settings holds configuration, not records: an empty table is a
// state the running application can never be in, so leaving it truncated would
// make every test run against a platform with no activation fee. Restoring it
// here keeps each test isolated - a test that changes a setting does not leak
// into the next - while still starting from a valid platform.
func restoreDefaults(t *testing.T, ctx context.Context, p *pgxpool.Pool) {
	t.Helper()
	if _, err := p.Exec(ctx, `
		INSERT INTO platform_settings (key, value, description)
		VALUES ('activation_fee_kzt', '"5000.00"'::jsonb,
		        'One-time paid-sales activation fee per event, in KZT (SRS 3.3).')
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
		t.Fatalf("restore platform settings: %v", err)
	}
}
