package store

import (
	"crypto/sha256"
	"database/sql"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// Test tenants used to share ONE hard-coded ULID for every slug. That made every
// cross-tenant isolation test in this package a single-tenant test the moment
// the queries started filtering on tenant_id: "demo" and "other" would have been
// the same tenant, and a repository returning both rows would still have passed.
//
// So each slug now gets its own identity, derived from the slug itself so it is
// stable across runs and readable in a failure message, and the same identity is
// written into the `tenant` table of every fixture database — the foreign key
// from every tenant-bound row points there.
//
// Where this is blind: a slug that no fixture seeds gets no row, and the first
// write that references it fails with a foreign-key error rather than silently
// writing an orphan. That is the intended failure mode — loud and local — but it
// does mean adding a new fixture slug means adding it to fixtureTenantSlugs.
var fixtureTenantSlugs = []string{
	"demo", "other", "haus-a", "haus-b", "haus-c", "anderes-haus", "stadtpark-home", "tenant-heroes",
}

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// testTenantID derives a stable, valid ULID from a slug. The first character is
// restricted to 0-7 because that is what a real 128-bit ULID can produce, and
// what both the SQLite CHECK and the PostgreSQL constraint enforce.
func testTenantID(slug string) string {
	sum := sha256.Sum256([]byte("hausv-test-tenant:" + slug))
	out := make([]byte, 26)
	out[0] = crockford[int(sum[0])%8]
	for i := 1; i < 26; i++ {
		out[i] = crockford[int(sum[i])%len(crockford)]
	}
	return string(out)
}

func testTenantRef(slug string) TenantRef {
	return TenantRef{ID: testTenantID(slug), Slug: slug}
}

// testDB opens a fixture database and gives the fixture tenants the identities
// testTenantRef hands out, so a bound repository in a test addresses rows the
// same way production does.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	database, _ := testLanes(t)
	return database
}

// testLanes returns the fixture pool AND the scoped seam over it, which is what
// a store now takes.
//
// The pool is still handed back because fixtures and assertions write and read
// directly, deliberately outside the store's lane: an assertion that ran through
// the same lane as the write could not tell "the row is scoped correctly" from
// "the lane hid a row that is scoped wrong".
//
// On PostgreSQL the seam is built from the DSN of this test's isolated schema,
// so every handle a store takes here is a REAL lane — a pool of its own, born
// with the scope in its startup packet — not the process pool wearing a wrapper.
// On SQLite there are no lanes and nothing to scope, and db.Scoped hands back
// the one pool for every accessor, so these tests behave exactly as before.
func testLanes(t *testing.T) (*sql.DB, *TenantDB) {
	t.Helper()
	database, cfg := dbtest.OpenWithConfig(t)
	seedFixtureTenants(t, database)
	return database, lanesOver(t, database, cfg)
}

// testStoreDB is testLanes for the tests that never touch the pool directly.
func testStoreDB(t *testing.T) *TenantDB {
	t.Helper()
	_, lanes := testLanes(t)
	return lanes
}

// LanesForTest is testLanes for this package's black-box tests, which cannot see
// an unexported helper. It is not named Test* on purpose: vet would then read it
// as a test with the wrong signature.
func LanesForTest(t *testing.T, database *sql.DB, cfg appdb.Config) *TenantDB {
	t.Helper()
	return lanesOver(t, database, cfg)
}

// lanesOver builds the scoped seam over a pool the caller already opened, for
// the tests that open their own database (reopen-after-close, two-database
// isolation checks) rather than going through testLanes.
func lanesOver(t *testing.T, database *sql.DB, cfg appdb.Config) *TenantDB {
	t.Helper()
	scoped, err := appdb.NewScoped(cfg, database)
	if err != nil {
		t.Fatalf("open scoped test lanes: %v", err)
	}
	// Lanes are pools; without this a package-sized test run leaks one set per
	// database it opens, and the server runs out of backends long before the
	// suite ends.
	t.Cleanup(func() { _ = scoped.Close() })
	return NewTenantDB(scoped)
}

func seedFixtureTenants(t *testing.T, database *sql.DB) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, slug := range fixtureTenantSlugs {
		if _, err := database.ExecContext(t.Context(),
			`INSERT INTO tenant(tenant_id,slug,name,created_at,updated_at) VALUES($1,$2,$3,$4,$5)`,
			testTenantID(slug), slug, slug, now, now); err != nil {
			t.Fatalf("seed fixture tenant %s: %v", slug, err)
		}
	}
}
