package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

const (
	// OrganisationRoleAdmin administers the Verwaltung itself: employees,
	// settings, and every house the organisation carries.
	OrganisationRoleAdmin = "admin"
	// OrganisationRoleClerk works the cases without administering the
	// organisation ("Sachbearbeiter").
	OrganisationRoleClerk = "sachbearbeiter"
)

// OrganisationMember is an employee of a Hausverwaltung.
//
// Granted is the undo record: for every house whose role this membership
// changed, the role that house had before ("" when the person had none). House
// roles themselves live where authorization reads them; this only remembers
// what to put back when the employee leaves.
type OrganisationMember struct {
	OrgKey    string
	Email     string
	Role      string
	Granted   map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganisationMemberRepository interface {
	List(context.Context) ([]OrganisationMember, error)
	Get(context.Context, string) (OrganisationMember, bool, error)
	Save(context.Context, OrganisationMember) error
	Delete(context.Context, string) (bool, error)
}

type sqlOrganisationMemberRepository struct {
	begin  func(context.Context, string) (*sql.Tx, error)
	orgKey string
}

func BindOrganisationMemberRepository(database *sql.DB, orgKey string) OrganisationMemberRepository {
	return &sqlOrganisationMemberRepository{
		begin:  func(ctx context.Context, key string) (*sql.Tx, error) { return beginOrgTx(ctx, database, key) },
		orgKey: textutil.Slug(orgKey),
	}
}

// NormalizeOrganisationRole maps anything unknown to the clerk role: a member
// row with an unreadable role must not silently become an administrator.
func NormalizeOrganisationRole(role string) string {
	if strings.ToLower(strings.TrimSpace(role)) == OrganisationRoleAdmin {
		return OrganisationRoleAdmin
	}
	return OrganisationRoleClerk
}

func normalizeOrganisationMember(orgKey string, item OrganisationMember) (OrganisationMember, error) {
	item.OrgKey = textutil.Slug(orgKey)
	if item.OrgKey == "" {
		return OrganisationMember{}, fmt.Errorf("organisation is required")
	}
	item.Email = strings.ToLower(strings.TrimSpace(item.Email))
	if item.Email == "" || !strings.Contains(item.Email, "@") {
		return OrganisationMember{}, fmt.Errorf("a member needs an e-mail address")
	}
	item.Role = NormalizeOrganisationRole(item.Role)
	granted := make(map[string]string, len(item.Granted))
	for slug, previous := range item.Granted {
		slug = textutil.Slug(slug)
		if slug == "" {
			continue
		}
		granted[slug] = strings.TrimSpace(previous)
	}
	item.Granted = granted
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	} else {
		item.CreatedAt = item.CreatedAt.UTC()
	}
	item.UpdatedAt = now
	return item, nil
}

func (r *sqlOrganisationMemberRepository) bound() error {
	if r.begin == nil || r.orgKey == "" {
		return fmt.Errorf("organisation member repository is not bound")
	}
	return nil
}

func (r *sqlOrganisationMemberRepository) List(ctx context.Context) ([]OrganisationMember, error) {
	if err := r.bound(); err != nil {
		return nil, err
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT email, role, granted, created_at, updated_at
		FROM organisation_members WHERE org_key=$1 ORDER BY email`, r.orgKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OrganisationMember{}
	for rows.Next() {
		item, err := scanOrganisationMember(rows, r.orgKey)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Email < out[j].Email })
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOrganisationMember(row rowScanner, orgKey string) (OrganisationMember, error) {
	item := OrganisationMember{OrgKey: orgKey, Granted: map[string]string{}}
	var granted, createdAt, updatedAt string
	if err := row.Scan(&item.Email, &item.Role, &granted, &createdAt, &updatedAt); err != nil {
		return OrganisationMember{}, err
	}
	if strings.TrimSpace(granted) != "" {
		if err := json.Unmarshal([]byte(granted), &item.Granted); err != nil {
			return OrganisationMember{}, fmt.Errorf("organisation member %s: unreadable grant record: %w", item.Email, err)
		}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		item.CreatedAt = parsed.UTC()
	}
	if parsed, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
		item.UpdatedAt = parsed.UTC()
	}
	item.Role = NormalizeOrganisationRole(item.Role)
	return item, nil
}

func (r *sqlOrganisationMemberRepository) Get(ctx context.Context, email string) (OrganisationMember, bool, error) {
	if err := r.bound(); err != nil {
		return OrganisationMember{}, false, err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return OrganisationMember{}, false, err
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `SELECT email, role, granted, created_at, updated_at
		FROM organisation_members WHERE org_key=$1 AND email=$2`, r.orgKey, email)
	item, err := scanOrganisationMember(row, r.orgKey)
	if err == sql.ErrNoRows {
		if err := tx.Commit(); err != nil {
			return OrganisationMember{}, false, err
		}
		return OrganisationMember{}, false, nil
	}
	if err != nil {
		return OrganisationMember{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return OrganisationMember{}, false, err
	}
	return item, true, nil
}

func (r *sqlOrganisationMemberRepository) Save(ctx context.Context, item OrganisationMember) error {
	if err := r.bound(); err != nil {
		return err
	}
	item, err := normalizeOrganisationMember(r.orgKey, item)
	if err != nil {
		return err
	}
	granted, err := json.Marshal(item.Granted)
	if err != nil {
		return err
	}
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO organisation_members(org_key,email,role,granted,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(org_key,email) DO UPDATE SET role=excluded.role, granted=excluded.granted, updated_at=excluded.updated_at`,
		item.OrgKey, item.Email, item.Role, string(granted),
		item.CreatedAt.Format(time.RFC3339Nano), item.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *sqlOrganisationMemberRepository) Delete(ctx context.Context, email string) (bool, error) {
	if err := r.bound(); err != nil {
		return false, err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	tx, err := r.begin(ctx, r.orgKey)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM organisation_members WHERE org_key=$1 AND email=$2`, r.orgKey, email)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return affected > 0, nil
}
