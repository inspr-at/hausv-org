package store_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/ulid"
)

func TestSQLHomePortalActivationIsAtomicIdempotentAndPersistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hausv.db")
	database, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	reservations := store.NewSQLHomeReservationStore(database)
	identity := store.NewSQLIdentityStore(database)
	portals := store.NewSQLHomePortalStore(database)
	now := time.Date(2026, time.August, 14, 12, 0, 0, 0, time.UTC)

	person, err := identity.UpsertPerson(store.Person{
		Email: "owner@example.com", FirstName: "Eva", AuthMethods: []string{store.AuthMethodOIDC},
	}, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := identity.SetMembership(store.HouseMembership{
		PersonID: person.ID, TenantSlug: "anderes-haus", Role: store.RoleRenter, Status: "Aktiv",
	}, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := reservations.Reserve(store.HomeReservation{
		Slug: "stadtpark-home", HouseholdName: "Zuhause am Stadtpark", OwnerEmail: "owner@example.com",
		AuthorizationConfirmed: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := reservations.Confirm("stadtpark-home", "owner@example.com", now.Add(time.Minute)); err != nil || !ok {
		t.Fatalf("confirm ok=%v err=%v", ok, err)
	}

	portal, created, err := portals.Activate("stadtpark-home", "owner@example.com", now.Add(2*time.Minute))
	if err != nil || !created || portal.HouseholdName != "Zuhause am Stadtpark" {
		t.Fatalf("activation portal=%+v created=%v err=%v", portal, created, err)
	}
	var tenantID string
	if err := database.QueryRow(`SELECT tenant_id FROM tenant WHERE slug=$1`, portal.Slug).Scan(&tenantID); err != nil || !ulid.Valid(tenantID) {
		t.Fatalf("activation tenant identity=%q err=%v", tenantID, err)
	}
	owner, ok := identity.PersonByEmail("owner@example.com")
	if !ok || owner.ID != person.ID || owner.FirstName != "Eva" || len(owner.AuthMethods) != 1 || owner.AuthMethods[0] != store.AuthMethodOIDC {
		t.Fatalf("existing identity was not preserved: %+v", owner)
	}
	// Both references are read back from the database rather than invented. A
	// fabricated id used to pass here because nothing filtered on it; now it
	// would address a tenant that does not exist, and the membership lookups
	// below would come back empty for a reason that had nothing to do with
	// activation.
	homeIdentity, _ := store.BindIdentityRepository(identity, tenantRefFromDB(t, database, "stadtpark-home"))
	otherIdentity, _ := store.BindIdentityRepository(identity, tenantRefFromDB(t, database, "anderes-haus"))
	membership, ok := homeIdentity.Membership(owner.ID)
	if !ok || membership.Role != store.RoleOwner || membership.Status != "Aktiv" {
		t.Fatalf("owner membership = %+v ok=%v", membership, ok)
	}
	if other, ok := otherIdentity.Membership(owner.ID); !ok || other.Role != store.RoleRenter {
		t.Fatalf("other membership changed: %+v ok=%v", other, ok)
	}
	reservation, _, _ := reservations.Get("stadtpark-home")
	if reservation.Status != store.HomeReservationActive {
		t.Fatalf("reservation status = %q", reservation.Status)
	}
	if _, ok, err := reservations.Confirm("stadtpark-home", "owner@example.com", now.Add(3*time.Minute)); err != nil || !ok {
		t.Fatalf("repeat confirm ok=%v err=%v", ok, err)
	}
	reservation, _, _ = reservations.Get("stadtpark-home")
	if reservation.Status != store.HomeReservationActive {
		t.Fatalf("repeat confirmation downgraded active reservation to %q", reservation.Status)
	}
	second, created, err := portals.Activate("stadtpark-home", "owner@example.com", now.Add(4*time.Minute))
	if err != nil || created || !second.ActivatedAt.Equal(portal.ActivatedAt) {
		t.Fatalf("repeat activation portal=%+v created=%v err=%v", second, created, err)
	}
	owned, err := portals.ListByOwner(" OWNER@example.com ")
	if err != nil || len(owned) != 1 || owned[0].Slug != "stadtpark-home" {
		t.Fatalf("owned portals=%+v err=%v", owned, err)
	}
	if foreign, err := portals.ListByOwner("other@example.com"); err != nil || len(foreign) != 0 {
		t.Fatalf("foreign portals=%+v err=%v", foreign, err)
	}

	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	database, err = db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	reopened, found, err := store.NewSQLHomePortalStore(database).Get("stadtpark-home")
	if err != nil || !found || reopened.OwnerEmail != "owner@example.com" || reopened.HouseholdName != portal.HouseholdName {
		t.Fatalf("reopened portal=%+v found=%v err=%v", reopened, found, err)
	}
	owned, err = store.NewSQLHomePortalStore(database).ListByOwner("owner@example.com")
	if err != nil || len(owned) != 1 || owned[0].HouseholdName != portal.HouseholdName {
		t.Fatalf("reopened owned portals=%+v err=%v", owned, err)
	}
}

func TestSQLHomePortalRejectsForeignUnconfirmedAndRollsBack(t *testing.T) {
	database := dbtest.Open(t)
	t.Cleanup(func() { _ = database.Close() })
	reservations := store.NewSQLHomeReservationStore(database)
	portals := store.NewSQLHomePortalStore(database)
	identity := store.NewSQLIdentityStore(database)
	now := time.Date(2026, time.August, 14, 12, 0, 0, 0, time.UTC)
	if _, err := reservations.Reserve(store.HomeReservation{
		Slug: "sicheres-home", HouseholdName: "Sicheres Home", OwnerEmail: "owner@example.com", AuthorizationConfirmed: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	for _, email := range []string{"owner@example.com", "other@example.com"} {
		if _, _, err := portals.Activate("sicheres-home", email, now); !errors.Is(err, store.ErrHomePortalActivationDenied) {
			t.Fatalf("activation for %s error=%v", email, err)
		}
	}
	if _, ok, err := reservations.Confirm("sicheres-home", "owner@example.com", now.Add(time.Minute)); err != nil || !ok {
		t.Fatalf("confirm ok=%v err=%v", ok, err)
	}
	rejectMembershipSQL := `CREATE TRIGGER reject_home_owner BEFORE INSERT ON house_memberships
		BEGIN SELECT RAISE(ABORT, 'test membership failure'); END`
	if dbtest.Backend() == db.BackendPostgres {
		rejectMembershipSQL = `CREATE FUNCTION reject_home_owner_fn() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'test membership failure'; END; $$;
		CREATE TRIGGER reject_home_owner BEFORE INSERT ON house_memberships
		FOR EACH ROW EXECUTE FUNCTION reject_home_owner_fn()`
	}
	if _, err := database.Exec(rejectMembershipSQL); err != nil {
		t.Fatal(err)
	}
	if _, _, err := portals.Activate("sicheres-home", "owner@example.com", now.Add(2*time.Minute)); err == nil {
		t.Fatal("activation succeeded despite membership failure")
	}
	if _, found, err := portals.Get("sicheres-home"); err != nil || found {
		t.Fatalf("partial portal survived found=%v err=%v", found, err)
	}
	if _, found := identity.PersonByEmail("owner@example.com"); found {
		t.Fatal("partial owner identity survived failed activation")
	}
	reservation, _, _ := reservations.Get("sicheres-home")
	if reservation.Status != store.HomeReservationEmailConfirmed {
		t.Fatalf("failed activation changed reservation to %q", reservation.Status)
	}
}

// tenantRefFromDB reads a tenant's real identity, so a test addresses rows the
// same way the application does.
func tenantRefFromDB(t *testing.T, database *sql.DB, slug string) store.TenantRef {
	t.Helper()
	var id string
	if err := database.QueryRow(`SELECT tenant_id FROM tenant WHERE slug=$1`, slug).Scan(&id); err != nil {
		t.Fatalf("tenant identity for %s: %v", slug, err)
	}
	return store.TenantRef{ID: id, Slug: slug}
}
