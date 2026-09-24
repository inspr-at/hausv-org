// Package dbtest opens a database for tests against PostgreSQL when CI (or a
// developer) sets HAUSV_STORE_TEST_POSTGRES and HAUSV_TEST_POSTGRES_DSN.
// Without those variables Open still uses an in-process SQLite file so `go test`
// on a laptop without Docker can run; that path is a test helper, not a product
// backend (HAUSV-757).
package dbtest

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// MaintenanceView returns the *sql.DB a package's fixtures and assertions read
// and write through: the maintenance lane, taken from the same seam the stores
// under test hold.
//
// Until PostgreSQL migration 0006 that role was played by the process pool. An
// unscoped session saw everything, so a fixture could seed any row and an
// assertion could count any table without going through the lane it was
// checking. 0006 made the unscoped session see NOTHING and write NOTHING — that
// is the whole point of it — so the pool can no longer play that role on the
// one engine that has row-level security. The maintenance lane can: it declares
// hausv.cross_tenant=on in its startup packet, the policy lets it see across
// every tenant, and it is still deliberately NOT the tenant lane a store writes
// on, so an assertion that reads through it can still tell "the row is scoped
// correctly" from "the lane hid a row that is scoped wrong".
//
// On SQLite there are no lanes and no policy; every accessor of the seam is the
// one process pool, and that is what comes back.
//
// Fixtures need a concrete pool for migration helpers as well as context
// methods. Reserve an explicit maintenance lease until test cleanup; ordinary
// store handles acquire and release their own leases per operation.
func MaintenanceView(t *testing.T, scoped *db.Scoped) *sql.DB {
	t.Helper()
	view, release, err := db.LeasePool(t.Context(), scoped.Unscoped("test fixtures and assertions read and write outside every tenant lane"))
	if err != nil {
		t.Fatalf("lease maintenance fixture pool: %v", err)
	}
	t.Cleanup(release)
	return view
}

// RollbackWindowClosed reports whether this engine refuses the row the
// rollback-window tests reproduce: a tenant-bound row carrying a tenant_slug and
// NO tenant_id — the shape the previous release wrote, and the shape the
// boot-time backfill and the HealOrphanReason upserts exist to repair.
//
// It measures instead of reading configuration. It attempts exactly that
// insert, inside a transaction it rolls back, and demands the engine's
// contractual answer:
//
//   - SQLite must ACCEPT it and this returns false. Production runs there, the
//     window is real, and the repair paths are live code; the caller goes on to
//     prove them.
//   - PostgreSQL must REFUSE it with SQLSTATE 23502 and this returns true:
//     migration 0006 made tenant_id NOT NULL on every governed table, so the
//     scenario cannot be constructed, and the caller returns — its remaining
//     assertions have no subject, and the refusal IS the property on this
//     engine. The attempt is made on the maintenance lane, declared with SET
//     LOCAL hausv.cross_tenant='on', so the row-level policy is out of the way
//     and only the column constraint answers; from an undeclared session the
//     policy would refuse first (42501) and say nothing about NOT NULL.
//
// Any other outcome is fatal and names the engine, so the two cannot drift apart
// silently: under the 0003 posture PostgreSQL accepts the row and this fails.
func RollbackWindowClosed(t *testing.T, database *sql.DB) bool {
	t.Helper()
	tx, err := database.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("begin rollback-window probe: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if Backend() == db.BackendPostgres {
		if _, err := tx.ExecContext(t.Context(), `SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
			t.Fatalf("declare the maintenance lane for the probe: %v", err)
		}
	}
	_, err = tx.ExecContext(t.Context(),
		`INSERT INTO announcements(tenant_slug, id, data) VALUES('rollback-window-probe', 'probe', '{}')`)
	switch Backend() {
	case db.BackendPostgres:
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23502" {
			t.Fatalf("PostgreSQL must refuse a tenant-bound row without an identity with SQLSTATE 23502 (migration 0006 made tenant_id NOT NULL); got %v", err)
		}
		return true
	default:
		if err != nil {
			t.Fatalf("SQLite must accept a tenant-bound row without an identity — the rollback window is real there; got %v", err)
		}
		return false
	}
}
