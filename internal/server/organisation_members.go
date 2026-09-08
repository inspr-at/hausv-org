package server

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/inspr-at/hausv-org/internal/store"
)

// organisationMemberView is one employee as a page shows them.
type organisationMemberView struct {
	Email     string
	Name      string
	Role      string
	RoleLabel string
	Houses    int
	Since     string
}

// houseRoleForOrganisationRole maps the organisation role to the role the
// employee receives in every house the Verwaltung carries. Authorization keeps
// reading the house role; the organisation only hands it out in one go.
func houseRoleForOrganisationRole(role string) string {
	if store.NormalizeOrganisationRole(role) == store.OrganisationRoleAdmin {
		return roleAdmin
	}
	return roleManager
}

func organisationRoleLabel(role string) string {
	if store.NormalizeOrganisationRole(role) == store.OrganisationRoleAdmin {
		return "Verwaltungs-Admin"
	}
	return "Sachbearbeiter"
}

func (a *app) organisationMembers(ctx context.Context, orgKey string) ([]organisationMemberView, error) {
	if a == nil || a.organisationMemberRepo == nil {
		return nil, nil
	}
	orgKey = normalizeSlug(orgKey)
	if orgKey == "" {
		return nil, nil
	}
	repo := a.organisationMemberRepo(orgKey)
	if repo == nil {
		return nil, nil
	}
	members, err := repo.List(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]organisationMemberView, 0, len(members))
	for _, member := range members {
		profile := a.profileFor(member.Email)
		name := strings.TrimSpace(profile.DisplayName())
		if name == "" || name == member.Email {
			name = ""
		}
		views = append(views, organisationMemberView{
			Email:     member.Email,
			Name:      name,
			Role:      member.Role,
			RoleLabel: organisationRoleLabel(member.Role),
			Houses:    len(member.Granted),
			Since:     member.CreatedAt.Local().Format("02.01.2006"),
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Email < views[j].Email })
	return views, nil
}

// addOrganisationMember gives one person the organisation's role in every house
// the Verwaltung carries, and records for each house what it was before so
// leaving can undo exactly this and nothing else.
func (a *app) addOrganisationMember(ctx context.Context, ac *authCtx, orgKey string, email string, role string) error {
	if a == nil || a.organisationMemberRepo == nil || a.inviteStore == nil {
		return fmt.Errorf("Mitarbeiterverwaltung ist nicht verfügbar.")
	}
	orgKey = normalizeSlug(orgKey)
	email = normalizeEmail(email)
	if orgKey == "" || email == "" {
		return fmt.Errorf("Bitte eine E-Mail-Adresse angeben.")
	}
	if _, ok := a.directoryProfile(email); !ok {
		return fmt.Errorf("Diese Person ist noch nicht angelegt. Bitte zuerst in einer Liegenschaft einladen.")
	}
	role = store.NormalizeOrganisationRole(role)
	houseRole := houseRoleForOrganisationRole(role)

	repo := a.organisationMemberRepo(orgKey)
	existing, found, err := repo.Get(ctx, email)
	if err != nil {
		return err
	}
	granted := map[string]string{}
	if found {
		// A role change keeps the original undo record: what matters is the
		// state before the organisation touched the house, not before the
		// change.
		for slug, previous := range existing.Granted {
			granted[slug] = previous
		}
	}

	houses := a.configuredOrganisationHouses(orgKey)
	if stored, ok, err := a.organisationRepo(orgKey).Get(ctx); err == nil && ok && len(stored.Houses) > 0 {
		houses = stored.Houses
	}
	changed := make([]string, 0, len(houses))
	for _, slug := range houses {
		if _, ok := a.tenantBySlug(slug); !ok {
			continue
		}
		current := normalizeRole(a.roleFor(email, slug))
		if current == houseRole {
			continue
		}
		if _, recorded := granted[slug]; !recorded {
			if profile, ok := a.directoryProfile(email); ok && profile.HasTenant(slug) {
				granted[slug] = current
			} else {
				granted[slug] = ""
			}
		}
		if _, _, err := a.inviteStore.SetTenantMembership(email, slug, houseRole, nil); err != nil {
			return err
		}
		changed = append(changed, slug)
	}

	member := store.OrganisationMember{OrgKey: orgKey, Email: email, Role: role, Granted: granted}
	if found {
		member.CreatedAt = existing.CreatedAt
	}
	if err := repo.Save(ctx, member); err != nil {
		return err
	}
	a.auditOrganisationMember(ac, orgKey, email, "Mitarbeiter aufgenommen", map[string]string{
		"organisation_role": role,
		"house_role":        houseRole,
		"houses_changed":    strings.Join(changed, ", "),
	})
	return nil
}

// removeOrganisationMember withdraws exactly the house roles this membership
// granted. A role someone set by hand afterwards is left alone: the record says
// what the organisation did, and only that is undone.
func (a *app) removeOrganisationMember(ctx context.Context, ac *authCtx, orgKey string, email string) error {
	if a == nil || a.organisationMemberRepo == nil || a.inviteStore == nil {
		return fmt.Errorf("Mitarbeiterverwaltung ist nicht verfügbar.")
	}
	orgKey = normalizeSlug(orgKey)
	email = normalizeEmail(email)
	repo := a.organisationMemberRepo(orgKey)
	member, found, err := repo.Get(ctx, email)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("Diese Person gehört nicht zur Verwaltung.")
	}
	houseRole := houseRoleForOrganisationRole(member.Role)

	slugs := make([]string, 0, len(member.Granted))
	for slug := range member.Granted {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	restored := make([]string, 0, len(slugs))
	kept := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		previous := normalizeRole(member.Granted[slug])
		if normalizeRole(a.roleFor(email, slug)) != houseRole {
			// Someone changed this house by hand since. Leave it.
			kept = append(kept, slug)
			continue
		}
		if member.Granted[slug] == "" {
			if _, _, err := a.inviteStore.RemoveTenant(email, slug); err != nil {
				return err
			}
		} else if _, _, err := a.inviteStore.SetTenantMembership(email, slug, previous, nil); err != nil {
			return err
		}
		restored = append(restored, slug)
	}
	if _, err := repo.Delete(ctx, email); err != nil {
		return err
	}
	a.auditOrganisationMember(ac, orgKey, email, "Mitarbeiter entfernt", map[string]string{
		"organisation_role": member.Role,
		"houses_restored":   strings.Join(restored, ", "),
		"houses_untouched":  strings.Join(kept, ", "),
	})
	return nil
}

func (a *app) auditOrganisationMember(ac *authCtx, orgKey string, email string, summary string, details map[string]string) {
	if a == nil || ac == nil {
		return
	}
	for _, tenant := range a.managedTenants(ac) {
		a.recordAudit(store.AuditEvent{
			TenantSlug: tenant.Ref.Slug,
			ActorEmail: ac.email,
			ActorRole:  tenant.Role,
			Action:     store.AuditActionVerwaltungSettings,
			TargetType: "organisation_member",
			TargetID:   email,
			Summary:    summary,
			Details:    details,
		})
	}
	_ = orgKey
}
