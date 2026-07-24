package store

import (
	"path/filepath"
	"testing"

	"github.com/markus-barta/hausv-org/internal/db"
)

// HAUSV-169 phase 3: the JSON profile store and the SQLite person/house model
// must behave identically through the surface the server uses — and BOTH must
// make a house-scoped edit or removal stop at that house.

func profileBackends() map[string]func(t *testing.T) ProfileStorage {
	return map[string]func(t *testing.T) ProfileStorage{
		"json": func(t *testing.T) ProfileStorage {
			s, err := NewInviteStore(filepath.Join(t.TempDir(), "invites.json"))
			if err != nil {
				t.Fatalf("json store: %v", err)
			}
			return s
		},
		"sqlite": func(t *testing.T) ProfileStorage {
			database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("db open: %v", err)
			}
			t.Cleanup(func() { database.Close() })
			return NewSQLIdentityStore(database)
		},
	}
}

func twoHouseProfile() UserProfile {
	return UserProfile{
		Email: "anna@example.com", Title: "Dr.", FirstName: "Anna", LastName: "Muster",
		Role: RoleRenter, Status: "Eingeladen",
		Tenants:     []string{"jhw22", "haus-b"},
		AuthMethods: []string{"email"},
		TenantMemberships: map[string]TenantMembership{
			"haus-b": {Role: RoleOwner},
		},
	}
}

func TestProfileStorageParity(t *testing.T) {
	for name, build := range profileBackends() {
		t.Run(name, func(t *testing.T) {
			s := build(t)

			added, err := s.Add(twoHouseProfile())
			if err != nil || !added {
				t.Fatalf("add: added=%v err=%v", added, err)
			}
			// Adding the same email again is refused without an error.
			if added, err := s.Add(twoHouseProfile()); err != nil || added {
				t.Fatalf("duplicate add: added=%v err=%v", added, err)
			}

			got, ok := s.Get("anna@example.com")
			if !ok {
				t.Fatal("profile not found")
			}
			if got.Title != "Dr." || got.FirstName != "Anna" {
				t.Fatalf("identity lost: %+v", got)
			}
			// Per-house resolution is what the app actually consumes.
			if r := got.ForTenant("jhw22").Role; r != RoleRenter {
				t.Fatalf("jhw22 role = %q, want %q", r, RoleRenter)
			}
			if r := got.ForTenant("haus-b").Role; r != RoleOwner {
				t.Fatalf("haus-b role = %q, want %q", r, RoleOwner)
			}
			if !got.HasTenant("jhw22") || !got.HasTenant("haus-b") {
				t.Fatalf("tenants lost: %+v", got.Tenants)
			}
			if got.HasTenant("haus-c") {
				t.Fatal("must not claim an unrelated house")
			}
			if len(s.List()) != 1 {
				t.Fatalf("list = %d", len(s.List()))
			}
			if _, ok := s.Get("nobody@example.com"); ok {
				t.Fatal("unknown email must not resolve")
			}
		})
	}
}

// The defect HAUSV-135 describes: a house admin's edit must not change the
// person's role in a house they do not administer.
func TestSetTenantMembershipDoesNotLeakToOtherHouseParity(t *testing.T) {
	for name, build := range profileBackends() {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			if _, err := s.Add(twoHouseProfile()); err != nil {
				t.Fatalf("add: %v", err)
			}

			// House jhw22 promotes Anna to manager, in its own house only.
			updated, ok, err := s.SetTenantMembership("anna@example.com", "jhw22", RoleManager, []string{PermissionParking})
			if err != nil || !ok {
				t.Fatalf("set membership: ok=%v err=%v", ok, err)
			}
			if r := updated.ForTenant("jhw22").Role; r != RoleManager {
				t.Fatalf("jhw22 role not applied: %q", r)
			}
			// The other house is unaffected.
			if r := updated.ForTenant("haus-b").Role; r != RoleOwner {
				t.Fatalf("haus-b role leaked: %q, want %q", r, RoleOwner)
			}
			// And it survives a re-read.
			reread, _ := s.Get("anna@example.com")
			if r := reread.ForTenant("haus-b").Role; r != RoleOwner {
				t.Fatalf("haus-b role leaked after re-read: %q", r)
			}
			if r := reread.ForTenant("jhw22").Role; r != RoleManager {
				t.Fatalf("jhw22 role lost after re-read: %q", r)
			}
			// Global identity untouched by a house-scoped edit.
			if reread.Title != "Dr." || reread.FirstName != "Anna" {
				t.Fatalf("house edit changed global identity: %+v", reread)
			}
			if _, ok, _ := s.SetTenantMembership("nobody@example.com", "jhw22", RoleManager, nil); ok {
				t.Fatal("unknown person must not be found")
			}
		})
	}
}

// Removing someone from one house must keep them in the others; only the last
// house removes the record entirely.
func TestRemoveTenantIsHouseScopedParity(t *testing.T) {
	for name, build := range profileBackends() {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			if _, err := s.Add(twoHouseProfile()); err != nil {
				t.Fatalf("add: %v", err)
			}

			removedProfile, found, err := s.RemoveTenant("anna@example.com", "jhw22")
			if err != nil || !found {
				t.Fatalf("remove: found=%v err=%v", found, err)
			}
			if removedProfile {
				t.Fatal("removing one of two houses must NOT delete the person")
			}
			after, ok := s.Get("anna@example.com")
			if !ok {
				t.Fatal("person disappeared after a house-scoped removal")
			}
			if after.HasTenant("jhw22") {
				t.Fatal("jhw22 membership should be gone")
			}
			if !after.HasTenant("haus-b") || after.ForTenant("haus-b").Role != RoleOwner {
				t.Fatalf("the other house was damaged: %+v", after)
			}

			// Removing the LAST house removes the record, matching the old delete.
			removedProfile, found, err = s.RemoveTenant("anna@example.com", "haus-b")
			if err != nil || !found || !removedProfile {
				t.Fatalf("last house: removedProfile=%v found=%v err=%v", removedProfile, found, err)
			}
			if _, ok := s.Get("anna@example.com"); ok {
				t.Fatal("record should be gone once no house is left")
			}
			if _, found, _ := s.RemoveTenant("anna@example.com", "haus-b"); found {
				t.Fatal("removing from an unknown person must report not found")
			}
		})
	}
}

func TestProfileUpdateDeleteMutateParity(t *testing.T) {
	for name, build := range profileBackends() {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			if _, err := s.Add(twoHouseProfile()); err != nil {
				t.Fatalf("add: %v", err)
			}

			// Mutate applies a field change without disturbing houses.
			mutated, ok, err := s.Mutate("anna@example.com", func(p *UserProfile) {
				p.LastName = "Neu"
			})
			if err != nil || !ok || mutated.LastName != "Neu" {
				t.Fatalf("mutate: ok=%v err=%v profile=%+v", ok, err, mutated)
			}
			if mutated.ForTenant("haus-b").Role != RoleOwner {
				t.Fatalf("mutate disturbed a membership: %+v", mutated)
			}

			// A global rename keeps both houses.
			renamed := twoHouseProfile()
			renamed.Email = "anna.neu@example.com"
			renamed.LastName = "Neu"
			changed, err := s.Update("anna@example.com", renamed)
			if err != nil || !changed {
				t.Fatalf("update: changed=%v err=%v", changed, err)
			}
			moved, ok := s.Get("anna.neu@example.com")
			if !ok {
				t.Fatal("renamed profile not found")
			}
			if !moved.HasTenant("jhw22") || !moved.HasTenant("haus-b") {
				t.Fatalf("rename lost houses: %+v", moved.Tenants)
			}
			if _, ok := s.Get("anna@example.com"); ok {
				t.Fatal("old email must no longer resolve")
			}
			if changed, err := s.Update("ghost@example.com", renamed); err != nil || changed {
				t.Fatalf("updating an unknown profile: changed=%v err=%v", changed, err)
			}

			// Global delete removes everything.
			if removed, err := s.Delete("anna.neu@example.com"); err != nil || !removed {
				t.Fatalf("delete: removed=%v err=%v", removed, err)
			}
			if removed, err := s.Delete("anna.neu@example.com"); err != nil || removed {
				t.Fatalf("second delete: removed=%v err=%v", removed, err)
			}
		})
	}
}

// Revoking a permission must be visible BOTH through per-house resolution and on
// the raw profile. The flat fields going stale after a house-scoped edit is a
// trap: ForTenant would still resolve correctly, so any code reading the raw
// profile would silently see a permission that was just revoked.
func TestRevokedPermissionIsNotVisibleOnRawProfileParity(t *testing.T) {
	for name, build := range profileBackends() {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			if _, err := s.Add(UserProfile{
				Email: "parker@example.com", Role: RoleRenter,
				Tenants: []string{"jhw22"}, Permissions: []string{PermissionParking},
			}); err != nil {
				t.Fatalf("add: %v", err)
			}
			if got, _ := s.Get("parker@example.com"); !got.HasPermission(PermissionParking) {
				t.Fatalf("precondition: permission should be granted, got %+v", got)
			}

			// Revoke it for that house.
			if _, ok, err := s.SetTenantMembership("parker@example.com", "jhw22", RoleRenter, nil); err != nil || !ok {
				t.Fatalf("revoke: ok=%v err=%v", ok, err)
			}
			got, ok := s.Get("parker@example.com")
			if !ok {
				t.Fatal("profile gone")
			}
			if got.ForTenant("jhw22").HasPermission(PermissionParking) {
				t.Fatalf("permission still effective after revoke: %+v", got)
			}
			if got.HasPermission(PermissionParking) {
				t.Fatalf("raw profile still reports the revoked permission: %+v", got)
			}
		})
	}
}

// The legacy data shape: houses listed in Tenants with NO explicit membership,
// so every house resolves through the flat top-level role. This is the shape
// that actually leaks — a house-scoped edit must still stop at its own house.
func TestHouseScopedEditOnLegacyShapeDoesNotLeakParity(t *testing.T) {
	legacy := func() UserProfile {
		return UserProfile{
			Email: "anna@example.com", Role: RoleRenter,
			Tenants: []string{"jhw22", "haus-b"},
		}
	}
	for name, build := range profileBackends() {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			if _, err := s.Add(legacy()); err != nil {
				t.Fatalf("add: %v", err)
			}
			// Precondition: both houses currently resolve to the flat default.
			before, _ := s.Get("anna@example.com")
			if before.ForTenant("haus-b").Role != RoleRenter {
				t.Fatalf("precondition: haus-b = %q", before.ForTenant("haus-b").Role)
			}

			if _, ok, err := s.SetTenantMembership("anna@example.com", "jhw22", RoleManager, nil); err != nil || !ok {
				t.Fatalf("scoped edit: ok=%v err=%v", ok, err)
			}
			after, _ := s.Get("anna@example.com")
			if got := after.ForTenant("jhw22").Role; got != RoleManager {
				t.Fatalf("jhw22 role = %q, want %q", got, RoleManager)
			}
			if got := after.ForTenant("haus-b").Role; got != RoleRenter {
				t.Fatalf("haus-b role leaked to %q — it had no explicit membership", got)
			}
		})
	}
}

// Same shape, for removal: detaching one house must not disturb the other.
func TestHouseScopedRemovalOnLegacyShapeParity(t *testing.T) {
	for name, build := range profileBackends() {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			if _, err := s.Add(UserProfile{
				Email: "anna@example.com", Role: RoleOwner,
				Tenants: []string{"jhw22", "haus-b"},
			}); err != nil {
				t.Fatalf("add: %v", err)
			}
			removedProfile, found, err := s.RemoveTenant("anna@example.com", "jhw22")
			if err != nil || !found || removedProfile {
				t.Fatalf("remove: removedProfile=%v found=%v err=%v", removedProfile, found, err)
			}
			after, ok := s.Get("anna@example.com")
			if !ok {
				t.Fatal("person removed globally")
			}
			if after.HasTenant("jhw22") {
				t.Fatal("jhw22 should be detached")
			}
			if got := after.ForTenant("haus-b").Role; got != RoleOwner {
				t.Fatalf("haus-b role damaged: %q", got)
			}
		})
	}
}

// HAUSV-178: the contact directory is rendered per house, so visibility is a
// per-membership choice. Unset means inherit, so nothing changes for people who
// never touched it.
func TestDirectoryVisibilityIsPerHouseParity(t *testing.T) {
	for name, build := range profileBackends() {
		t.Run(name, func(t *testing.T) {
			s := build(t)
			if _, err := s.Add(twoHouseProfile()); err != nil {
				t.Fatalf("add: %v", err)
			}

			// Unset: both houses inherit whatever the person-wide value is.
			got, _ := s.Get("anna@example.com")
			for _, house := range []string{"jhw22", "haus-b"} {
				resolved := got
				resolved.DirectoryOptIn = true // person-wide value, as the overlay supplies it
				if !resolved.ForTenant(house).DirectoryOptIn {
					t.Fatalf("%s should inherit the person-wide value while unset", house)
				}
			}

			// Opt out in jhw22 only.
			if found, err := s.SetTenantDirectoryOptIn("anna@example.com", "jhw22", false); err != nil || !found {
				t.Fatalf("set: found=%v err=%v", found, err)
			}
			got, _ = s.Get("anna@example.com")
			withPersonWideOptIn := got
			withPersonWideOptIn.DirectoryOptIn = true
			if withPersonWideOptIn.ForTenant("jhw22").DirectoryOptIn {
				t.Fatal("jhw22 should now be hidden despite the person-wide opt-in")
			}
			if !withPersonWideOptIn.ForTenant("haus-b").DirectoryOptIn {
				t.Fatal("haus-b must still inherit the person-wide opt-in")
			}

			// A later role edit must not reset the visibility choice.
			if _, ok, err := s.SetTenantMembership("anna@example.com", "jhw22", RoleManager, nil); err != nil || !ok {
				t.Fatalf("role edit: ok=%v err=%v", ok, err)
			}
			got, _ = s.Get("anna@example.com")
			afterEdit := got
			afterEdit.DirectoryOptIn = true
			if afterEdit.ForTenant("jhw22").DirectoryOptIn {
				t.Fatal("a role edit silently reset the house's directory visibility")
			}

			if found, err := s.SetTenantDirectoryOptIn("nobody@example.com", "jhw22", true); err != nil || found {
				t.Fatalf("unknown person: found=%v err=%v", found, err)
			}
		})
	}
}
