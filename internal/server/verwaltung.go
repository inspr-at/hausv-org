package server

import (
	"bytes"
	"net/http"
	"sort"
	"strings"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/version"
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

func (a *app) showVerwaltungNav(ac authCtx) bool {
	managed := a.managedTenants(&ac)
	if len(managed) >= 2 {
		return true
	}
	return normalizeSlug(ac.tenant.Organisation) != ""
}

func (a *app) verwaltungShell(ac *authCtx, active string) web.VerwaltungShell {
	managed := a.managedTenants(ac)
	organisationName := "Verwaltung"
	if organisation, ok := a.organisationFor(ac); ok {
		organisationName = organisation.Name
	}
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	roleLabel := ac.role
	houses := make([]web.VerwaltungHouse, 0, len(managed))
	for _, tenant := range managed {
		if tenant.Config.Slug == ac.tenant.Slug {
			roleLabel = tenant.Role
		}
		houses = append(houses, web.VerwaltungHouse{
			Slug: tenant.Config.Slug, Name: houseDisplayName(tenant.Config), Address: tenant.Config.Address, Role: tenant.Role,
		})
	}
	if roleLabel != roleAdmin && roleLabel != roleManager && len(managed) > 0 {
		roleLabel = managed[0].Role
	}
	contexts := a.portalContextsFor(ac.email, ac.tenant.Slug, ac.role)
	portalContexts := make([]web.PortalContext, 0, len(contexts))
	for _, context := range contexts {
		portalContexts = append(portalContexts, web.PortalContext{
			TenantSlug: context.TenantSlug, HouseName: context.HouseName, Address: context.Address, Role: context.Role, Current: context.Current,
		})
	}
	return web.VerwaltungShell{
		OrganisationName: organisationName,
		RoleLabel:        roleLabel,
		DisplayName:      profile.DisplayName(),
		Initials:         profile.Initials(),
		Active:           active,
		InboxOpenCount:   a.inboxOpenCount(ac),
		Houses:           houses,
		Contexts:         portalContexts,
		DisplayVersion:   version.DisplayVersion(version.Version),
		ReleaseNotes:     version.Notes(),
	}
}

func (a *app) requireVerwaltung(next authedHandler) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, ac authCtx) {
		managed := a.managedTenants(&ac)
		allowed := false
		for _, tenant := range managed {
			if can(actorFor(ac.email, tenant.Ref.Slug, tenant.Role), capabilityManageIssues, resourceFor(tenant.Ref.Slug)) {
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
	if err := web.PortalVerwaltungPage(a.verwaltungShell(&ac, active), title, web.VerwaltungPlaceholder()).Render(r.Context(), &rendered); err != nil {
		logError("templ verwaltung render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug)))
}
