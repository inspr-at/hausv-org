package demo

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/textutil"
)

// seedOrganisationMembers installs the fixture's declared before/after state,
// never inferring historical permissions from legacy demo rows. Repeating a
// seed (including the settings reset) replaces only its named employees and
// houses. Each member and its complete undo/origin pair commit together.
func seedOrganisationMembers(ctx context.Context, database *sql.DB, dir string, org seedOrg, houses []seedHouse, identities map[string]store.TenantIdentity) error {
	if len(org.Members) == 0 {
		return nil
	}
	var people []seedPerson
	if err := readJSON(filepath.Join(dir, "persons.json"), &people); err != nil {
		return err
	}
	names := map[string]string{}
	for _, person := range people {
		names[textutil.Email(person.Email)] = person.Name
	}
	owned := map[string]store.TenantIdentity{}
	for _, house := range houses {
		if house.Organisation == org.Key {
			owned[house.Slug] = identities[house.Slug]
		}
	}
	for _, member := range org.Members {
		if err := seedOrganisationMember(ctx, database, org.Key, member, names, owned); err != nil {
			return fmt.Errorf("seed member %s: %w", member.Email, err)
		}
	}
	return nil
}

func seedOrganisationMember(ctx context.Context, database *sql.DB, orgKey string, member seedMember, names map[string]string, houses map[string]store.TenantIdentity) error {
	email := textutil.Email(member.Email)
	name, found := names[email]
	if !found || !strings.Contains(email, "@") || len(houses) == 0 || len(member.Previous) != len(houses) {
		return fmt.Errorf("fixture needs a person and a complete previous_memberships map")
	}
	for slug, previous := range member.Previous {
		identity, found := houses[slug]
		if !found || identity.ID == "" {
			return fmt.Errorf("previous membership names unknown fixture house %s", slug)
		}
		if previous != nil && (previous.Role == "" || previous.Permissions == nil || previous.Status == "") {
			return fmt.Errorf("previous membership for %s needs role, explicit permissions and status", slug)
		}
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if strings.Contains(fmt.Sprintf("%T", database.Driver()), "stdlib") {
		if _, err := tx.ExecContext(ctx, `SET LOCAL hausv.cross_tenant = 'on'`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `SELECT set_config('app.org_key',$1,true)`, orgKey); err != nil {
			return err
		}
	}
	at := time.Now().UTC()
	stamp := at.Format(time.RFC3339Nano)
	first, last, _ := strings.Cut(name, " ")
	if _, err := tx.ExecContext(ctx, `INSERT INTO persons(id,email,first_name,last_name,auth_methods,created_at,updated_at)
		VALUES($1,$2,$3,$4,'["email"]',$5,$5) ON CONFLICT(email) DO NOTHING`, rand.Text(), email, first, last, stamp); err != nil {
		return err
	}
	// Same stable-person lock as the service; this also takes SQLite's writer
	// lock before inspecting a house grant.
	if _, err := tx.ExecContext(ctx, `UPDATE persons SET id=id WHERE email=$1`, email); err != nil {
		return err
	}
	var personID string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM persons WHERE email=$1`, email).Scan(&personID); err != nil {
		return err
	}
	role := store.NormalizeOrganisationRole(member.Role)
	houseRole := store.RoleManager
	if role == store.OrganisationRoleAdmin {
		houseRole = store.RoleAdmin
	}
	granted := map[string]string{}
	undo := map[string]store.OrganisationMembershipUndo{}
	for slug, identity := range houses {
		var origin string
		err := tx.QueryRowContext(ctx, `SELECT grant_origin FROM house_memberships WHERE person_id=$1 AND tenant_id=$2`, personID, identity.ID).Scan(&origin)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if origin != "" && origin != orgKey {
			return fmt.Errorf("fixture house %s has a grant from another organisation", slug)
		}
		before := member.Previous[slug]
		snapshot := store.OrganisationMembershipUndo{TenantSlug: slug}
		permissions, status := []string{}, "Aktiv"
		var directory *bool
		granted[slug] = ""
		if before != nil {
			permissions, status, directory = before.Permissions, before.Status, before.DirectoryOptIn
			snapshot.Previous = &store.HouseMembership{
				PersonID: personID, TenantID: identity.ID, TenantSlug: slug,
				Role: before.Role, Permissions: permissions, Status: status,
				DirectoryOptIn: directory, CreatedAt: at, UpdatedAt: at,
			}
			granted[slug] = before.Role
		}
		undo[identity.ID] = snapshot
		encodedPermissions, err := json.Marshal(permissions)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO house_memberships(person_id,tenant_id,tenant_slug,role,permissions,status,directory_opt_in,grant_origin,created_at,updated_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9) ON CONFLICT(person_id,tenant_slug) DO NOTHING`,
			personID, identity.ID, slug, houseRole, string(encodedPermissions), status, directory, orgKey, stamp); err != nil {
			return err
		}
		// As with the other demo fixtures, insert missing rows and then replace
		// the known fixture identity. Never re-point a conflicting tenant ID.
		result, err := tx.ExecContext(ctx, `UPDATE house_memberships SET role=$1,permissions=$2,status=$3,directory_opt_in=$4,grant_origin=$5,updated_at=$6
			WHERE person_id=$7 AND tenant_id=$8 AND tenant_slug=$9`, houseRole, string(encodedPermissions), status, directory, orgKey, stamp, personID, identity.ID, slug)
		if err != nil {
			return err
		}
		if n, err := result.RowsAffected(); err != nil || n != 1 {
			return fmt.Errorf("fixture membership identity conflict for %s: updated %d rows (%v)", slug, n, err)
		}
	}
	encodedGranted, err := json.Marshal(granted)
	if err != nil {
		return err
	}
	encodedUndo, err := json.Marshal(undo)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_members(org_key,email,role,granted,undo,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$6) ON CONFLICT(org_key,email) DO UPDATE SET
		role=excluded.role,granted=excluded.granted,undo=excluded.undo,updated_at=excluded.updated_at`,
		orgKey, email, role, string(encodedGranted), string(encodedUndo), stamp); err != nil {
		return err
	}
	return tx.Commit()
}
