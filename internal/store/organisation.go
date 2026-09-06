package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// Organisation is the Hausverwaltung as a stored entity: a name, how to reach
// it, and the houses it administers. Authorization still comes from a person's
// membership in a house; this is the administrative level above that, and the
// row the employee roster will attach to.
type Organisation struct {
	Key          string
	Name         string
	ContactName  string
	ContactEmail string
	ContactPhone string
	// Houses holds tenant slugs, sorted, without duplicates.
	Houses    []string
	UpdatedAt time.Time
}

type OrganisationRepository interface {
	// Get reports the stored organisation. The second value is false when no
	// row exists yet, which callers treat as "fall back to configuration".
	Get(context.Context) (Organisation, bool, error)
	Save(context.Context, Organisation) error
	// SetHouses replaces the administered house set without touching the
	// organisation's own fields, so a boot-time reconcile cannot overwrite
	// contact data someone edited in the app.
	SetHouses(context.Context, []string) error
}

type sqlOrganisationRepository struct {
	begin  func(context.Context, string) (*sql.Tx, error)
	orgKey string
}

func BindOrganisationRepository(database *sql.DB, orgKey string) OrganisationRepository {
	return &sqlOrganisationRepository{
		begin:  func(ctx context.Context, key string) (*sql.Tx, error) { return beginOrgTx(ctx, database, key) },
		orgKey: textutil.Slug(orgKey),
	}
}

// NormalizeHouseSlugs returns the slugs sorted, de-duplicated and empty-free.
func NormalizeHouseSlugs(slugs []string) []string {
	seen := make(map[string]struct{}, len(slugs))
	out := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		slug = textutil.Slug(slug)
		if slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	sort.Strings(out)
	return out
}

func normalizeOrganisation(orgKey string, item Organisation) (Organisation, error) {
	item.Key = textutil.Slug(orgKey)
	if item.Key == "" {
		return Organisation{}, fmt.Errorf("organisation is required")
	}
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return Organisation{}, fmt.Errorf("organisation name is required")
	}
	item.ContactName = strings.TrimSpace(item.ContactName)
	item.ContactEmail = strings.ToLower(strings.TrimSpace(item.ContactEmail))
	item.ContactPhone = strings.TrimSpace(item.ContactPhone)
	item.Houses = NormalizeHouseSlugs(item.Houses)
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = time.Now().UTC()
	} else {
		item.UpdatedAt = item.UpdatedAt.UTC()
	}
	return item, nil
}

func (r *sqlOrganisationRepository) Get(ctx context.Context) (Organisation, bool, error) {
	if r.begin == nil || r.orgKey == "" {
		return Organisation{}, false, fmt.Errorf("organisation repository is not bound")
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return Organisation{}, false, err
	}
	defer tx.Rollback()

	item := Organisation{Key: r.orgKey}
	var updatedAt string
	err = tx.QueryRowContext(ctx, `SELECT name, contact_name, contact_email, contact_phone, updated_at
		FROM organisations WHERE org_key=$1`, r.orgKey).
		Scan(&item.Name, &item.ContactName, &item.ContactEmail, &item.ContactPhone, &updatedAt)
	if err == sql.ErrNoRows {
		if err := tx.Commit(); err != nil {
			return Organisation{}, false, err
		}
		return Organisation{}, false, nil
	}
	if err != nil {
		return Organisation{}, false, err
	}
	if parsed, parseErr := time.Parse(time.RFC3339Nano, updatedAt); parseErr == nil {
		item.UpdatedAt = parsed.UTC()
	}

	rows, err := tx.QueryContext(ctx, `SELECT tenant_slug FROM organisation_houses
		WHERE org_key=$1 ORDER BY tenant_slug`, r.orgKey)
	if err != nil {
		return Organisation{}, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return Organisation{}, false, err
		}
		item.Houses = append(item.Houses, slug)
	}
	if err := rows.Err(); err != nil {
		return Organisation{}, false, err
	}
	if err := rows.Close(); err != nil {
		return Organisation{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Organisation{}, false, err
	}
	item.Houses = NormalizeHouseSlugs(item.Houses)
	return item, true, nil
}

func (r *sqlOrganisationRepository) Save(ctx context.Context, item Organisation) error {
	if r.begin == nil || r.orgKey == "" {
		return fmt.Errorf("organisation repository is not bound")
	}
	item, err := normalizeOrganisation(r.orgKey, item)
	if err != nil {
		return err
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO organisations(org_key,name,contact_name,contact_email,contact_phone,updated_at)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(org_key) DO UPDATE SET name=excluded.name, contact_name=excluded.contact_name,
			contact_email=excluded.contact_email, contact_phone=excluded.contact_phone, updated_at=excluded.updated_at`,
		item.Key, item.Name, item.ContactName, item.ContactEmail, item.ContactPhone,
		item.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if err := replaceHouses(ctx, tx, item.Key, item.Houses, item.UpdatedAt); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *sqlOrganisationRepository) SetHouses(ctx context.Context, slugs []string) error {
	if r.begin == nil || r.orgKey == "" {
		return fmt.Errorf("organisation repository is not bound")
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := replaceHouses(ctx, tx, r.orgKey, NormalizeHouseSlugs(slugs), time.Now().UTC()); err != nil {
		return err
	}
	return tx.Commit()
}

func replaceHouses(ctx context.Context, tx *sql.Tx, orgKey string, slugs []string, at time.Time) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM organisation_houses WHERE org_key=$1`, orgKey); err != nil {
		return err
	}
	stamp := at.UTC().Format(time.RFC3339Nano)
	for _, slug := range slugs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_houses(org_key,tenant_slug,added_at)
			VALUES($1,$2,$3)`, orgKey, slug, stamp); err != nil {
			return err
		}
	}
	return nil
}
