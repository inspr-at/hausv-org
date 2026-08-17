package store

import (
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// TestConcurrentBootsMintOneIdentityPerSlug is the multi-replica cutover
// hazard, reproduced with real concurrency against a real database.
//
// EnsureTenantIdentities is the FIRST thing a boot runs, and it minted with its
// own SELECT-then-INSERT: no ON CONFLICT, no recovery. Eight boots against one
// PostgreSQL reproduced the consequence one time in eight —
//
//	store: insert tenant demo: ERROR: duplicate key value violates unique
//	constraint "tenant_slug_key" (SQLSTATE 23505)
//
// — and that error aborts newApp, so the replica does not start. The backfill's
// own mint had already been moved onto tenantid.Ensure for exactly this reason;
// this one had not, and it runs first.
//
// The property asserted is the one that matters after the race resolves: one
// slug, one identity. A second `tenant` row for one slug is unrecoverable —
// every row already written points at the first — so the resolution has to be
// re-reading the winner's row, not retrying the insert.
//
// Where it is blind: it drives goroutines in one process against one pool, not
// two containers against one database, so it exercises the SQL-level race and
// not the process-level one. On SQLite it tolerates the engine's own writer
// contention: a deferred transaction that reads and then writes while another
// has written returns SQLITE_BUSY_SNAPSHOT immediately, which busy_timeout does
// not retry. That is inherent to every concurrent write in this codebase and is
// not what this test is about — production SQLite is one container by
// construction, and the multi-replica cutover is PostgreSQL.
func TestConcurrentBootsMintOneIdentityPerSlug(t *testing.T) {
	database := dbtest.Open(t)
	configured := []TenantIdentity{{Slug: "demo", Name: "Demo"}, {Slug: "haus-a", Name: "Haus A"}}

	const replicas = 8
	errs := make([]error, replicas)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range replicas {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = EnsureTenantIdentities(t.Context(), database, configured)
		}()
	}
	close(start)
	wg.Wait()

	booted := 0
	for i, err := range errs {
		switch {
		case err == nil:
			booted++
		case isEngineWriterContention(err):
			t.Logf("replica %d lost the engine's writer lock (tolerated): %v", i, err)
		default:
			t.Errorf("replica %d failed to boot: %v", i, err)
		}
	}
	if booted == 0 {
		t.Fatal("no replica booted at all — the test proves nothing about the race")
	}
	for _, tenant := range configured {
		var rows int
		if err := database.QueryRowContext(t.Context(),
			`SELECT count(*) FROM tenant WHERE slug=$1`, tenant.Slug).Scan(&rows); err != nil {
			t.Fatalf("count tenant rows for %s: %v", tenant.Slug, err)
		}
		if rows != 1 {
			t.Errorf("tenant %s has %d rows, want exactly 1: a second identity orphans every row "+
				"already pointing at the first", tenant.Slug, rows)
		}
	}
}

// isEngineWriterContention reports SQLite's single-writer refusal, which is the
// one failure this test is allowed to ignore. A unique violation is deliberately
// NOT in here: that is the defect.
func isEngineWriterContention(err error) bool {
	if dbtest.Backend() != "sqlite" {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "sqlite_busy")
}

// racingQuerier runs a hook exactly once, immediately before the statement the
// caller believes it is the first to issue. That is the whole of the race the
// backfill can lose: its SELECT found no tenant row, another replica committed
// one, and its INSERT arrives second.
//
// It wraps the real transaction, so the collision is a real unique violation
// raised by the real engine rather than a stubbed error.
type racingQuerier struct {
	inner  tenantIDQuerier
	before func()
}

func (r *racingQuerier) QueryRow(query string, args ...any) *sql.Row {
	return r.inner.QueryRow(query, args...)
}

func (r *racingQuerier) Exec(query string, args ...any) (sql.Result, error) {
	if r.before != nil {
		hook := r.before
		r.before = nil
		hook()
	}
	return r.inner.Exec(query, args...)
}

// TestBackfillMintSurvivesALostRaceForTheSameSlug is the multi-replica hazard,
// reproduced without a second process.
//
// The backfill mints an identity for a slug that has none. It used to do that
// with a bare INSERT: two containers booting against one PostgreSQL both find no
// `tenant` row, both insert, and the loser takes a unique violation on
// tenant.slug. That error aborts BackfillTenantIDs, which aborts
// EnsureTenantIdentities, which aborts newApp — the replica does not start, and
// the deploy that added the second replica is the deploy that discovers it.
//
// The resolution is the one internal/tenantid already used: the winner's row IS
// the answer, so re-reading it resolves the race. Doing that inside a
// transaction needs the insert not to raise at all, because on PostgreSQL a
// failed statement poisons the surrounding transaction and the re-read fails
// too — which is why the shared resolver inserts with ON CONFLICT DO NOTHING.
//
// Where it is blind: it drives one goroutine with a deterministic interleaving
// instead of two real replicas, so it proves the recovery works when the race is
// lost and says nothing about how often it is lost. It covers the backfill's
// mint; the same resolver is reached from internal/energy and internal/server on
// a pooled *sql.DB, where a failed statement does not poison anything, and that
// path is not exercised here.
func TestBackfillMintSurvivesALostRaceForTheSameSlug(t *testing.T) {
	database := dbtest.Open(t)
	tx, err := database.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	winner := testTenantID("race-winner")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	querier := &racingQuerier{inner: tx, before: func() {
		// The other replica commits its mint first.
		if _, err := tx.Exec(
			`INSERT INTO tenant(tenant_id,slug,name,created_at,updated_at) VALUES($1,$2,$3,$4,$5)`,
			winner, "demo", "Demo", now, now); err != nil {
			t.Fatalf("seed the winning replica's tenant row: %v", err)
		}
	}}

	id, ok, err := identityForSlug(querier, "demo", true)
	if err != nil {
		t.Fatalf("losing the mint race must not abort the boot: %v", err)
	}
	if !ok {
		t.Fatal("no identity was resolved after losing the race")
	}
	if id != winner {
		t.Fatalf("identity = %q, want the winner's %q: a second row for one slug cannot exist", id, winner)
	}
	// And the transaction is still usable, which is the half that PostgreSQL
	// decides: a statement that raised would have poisoned it and every later
	// statement of the backfill with it.
	var count int
	if err := tx.QueryRow(`SELECT count(*) FROM tenant WHERE slug='demo'`).Scan(&count); err != nil {
		t.Fatalf("the transaction did not survive the collision: %v", err)
	}
	if count != 1 {
		t.Fatalf("tenant rows for slug demo = %d, want 1", count)
	}
}
