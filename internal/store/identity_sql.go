package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/markus-barta/hausv-org/internal/textutil"
)

// Person is the global identity: who someone is, and whether they may sign in
// at all. It deliberately carries NOTHING house-specific (HAUSV-169).
type Person struct {
	ID          string
	Email       string
	Title       string
	FirstName   string
	LastName    string
	AuthMethods []string
	// Deactivated is a GLOBAL login suspension ("cannot sign in via any
	// method"). Suspending someone in a single house is a membership concern.
	Deactivated bool
	Adopted     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// HouseMembership is the person<->house join: everything that is true of a
// person *within one house* and nowhere else (HAUSV-169).
type HouseMembership struct {
	PersonID    string
	TenantSlug  string
	Role        string
	Permissions []string
	Status      string
	// DirectoryOptIn overrides person-wide directory visibility for this house;
	// nil means inherit (HAUSV-178).
	DirectoryOptIn *bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// HouseMember pairs a person with their membership in one specific house — what
// a house admin screen actually lists.
type HouseMember struct {
	Person     Person
	Membership HouseMembership
}

// SQLIdentityStore implements the person/house N:N model. Every mutating method
// is scoped to EITHER the global person OR one membership — there is no
// whole-profile update or delete, which is exactly the defect HAUSV-135
// described. Tables from migration 0017.
type SQLIdentityStore struct {
	db *sql.DB
}

func NewSQLIdentityStore(db *sql.DB) *SQLIdentityStore {
	return &SQLIdentityStore{db: db}
}

func identityTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseIdentityTime(raw string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func encodeStringList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	blob, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	return string(blob)
}

func decodeStringList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func scanPerson(scan func(dest ...any) error) (Person, error) {
	var p Person
	var authMethods, createdAt, updatedAt string
	var deactivated, adopted int
	if err := scan(&p.ID, &p.Email, &p.Title, &p.FirstName, &p.LastName, &authMethods, &deactivated, &adopted, &createdAt, &updatedAt); err != nil {
		return Person{}, err
	}
	p.AuthMethods = decodeStringList(authMethods)
	p.Deactivated = deactivated != 0
	p.Adopted = adopted != 0
	p.CreatedAt = parseIdentityTime(createdAt)
	p.UpdatedAt = parseIdentityTime(updatedAt)
	return p, nil
}

const personColumns = `id, email, title, first_name, last_name, auth_methods, deactivated, adopted, created_at, updated_at`

// PersonByEmail looks a person up by their normalized login email.
func (s *SQLIdentityStore) PersonByEmail(email string) (Person, bool) {
	if s == nil {
		return Person{}, false
	}
	email = textutil.Email(email)
	if email == "" {
		return Person{}, false
	}
	row := s.db.QueryRow(`SELECT `+personColumns+` FROM persons WHERE email=?`, email)
	p, err := scanPerson(row.Scan)
	if err != nil {
		return Person{}, false
	}
	return p, true
}

func (s *SQLIdentityStore) PersonByID(id string) (Person, bool) {
	if s == nil {
		return Person{}, false
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Person{}, false
	}
	row := s.db.QueryRow(`SELECT `+personColumns+` FROM persons WHERE id=?`, id)
	p, err := scanPerson(row.Scan)
	if err != nil {
		return Person{}, false
	}
	return p, true
}

func (s *SQLIdentityStore) ListPersons() []Person {
	if s == nil {
		return nil
	}
	rows, err := s.db.Query(`SELECT ` + personColumns + ` FROM persons ORDER BY email`)
	if err != nil {
		return []Person{}
	}
	defer rows.Close()
	out := []Person{}
	for rows.Next() {
		p, err := scanPerson(rows.Scan)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out
}

// UpsertPerson creates or updates ONLY the global identity. It never touches a
// membership. A person is identified by email; an existing person keeps their
// ID (and thus every membership) across a name change.
func (s *SQLIdentityStore) UpsertPerson(p Person, at time.Time) (Person, error) {
	if s == nil {
		return Person{}, fmt.Errorf("identity store unavailable")
	}
	p.Email = textutil.Email(p.Email)
	if p.Email == "" {
		return Person{}, fmt.Errorf("person requires an email")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return Person{}, err
	}
	defer tx.Rollback()

	var existingID, createdAt string
	err = tx.QueryRow(`SELECT id, created_at FROM persons WHERE email=?`, p.Email).Scan(&existingID, &createdAt)
	switch {
	case err == nil:
		p.ID = existingID
		p.CreatedAt = parseIdentityTime(createdAt)
	case err == sql.ErrNoRows:
		if strings.TrimSpace(p.ID) == "" {
			id, idErr := randomToken(12)
			if idErr != nil {
				return Person{}, idErr
			}
			p.ID = id
		}
		p.CreatedAt = at
	default:
		return Person{}, err
	}
	p.UpdatedAt = at

	if _, err := tx.Exec(
		`INSERT INTO persons(`+personColumns+`) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   email=excluded.email, title=excluded.title, first_name=excluded.first_name,
		   last_name=excluded.last_name, auth_methods=excluded.auth_methods,
		   deactivated=excluded.deactivated, adopted=excluded.adopted,
		   updated_at=excluded.updated_at`,
		p.ID, p.Email, strings.TrimSpace(p.Title), strings.TrimSpace(p.FirstName), strings.TrimSpace(p.LastName),
		encodeStringList(p.AuthMethods), boolToInt(p.Deactivated), boolToInt(p.Adopted),
		identityTime(p.CreatedAt), identityTime(p.UpdatedAt),
	); err != nil {
		return Person{}, err
	}
	if err := tx.Commit(); err != nil {
		return Person{}, err
	}
	// Re-read so the returned value is byte-for-byte what a later read gives;
	// otherwise an empty slice here and a nil there compare unequal.
	if fresh, ok := s.PersonByID(p.ID); ok {
		return fresh, nil
	}
	return p, nil
}

// ChangePersonEmail moves a person's global login email, keeping their ID and
// therefore every membership. It refuses to collide with another person, so it
// can never absorb someone else's relationships (HAUSV-135).
func (s *SQLIdentityStore) ChangePersonEmail(personID string, newEmail string, at time.Time) (Person, error) {
	if s == nil {
		return Person{}, fmt.Errorf("identity store unavailable")
	}
	personID = strings.TrimSpace(personID)
	newEmail = textutil.Email(newEmail)
	if personID == "" || newEmail == "" {
		return Person{}, fmt.Errorf("invalid email change")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return Person{}, err
	}
	defer tx.Rollback()
	var otherID string
	if err := tx.QueryRow(`SELECT id FROM persons WHERE email=?`, newEmail).Scan(&otherID); err == nil && otherID != personID {
		return Person{}, fmt.Errorf("email already belongs to another person")
	}
	res, err := tx.Exec(`UPDATE persons SET email=?, updated_at=? WHERE id=?`, newEmail, identityTime(at), personID)
	if err != nil {
		return Person{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Person{}, fmt.Errorf("person not found")
	}
	if err := tx.Commit(); err != nil {
		return Person{}, err
	}
	p, _ := s.PersonByID(personID)
	return p, nil
}

func scanMembership(scan func(dest ...any) error) (HouseMembership, error) {
	var m HouseMembership
	var permissions, createdAt, updatedAt string
	var directoryOptIn *int64
	if err := scan(&m.PersonID, &m.TenantSlug, &m.Role, &permissions, &m.Status, &directoryOptIn, &createdAt, &updatedAt); err != nil {
		return HouseMembership{}, err
	}
	if directoryOptIn != nil {
		v := *directoryOptIn != 0
		m.DirectoryOptIn = &v
	}
	m.Permissions = decodeStringList(permissions)
	m.CreatedAt = parseIdentityTime(createdAt)
	m.UpdatedAt = parseIdentityTime(updatedAt)
	return m, nil
}

const membershipColumns = `person_id, tenant_slug, role, permissions, status, directory_opt_in, created_at, updated_at`

// SetMembership creates or updates exactly ONE person<->house relation. Other
// houses' memberships are not read, not rewritten and not touched.
func (s *SQLIdentityStore) SetMembership(m HouseMembership, at time.Time) (HouseMembership, error) {
	if s == nil {
		return HouseMembership{}, fmt.Errorf("identity store unavailable")
	}
	m.PersonID = strings.TrimSpace(m.PersonID)
	m.TenantSlug = textutil.Slug(m.TenantSlug)
	if m.PersonID == "" || m.TenantSlug == "" {
		return HouseMembership{}, fmt.Errorf("membership requires a person and a house")
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return HouseMembership{}, err
	}
	defer tx.Rollback()
	var personExists int
	if err := tx.QueryRow(`SELECT 1 FROM persons WHERE id=?`, m.PersonID).Scan(&personExists); err != nil {
		return HouseMembership{}, fmt.Errorf("person not found")
	}
	var createdAt string
	if err := tx.QueryRow(
		`SELECT created_at FROM house_memberships WHERE person_id=? AND tenant_slug=?`, m.PersonID, m.TenantSlug,
	).Scan(&createdAt); err == nil {
		m.CreatedAt = parseIdentityTime(createdAt)
	} else {
		m.CreatedAt = at
	}
	m.UpdatedAt = at
	var directoryOptIn any
	if m.DirectoryOptIn != nil {
		directoryOptIn = boolToInt(*m.DirectoryOptIn)
	}
	if _, err := tx.Exec(
		`INSERT INTO house_memberships(`+membershipColumns+`) VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(person_id, tenant_slug) DO UPDATE SET
		   role=excluded.role, permissions=excluded.permissions, status=excluded.status,
		   directory_opt_in=excluded.directory_opt_in, updated_at=excluded.updated_at`,
		m.PersonID, m.TenantSlug, NormalizeRole(m.Role), encodeStringList(NormalizePermissions(m.Permissions)),
		strings.TrimSpace(m.Status), directoryOptIn, identityTime(m.CreatedAt), identityTime(m.UpdatedAt),
	); err != nil {
		return HouseMembership{}, err
	}
	if err := tx.Commit(); err != nil {
		return HouseMembership{}, err
	}
	// Re-read for the same reason as UpsertPerson.
	if fresh, ok := s.Membership(m.PersonID, m.TenantSlug); ok {
		return fresh, nil
	}
	m.Role = NormalizeRole(m.Role)
	m.Permissions = NormalizePermissions(m.Permissions)
	return m, nil
}

func (s *SQLIdentityStore) Membership(personID string, tenantSlug string) (HouseMembership, bool) {
	if s == nil {
		return HouseMembership{}, false
	}
	row := s.db.QueryRow(
		`SELECT `+membershipColumns+` FROM house_memberships WHERE person_id=? AND tenant_slug=?`,
		strings.TrimSpace(personID), textutil.Slug(tenantSlug),
	)
	m, err := scanMembership(row.Scan)
	if err != nil {
		return HouseMembership{}, false
	}
	return m, true
}

func (s *SQLIdentityStore) MembershipsForPerson(personID string) []HouseMembership {
	if s == nil {
		return nil
	}
	rows, err := s.db.Query(
		`SELECT `+membershipColumns+` FROM house_memberships WHERE person_id=? ORDER BY tenant_slug`,
		strings.TrimSpace(personID),
	)
	if err != nil {
		return []HouseMembership{}
	}
	defer rows.Close()
	out := []HouseMembership{}
	for rows.Next() {
		m, err := scanMembership(rows.Scan)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

// ListHouseMembers returns the people of ONE house with their membership there —
// the only view a house admin is entitled to manage.
func (s *SQLIdentityStore) ListHouseMembers(tenantSlug string) []HouseMember {
	if s == nil {
		return nil
	}
	tenantSlug = textutil.Slug(tenantSlug)
	rows, err := s.db.Query(
		`SELECT p.id, p.email, p.title, p.first_name, p.last_name, p.auth_methods,
		        p.deactivated, p.adopted, p.created_at, p.updated_at,
		        m.person_id, m.tenant_slug, m.role, m.permissions, m.status,
		        m.directory_opt_in, m.created_at, m.updated_at
		   FROM house_memberships m JOIN persons p ON p.id = m.person_id
		  WHERE m.tenant_slug=? ORDER BY p.email`, tenantSlug)
	if err != nil {
		return []HouseMember{}
	}
	defer rows.Close()
	out := []HouseMember{}
	for rows.Next() {
		var p Person
		var m HouseMembership
		var authMethods, pCreated, pUpdated, permissions, mCreated, mUpdated string
		var deactivated, adopted int
		var mDirectory *int64
		if err := rows.Scan(
			&p.ID, &p.Email, &p.Title, &p.FirstName, &p.LastName, &authMethods, &deactivated, &adopted, &pCreated, &pUpdated,
			&m.PersonID, &m.TenantSlug, &m.Role, &permissions, &m.Status, &mDirectory, &mCreated, &mUpdated,
		); err != nil {
			continue
		}
		p.AuthMethods = decodeStringList(authMethods)
		p.Deactivated = deactivated != 0
		p.Adopted = adopted != 0
		p.CreatedAt, p.UpdatedAt = parseIdentityTime(pCreated), parseIdentityTime(pUpdated)
		m.Permissions = decodeStringList(permissions)
		if mDirectory != nil {
			v := *mDirectory != 0
			m.DirectoryOptIn = &v
		}
		m.CreatedAt, m.UpdatedAt = parseIdentityTime(mCreated), parseIdentityTime(mUpdated)
		out = append(out, HouseMember{Person: p, Membership: m})
	}
	return out
}

// RemoveMembership detaches a person from ONE house. The person and every other
// membership survive — this is what "delete" in a house admin screen must mean
// (HAUSV-135).
func (s *SQLIdentityStore) RemoveMembership(personID string, tenantSlug string) (bool, error) {
	if s == nil {
		return false, nil
	}
	res, err := s.db.Exec(
		`DELETE FROM house_memberships WHERE person_id=? AND tenant_slug=?`,
		strings.TrimSpace(personID), textutil.Slug(tenantSlug),
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// DeletePerson removes the global identity and, by cascade, every membership.
// Deliberately separate from RemoveMembership: this is a platform-level act, not
// something a single house's admin performs.
func (s *SQLIdentityStore) DeletePerson(personID string) (bool, error) {
	if s == nil {
		return false, nil
	}
	res, err := s.db.Exec(`DELETE FROM persons WHERE id=?`, strings.TrimSpace(personID))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ImportProfiles migrates the email-keyed UserProfile records into person +
// memberships, clobber-safe and idempotent (HAUSV-169/170).
//
// Role and permissions per house are resolved the same way UserProfile.ForTenant
// resolves them at read time: the top-level pair is the default, and a
// TenantMemberships entry overrides it for that house.
func (s *SQLIdentityStore) ImportProfiles(src *InviteStore, at time.Time) error {
	if src == nil {
		return nil
	}
	if at.IsZero() {
		at = time.Now()
	}
	at = at.UTC()
	for _, profile := range src.List() {
		email := textutil.Email(profile.Email)
		if email == "" {
			continue
		}
		person, exists := s.PersonByEmail(email)
		if !exists {
			created, err := s.UpsertPerson(Person{
				Email:       email,
				Title:       profile.Title,
				FirstName:   profile.FirstName,
				LastName:    profile.LastName,
				AuthMethods: profile.AuthMethods,
				Deactivated: profile.Deactivated,
				Adopted:     profile.Adopted,
			}, at)
			if err != nil {
				return err
			}
			person = created
		}
		for _, tenant := range tenantsOfProfile(profile) {
			if _, already := s.Membership(person.ID, tenant); already {
				continue
			}
			resolved := profile.ForTenant(tenant)
			if _, err := s.SetMembership(HouseMembership{
				PersonID:    person.ID,
				TenantSlug:  tenant,
				Role:        resolved.Role,
				Permissions: resolved.Permissions,
				Status:      profile.Status,
			}, at); err != nil {
				return err
			}
		}
	}
	return nil
}

// tenantsOfProfile is the union of the flat Tenants list and the keys of
// TenantMemberships — a profile may carry a membership for a house that is not
// repeated in Tenants.
func tenantsOfProfile(profile UserProfile) []string {
	seen := map[string]struct{}{}
	out := []string{}
	add := func(raw string) {
		slug := textutil.Slug(raw)
		if slug == "" {
			return
		}
		if _, ok := seen[slug]; ok {
			return
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	for _, tenant := range profile.Tenants {
		add(tenant)
	}
	for tenant := range profile.TenantMemberships {
		add(tenant)
	}
	sort.Strings(out)
	return out
}
