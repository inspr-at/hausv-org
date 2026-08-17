// Package dbtest opens a database for tests against whichever backend is
// configured, so the same store tests can prove themselves on both engines.
//
// Until now every store test opened SQLite directly. That made "the stores work
// on PostgreSQL" an untested claim: the SQL looked portable, nothing checked it.
// With HAUSV_TEST_POSTGRES_DSN set, the same tests run against PostgreSQL in an
// isolated schema; without it they behave exactly as before.
package dbtest

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/inspr-at/hausv-org/internal/db"
)

// Backend reports which engine Open will use, for tests that need to skip a
// case that is genuinely engine-specific rather than a portability defect.
func Backend() db.Backend {
	if strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN")) != "" {
		return db.BackendPostgres
	}
	return db.BackendSQLite
}

// Open returns a migrated, empty database. On PostgreSQL each call gets its own
// schema, dropped on cleanup, so tests stay independent and can run in parallel.
func Open(t *testing.T) *sql.DB {
	t.Helper()
	baseDSN := strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
		if err != nil {
			t.Fatalf("open sqlite test database: %v", err)
		}
		t.Cleanup(func() { _ = database.Close() })
		return database
	}

	dsn := isolatedSchema(t, baseDSN)
	database, err := db.OpenConfig(t.Context(), db.Config{
		Backend:          db.BackendPostgres,
		DSN:              dsn,
		ConnectTimeout:   3 * time.Second,
		StatementTimeout: 15 * time.Second,
		MaxOpenConns:     2,
		MaxIdleConns:     2,
		ConnMaxLifetime:  time.Minute,
		ConnMaxIdleTime:  time.Minute,
	})
	if err != nil {
		// Never include the DSN: it carries the role password.
		t.Fatalf("open postgres test database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func isolatedSchema(t *testing.T, dsn string) string {
	t.Helper()
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse test postgres DSN: %v", err)
	}
	admin := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { _ = admin.Close() })
	// Nanoseconds alone collide when two tests start in the same tick, which is
	// ordinary under -parallel; the test name keeps them apart.
	schema := fmt.Sprintf("hausv_t_%d_%s", time.Now().UnixNano(), sanitise(t.Name()))
	if len(schema) > 63 {
		schema = schema[:63]
	}
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatalf("create isolated test schema: %v", err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`) })

	if parsed, err := url.Parse(dsn); err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
		query := parsed.Query()
		query.Set("search_path", schema)
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}
	return dsn + " search_path=" + schema
}

func sanitise(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
