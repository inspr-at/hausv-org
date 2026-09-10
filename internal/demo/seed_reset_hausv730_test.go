package demo

import (
	"path/filepath"
	"testing"

	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

// HAUSV-730: rights a demo visitor configured must not survive the daily
// reseed. A plain load keeps them; a reset returns to the standard matrix.
func TestResetClearsConfiguredRightsHAUSV730(t *testing.T) {
	database := dbtest.Open(t)
	dir := filepath.Join("testdata")
	if _, err := Load(t.Context(), database, dir, SeedOptions{}); err != nil {
		t.Fatal(err)
	}
	var org seedOrg
	if err := readJSON(filepath.Join(dir, "org.json"), &org); err != nil {
		t.Fatal(err)
	}
	rights := store.BindCapabilityRepository(database, org.Key)
	if err := rights.SetOverride(t.Context(), store.CapabilityOverride{RoleFamily: "bewohner", Capability: "manage-documents", Allowed: true, UpdatedBy: "vera@example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := rights.SetGrant(t.Context(), store.UserCapabilityGrant{Email: "gast@example.com", Capability: "manage-announcements", Effect: "deny", UpdatedBy: "vera@example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := rights.SaveProfile(t.Context(), store.CapabilityProfile{Name: "Dokumente", Capabilities: []store.UserCapabilityGrant{{Capability: "manage-documents", Effect: "grant"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(t.Context(), database, dir, SeedOptions{}); err != nil {
		t.Fatal(err)
	}
	kept, err := rights.Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(kept.Overrides) != 1 || len(kept.Grants) != 1 || len(kept.Profiles) != 1 {
		t.Fatalf("a plain load must keep configured rights: %+v", kept)
	}
	if _, err := Load(t.Context(), database, dir, SeedOptions{Reset: true}); err != nil {
		t.Fatal(err)
	}
	cleared, err := rights.Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(cleared.Overrides) != 0 || len(cleared.Grants) != 0 || len(cleared.Profiles) != 0 {
		t.Fatalf("reset left configured rights behind: %+v", cleared)
	}
}
