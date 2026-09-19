package api

// Test database fixtures. Every integration test in this package runs against
// a real PostgreSQL database (biletflow_test), created and given the db/init
// schema on first use so the development data is never touched.

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
	testPoolOnce sync.Once
	sharedPool   *pgxpool.Pool
	testPoolErr  error
)

// adminURL is the maintenance database used to CREATE DATABASE. Override it
// with TEST_ADMIN_DATABASE_URL.
func adminURL() string {
	if v := os.Getenv("TEST_ADMIN_DATABASE_URL"); v != "" {
		return v
	}
	return defaultAdminURL
}

func testDatabaseURL() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return strings.Replace(adminURL(), "/postgres?", "/"+testDBName+"?", 1)
}

// testPool returns a pool on the schema'd test database. The test fails rather
// than skips when the database is down: a suite that quietly skips its
// integration coverage is worse than one that says the database is not running.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	testPoolOnce.Do(func() { sharedPool, testPoolErr = openTestPool() })
	if testPoolErr != nil {
		t.Fatalf("test database unavailable: %v\n\nStart it with: make up", testPoolErr)
	}
	return sharedPool
}

func openTestPool() (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, adminURL())
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", adminURL(), err)
	}
	var exists bool
	if err := admin.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, testDBName).Scan(&exists); err != nil {
		_ = admin.Close(ctx)
		return nil, err
	}
	if !exists {
		if _, err := admin.Exec(ctx, `CREATE DATABASE `+pgx.Identifier{testDBName}.Sanitize()); err != nil {
			_ = admin.Close(ctx)
			return nil, fmt.Errorf("create %s: %w", testDBName, err)
		}
	}
	_ = admin.Close(ctx)

	if err := applySchema(ctx); err != nil {
		return nil, err
	}
	return pgxpool.New(ctx, testDatabaseURL())
}

// applySchema runs every db/init script. They are idempotent.
func applySchema(ctx context.Context) error {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "db", "init")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read %s: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	// Multi-statement scripts with dollar quoting need the simple protocol.
	cfg, err := pgx.ParseConfig(testDatabaseURL())
	if err != nil {
		return err
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return err
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

// resetDB empties every table before and after a test.
func resetDB(t *testing.T, p *pgxpool.Pool) {
	t.Helper()
	truncateAll(t, p)
	t.Cleanup(func() { truncateAll(t, p) })
}

func truncateAll(t *testing.T, p *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := p.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname = 'public'`)
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
	// TRUNCATE bypasses row triggers, so it also clears append-only tables.
	if _, err := p.Exec(ctx, "TRUNCATE TABLE "+strings.Join(names, ", ")+" RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
