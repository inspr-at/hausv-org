package store

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func testTenantDB(t *testing.T) (*TenantDB, appdb.Handle) {
	t.Helper()
	database, err := appdb.Open(filepath.Join(t.TempDir(), "scoped.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	scoped, err := appdb.NewScoped(appdb.Config{DSN: "unused"}, database)
	if err != nil {
		t.Fatalf("new scoped: %v", err)
	}
	return NewTenantDB(scoped), database
}

// The property the whole design rests on is an ABSENCE: TenantDB must not be a
// database handle. If it were, a store could keep calling tenantDB.Query(...)
// and compile, and the scope would be quietly optional again. Making the
// unscoped call impossible to write by accident is the enforcement mechanism —
// there is no linter, no review checklist and no runtime guard behind it.
func TestTenantDBIsNotItselfADatabaseHandle(t *testing.T) {
	tenantDB, _ := testTenantDB(t)
	if _, isHandle := any(tenantDB).(appdb.Handle); isHandle {
		t.Fatal("TenantDB must NOT satisfy db.Handle: a store could then query it without naming a tenant")
	}
}

// And the method set is exactly the two accessors, so a later convenience
// method cannot reopen the hole one call at a time.
func TestTenantDBExposesOnlyTheTwoAccessors(t *testing.T) {
	typ := reflect.TypeOf(&TenantDB{})
	got := make([]string, 0, typ.NumMethod())
	for i := range typ.NumMethod() {
		got = append(got, typ.Method(i).Name)
	}
	sort.Strings(got)
	want := []string{"For", "Unscoped"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("TenantDB methods = %v, want exactly %v", got, want)
	}
}

// On SQLite there is one pool and it must be THE pool the stores already hold —
// pointer identity, because a second *sql.DB over one SQLite file is a second
// write lock and a different transaction view.
func TestTenantDBOnSQLiteAlwaysReturnsTheProcessPool(t *testing.T) {
	tenantDB, database := testTenantDB(t)
	refA := TenantRef{ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Slug: "haus-a"}
	refB := TenantRef{ID: "01BX5ZZKBKACTAV9WEVGEMMVRZ", Slug: "haus-b"}
	for name, handle := range map[string]appdb.Handle{
		"For(A)":    tenantDB.For(refA),
		"For(B)":    tenantDB.For(refB),
		"For(zero)": tenantDB.For(TenantRef{}),
		"Unscoped":  tenantDB.Unscoped("test"),
	} {
		if handle != database {
			t.Fatalf("%s returned a different pool than the one the stores hold", name)
		}
	}
}

// Every store in this package now takes a *TenantDB, and every one of the store
// tests builds one with testLanes. That makes this the load-bearing question for
// the whole suite: is the handle those stores receive a REAL lane, or the
// process pool wearing a wrapper?
//
// It matters because the difference is invisible to an ordinary assertion. Each
// store's SQL already carries `WHERE tenant_id = $1`, so a converted store
// returns identical rows either way; a suite handed the plain pool would be
// green while proving nothing about the mechanism it exists to prove. What only
// a real lane has is the scope in the SESSION — set in the startup packet, which
// is what row-level security reads — so that is what is asked for here.
func TestStoreTestsRunOnRealLanes(t *testing.T) {
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")

	if dbtest.Backend() != appdb.BackendPostgres {
		// SQLite has no lanes and no RLS to need them; db.Scoped hands back the
		// one pool for every accessor, which is what keeps this suite's SQLite
		// run behaviour-identical. Pointer identity is asserted in full by
		// TestTenantDBOnSQLiteAlwaysReturnsTheProcessPool.
		if lanes.For(tenant) != appdb.Handle(database) {
			t.Fatal("on SQLite a store must be handed the one process pool")
		}
		return
	}

	if lanes.For(tenant) == appdb.Handle(database) {
		t.Fatal("the stores were handed the process pool, so nothing in this suite exercises a lane")
	}
	var scope string
	if err := lanes.For(tenant).QueryRow(`SELECT current_setting('hausv.tenant_id', true)`).Scan(&scope); err != nil {
		t.Fatalf("read the lane scope: %v", err)
	}
	if scope != tenant.ID {
		t.Fatalf("tenant lane session scope = %q, want the tenant id %q", scope, tenant.ID)
	}

	var declared string
	if err := lanes.Unscoped("prove the maintenance lane declares itself").
		QueryRow(`SELECT current_setting('hausv.cross_tenant', true)`).Scan(&declared); err != nil {
		t.Fatalf("read the maintenance declaration: %v", err)
	}
	if declared != "on" {
		t.Fatalf("maintenance lane declaration = %q, want \"on\"", declared)
	}
	// And the maintenance lane must NOT be carrying a tenant scope, or it would
	// be a tenant lane with a different name.
	var leaked string
	if err := lanes.Unscoped("prove the maintenance lane carries no tenant").
		QueryRow(`SELECT coalesce(current_setting('hausv.tenant_id', true), '')`).Scan(&leaked); err != nil {
		t.Fatalf("read the maintenance lane tenant: %v", err)
	}
	if leaked != "" {
		t.Fatalf("maintenance lane carries tenant scope %q", leaked)
	}
}

// The tightening the conversion actually buys, stated as behaviour: a row with
// no tenant_id is reachable from the maintenance lane and from NO tenant lane.
//
// This is the reason HealOrphanReason exists, pinned as a test rather than left
// as a paragraph. If a later change makes a tenant lane able to see an orphan
// row, every site naming HealOrphanReason can go back to For(tenant) — and this
// test is what will say so, by failing.
func TestATenantLaneCannotReachARowWithoutAnIdentity(t *testing.T) {
	if dbtest.Backend() != appdb.BackendPostgres {
		t.Skip("row-level security exists only on PostgreSQL; SQLite has one pool and no policy")
	}
	database, lanes := testLanes(t)
	tenant := testTenantRef("demo")
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Exactly the shape the previous release wrote: no tenant_id column value.
	if _, err := database.ExecContext(t.Context(),
		`INSERT INTO unit_payment_status(tenant_slug, unit_id, status, updated_at, updated_by)
		 VALUES($1,$2,$3,$4,$5)`,
		tenant.Slug, "u1", UnitPaymentStatusOpen, now, "old@example.com"); err != nil {
		t.Fatalf("seed the orphan row: %v", err)
	}

	var fromLane int
	if err := lanes.For(tenant).QueryRow(
		`SELECT count(*) FROM unit_payment_status WHERE unit_id=$1`, "u1").Scan(&fromLane); err != nil {
		t.Fatalf("count from the tenant lane: %v", err)
	}
	if fromLane != 0 {
		t.Fatalf("the tenant lane can see %d orphan rows; migration 0003 says it must see none", fromLane)
	}

	var fromMaintenance int
	if err := lanes.Unscoped("prove the orphan is reachable somewhere").QueryRow(
		`SELECT count(*) FROM unit_payment_status WHERE unit_id=$1`, "u1").Scan(&fromMaintenance); err != nil {
		t.Fatalf("count from the maintenance lane: %v", err)
	}
	if fromMaintenance != 1 {
		t.Fatalf("the maintenance lane sees %d orphan rows, want 1: nothing could heal it", fromMaintenance)
	}
}
