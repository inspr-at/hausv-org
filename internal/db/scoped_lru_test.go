package db

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// laneID is a valid ULID that differs per index, so a test can ask for as many
// distinct tenants as it likes without hand-writing them.
func laneID(index int) string {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	out := []byte("01ARZ3NDEKTSV4RRFFQ69G5F00")
	out[24] = alphabet[index/32%32]
	out[25] = alphabet[index%32]
	return string(out)
}

// offlineScoped builds the lane factory against a DSN nothing listens on.
// database/sql connects lazily, so every lane here is a real pool that has never
// opened a socket — which is exactly what the cache bookkeeping operates on, and
// it lets the eviction rules be tested without a server.
func offlineScoped(t *testing.T, laneCap int) *Scoped {
	t.Helper()
	const dsn = "postgres://nobody:nothing@127.0.0.1:1/none?sslmode=disable"
	placeholder, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse offline DSN: %v", err)
	}
	process := stdlib.OpenDB(*placeholder)
	t.Cleanup(func() { _ = process.Close() })
	scoped, err := NewScoped(Config{
		Backend:      BackendPostgres,
		DSN:          dsn,
		LaneCap:      laneCap,
		LaneMaxConns: 3,
	}, process)
	if err != nil {
		t.Fatalf("new scoped: %v", err)
	}
	t.Cleanup(func() { _ = scoped.Close() })
	return scoped
}

func laneIsClosed(t *testing.T, handle Handle) bool {
	t.Helper()
	err := handle.QueryRow(`SELECT 1`).Scan(new(int))
	return err != nil && strings.Contains(err.Error(), "database is closed")
}

// Unbounded lanes are how a process runs a server out of backends: one pool per
// tenant, forever, each holding its own connections. The cache is capped.
func TestLaneCacheStaysWithinItsCap(t *testing.T) {
	scoped := offlineScoped(t, 3)
	for i := range 12 {
		scoped.For(laneID(i))
	}
	if got := scoped.openLanes(); got > 3 {
		t.Fatalf("lane cache holds %d pools, cap is 3", got)
	}
}

// Least-recently-USED, not least-recently-created: a tenant that keeps working
// must not be evicted because it was first through the door.
func TestLaneCacheEvictsTheLeastRecentlyUsedLane(t *testing.T) {
	scoped := offlineScoped(t, 2)
	laneA := scoped.For(laneID(0))
	laneB := scoped.For(laneID(1))
	scoped.For(laneID(0)) // A is used again, so B becomes the oldest
	scoped.For(laneID(2)) // forces one eviction

	if !laneIsClosed(t, laneB) {
		t.Fatal("the least recently used lane was not evicted")
	}
	if laneIsClosed(t, laneA) {
		t.Fatal("the recently used lane was evicted instead")
	}
}

// Evicting a pool means Close(), and the caller is still holding the handle it
// was given. A store does not re-ask the cache between statements: it takes a
// Handle, opens a transaction, runs several statements, commits. Close the pool
// underneath that and the next statement fails with "sql: database is closed" —
// a request killed by a cache-size policy, which is never an acceptable trade.
//
// So a lane with connections checked out is skipped, and the cache is allowed to
// run over its cap until they come back.
func TestLaneCacheNeverEvictsAPoolWithWorkInFlight(t *testing.T) {
	scoped, _ := scopedPostgres(t)
	scoped.laneCap = 2

	busy := scoped.For(scopedTenantA)
	tx, err := busy.Begin()
	if err != nil {
		t.Fatalf("begin on the busy lane: %v", err)
	}
	defer tx.Rollback()

	for i := range 8 {
		scoped.For(laneID(i + 10))
	}

	// The handle the caller still holds has to keep working: this is what an
	// eviction takes away.
	if err := busy.QueryRow(`SELECT 1`).Scan(new(int)); err != nil {
		t.Fatalf("the busy lane was evicted while its holder was still using it: %v", err)
	}
	var scope string
	if err := tx.QueryRow(`SELECT coalesce(current_setting('hausv.tenant_id', true), '')`).Scan(&scope); err != nil {
		t.Fatalf("the in-flight transaction was killed by an eviction: %v", err)
	}
	if scope != scopedTenantA {
		t.Fatalf("in-flight transaction scope = %q, want %q", scope, scopedTenantA)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit after evictions: %v", err)
	}

	// Once the work is done the lane is evictable again, so the overflow is
	// temporary rather than a leak.
	for i := range 8 {
		scoped.For(laneID(i + 30))
	}
	if got := scoped.openLanes(); got > 2 {
		t.Fatalf("lane cache stayed at %d pools after the busy lane went idle, cap is 2", got)
	}
}

// The cap is a number someone has to be able to change, and a number that is
// wrong must stop the boot rather than surface as connection refusals later.
func TestLaneCapIsDefaultedAndBudgeted(t *testing.T) {
	cfg := Config{Backend: BackendPostgres}
	applyPostgresDefaults(&cfg)
	if cfg.LaneCap <= 0 || cfg.LaneMaxConns <= 0 {
		t.Fatalf("lane cap defaults missing: cap=%d per-lane=%d", cfg.LaneCap, cfg.LaneMaxConns)
	}
	budget := ConnectionBudget{LaneCap: cfg.LaneCap, PerLaneMax: cfg.LaneMaxConns, ProcessPool: cfg.MaxOpenConns}
	if err := budget.fits(100, 3); err != nil {
		t.Fatalf("the shipped defaults must fit a stock max_connections=100 server: %v", err)
	}
}

func TestScopedBudgetReportsTheConfiguredPlan(t *testing.T) {
	scoped := offlineScoped(t, 7)
	budget := scoped.Budget()
	if budget.LaneCap != 7 || budget.PerLaneMax != 3 || budget.ProcessPool != defaultMaxOpenConns {
		t.Fatalf("budget = %+v, want 7 lanes x 3 plus the %d-connection process pool", budget, defaultMaxOpenConns)
	}
}

// The startup assertion, against a real server, with the numbers this process
// actually ships.
func TestVerifyBudgetAcceptsTheShippedDefaultsAndRefusesAnAbsurdCap(t *testing.T) {
	scoped, _ := scopedPostgres(t)
	if err := scoped.VerifyBudget(t.Context()); err != nil {
		t.Fatalf("the shipped lane plan must fit the test server: %v", err)
	}
	scoped.laneCap = 1_000_000
	if err := scoped.VerifyBudget(t.Context()); err == nil {
		t.Fatal("a million lanes must stop the boot")
	}
}

// SQLite has no lanes to cap and no max_connections to check.
func TestVerifyBudgetIsANoOpOnSQLite(t *testing.T) {
	database, err := Open(t.TempDir() + "/budget.db")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	scoped, err := NewScoped(Config{DSN: "unused", LaneCap: 1_000_000}, database)
	if err != nil {
		t.Fatalf("new scoped: %v", err)
	}
	if err := scoped.VerifyBudget(t.Context()); err != nil {
		t.Fatalf("sqlite has no connection budget to blow: %v", err)
	}
}
