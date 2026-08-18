package store

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	appdb "github.com/inspr-at/hausv-org/internal/db"
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
