package store

import (
	"database/sql"
	"testing"
	"time"
)

// RemoveTenant is the one call in this package where the lane conversion's
// mechanical answer and the safe answer could come apart, so it is pinned as
// behaviour rather than left to a comment.
//
// Its signature is RemoveTenant(email, tenantSlug string): no TenantRef, so
// Unscoped is the only thing that compiles inside it. That looks like an
// oversight, and the obvious tidy-up — thread the tenant through, scope the
// calls, delete an Unscoped — is exactly the change that destroys data.
//
// The mechanism, measured rather than argued: RemoveTenant deletes ONE
// membership, then asks MembershipsForPerson whether any house is left, and
// deletes the person only when none is. That question is cross-tenant by
// construction. Scope it to the house being left and the answer is always zero,
// because migration 0003's policy hides every other tenant's row from a tenant
// lane — so RemoveTenant deletes the person, and the FK cascade from persons
// takes the OTHER house's membership with it. The measurement that produced this
// test read "tenant B memberships surviving: 0 (was 1)".
//
// This test passes today. It exists to fail then.
//
// WHERE IT IS BLIND:
//
//   - On SQLite it proves the SQL, not the lane. db.Scoped hands back the one
//     process pool for every accessor there, so scoping MembershipsForPerson
//     changes nothing and this test stays green through the sabotage. Only the
//     PostgreSQL run — the engine that has RLS — actually holds the line, and
//     that is where the sabotage was verified.
//   - It pins ONE shape of the regression: the person-survives decision going
//     tenant-scoped. A different route to the same damage — say a cascade added
//     to a new table, or DeletePerson gaining a tenant filter that matches
//     nothing — is not covered by these assertions.
//   - It asserts on rows, through the pool, deliberately outside any lane.
//     That is what lets it tell "the membership survived" from "a lane is
//     hiding the membership from me", but it also means it says nothing about
//     what the application would SEE afterwards.
func TestRemoveTenantLeavesTheOtherHousesMembershipIntact(t *testing.T) {
	database, lanes := testLanes(t)
	store := NewSQLIdentityStore(lanes)

	leaving := testTenantRef("demo")
	staying := testTenantRef("haus-b")
	const email = "anna@example.com"
	now := time.Now().UTC()

	person, err := store.UpsertPerson(Person{
		Email: email, FirstName: "Anna", LastName: "Muster", AuthMethods: []string{"email"},
	}, now)
	if err != nil {
		t.Fatalf("create the person: %v", err)
	}
	for _, tenant := range []TenantRef{leaving, staying} {
		if _, err := store.SetMembership(HouseMembership{
			PersonID: person.ID, TenantID: tenant.ID, TenantSlug: tenant.Slug,
			Role: RoleOwner, Status: "Aktiv",
		}, now); err != nil {
			t.Fatalf("create the %s membership: %v", tenant.Slug, err)
		}
	}
	// The premise has to be true before the assertion means anything: one
	// person, two houses, both memberships on disk.
	if got := countMemberships(t, database, person.ID, leaving.ID); got != 1 {
		t.Fatalf("setup: %s memberships = %d, want 1", leaving.Slug, got)
	}
	if got := countMemberships(t, database, person.ID, staying.ID); got != 1 {
		t.Fatalf("setup: %s memberships = %d, want 1", staying.Slug, got)
	}

	removedProfile, found, err := store.RemoveTenant(email, leaving.Slug)
	if err != nil || !found {
		t.Fatalf("RemoveTenant: found=%v err=%v", found, err)
	}

	if got := countMemberships(t, database, person.ID, leaving.ID); got != 0 {
		t.Fatalf("the house being left still has %d memberships, want 0: the removal did not happen", got)
	}
	// THE PIN. Read straight from the pool, outside every lane, so this cannot
	// be satisfied by a lane hiding the row instead of the row existing.
	if got := countMemberships(t, database, person.ID, staying.ID); got != 1 {
		t.Fatalf("tenant %s memberships surviving: %d (was 1).\n\n"+
			"Removing someone from one house destroyed their membership in ANOTHER. This is the\n"+
			"measured failure that keeps RemoveTenant's internals cross-tenant: scoping the\n"+
			"\"does any house remain?\" question to the house being left always answers no, so the\n"+
			"person is deleted and the FK cascade from persons takes this row with it.\n"+
			"Do not thread a TenantRef into RemoveTenant.", staying.Slug, got)
	}
	var persons int
	if err := database.QueryRowContext(t.Context(),
		`SELECT count(*) FROM persons WHERE id=$1`, person.ID).Scan(&persons); err != nil {
		t.Fatalf("count persons: %v", err)
	}
	if persons != 1 {
		t.Fatalf("the person was deleted (%d rows) while %s still had them", persons, staying.Slug)
	}
	if removedProfile {
		t.Fatal("RemoveTenant reported it deleted the person while another house still had them")
	}

	// And the counterpart, so the pin above cannot be satisfied by RemoveTenant
	// simply never deleting anything: leaving the LAST house does remove the
	// person, which is the behaviour the old whole-profile delete had.
	removedProfile, found, err = store.RemoveTenant(email, staying.Slug)
	if err != nil || !found || !removedProfile {
		t.Fatalf("leaving the last house: removedProfile=%v found=%v err=%v", removedProfile, found, err)
	}
	if err := database.QueryRowContext(t.Context(),
		`SELECT count(*) FROM persons WHERE id=$1`, person.ID).Scan(&persons); err != nil {
		t.Fatalf("count persons: %v", err)
	}
	if persons != 0 {
		t.Fatalf("person rows after the last house: %d, want 0", persons)
	}
}

func countMemberships(t *testing.T, database *sql.DB, personID string, tenantID string) int {
	t.Helper()
	var n int
	if err := database.QueryRowContext(t.Context(),
		`SELECT count(*) FROM house_memberships WHERE person_id=$1 AND tenant_id=$2`,
		personID, tenantID).Scan(&n); err != nil {
		t.Fatalf("count memberships for tenant %s: %v", tenantID, err)
	}
	return n
}
