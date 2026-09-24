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

// Organisation membership writes have one store-owned transaction. Returned
// audit events are committed already; JSONL remains the existing UI mirror.
func (a *app) addOrganisationMember(ctx context.Context, ac *authCtx, orgKey, email, role string) error {
	service, err := a.organisationMembershipService(orgKey)
	if err != nil {
		return err
	}
	events, err := service.Add(ctx, email, role, organisationMemberActor(ac))
	if err != nil {
		return err
	}
	for _, event := range events {
		a.recordAudit(event)
	}
	return nil
}

func (a *app) removeOrganisationMember(ctx context.Context, ac *authCtx, orgKey, email string) error {
	service, err := a.organisationMembershipService(orgKey)
	if err != nil {
		return err
	}
	events, err := service.Remove(ctx, email, organisationMemberActor(ac))
	if err != nil {
		return err
	}
	for _, event := range events {
		a.recordAudit(event)
	}
	return nil
}

func (a *app) organisationMembershipService(orgKey string) (*store.OrganisationMembershipService, error) {
	if a == nil || a.organisationMemberRepo == nil || a.identityStore == nil {
		return nil, fmt.Errorf("Mitarbeiterverwaltung ist nicht verfügbar.")
	}
	return store.NewOrganisationMembershipService(a.organisationMemberRepo(normalizeSlug(orgKey)), a.identityStore)
}

func organisationMemberActor(ac *authCtx) store.AuditEvent {
	if ac == nil {
		return store.AuditEvent{}
	}
	return store.AuditEvent{ActorEmail: ac.email, ActorRole: ac.role}
}
