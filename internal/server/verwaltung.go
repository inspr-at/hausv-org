package server

import (
	"bytes"
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

type managedTenant struct {
	Ref    store.TenantRef
	Config config.TenantConfig
	Role   string
}

func (a *app) managedTenants(ac *authCtx) []managedTenant {
	if a == nil || ac == nil {
		return nil
	}
	// A role preview narrows the effective role to Eigentümer or Bewohner; neither
	// manages houses, so the Verwaltung layer disappears for the duration.
	if ac.preview != nil {
		return nil
	}
	email := normalizeEmail(ac.email)
	_, breakGlass := a.admins[email]
	slugs := a.ownTenantSlugs(email)
	if breakGlass {
		slugs = make([]string, 0, len(a.tenants))
		for slug := range a.tenants {
			slugs = append(slugs, normalizeSlug(slug))
		}
	}
	seen := map[string]struct{}{}
	managed := make([]managedTenant, 0, len(slugs))
	for _, slug := range slugs {
		slug = normalizeSlug(slug)
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		tenant, ok := a.tenantBySlug(slug)
		if !ok {
			continue
		}
		identity, ok := a.tenantIdentity(slug)
		if !ok {
			continue
		}
		role := normalizeRole(a.roleFor(email, slug))
		if breakGlass {
			role = roleAdmin
		}
		if role != roleAdmin && role != roleManager {
			continue
		}
		managed = append(managed, managedTenant{Ref: identity.Ref(), Config: tenant, Role: role})
	}
	sort.Slice(managed, func(i, j int) bool {
		left := strings.ToLower(houseDisplayName(managed[i].Config))
		right := strings.ToLower(houseDisplayName(managed[j].Config))
		if left == right {
			return managed[i].Config.Slug < managed[j].Config.Slug
		}
		return left < right
	})
	return managed
}

// repositoriesFor returns a fresh repository bundle bound to exactly ref.
// Cross-house pages must call it once per managed tenant and never combine
// tenant identifiers in a store query.
func (a *app) repositoriesFor(ref store.TenantRef) requestRepositories {
	return a.repositoriesForTenant(ref)
}

func (a *app) organisationFor(ac *authCtx) (config.OrganisationConfig, bool) {
	if a == nil || ac == nil {
		return config.OrganisationConfig{}, false
	}
	lookup := func(tenant config.TenantConfig) (config.OrganisationConfig, bool) {
		organisation, ok := a.organisations[normalizeSlug(tenant.Organisation)]
		return organisation, ok
	}
	if organisation, ok := lookup(ac.tenant); ok {
		return organisation, true
	}
	managed := a.managedTenants(ac)
	if len(managed) == 0 {
		return config.OrganisationConfig{}, false
	}
	return lookup(managed[0].Config)
}

func (a *app) isOrganisationAdmin(ac *authCtx) bool {
	if a == nil || ac == nil || ac.preview != nil {
		return false
	}
	email := normalizeEmail(ac.email)
	if _, ok := a.admins[email]; ok {
		return true
	}
	organisation, ok := a.organisationFor(ac)
	if !ok {
		return false
	}
	found := false
	for slug, tenant := range a.tenants {
		if normalizeSlug(tenant.Organisation) != normalizeSlug(organisation.Key) {
			continue
		}
		found = true
		if normalizeRole(a.roleFor(email, slug)) != roleAdmin {
			return false
		}
	}
	return found
}

func (a *app) showVerwaltungNav(ac authCtx) bool {
	if ac.preview != nil {
		return false
	}
	managed := a.managedTenants(&ac)
	if len(managed) >= 2 {
		return true
	}
	return normalizeSlug(ac.tenant.Organisation) != ""
}

// The authenticated session retains the last selected house when opening an
// organisation route. Keep that house's permissions and module links; only the
// switcher's presentation changes to the portfolio overview.
func (a *app) verwaltungShell(ctx context.Context, ac *authCtx, active string) web.PortalPageData {
	house := a.organisationHouseContext(ctx, ac)
	data := a.portalBaseData(house, "organisation-"+active, "Verwaltung")
	if house.tenant.Slug != ac.tenant.Slug || house.role != ac.role {
		data.NavigationTenant = house.tenant.Slug
		data.NavigationRole = house.role
	}
	if organisation, ok := a.organisationRecordFor(ctx, ac); ok {
		data.Organisation.OrganisationName = organisation.Name
		data.Shell.OrganisationName = organisation.Name
	}
	data.Organisation.Active = active
	data.Overview = true
	data.Shell.Context = a.scopeContext(ac, data.Organisation.OrganisationName, true)
	return data
}

func (a *app) requireVerwaltung(next authedHandler) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, ac authCtx) {
		managed := a.managedTenants(&ac)
		allowed := false
		for _, tenant := range managed {
			// HAUSV-699: the organisation administration (non-delegable rights) keeps
			// the Verwaltung area even when it switches off delegable rights for
			// its own family; otherwise it could lock itself out of the rights page.
			actor, resource := a.actorFor(ac.email, tenant.Ref.Slug, tenant.Role), resourceFor(tenant.Ref.Slug)
			if can(actor, capabilityManageIssues, resource) || can(actor, capabilityManageUsers, resource) || can(actor, capabilityPlatformAdmin, resource) {
				allowed = true
				break
			}
		}
		if !allowed {
			http.Error(w, "Dieser Bereich ist der Verwaltung vorbehalten.", http.StatusForbidden)
			return
		}
		next(w, r, ac)
	}
}

func (a *app) renderVerwaltungPage(w http.ResponseWriter, r *http.Request, ac authCtx, active, title string) {
	var rendered bytes.Buffer
	if err := web.PortalVerwaltungPage(a.verwaltungShell(r.Context(), &ac, active), title, web.VerwaltungPlaceholder()).Render(r.Context(), &rendered); err != nil {
		logError("templ verwaltung render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug)))
}

// A community session is already the last opened house. If the person visits
// their private Home in between, the existing, durable context-switch audit
// retains the last house per person. Revalidate every candidate against today's
// assignments; a revoked role or house can never be revived by history.
func (a *app) organisationHouseContext(ctx context.Context, ac *authCtx) authCtx {
	if a.isSwitcherProperty(ac.tenant.Slug) {
		return *ac
	}
	contexts := a.portalContextsFor(ac.email, ac.tenant.Slug, ac.role)
	selectHouse := func(slug, role string) (authCtx, bool) {
		if !a.isSwitcherProperty(slug) {
			return authCtx{}, false
		}
		for _, candidate := range contexts {
			if candidate.TenantSlug != slug || candidate.Role != role {
				continue
			}
			tenant, found := a.tenantBySlug(slug)
			identity, identified := a.tenantIdentity(slug)
			if !found || !identified {
				continue
			}
			house := *ac
			house.tenant, house.tenantRef, house.role = tenant, identity.Ref(), role
			house.repositories = a.repositoriesFor(identity.Ref())
			house.policy = a.capabilityPolicy(ctx, tenant, ac.preview != nil)
			return house, true
		}
		return authCtx{}, false
	}
	if a.auditStore != nil {
		for _, event := range a.auditStore.List(auditFilter{Action: auditActionContextSwitch, Query: ac.email, Limit: 100}) {
			if normalizeEmail(event.ActorEmail) != normalizeEmail(ac.email) {
				continue
			}
			if house, ok := selectHouse(event.Details["tenant_to"], event.Details["role_to"]); ok {
				return house
			}
			if house, ok := selectHouse(event.Details["tenant_from"], event.Details["role_from"]); ok {
				return house
			}
		}
	}
	for _, tenant := range a.organisationManagedTenants(ctx, ac) {
		if house, ok := selectHouse(tenant.Config.Slug, tenant.Role); ok {
			return house
		}
	}
	return *ac
}
