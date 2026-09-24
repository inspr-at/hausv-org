package server

import "github.com/inspr-at/hausv-org/internal/store"

// organisationAccessNoHouse is a tenant slug no house can be given. An empty
// house list must not become "every row in the organisation": intakeWhere adds
// a tenant predicate only when a slug or slug list is present.
const organisationAccessNoHouse = "\x00"

// organisationAccess is the authorization decision for one Verwaltung request.
// The organisation is the selected house's organisation. Management rights in a
// different organisation do not admit the actor, and they do not contribute
// houses or unassigned items.
//
// SeesUnassigned follows the existing inbox rule, not organisation-admin
// status. managedIntakeFilter and actorCanAccessIntake show unassigned items
// to every manager of the organisation who is not in a support view.
// TestInboxQueueAndCountsAreScopedToManagedHouses locks that: a house-only
// manager sees the organisation's unassigned queue and only the assigned items
// of houses they manage. A support view never includes unassigned items.
type organisationAccess struct {
	Key            string
	Name           string
	Houses         []managedTenant
	SeesUnassigned bool
}

func (a *app) organisationAccessFor(ac *authCtx) organisationAccess {
	if a == nil || ac == nil || ac.preview != nil {
		return organisationAccess{}
	}
	selectedOrg := normalizeSlug(ac.tenant.Organisation)
	access := organisationAccess{}
	if selectedOrg != "" {
		if organisation, ok := a.organisations[selectedOrg]; ok && normalizeSlug(organisation.Key) != "" {
			access.Key = normalizeSlug(organisation.Key)
			access.Name = organisation.Name
			selectedOrg = access.Key
		}
	}
	for _, house := range a.managedTenants(ac) {
		if normalizeSlug(house.Config.Organisation) != selectedOrg {
			continue
		}
		access.Houses = append(access.Houses, house)
	}
	access.SeesUnassigned = access.Key != "" && ac.supportView == nil && len(access.Houses) > 0
	return access
}

func (access organisationAccess) allowed() bool {
	return len(access.Houses) > 0
}

func (access organisationAccess) managesHouse(slug string) bool {
	slug = normalizeSlug(slug)
	if slug == "" {
		return false
	}
	for _, house := range access.Houses {
		if normalizeSlug(house.Ref.Slug) == slug || normalizeSlug(house.Config.Slug) == slug {
			return true
		}
	}
	return false
}

func (access organisationAccess) intakeFilter(statuses []store.IntakeStatus) store.IntakeFilter {
	filter := store.IntakeFilter{Statuses: statuses, IncludeUnassigned: access.SeesUnassigned}
	for _, house := range access.Houses {
		if slug := normalizeSlug(house.Ref.Slug); slug != "" {
			filter.TenantSlugs = append(filter.TenantSlugs, slug)
		}
	}
	if len(filter.TenantSlugs) == 0 {
		filter.IncludeUnassigned = false
		filter.TenantSlug = organisationAccessNoHouse
	}
	return filter
}
