package demo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestDemoOrganisationMembershipRoundTrip(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run(map[bool]string{false: "committed-fixture", true: "explicit-permissions-and-absent-membership"}[custom], func(t *testing.T) {
			database, config := dbtest.OpenWithConfig(t)
			scoped, err := db.NewScoped(config, database)
			if err != nil {
				t.Fatal(err)
			}
			defer scoped.Close()
			identity := store.NewSQLIdentityStore(store.NewTenantDB(scoped))
			members := store.BindOrganisationMemberRepository(database, "musterstadt")
			service, err := store.NewOrganisationMembershipService(members, identity)
			if err != nil {
				t.Fatal(err)
			}
			dir := "../../scripts/demo/seed"
			var org seedOrg
			if err := readJSON(filepath.Join(dir, "org.json"), &org); err != nil {
				t.Fatal(err)
			}
			if custom {
				copyDir := t.TempDir()
				if err := os.CopyFS(copyDir, os.DirFS(dir)); err != nil {
					t.Fatal(err)
				}
				dir = copyDir
				optIn := true
				org.Members[1].Previous["janusbergweg-123"] = &seedMemberMembership{
					Role: store.RoleOwner, Permissions: []string{store.PermissionParking}, Status: "Eingeladen", DirectoryOptIn: &optIn,
				}
				org.Members[1].Previous["musterstrasse-12"] = nil
				data, err := json.Marshal(org)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "org.json"), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			options := SeedOptions{DocumentDir: t.TempDir()}
			for _, reset := range []bool{false, false, true} {
				options.Reset, options.DiscardAnnualStatements = reset, reset
				if _, err := Load(t.Context(), database, dir, options); err != nil {
					t.Fatal(err)
				}
				for _, fixture := range org.Members {
					person, found := identity.PersonByEmail(fixture.Email)
					if !found {
						t.Fatal("seed did not create employee identity")
					}
					member, found, err := members.Get(t.Context(), fixture.Email)
					if err != nil || !found || len(member.Undo) != 13 || len(member.Granted) != 13 {
						t.Fatalf("seeded member: %+v %v", member, err)
					}
					for id, undo := range member.Undo {
						var origin string
						if err := dbtest.MaintenanceView(t, scoped).QueryRow(`SELECT grant_origin FROM house_memberships WHERE person_id=$1 AND tenant_id=$2`, person.ID, id).Scan(&origin); err != nil || origin != org.Key {
							t.Fatalf("seeded origin for %s: %q %v", undo.TenantSlug, origin, err)
						}
					}
					// Changing a seeded employee must retain the original snapshot.
					if _, err := service.ChangeRole(t.Context(), fixture.Email, store.OrganisationRoleAdmin, store.AuditEvent{}); err != nil {
						t.Fatal(err)
					}
					for cycle := 0; cycle < 2; cycle++ {
						if _, err := service.Remove(t.Context(), fixture.Email, store.AuditEvent{}); err != nil {
							t.Fatalf("remove seeded/re-added employee: %v", err)
						}
						for id, undo := range member.Undo {
							repo, _ := store.BindIdentityRepository(identity, store.TenantRef{ID: id, Slug: undo.TenantSlug})
							got, exists := repo.Membership(person.ID)
							want := fixture.Previous[undo.TenantSlug]
							if want == nil {
								if exists {
									t.Fatalf("organisation-created membership survived removal: %s", undo.TenantSlug)
								}
								continue
							}
							if !exists || got.Role != want.Role || len(got.Permissions) != len(want.Permissions) || (len(want.Permissions) > 0 && !reflect.DeepEqual(got.Permissions, want.Permissions)) || got.Status != want.Status || !reflect.DeepEqual(got.DirectoryOptIn, want.DirectoryOptIn) {
								t.Fatalf("restored %s: %+v, want %+v", undo.TenantSlug, got, want)
							}
						}
						if _, err := service.Add(t.Context(), fixture.Email, fixture.Role, store.AuditEvent{}); err != nil {
							t.Fatalf("re-add employee: %v", err)
						}
					}
				}
			}
		})
	}
}

func TestDemoOrganisationMemberSeedRollsBackHouseGrants(t *testing.T) {
	database, config := dbtest.OpenWithConfig(t)
	scoped, err := db.NewScoped(config, database)
	if err != nil {
		t.Fatal(err)
	}
	defer scoped.Close()
	identity := store.NewSQLIdentityStore(store.NewTenantDB(scoped))
	dir := "../../scripts/demo/seed"
	if _, err := Load(t.Context(), database, dir, SeedOptions{DocumentDir: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	var org seedOrg
	if err := readJSON(filepath.Join(dir, "org.json"), &org); err != nil {
		t.Fatal(err)
	}
	fixture := org.Members[1]
	person, _ := identity.PersonByEmail(fixture.Email)
	before := identity.MembershipsForPerson(person.ID)
	repo := store.BindOrganisationMemberRepository(database, org.Key)
	memberBefore, _, err := repo.Get(t.Context(), fixture.Email)
	if err != nil {
		t.Fatal(err)
	}
	houses := map[string]store.TenantIdentity{}
	for _, membership := range before {
		houses[membership.TenantSlug] = store.TenantIdentity{ID: membership.TenantID, Slug: membership.TenantSlug}
	}
	fixture.Role = store.OrganisationRoleAdmin
	fixture.Previous["janusbergweg-123"].Permissions = []string{store.PermissionParking}
	// Fail the final member-row write, after every house has been replaced.
	queries := []string{`CREATE TRIGGER reject_fixture_member BEFORE UPDATE ON organisation_members BEGIN SELECT RAISE(ABORT, 'reject fixture member'); END`}
	if config.Backend == db.BackendPostgres {
		queries = []string{
			`CREATE FUNCTION reject_fixture_member() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'reject fixture member'; END $$`,
			`CREATE TRIGGER reject_fixture_member BEFORE UPDATE ON organisation_members FOR EACH ROW EXECUTE FUNCTION reject_fixture_member()`,
		}
	}
	for _, query := range queries {
		if _, err := database.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	err = seedOrganisationMember(t.Context(), database, org.Key, fixture, map[string]string{fixture.Email: "Paul Sommer"}, houses)
	if err == nil || !strings.Contains(err.Error(), "reject fixture member") {
		t.Fatalf("expected final write failure, got %v", err)
	}
	if after := identity.MembershipsForPerson(person.ID); !reflect.DeepEqual(before, after) {
		t.Fatalf("failed seed changed house grants: before=%+v after=%+v", before, after)
	}
	memberAfter, _, err := repo.Get(t.Context(), fixture.Email)
	if err != nil || !reflect.DeepEqual(memberBefore, memberAfter) {
		t.Fatalf("failed seed changed member/undo: %+v %v", memberAfter, err)
	}
}
