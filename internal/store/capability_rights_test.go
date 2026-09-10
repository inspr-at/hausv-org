package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
)

func TestCapabilityRepositorySQLiteHAUSV699(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "rights.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	testCapabilityRepository(t, database)
}

func TestCapabilityRepositoryPostgresHAUSV699(t *testing.T) {
	if strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN")) == "" {
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to verify PostgreSQL rights and RLS")
	}
	t.Setenv("HAUSV_STORE_TEST_POSTGRES", "1")
	database := dbtest.Open(t)
	testCapabilityRepository(t, database)
	for _, table := range []string{"organisation_capability_overrides", "user_capability_grants", "capability_profiles"} {
		var enabled, forced bool
		if err := database.QueryRow(`SELECT relrowsecurity,relforcerowsecurity FROM pg_class WHERE oid=$1::regclass`, table).Scan(&enabled, &forced); err != nil || !enabled || !forced {
			t.Fatalf("RLS %s: %v/%v %v", table, enabled, forced, err)
		}
		var count int
		if err := database.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("unscoped %s = %d, %v", table, count, err)
		}
	}
	tx, err := beginOrgTx(t.Context(), database, "other")
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO capability_profiles(org_key,name,capabilities) VALUES('org','intrusion','[]')`); err == nil {
		t.Fatal("cross-organisation insert passed RLS")
	}
}

func testCapabilityRepository(t *testing.T, database *sql.DB) {
	t.Helper()
	ctx := t.Context()
	repo := BindCapabilityRepository(database, "org")
	other := BindCapabilityRepository(database, "other")
	initial, err := repo.Get(ctx)
	if err != nil || len(initial.Grants)+len(initial.Overrides)+len(initial.Profiles) != 0 {
		t.Fatalf("initial: %+v %v", initial, err)
	}
	override := CapabilityOverride{RoleFamily: "bewohner", Capability: "manage-documents", Allowed: true, UpdatedBy: "admin@example.com"}
	if err := repo.SetOverride(ctx, override); err != nil {
		t.Fatal(err)
	}
	override.Allowed = false
	if err := repo.SetOverride(ctx, override); err != nil {
		t.Fatal(err)
	}
	grant := UserCapabilityGrant{Email: " PERSON@Example.com ", Capability: "manage-documents", Effect: "grant", UpdatedBy: "admin@example.com"}
	if err := repo.SetGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	grant.Effect = "deny"
	grant.TenantSlug = "house"
	if err := repo.SetGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	rules, err := repo.Get(ctx)
	if err != nil || len(rules.Overrides) != 1 || rules.Overrides[0].Allowed || len(rules.Grants) != 2 {
		t.Fatalf("round trip: %+v %v", rules, err)
	}
	if rules.Overrides[0].OrgKey != "org" || rules.Overrides[0].UpdatedAt == "" || rules.Grants[0].Email != "person@example.com" || rules.Grants[0].UpdatedBy != "admin@example.com" {
		t.Fatalf("metadata: %+v", rules)
	}
	isolated, err := other.Get(ctx)
	if err != nil || len(isolated.Overrides)+len(isolated.Grants)+len(isolated.Profiles) != 0 {
		t.Fatalf("org leak: %+v %v", isolated, err)
	}
	for _, capability := range []string{"manage-users", "platform-admin", "unknown"} {
		override.Capability = capability
		if err := repo.SetOverride(ctx, override); err == nil {
			t.Fatalf("accepted override %s", capability)
		}
		grant.Capability = capability
		if err := repo.SetGrant(ctx, grant); err == nil {
			t.Fatalf("accepted user right %s", capability)
		}
		if err := repo.SaveProfile(ctx, CapabilityProfile{Name: "Bad", Capabilities: []UserCapabilityGrant{{Capability: capability, Effect: "grant"}}}); err == nil {
			t.Fatalf("accepted profile %s", capability)
		}
	}
	if err := repo.ReplaceUserGrants(ctx, "person@example.com", "house", []UserCapabilityGrant{{Capability: "manage-users", Effect: "grant"}}, "admin@example.com"); err == nil {
		t.Fatal("invalid replacement accepted")
	}
	rules, err = repo.Get(ctx)
	if err != nil || len(rules.Grants) != 2 {
		t.Fatal("failed profile application erased grants")
	}
	profile := CapabilityProfile{Name: "Dokumente", Capabilities: []UserCapabilityGrant{{Capability: "manage-documents", Effect: "deny"}, {Capability: "vote", Effect: "grant"}}}
	if err := repo.SaveProfile(ctx, profile); err != nil {
		t.Fatal(err)
	}
	rules, err = repo.Get(ctx)
	if err != nil || len(rules.Profiles) != 1 || len(rules.Profiles[0].Capabilities) != 2 {
		t.Fatalf("profile round trip: %+v %v", rules, err)
	}
	if err := repo.ReplaceUserGrants(ctx, "person@example.com", "house", rules.Profiles[0].Capabilities, "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	rules, err = repo.Get(ctx)
	if err != nil || len(rules.Grants) != 3 {
		t.Fatalf("scoped profile: %+v %v", rules, err)
	}
	if err := repo.ReplaceUserGrants(ctx, "person@example.com", "house", nil, "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	rules, err = repo.Get(ctx)
	if err != nil || len(rules.Grants) != 1 || rules.Grants[0].TenantSlug != "" {
		t.Fatal("standard profile touched global scope")
	}
	if err := repo.ResetOverrides(ctx); err != nil {
		t.Fatal(err)
	}
	rules, err = repo.Get(ctx)
	if err != nil || len(rules.Overrides) != 0 || len(rules.Grants) != 1 || len(rules.Profiles) != 1 {
		t.Fatalf("reset affected unrelated records: %+v %v", rules, err)
	}
	grant.Capability = "manage-documents"
	grant.TenantSlug = ""
	grant.Effect = ""
	if err := repo.SetGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	rules, err = repo.Get(ctx)
	if err != nil || len(rules.Grants) != 0 {
		t.Fatal("remove override failed")
	}
	// Keep one row in every org table for the unscoped PostgreSQL probes.
	override.Capability = "manage-documents"
	if err := repo.SetOverride(ctx, override); err != nil {
		t.Fatal(err)
	}
	grant.Effect = "grant"
	if err := repo.SetGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	if err := BindCapabilityRepository(database, "").SetGrant(ctx, grant); err == nil {
		t.Fatal("empty organisation accepted")
	}
}
