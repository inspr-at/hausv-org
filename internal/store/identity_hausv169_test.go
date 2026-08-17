package store

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// HAUSV-169 / HAUSV-135: person and house are separate aggregates joined N:N.
// These tests pin the invariants the old email-keyed whole-profile CRUD broke:
// a house may only ever touch its OWN membership.

func newIdentityStore(t *testing.T) *SQLIdentityStore {
	t.Helper()
	database := dbtest.Open(t)
	t.Cleanup(func() { database.Close() })
	return NewSQLIdentityStore(database)
}

func TestPersonIsGlobalAndUniqueByEmail(t *testing.T) {
	s := newIdentityStore(t)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	if _, err := s.UpsertPerson(Person{Email: ""}, now); err == nil {
		t.Fatal("a person without an email must be rejected")
	}

	created, err := s.UpsertPerson(Person{
		Email: "Anna@Example.COM", Title: "Dr.", FirstName: "Anna", LastName: "Muster",
		AuthMethods: []string{"email"},
	}, now)
	if err != nil {
		t.Fatalf("create person: %v", err)
	}
	if created.ID == "" || created.Email != "anna@example.com" {
		t.Fatalf("person not normalized: %+v", created)
	}

	// Upserting the same email must reuse the person, not mint a second one.
	again, err := s.UpsertPerson(Person{Email: "anna@example.com", FirstName: "Anna B"}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	if again.ID != created.ID {
		t.Fatalf("same email produced a second person: %q vs %q", again.ID, created.ID)
	}
	if !again.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("CreatedAt must be preserved: %v vs %v", again.CreatedAt, created.CreatedAt)
	}
	if got := s.ListPersons(); len(got) != 1 {
		t.Fatalf("expected exactly one person, got %d", len(got))
	}
}

// The core N:N property: a second invitation links a NEW membership rather than
// creating or overwriting a second profile.
func TestSecondHouseLinksMembershipNotSecondPerson(t *testing.T) {
	s := newIdentityStore(t)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	person, err := s.UpsertPerson(Person{Email: "anna@example.com", FirstName: "Anna"}, now)
	if err != nil {
		t.Fatalf("person: %v", err)
	}
	if _, err := s.SetMembership(HouseMembership{
		PersonID: person.ID, TenantSlug: "demo", Role: RoleOwner, Status: "Eingeladen",
	}, now); err != nil {
		t.Fatalf("membership a: %v", err)
	}
	if _, err := s.SetMembership(HouseMembership{
		PersonID: person.ID, TenantSlug: "haus-b", Role: RoleRenter, Status: "Aktiv",
	}, now); err != nil {
		t.Fatalf("membership b: %v", err)
	}

	if got := s.ListPersons(); len(got) != 1 {
		t.Fatalf("two houses must not create two persons: %d", len(got))
	}
	if got := s.MembershipsForPerson(person.ID); len(got) != 2 {
		t.Fatalf("expected 2 memberships, got %d", len(got))
	}
	// Each house sees only its own member list.
	demo, _ := BindIdentityRepository(s, "demo")
	hausBRepo, _ := BindIdentityRepository(s, "haus-b")
	if got := demo.ListHouseMembers(); len(got) != 1 || got[0].Membership.Role != RoleOwner {
		t.Fatalf("demo members = %+v", got)
	}
	if got := hausBRepo.ListHouseMembers(); len(got) != 1 || got[0].Membership.Role != RoleRenter {
		t.Fatalf("haus-b members = %+v", got)
	}
	// A membership needs a real person.
	if _, err := s.SetMembership(HouseMembership{PersonID: "ghost", TenantSlug: "demo"}, now); err == nil {
		t.Fatal("membership for an unknown person must be rejected")
	}
}

// AC7: editing one house's membership must leave the other byte-identical.
func TestEditingOneMembershipLeavesOthersUntouched(t *testing.T) {
	s := newIdentityStore(t)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	person, _ := s.UpsertPerson(Person{Email: "anna@example.com"}, now)
	if _, err := s.SetMembership(HouseMembership{
		PersonID: person.ID, TenantSlug: "demo", Role: RoleOwner,
		Permissions: []string{PermissionParking}, Status: "Aktiv",
	}, now); err != nil {
		t.Fatalf("membership a: %v", err)
	}
	other, err := s.SetMembership(HouseMembership{
		PersonID: person.ID, TenantSlug: "haus-b", Role: RoleRenter, Status: "Eingeladen",
	}, now)
	if err != nil {
		t.Fatalf("membership b: %v", err)
	}

	// House demo changes role AND permissions.
	if _, err := s.SetMembership(HouseMembership{
		PersonID: person.ID, TenantSlug: "demo", Role: RoleManager, Status: "Aktiv",
	}, now.Add(time.Hour)); err != nil {
		t.Fatalf("edit a: %v", err)
	}

	hausBRepo, _ := BindIdentityRepository(s, "haus-b")
	afterEdit, ok := hausBRepo.Membership(person.ID)
	if !ok {
		t.Fatal("the other membership disappeared")
	}
	if !reflect.DeepEqual(afterEdit, other) {
		t.Fatalf("the other house's membership changed:\n before %+v\n after  %+v", other, afterEdit)
	}
}

// AC9: removing a person from one house keeps the person and every other house.
func TestRemoveMembershipKeepsPersonAndOtherHouses(t *testing.T) {
	s := newIdentityStore(t)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	person, _ := s.UpsertPerson(Person{Email: "anna@example.com"}, now)
	_, _ = s.SetMembership(HouseMembership{PersonID: person.ID, TenantSlug: "demo", Role: RoleOwner}, now)
	_, _ = s.SetMembership(HouseMembership{PersonID: person.ID, TenantSlug: "haus-b", Role: RoleRenter}, now)

	demo, _ := BindIdentityRepository(s, "demo")
	hausBRepo, _ := BindIdentityRepository(s, "haus-b")
	removed, err := demo.RemoveMembership(person.ID)
	if err != nil || !removed {
		t.Fatalf("remove: removed=%v err=%v", removed, err)
	}
	if _, ok := s.PersonByEmail("anna@example.com"); !ok {
		t.Fatal("removing a membership must NOT delete the person")
	}
	if _, ok := hausBRepo.Membership(person.ID); !ok {
		t.Fatal("removing one membership must not affect another house")
	}
	if got := demo.ListHouseMembers(); len(got) != 0 {
		t.Fatalf("demo should have no members left: %+v", got)
	}
	// Removing again is a no-op, not an error.
	if removed, err := demo.RemoveMembership(person.ID); err != nil || removed {
		t.Fatalf("second remove: removed=%v err=%v", removed, err)
	}

	// Deleting the person is a separate, deliberate act that cascades.
	if deleted, err := s.DeletePerson(person.ID); err != nil || !deleted {
		t.Fatalf("delete person: deleted=%v err=%v", deleted, err)
	}
	if got := s.MembershipsForPerson(person.ID); len(got) != 0 {
		t.Fatalf("deleting a person must cascade its memberships: %+v", got)
	}
}

// AC10: a global email change keeps every membership and cannot take over
// another person's relationships.
func TestChangePersonEmailKeepsMembershipsAndCannotCollide(t *testing.T) {
	s := newIdentityStore(t)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	anna, _ := s.UpsertPerson(Person{Email: "anna@example.com"}, now)
	_, _ = s.SetMembership(HouseMembership{PersonID: anna.ID, TenantSlug: "demo", Role: RoleOwner}, now)
	_, _ = s.SetMembership(HouseMembership{PersonID: anna.ID, TenantSlug: "haus-b", Role: RoleRenter}, now)
	bob, _ := s.UpsertPerson(Person{Email: "bob@example.com"}, now)
	_, _ = s.SetMembership(HouseMembership{PersonID: bob.ID, TenantSlug: "demo", Role: RoleManager}, now)

	// Cannot take over an existing person's email.
	if _, err := s.ChangePersonEmail(anna.ID, "bob@example.com", now.Add(time.Hour)); err == nil {
		t.Fatal("changing to an email that belongs to someone else must be refused")
	}
	if got, _ := s.PersonByEmail("bob@example.com"); got.ID != bob.ID {
		t.Fatalf("bob's identity was affected: %+v", got)
	}

	moved, err := s.ChangePersonEmail(anna.ID, "anna.neu@example.com", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("email change: %v", err)
	}
	if moved.ID != anna.ID {
		t.Fatalf("email change must keep the person id: %q vs %q", moved.ID, anna.ID)
	}
	if got := s.MembershipsForPerson(anna.ID); len(got) != 2 {
		t.Fatalf("email change lost memberships: %+v", got)
	}
	if _, ok := s.PersonByEmail("anna@example.com"); ok {
		t.Fatal("the old email must no longer resolve")
	}
}

// AC12: existing profiles migrate losslessly and idempotently.
func TestImportProfilesIsLosslessAndIdempotent(t *testing.T) {
	jsonStore, err := NewInviteStore(filepath.Join(t.TempDir(), "invites.json"))
	if err != nil {
		t.Fatalf("invite store: %v", err)
	}
	// A multi-house person whose role differs per house — exactly the case the
	// old whole-profile CRUD could corrupt.
	if _, err := jsonStore.Add(UserProfile{
		Email: "anna@example.com", Title: "Dr.", FirstName: "Anna", LastName: "Muster",
		Role: RoleRenter, Status: "Eingeladen", Tenants: []string{"demo", "haus-b"},
		Permissions: []string{PermissionParking},
		TenantMemberships: map[string]TenantMembership{
			"haus-b": {Role: RoleOwner},
		},
		AuthMethods: []string{"email"}, Deactivated: true, Adopted: true,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	s := newIdentityStore(t)
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		if err := s.ImportProfiles(jsonStore, now); err != nil {
			t.Fatalf("import %d: %v", i, err)
		}
	}

	if got := s.ListPersons(); len(got) != 1 {
		t.Fatalf("import must be idempotent, got %d persons", len(got))
	}
	person, ok := s.PersonByEmail("anna@example.com")
	if !ok {
		t.Fatal("person missing after import")
	}
	// Global identity landed on the person.
	if person.Title != "Dr." || person.FirstName != "Anna" || !person.Deactivated || !person.Adopted {
		t.Fatalf("global identity lost: %+v", person)
	}
	// Both houses became memberships, each with its OWN resolved role.
	memberships := s.MembershipsForPerson(person.ID)
	if len(memberships) != 2 {
		t.Fatalf("expected 2 memberships, got %+v", memberships)
	}
	demo, _ := BindIdentityRepository(s, "demo")
	hausBRepo, _ := BindIdentityRepository(s, "haus-b")
	demoMembership, ok := demo.Membership(person.ID)
	if !ok || demoMembership.Role != RoleRenter {
		t.Fatalf("demo membership should inherit the default role: %+v", demoMembership)
	}
	hausB, ok := hausBRepo.Membership(person.ID)
	if !ok || hausB.Role != RoleOwner {
		t.Fatalf("haus-b membership should use its per-tenant override: %+v", hausB)
	}
	if demoMembership.Status != "Eingeladen" {
		t.Fatalf("membership status lost: %+v", demoMembership)
	}
}

// A house listed only in TenantMemberships (not repeated in Tenants) must still
// become a membership.
func TestImportPicksUpMembershipOnlyHouses(t *testing.T) {
	jsonStore, err := NewInviteStore(filepath.Join(t.TempDir(), "invites.json"))
	if err != nil {
		t.Fatalf("invite store: %v", err)
	}
	if _, err := jsonStore.Add(UserProfile{
		Email: "solo@example.com", Role: RoleOwner,
		TenantMemberships: map[string]TenantMembership{"haus-c": {Role: RoleManager}},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	s := newIdentityStore(t)
	if err := s.ImportProfiles(jsonStore, time.Now()); err != nil {
		t.Fatalf("import: %v", err)
	}
	person, ok := s.PersonByEmail("solo@example.com")
	if !ok {
		t.Fatal("person missing")
	}
	hausC, _ := BindIdentityRepository(s, "haus-c")
	if got, ok := hausC.Membership(person.ID); !ok || got.Role != RoleManager {
		t.Fatalf("membership-only house was dropped: %+v ok=%v", got, ok)
	}
}
