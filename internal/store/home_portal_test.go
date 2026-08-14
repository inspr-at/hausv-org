package store_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/store"
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
	owner, ok := identity.PersonByEmail("owner@example.com")
	if !ok || owner.ID != person.ID || owner.FirstName != "Eva" || len(owner.AuthMethods) != 1 || owner.AuthMethods[0] != store.AuthMethodOIDC {
		t.Fatalf("existing identity was not preserved: %+v", owner)
	}
	membership, ok := identity.Membership(owner.ID, "stadtpark-home")
	if !ok || membership.Role != store.RoleOwner || membership.Status != "Aktiv" {
		t.Fatalf("owner membership = %+v ok=%v", membership, ok)
	}
	if other, ok := identity.Membership(owner.ID, "anderes-haus"); !ok || other.Role != store.RoleRenter {
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
}

func TestSQLHomePortalRejectsForeignUnconfirmedAndRollsBack(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "hausv.db"))
	if err != nil {
		t.Fatal(err)
	}
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
	if _, err := database.Exec(`CREATE TRIGGER reject_home_owner BEFORE INSERT ON house_memberships
		BEGIN SELECT RAISE(ABORT, 'test membership failure'); END`); err != nil {
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
