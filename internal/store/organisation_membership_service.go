package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/textutil"
)

// OrganisationMembershipUndo records existence as well as the complete house
// membership before the organisation took ownership. A nil Previous means the
// house membership was created by this organisation.
type OrganisationMembershipUndo struct {
	TenantSlug string           `json:"tenant_slug"`
	Previous   *HouseMembership `json:"previous"`
}

var ErrLegacyOrganisationUndo = errors.New("Die bisherigen Verwaltungsrechte müssen vor der Änderung abgeglichen werden: Der alte Datensatz enthält keine vollständige Sicherung.")

// OrganisationMembershipService is the sole bundled membership writer. The
// caller must authorize administration of the selected organisation first.
// The repository supplies organisation scope and dialect; identity supplies
// the shared database's declared cross-tenant lane.
type OrganisationMembershipService struct {
	repo     *sqlOrganisationMemberRepository
	identity *SQLIdentityStore
	// fault is test-only and runs inside the transaction after each named stage.
	fault func(stage string) error
}

func NewOrganisationMembershipService(repo OrganisationMemberRepository, identity *SQLIdentityStore) (*OrganisationMembershipService, error) {
	r, ok := repo.(*sqlOrganisationMemberRepository)
	if !ok || r == nil || identity == nil || identity.db == nil || r.orgKey == "" {
		return nil, fmt.Errorf("Mitarbeiterverwaltung ist nicht verfügbar.")
	}
	return &OrganisationMembershipService{repo: r, identity: identity}, nil
}

// Add also changes the role of an existing member without replacing their
// original undo snapshot. This keeps the roster form's upsert behavior atomic.
func (s *OrganisationMembershipService) Add(ctx context.Context, email, role string, actor AuditEvent) ([]AuditEvent, error) {
	return s.change(ctx, email, role, "add", actor)
}
func (s *OrganisationMembershipService) ChangeRole(ctx context.Context, email, role string, actor AuditEvent) ([]AuditEvent, error) {
	return s.change(ctx, email, role, "role", actor)
}
func (s *OrganisationMembershipService) Remove(ctx context.Context, email string, actor AuditEvent) ([]AuditEvent, error) {
	return s.change(ctx, email, "", "remove", actor)
}
func (s *OrganisationMembershipService) checkpoint(stage string) error {
	if s.fault != nil {
		return s.fault(stage)
	}
	return nil
}

func (s *OrganisationMembershipService) change(ctx context.Context, email, role, operation string, actor AuditEvent) ([]AuditEvent, error) {
	email = textutil.Email(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("Bitte eine E-Mail-Adresse angeben.")
	}
	tx, err := s.identity.db.Unscoped("organisation membership transaction spans the selected organisation's stored house set and its origin-owned undo records, member row and durable audit").Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	lock := ""
	if s.repo.postgres {
		lock = " FOR UPDATE"
	} else {
		// SQLite must acquire the write lock before the first read, so a competing
		// writer waits instead of trying to upgrade an obsolete read snapshot.
		if _, err := tx.ExecContext(ctx, `UPDATE persons SET id=id WHERE email=$1`, email); err != nil {
			return nil, err
		}
	}
	// The person exists even before Add and after Remove. Locking only the member
	// row cannot serialize an insertion racing a removal of an absent row.
	var personID string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM persons WHERE email=$1`+lock, email).Scan(&personID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("Diese Person ist noch nicht angelegt. Bitte zuerst in einer Liegenschaft einladen.")
		}
		return nil, err
	}
	member, err := scanOrganisationMember(tx.QueryRowContext(ctx, `SELECT email,role,granted,undo,created_at,updated_at FROM organisation_members WHERE org_key=$1 AND email=$2`+lock, s.repo.orgKey, email), s.repo.orgKey)
	found := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if !found && operation != "add" {
		return nil, fmt.Errorf("Diese Person gehört nicht zur Verwaltung.")
	}
	if found {
		if len(member.Granted) != len(member.Undo) {
			return nil, ErrLegacyOrganisationUndo
		}
		for id, undo := range member.Undo {
			if _, ok := validTenantRef(TenantRef{ID: id, Slug: undo.TenantSlug}); !ok {
				return nil, fmt.Errorf("unreadable membership undo identity")
			}
			previousRole, recorded := member.Granted[undo.TenantSlug]
			if !recorded || (undo.Previous == nil && previousRole != "") || (undo.Previous != nil && previousRole != undo.Previous.Role) {
				return nil, fmt.Errorf("incomplete membership undo record")
			}
		}
	}
	if !found {
		member = OrganisationMember{OrgKey: s.repo.orgKey, Email: email, Granted: map[string]string{}, Undo: map[string]OrganisationMembershipUndo{}, CreatedAt: time.Now().UTC()}
	}
	if member.Undo == nil {
		member.Undo = map[string]OrganisationMembershipUndo{}
	}
	if member.Granted == nil {
		member.Granted = map[string]string{}
	}
	if err := s.checkpoint("locked"); err != nil {
		return nil, err
	}

	houses, err := s.houses(ctx, tx)
	if err != nil {
		return nil, err
	}
	at := time.Now().UTC()
	changed, kept := []string{}, []string{}
	if operation == "remove" {
		ids := make([]string, 0, len(member.Undo))
		for id := range member.Undo {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			undo := member.Undo[id]
			current, origin, err := loadOrganisationHouse(ctx, tx, personID, id, lock)
			if errors.Is(err, sql.ErrNoRows) {
				kept = append(kept, undo.TenantSlug)
				continue
			}
			if err != nil {
				return nil, err
			}
			if origin != s.repo.orgKey {
				kept = append(kept, current.TenantSlug)
				continue
			}
			if undo.Previous == nil {
				_, err = tx.ExecContext(ctx, `DELETE FROM house_memberships WHERE person_id=$1 AND tenant_id=$2 AND grant_origin=$3`, personID, id, s.repo.orgKey)
			} else {
				previous := *undo.Previous
				// Identity keys come from the locked target, never from JSON payloads.
				previous.PersonID, previous.TenantID, previous.TenantSlug = personID, id, current.TenantSlug
				// The organisation changed privileges only. Keep any subsequent
				// directory/status edits rather than restoring unrelated fields.
				previous.Status, previous.DirectoryOptIn = current.Status, current.DirectoryOptIn
				_, err = s.identity.setMembershipTx(tx, previous, at)
			}
			if err != nil {
				return nil, err
			}
			houses[id] = current.TenantSlug
			changed = append(changed, current.TenantSlug)
			if err := s.checkpoint("house:" + current.TenantSlug); err != nil {
				return nil, err
			}
		}
	} else {
		role = NormalizeOrganisationRole(role)
		houseRole := RoleManager
		if role == OrganisationRoleAdmin {
			houseRole = RoleAdmin
		}
		ids := make([]string, 0, len(houses))
		for id := range houses {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			slug := houses[id]
			current, origin, err := loadOrganisationHouse(ctx, tx, personID, id, lock)
			exists := err == nil
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			_, tracked := member.Undo[id]
			if tracked && (!exists || origin != s.repo.orgKey) {
				kept = append(kept, slug)
				continue // a later manual grant/deletion wins
			}
			if exists && origin != "" && origin != s.repo.orgKey {
				return nil, fmt.Errorf("Liegenschaft %s hat bereits Rechte aus einer anderen Verwaltung.", slug)
			}
			if !tracked && origin == s.repo.orgKey {
				return nil, fmt.Errorf("organisation grant has no undo record")
			}
			if !tracked {
				undo := OrganisationMembershipUndo{TenantSlug: slug}
				if exists {
					previous := current
					undo.Previous = &previous
					member.Granted[slug] = current.Role
				} else {
					member.Granted[slug] = ""
				}
				member.Undo[id] = undo
			}
			if !exists {
				current = HouseMembership{PersonID: personID, TenantID: id, TenantSlug: slug}
			}
			current.Role = houseRole // retain explicit permissions, status and directory preference
			if _, err := s.identity.setMembershipTx(tx, current, at); err != nil {
				return nil, err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE house_memberships SET grant_origin=$1 WHERE person_id=$2 AND tenant_id=$3`, s.repo.orgKey, personID, id); err != nil {
				return nil, err
			}
			changed = append(changed, slug)
			if err := s.checkpoint("house:" + slug); err != nil {
				return nil, err
			}
		}
		member.Role = role
	}
	if err := s.checkpoint("houses"); err != nil {
		return nil, err
	}
	if operation == "remove" {
		if _, err := tx.ExecContext(ctx, `DELETE FROM organisation_members WHERE org_key=$1 AND email=$2`, s.repo.orgKey, email); err != nil {
			return nil, err
		}
	} else {
		granted, err := json.Marshal(member.Granted)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_members(org_key,email,role,granted,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(org_key,email) DO UPDATE SET role=excluded.role,granted=excluded.granted,updated_at=excluded.updated_at`, s.repo.orgKey, email, member.Role, string(granted), identityTime(member.CreatedAt), identityTime(at)); err != nil {
			return nil, err
		}
	}
	if err := s.checkpoint("member"); err != nil {
		return nil, err
	}
	if operation != "remove" {
		undo, err := json.Marshal(member.Undo)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE organisation_members SET undo=$1 WHERE org_key=$2 AND email=$3`, string(undo), s.repo.orgKey, email); err != nil {
			return nil, err
		}
	}
	if err := s.checkpoint("undo"); err != nil {
		return nil, err
	}
	sort.Strings(changed)
	sort.Strings(kept)
	summary := "Mitarbeiter aufgenommen"
	if operation == "remove" {
		summary = "Mitarbeiter entfernt"
	} else if found {
		summary = "Mitarbeiterrolle geändert"
	}
	details := map[string]string{"org_key": s.repo.orgKey, "organisation_role": member.Role, "houses_changed": strings.Join(changed, ", "), "houses_untouched": strings.Join(kept, ", ")}
	slugs := make([]string, 0, len(houses))
	for _, slug := range houses {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	events := make([]AuditEvent, 0, len(slugs))
	for _, slug := range slugs {
		events = append(events, AuditEvent{At: at, TenantSlug: slug, ActorEmail: actor.ActorEmail, ActorRole: actor.ActorRole, Action: AuditActionVerwaltungSettings, TargetType: "organisation_member", TargetID: email, Summary: summary, Details: details})
	}
	// Retain an organisation event even when its last house has been detached.
	if len(events) == 0 {
		events = append(events, AuditEvent{At: at, ActorEmail: actor.ActorEmail, ActorRole: actor.ActorRole, Action: AuditActionVerwaltungSettings, TargetType: "organisation_member", TargetID: email, Summary: summary, Details: details})
	}
	data, err := json.Marshal(events)
	if err != nil {
		return nil, err
	}
	id, err := randomToken(16)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_membership_audit(org_key,id,events,created_at) VALUES($1,$2,$3,$4)`, s.repo.orgKey, id, string(data), identityTime(at)); err != nil {
		return nil, err
	}
	if err := s.checkpoint("audit"); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.checkpoint("commit"); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *OrganisationMembershipService) houses(ctx context.Context, tx *sql.Tx) (map[string]string, error) {
	var one int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM organisations WHERE org_key=$1`, s.repo.orgKey).Scan(&one); err != nil {
		return nil, fmt.Errorf("Verwaltung nicht gefunden: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT t.tenant_id,h.tenant_slug FROM organisation_houses h LEFT JOIN tenant t ON t.slug=h.tenant_slug WHERE h.org_key=$1 ORDER BY h.tenant_slug`, s.repo.orgKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id sql.NullString
		var slug string
		if err := rows.Scan(&id, &slug); err != nil {
			return nil, err
		}
		if !id.Valid {
			return nil, fmt.Errorf("Liegenschaft %s hat keine Datenbankidentität.", slug)
		}
		out[id.String] = slug
	}
	return out, rows.Err()
}

func loadOrganisationHouse(ctx context.Context, tx *sql.Tx, personID, tenantID, lock string) (HouseMembership, string, error) {
	var m HouseMembership
	var permissions, created, updated, origin string
	err := tx.QueryRowContext(ctx, `SELECT `+membershipColumns+`,grant_origin FROM house_memberships WHERE person_id=$1 AND tenant_id=$2`+lock, personID, tenantID).Scan(&m.PersonID, &m.TenantID, &m.TenantSlug, &m.Role, &permissions, &m.Status, &m.DirectoryOptIn, &created, &updated, &origin)
	if err != nil {
		return m, "", err
	}
	if permissions != "" {
		if err := json.Unmarshal([]byte(permissions), &m.Permissions); err != nil {
			return m, "", fmt.Errorf("unreadable membership permissions: %w", err)
		}
	}
	m.CreatedAt, m.UpdatedAt = parseIdentityTime(created), parseIdentityTime(updated)
	return m, origin, nil
}

// AuditEvents reads the durable source independently of the JSONL mirror.
func (s *OrganisationMembershipService) AuditEvents(ctx context.Context) ([]AuditEvent, error) {
	tx, err := s.repo.begin(ctx, s.repo.orgKey)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT events FROM organisation_membership_audit WHERE org_key=$1 ORDER BY created_at,id`, s.repo.orgKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEvent{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var events []AuditEvent
		if err := json.Unmarshal([]byte(raw), &events); err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return out, tx.Commit()
}
