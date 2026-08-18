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

// The structural reason this second switch existed is gone: the store layer now
// addresses rows by tenant_id, and the whole store suite passes against
// PostgreSQL. It is kept as a separate switch rather than removed because
// turning it on changes what CI runs, and that is a deliberate decision with a
// cost (every store test then needs a database) rather than a side effect of a
// code change.
//
// It is deliberately a switch and not a skip: a test that always skips reports a
// pass it never earned. Setting HAUSV_STORE_TEST_POSTGRES alongside
// HAUSV_TEST_POSTGRES_DSN is what closes HAUSV-555's last acceptance criterion.
const storeOptIn = "HAUSV_STORE_TEST_POSTGRES"

// Backend reports which engine Open will use, for tests that need to skip a
// case that is genuinely engine-specific rather than a portability defect.
func Backend() db.Backend {
	if postgresDSN() != "" {
		return db.BackendPostgres
	}
	return db.BackendSQLite
}

// postgresDSN returns the DSN only when the store suite has been asked to run on
// PostgreSQL. Asking for it without a DSN is a configuration error, not a
// silent fallback to SQLite — that would report a green PostgreSQL run that
// never touched PostgreSQL.
func postgresDSN() string {
	if strings.TrimSpace(os.Getenv(storeOptIn)) == "" {
		return ""
	}
	dsn := strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN"))
	if dsn == "" {
		panic(storeOptIn + " is set but HAUSV_TEST_POSTGRES_DSN is empty")
	}
	return dsn
}

// Open returns a migrated, empty database. On PostgreSQL each call gets its own
// schema, dropped on cleanup, so tests stay independent and can run in parallel.
func Open(t *testing.T) *sql.DB {
	t.Helper()
	database, _ := OpenWithConfig(t)
	return database
}

// OpenWithConfig returns the same database Open does, together with the exact
// db.Config it was opened from — on PostgreSQL that config carries the DSN of
// this test's isolated schema.
//
// It exists because a store now takes a *store.TenantDB instead of a pool, and
// building one means building a db.Scoped, which needs the DSN: every lane it
// hands out is a NEW pool dialled from that DSN, with the scope in the startup
// packet. Handing tests the plain pool instead would let them compile while
// never opening a lane, so the conversion this supports would be asserted by
// nothing on the one engine that has RLS. The isolated schema rides along in
// the DSN's search_path, so lanes land in the same schema the fixtures write.
func OpenWithConfig(t *testing.T) (*sql.DB, db.Config) {
	t.Helper()
	baseDSN := postgresDSN()
	if baseDSN == "" {
		database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
		if err != nil {
			t.Fatalf("open sqlite test database: %v", err)
		}
		t.Cleanup(func() { _ = database.Close() })
		return database, db.Config{Backend: db.BackendSQLite}
	}

	// A lane is a pool of its own, so the per-test connection plan is
	// MaxOpenConns + LaneCap*LaneMaxConns. Both lane numbers are kept small
	// deliberately: `go test ./...` runs packages concurrently, and the
	// disposable test server runs the stock max_connections.
	cfg := db.Config{
		Backend:          db.BackendPostgres,
		DSN:              isolatedSchema(t, baseDSN),
		ConnectTimeout:   3 * time.Second,
		StatementTimeout: 15 * time.Second,
		MaxOpenConns:     2,
		MaxIdleConns:     2,
		ConnMaxLifetime:  time.Minute,
		ConnMaxIdleTime:  time.Minute,
		LaneCap:          4,
		LaneMaxConns:     2,
	}
	database, err := db.OpenConfig(t.Context(), cfg)
	if err != nil {
		// Never include the DSN: it carries the role password.
		t.Fatalf("open postgres test database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database, cfg
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
