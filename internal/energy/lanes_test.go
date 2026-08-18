package energy_test

import (
	"database/sql"
	"testing"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/store"
)

// openEnergyLanes opens a migrated, empty database and the lane seam over it,
// which is what SQLStore now takes.
//
// The pool is handed back too, on purpose: fixtures and assertions in this
// package write and read the database directly, OUTSIDE the store's lane. An
// assertion that read back through the same lane as the write could not tell
// "the row is scoped correctly" from "the lane hid a row that is scoped wrong",
// and a seeded orphan (NULL tenant_id) can only be planted from the pool.
//
// On PostgreSQL every handle the store takes here is a real lane — a pool of
// its own, born with the scope in its startup packet — so the tests in this
// package prove the lane decisions on the one engine that has RLS. On SQLite
// db.Scoped hands back the one pool for every accessor, and nothing changes.
func openEnergyLanes(t *testing.T) (*sql.DB, *store.TenantDB) {
	t.Helper()
	database, cfg := dbtest.OpenWithConfig(t)
	scoped, err := appdb.NewScoped(cfg, database)
	if err != nil {
		t.Fatalf("open scoped test lanes: %v", err)
	}
	// Lanes are pools; without this a package-sized run leaks one set per
	// database it opens.
	t.Cleanup(func() { _ = scoped.Close() })
	return database, store.NewTenantDB(scoped)
}

// openEnergyStore is openEnergyLanes for the tests that never touch the pool.
func openEnergyStore(t *testing.T) *energy.SQLStore {
	t.Helper()
	_, lanes := openEnergyLanes(t)
	return energy.NewSQLStore(lanes)
}
