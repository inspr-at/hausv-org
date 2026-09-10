package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

type CapabilityOverride struct {
	OrgKey, RoleFamily, Capability string
	Allowed                        bool
	UpdatedAt, UpdatedBy           string
}

type UserCapabilityGrant struct {
	OrgKey     string `json:"-"`
	Email      string `json:"-"`
	Capability string `json:"capability"`
	Effect     string `json:"effect"`
	TenantSlug string `json:"-"`
	UpdatedAt  string `json:"-"`
	UpdatedBy  string `json:"-"`
}

type CapabilityProfile struct {
	Name         string
	Capabilities []UserCapabilityGrant
}

type CapabilityRules struct {
	OrgKey    string
	Overrides []CapabilityOverride
	Grants    []UserCapabilityGrant
	Profiles  []CapabilityProfile
}

type CapabilityRepository interface {
	Get(context.Context) (CapabilityRules, error)
	SetOverride(context.Context, CapabilityOverride) error
	ResetOverrides(context.Context) error
	SetGrant(context.Context, UserCapabilityGrant) error
	ReplaceUserGrants(context.Context, string, string, []UserCapabilityGrant, string) error
	SaveProfile(context.Context, CapabilityProfile) error
}

type sqlCapabilityRepository struct {
	begin  func(context.Context, string) (*sql.Tx, error)
	orgKey string
}

func BindCapabilityRepository(database *sql.DB, orgKey string) CapabilityRepository {
	return &sqlCapabilityRepository{begin: func(ctx context.Context, key string) (*sql.Tx, error) { return beginOrgTx(ctx, database, key) }, orgKey: textutil.Slug(orgKey)}
}

// Store validation is independent of authz, which consumes these records.
func DelegableCapability(capability string) bool {
	switch capability {
	case "manage-parking", "manage-announcements", "manage-documents", "manage-issues", "manage-votes", "manage-building", "owner-documents", "vote", "oversight", "manage-energy", "control-energy":
		return true
	default:
		return false
	}
}

func (r *sqlCapabilityRepository) tx(ctx context.Context) (*sql.Tx, error) {
	if r.begin == nil || r.orgKey == "" {
		return nil, fmt.Errorf("capability repository is not bound")
	}
	return r.begin(ctx, r.orgKey)
}

func (r *sqlCapabilityRepository) Get(ctx context.Context) (CapabilityRules, error) {
	result := CapabilityRules{OrgKey: r.orgKey}
	tx, err := r.tx(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT role_family,capability,allowed,updated_at,updated_by FROM organisation_capability_overrides WHERE org_key=$1 ORDER BY role_family,capability`, r.orgKey)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		item := CapabilityOverride{OrgKey: r.orgKey}
		if err := rows.Scan(&item.RoleFamily, &item.Capability, &item.Allowed, &item.UpdatedAt, &item.UpdatedBy); err != nil {
			rows.Close()
			return result, err
		}
		result.Overrides = append(result.Overrides, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT email,capability,effect,COALESCE(tenant_slug,''),updated_at,updated_by FROM user_capability_grants WHERE org_key=$1 ORDER BY email,capability,tenant_slug`, r.orgKey)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		item := UserCapabilityGrant{OrgKey: r.orgKey}
		if err := rows.Scan(&item.Email, &item.Capability, &item.Effect, &item.TenantSlug, &item.UpdatedAt, &item.UpdatedBy); err != nil {
			rows.Close()
			return result, err
		}
		result.Grants = append(result.Grants, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT name,capabilities FROM capability_profiles WHERE org_key=$1 ORDER BY name`, r.orgKey)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item CapabilityProfile
		var data string
		if err := rows.Scan(&item.Name, &data); err != nil {
			rows.Close()
			return result, err
		}
		if err := json.Unmarshal([]byte(data), &item.Capabilities); err != nil {
			rows.Close()
			return result, err
		}
		result.Profiles = append(result.Profiles, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	return result, tx.Commit()
}

func (r *sqlCapabilityRepository) SetOverride(ctx context.Context, item CapabilityOverride) error {
	if !DelegableCapability(item.Capability) || (item.RoleFamily != "verwaltung" && item.RoleFamily != "eigentuemer" && item.RoleFamily != "bewohner") || strings.TrimSpace(item.UpdatedBy) == "" {
		return fmt.Errorf("invalid capability override")
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO organisation_capability_overrides(org_key,role_family,capability,allowed,updated_at,updated_by) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(org_key,role_family,capability) DO UPDATE SET allowed=excluded.allowed,updated_at=excluded.updated_at,updated_by=excluded.updated_by`, r.orgKey, item.RoleFamily, item.Capability, item.Allowed, time.Now().UTC().Format(time.RFC3339Nano), strings.ToLower(strings.TrimSpace(item.UpdatedBy)))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *sqlCapabilityRepository) ResetOverrides(ctx context.Context) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM organisation_capability_overrides WHERE org_key=$1`, r.orgKey); err != nil {
		return err
	}
	return tx.Commit()
}

func validUserGrant(item UserCapabilityGrant, allowDefault bool) bool {
	return DelegableCapability(item.Capability) && (item.Effect == "grant" || item.Effect == "deny" || (allowDefault && item.Effect == ""))
}

func (r *sqlCapabilityRepository) writeGrant(ctx context.Context, tx *sql.Tx, item UserCapabilityGrant) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_capability_grants WHERE org_key=$1 AND email=$2 AND capability=$3 AND COALESCE(tenant_slug,'')=$4`, r.orgKey, item.Email, item.Capability, item.TenantSlug); err != nil {
		return err
	}
	if item.Effect == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO user_capability_grants(org_key,email,capability,effect,tenant_slug,updated_at,updated_by) VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7)`, r.orgKey, item.Email, item.Capability, item.Effect, item.TenantSlug, time.Now().UTC().Format(time.RFC3339Nano), item.UpdatedBy)
	return err
}

func (r *sqlCapabilityRepository) SetGrant(ctx context.Context, item UserCapabilityGrant) error {
	item.Email = strings.ToLower(strings.TrimSpace(item.Email))
	item.UpdatedBy = strings.ToLower(strings.TrimSpace(item.UpdatedBy))
	if !validUserGrant(item, true) || !strings.Contains(item.Email, "@") || item.UpdatedBy == "" || item.TenantSlug != textutil.Slug(item.TenantSlug) {
		return fmt.Errorf("invalid user capability")
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = r.writeGrant(ctx, tx, item); err != nil {
		return err
	}
	return tx.Commit()
}

// ReplaceUserGrants applies a template atomically within precisely one scope.
func (r *sqlCapabilityRepository) ReplaceUserGrants(ctx context.Context, email, tenant string, grants []UserCapabilityGrant, by string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	by = strings.ToLower(strings.TrimSpace(by))
	if !strings.Contains(email, "@") || by == "" || tenant != textutil.Slug(tenant) {
		return fmt.Errorf("invalid user capability scope")
	}
	seen := map[string]bool{}
	for _, item := range grants {
		if !validUserGrant(item, false) || seen[item.Capability] {
			return fmt.Errorf("invalid profile capability")
		}
		seen[item.Capability] = true
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM user_capability_grants WHERE org_key=$1 AND email=$2 AND COALESCE(tenant_slug,'')=$3`, r.orgKey, email, tenant); err != nil {
		return err
	}
	for _, item := range grants {
		item.Email = email
		item.TenantSlug = tenant
		item.UpdatedBy = by
		if err = r.writeGrant(ctx, tx, item); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *sqlCapabilityRepository) SaveProfile(ctx context.Context, item CapabilityProfile) error {
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" || len(item.Name) > 100 || strings.EqualFold(item.Name, "Standard") {
		return fmt.Errorf("invalid profile name")
	}
	seen := map[string]bool{}
	for _, grant := range item.Capabilities {
		if !validUserGrant(grant, false) || seen[grant.Capability] {
			return fmt.Errorf("invalid profile capability")
		}
		seen[grant.Capability] = true
	}
	data, err := json.Marshal(item.Capabilities)
	if err != nil {
		return err
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO capability_profiles(org_key,name,capabilities) VALUES($1,$2,$3) ON CONFLICT(org_key,name) DO UPDATE SET capabilities=excluded.capabilities`, r.orgKey, item.Name, string(data))
	if err != nil {
		return err
	}
	return tx.Commit()
}
