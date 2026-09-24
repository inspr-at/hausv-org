package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
)

const memberEmail = "employee@example.com"

func membershipFixture(t *testing.T) (*OrganisationMembershipService, *SQLIdentityStore, *sql.DB) {
	t.Helper()
	database, lanes := testLanes(t)
	identity := NewSQLIdentityStore(lanes)
	optIn := true
	_, err := identity.Add(UserProfile{Email: memberEmail, Role: RoleOwner, Tenants: []string{"demo"}, Status: "Eingeladen", TenantMemberships: map[string]TenantMembership{"demo": {Role: RoleOwner, Permissions: []string{PermissionParking}, DirectoryOptIn: &optIn}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := BindOrganisationRepository(database, "org-a").Save(t.Context(), Organisation{Name: "A", Houses: []string{"demo", "haus-a"}}); err != nil {
		t.Fatal(err)
	}
	if err := BindOrganisationRepository(database, "org-b").Save(t.Context(), Organisation{Name: "B", Houses: []string{"haus-b"}}); err != nil {
		t.Fatal(err)
	}
	s, err := NewOrganisationMembershipService(BindOrganisationMemberRepository(database, "org-a"), identity)
	if err != nil {
		t.Fatal(err)
	}
	return s, identity, database
}

func membershipState(t *testing.T, database *sql.DB) string {
	t.Helper()
	state := map[string][]string{}
	for table, query := range map[string]string{
		"person": `SELECT id,email,title,first_name,last_name,auth_methods,deactivated,adopted,created_at,updated_at FROM persons ORDER BY id`,
		"house":  `SELECT person_id,tenant_id,tenant_slug,role,permissions,status,directory_opt_in,created_at,updated_at,grant_origin FROM house_memberships ORDER BY person_id,tenant_id`,
		"member": `SELECT org_key,email,role,granted,undo,created_at,updated_at FROM organisation_members ORDER BY org_key,email`,
		"audit":  `SELECT org_key,id,events,created_at FROM organisation_membership_audit ORDER BY org_key,id`,
	} {
		rows, err := database.Query(query)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]any, len(columns))
			dest := make([]any, len(columns))
			for i := range values {
				dest[i] = &values[i]
			}
			if err := rows.Scan(dest...); err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(values)
			if err != nil {
				t.Fatal(err)
			}
			state[table] = append(state[table], string(data))
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestOrganisationMembershipAtomicAtEveryStage(t *testing.T) {
	for _, operation := range []string{"add", "role", "remove"} {
		for _, stage := range []string{"locked", "house:demo", "house:haus-a", "houses", "member", "undo", "audit", "commit"} {
			t.Run(operation+"/"+stage, func(t *testing.T) {
				s, _, database := membershipFixture(t)
				if operation != "add" {
					if _, err := s.Add(t.Context(), memberEmail, OrganisationRoleClerk, AuditEvent{}); err != nil {
						t.Fatal(err)
					}
				}
				before := membershipState(t, database)
				injected := errors.New("injected " + stage)
				s.fault = func(at string) error {
					if at == stage {
						return injected
					}
					return nil
				}
				var events []AuditEvent
				var err error
				switch operation {
				case "add":
					events, err = s.Add(t.Context(), memberEmail, OrganisationRoleAdmin, AuditEvent{})
				case "role":
					events, err = s.ChangeRole(t.Context(), memberEmail, OrganisationRoleAdmin, AuditEvent{})
				case "remove":
					events, err = s.Remove(t.Context(), memberEmail, AuditEvent{})
				}
				if !errors.Is(err, injected) || len(events) != 0 {
					t.Fatalf("events=%v error=%v", events, err)
				}
				if after := membershipState(t, database); after != before {
					t.Fatalf("partial state survived %s", stage)
				}
			})
		}
	}
}

func TestOrganisationMembershipRoundTripAndAudit(t *testing.T) {
	s, identity, database := membershipFixture(t)
	person, _ := identity.PersonByEmail(memberEmail)
	before, _ := identity.membershipBySlug(person.ID, "demo")
	for _, role := range []string{OrganisationRoleClerk, OrganisationRoleAdmin, OrganisationRoleClerk} {
		if _, err := s.Add(t.Context(), memberEmail, role, AuditEvent{ActorEmail: "admin@example.com", ActorRole: RoleAdmin}); err != nil {
			t.Fatal(err)
		}
		got, _ := identity.membershipBySlug(person.ID, "demo")
		if !reflect.DeepEqual(got.Permissions, before.Permissions) {
			t.Fatalf("grant discarded explicit permissions: %v", got.Permissions)
		}
	}
	if _, err := s.Remove(t.Context(), memberEmail, AuditEvent{ActorEmail: "admin@example.com"}); err != nil {
		t.Fatal(err)
	}
	after, ok := identity.membershipBySlug(person.ID, "demo")
	after.UpdatedAt = before.UpdatedAt
	if !ok || !reflect.DeepEqual(after, before) {
		t.Fatalf("undo = %+v, want %+v", after, before)
	}
	if _, ok := identity.membershipBySlug(person.ID, "haus-a"); ok {
		t.Fatal("derived membership survived removal")
	}
	if _, ok := identity.PersonByEmail(memberEmail); !ok {
		t.Fatal("removal deleted global identity")
	}
	events, err := s.AuditEvents(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 8 {
		t.Fatalf("events=%d, want 4 committed operations x 2 houses", len(events))
	}
	for _, event := range events {
		if event.TenantSlug == "haus-b" || event.Details["org_key"] != "org-a" {
			t.Fatalf("audit escaped organisation: %+v", event)
		}
	}
	var other int
	if err := database.QueryRow(`SELECT count(*) FROM house_memberships WHERE tenant_slug='haus-b'`).Scan(&other); err != nil || other != 0 {
		t.Fatalf("foreign grants=%d, %v", other, err)
	}
}

func TestOrganisationMembershipManualChangesSurvive(t *testing.T) {
	for _, edit := range []string{"same-role", "other-role", "permissions", "delete", "recreate"} {
		t.Run(edit, func(t *testing.T) {
			s, identity, _ := membershipFixture(t)
			if _, err := s.Add(t.Context(), memberEmail, OrganisationRoleClerk, AuditEvent{}); err != nil {
				t.Fatal(err)
			}
			person, _ := identity.PersonByEmail(memberEmail)
			var err error
			switch edit {
			case "same-role":
				_, _, err = identity.SetTenantMembership(memberEmail, "haus-a", RoleManager, []string{PermissionParking})
			case "other-role":
				_, _, err = identity.SetTenantMembership(memberEmail, "haus-a", RoleOwner, nil)
			case "permissions":
				_, _, err = identity.MutateTenantPermissions(memberEmail, "haus-a", func([]string) []string { return []string{PermissionParking} })
			case "delete", "recreate":
				_, err = identity.removeMembershipBySlug(person.ID, "haus-a")
				if edit == "recreate" && err == nil {
					_, _, err = identity.SetTenantMembership(memberEmail, "haus-a", RoleManager, nil)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			want, existed := identity.membershipBySlug(person.ID, "haus-a")
			if _, err := s.ChangeRole(t.Context(), memberEmail, OrganisationRoleAdmin, AuditEvent{}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Remove(t.Context(), memberEmail, AuditEvent{}); err != nil {
				t.Fatal(err)
			}
			got, exists := identity.membershipBySlug(person.ID, "haus-a")
			if exists != existed || !reflect.DeepEqual(got, want) {
				t.Fatalf("manual edit lost: got %+v/%v, want %+v/%v", got, exists, want, existed)
			}
		})
	}
}

func TestOrganisationMembershipSerializesConcurrentAddRemove(t *testing.T) {
	for _, first := range []string{"add", "remove"} {
		t.Run(first, func(t *testing.T) {
			s, identity, database := membershipFixture(t)
			if first == "remove" {
				if _, err := s.Add(t.Context(), memberEmail, OrganisationRoleClerk, AuditEvent{}); err != nil {
					t.Fatal(err)
				}
			}
			locked, release := make(chan struct{}), make(chan struct{})
			s.fault = func(stage string) error {
				if stage == "locked" {
					close(locked)
					<-release
				}
				return nil
			}
			second, err := NewOrganisationMembershipService(BindOrganisationMemberRepository(database, "org-a"), identity)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			firstDone, secondDone := make(chan error, 1), make(chan error, 1)
			go func() {
				var err error
				if first == "add" {
					_, err = s.Add(ctx, memberEmail, OrganisationRoleClerk, AuditEvent{})
				} else {
					_, err = s.Remove(ctx, memberEmail, AuditEvent{})
				}
				firstDone <- err
			}()
			select {
			case <-locked:
			case err := <-firstDone:
				t.Fatalf("first failed: %v", err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			go func() {
				var err error
				if first == "add" {
					_, err = second.Remove(ctx, memberEmail, AuditEvent{})
				} else {
					_, err = second.Add(ctx, memberEmail, OrganisationRoleAdmin, AuditEvent{})
				}
				secondDone <- err
			}()
			select {
			case err := <-secondDone:
				close(release)
				t.Fatalf("competing operation did not wait: %v", err)
			case <-time.After(100 * time.Millisecond):
			}
			close(release)
			if err := <-firstDone; err != nil {
				t.Fatal(err)
			}
			if err := <-secondDone; err != nil {
				t.Fatal(err)
			}
			_, found, err := BindOrganisationMemberRepository(database, "org-a").Get(ctx, memberEmail)
			if err != nil || found != (first == "remove") {
				t.Fatalf("member after %s = %v, %v", first, found, err)
			}
			person, _ := identity.PersonByEmail(memberEmail)
			house, exists := identity.membershipBySlug(person.ID, "haus-a")
			if exists != found || (exists && house.Role != RoleAdmin) {
				t.Fatalf("house inconsistent: %+v / %v", house, exists)
			}
		})
	}
}

func TestOrganisationMembershipRejectsLegacyAndCorruptUndo(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(fmt.Sprint(corrupt), func(t *testing.T) {
			s, _, database := membershipFixture(t)
			if err := BindOrganisationMemberRepository(database, "org-a").Save(t.Context(), OrganisationMember{Email: memberEmail, Role: OrganisationRoleClerk, Granted: map[string]string{"demo": RoleOwner}}); err != nil {
				t.Fatal(err)
			}
			if corrupt {
				if _, err := database.Exec(`UPDATE organisation_members SET undo='broken'`); err != nil {
					t.Fatal(err)
				}
			}
			before := membershipState(t, database)
			if _, err := s.Remove(t.Context(), memberEmail, AuditEvent{}); err == nil {
				t.Fatal("unrestorable undo accepted")
			}
			if after := membershipState(t, database); after != before {
				t.Fatal("failed reconciliation changed state")
			}
		})
	}
}

func TestOrganisationMembershipAuditRLS(t *testing.T) {
	if dbtest.Backend() != appdb.BackendPostgres {
		t.Skip("PostgreSQL RLS")
	}
	s, _, database := membershipFixture(t)
	if _, err := s.Add(t.Context(), memberEmail, OrganisationRoleClerk, AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	// Explicitly override the fixture maintenance lane inside this transaction.
	tx, err := database.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SELECT set_config('hausv.cross_tenant','off',true),set_config('app.org_key','org-b',true)`); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := tx.QueryRow(`SELECT count(*) FROM organisation_membership_audit`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("foreign audit visible: %d %v", count, err)
	}
	_, err = tx.Exec(`INSERT INTO organisation_membership_audit(org_key,id,events,created_at) VALUES('org-a','forbidden','[]','now')`)
	if err == nil || !strings.Contains(err.Error(), "row-level security") {
		t.Fatalf("foreign audit write: %v", err)
	}
}

func TestOrganisationMembershipProfileEditsPreserveOtherHouseOrigins(t *testing.T) {
	s, identity, _ := membershipFixture(t)
	if _, err := s.Add(t.Context(), memberEmail, OrganisationRoleClerk, AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := identity.SetTenantMembership(memberEmail, "demo", RoleOwner, []string{PermissionParking}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := identity.Mutate(memberEmail, func(profile *UserProfile) { profile.FirstName = "Updated" }); err != nil {
		t.Fatal(err)
	}
	if _, err := identity.SetTenantDirectoryOptIn(memberEmail, "haus-a", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Remove(t.Context(), memberEmail, AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	person, _ := identity.PersonByEmail(memberEmail)
	if person.FirstName != "Updated" {
		t.Fatal("global edit lost")
	}
	if _, exists := identity.membershipBySlug(person.ID, "haus-a"); exists {
		t.Fatal("global metadata edit or directory preference severed derived grant origin")
	}
	if got, exists := identity.membershipBySlug(person.ID, "demo"); !exists || got.Role != RoleOwner {
		t.Fatal("manual house edit lost")
	}
}

func TestOrganisationMembershipDetachedHouseIsRevokedAndForeignOriginRefused(t *testing.T) {
	s, identity, database := membershipFixture(t)
	if _, err := s.Add(t.Context(), memberEmail, OrganisationRoleClerk, AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	if err := BindOrganisationRepository(database, "org-b").SetHouses(t.Context(), []string{"haus-a", "haus-b"}); err != nil {
		t.Fatal(err)
	}
	other, err := NewOrganisationMembershipService(BindOrganisationMemberRepository(database, "org-b"), identity)
	if err != nil {
		t.Fatal(err)
	}
	before := membershipState(t, database)
	if _, err := other.Add(t.Context(), memberEmail, OrganisationRoleAdmin, AuditEvent{}); err == nil {
		t.Fatal("foreign origin stolen")
	}
	if after := membershipState(t, database); after != before {
		t.Fatal("conflicting grant partially committed")
	}
	if err := BindOrganisationRepository(database, "org-a").SetHouses(t.Context(), []string{"demo"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Remove(t.Context(), memberEmail, AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	person, _ := identity.PersonByEmail(memberEmail)
	if _, exists := identity.membershipBySlug(person.ID, "haus-a"); exists {
		t.Fatal("detached house retained organisation privileges")
	}
}
