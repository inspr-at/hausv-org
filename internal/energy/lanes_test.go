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
// A fixture handle is handed back too, on purpose: fixtures and assertions in
// this package write and read the database directly, OUTSIDE the store's tenant
// lane. An assertion that read back through the same lane as the write could
// not tell "the row is scoped correctly" from "the lane hid a row that is
// scoped wrong". That handle is the maintenance view (dbtest.MaintenanceView):
// it used to be the process pool, and PostgreSQL migration 0006 made an
// undeclared session see and write nothing, so on that engine only the lane
// that declares itself cross-tenant can still seed and count a governed table.
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
	return dbtest.MaintenanceView(t, scoped), store.NewTenantDB(scoped)
}

// openEnergyStore is openEnergyLanes for the tests that never touch the pool.
func openEnergyStore(t *testing.T) *energy.SQLStore {
	t.Helper()
	_, lanes := openEnergyLanes(t)
	return energy.NewSQLStore(lanes)
}
