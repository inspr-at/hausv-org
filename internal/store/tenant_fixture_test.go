package store

import (
	"crypto/sha256"
	"database/sql"
	"testing"
	"time"

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
	database := dbtest.Open(t)
	seedFixtureTenants(t, database)
	return database
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
