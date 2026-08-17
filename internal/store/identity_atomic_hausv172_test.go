package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
)

// HAUSV-172: multi-part operations must be all-or-nothing. Inviting someone
// creates a person AND their house membership; if the second write fails, the
// first must not survive, or the person is stranded in no house at all.

func identityStoreWithDB(t *testing.T) (*SQLIdentityStore, *sql.DB) {
	t.Helper()
	database := dbtest.Open(t)
	t.Cleanup(func() { database.Close() })
	return NewSQLIdentityStore(database), database
}

// The failure is simulated by removing the memberships table: the person INSERT
// succeeds, the membership INSERT then fails inside the same transaction. That
// is a real mid-operation failure, not a mock.
func TestInviteRollsBackPersonWhenMembershipWriteFails(t *testing.T) {
	s, database := identityStoreWithDB(t)
	if _, err := database.Exec(`DROP TABLE house_memberships`); err != nil {
		t.Fatalf("arrange failure: %v", err)
	}

	added, err := s.Add(UserProfile{
		Email: "anna@example.com", Role: RoleOwner, Tenants: []string{"demo"},
	})
	if err == nil {
		t.Fatal("the invite must fail when the membership cannot be written")
	}
	if added {
		t.Fatal("a failed invite must not report success")
	}

	// The whole point: no half-created person survives.
	var persons int
	if err := database.QueryRow(`SELECT count(*) FROM persons`).Scan(&persons); err != nil {
		t.Fatalf("count persons: %v", err)
	}
	if persons != 0 {
		t.Fatalf("person row survived a failed invite: %d rows — the operation was not atomic", persons)
	}
}

// Same guarantee for an edit: a failing reconcile must not leave the rename or a
// partial membership set behind.
func TestProfileUpdateRollsBackEntirelyOnFailure(t *testing.T) {
	s, database := identityStoreWithDB(t)
	if _, err := s.Add(UserProfile{
		Email: "anna@example.com", Role: RoleOwner, Tenants: []string{"demo"},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := database.Exec(`DROP TABLE house_memberships`); err != nil {
		t.Fatalf("arrange failure: %v", err)
	}

	if _, err := s.Update("anna@example.com", UserProfile{
		Email: "anna.neu@example.com", Role: RoleManager, Tenants: []string{"demo"},
	}); err == nil {
		t.Fatal("the update must fail when memberships cannot be reconciled")
	}

	// The rename must have rolled back with the rest.
	var email string
	if err := database.QueryRow(`SELECT email FROM persons`).Scan(&email); err != nil {
		t.Fatalf("read person: %v", err)
	}
	if email != "anna@example.com" {
		t.Fatalf("email is %q — the rename survived a failed update, so it was not atomic", email)
	}
}

// A successful invite really does write both halves.
func TestInviteCreatesPersonAndMembershipTogether(t *testing.T) {
	s, database := identityStoreWithDB(t)
	if _, err := s.Add(UserProfile{
		Email: "anna@example.com", Role: RoleOwner, Tenants: []string{"demo"},
		Status: "Eingeladen",
	}); err != nil {
		t.Fatalf("invite: %v", err)
	}
	var persons, memberships int
	if err := database.QueryRow(`SELECT count(*) FROM persons`).Scan(&persons); err != nil {
		t.Fatalf("count persons: %v", err)
	}
	if err := database.QueryRow(`SELECT count(*) FROM house_memberships`).Scan(&memberships); err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	if persons != 1 || memberships != 1 {
		t.Fatalf("expected 1 person and 1 membership, got %d/%d", persons, memberships)
	}
	_ = time.Now
}
