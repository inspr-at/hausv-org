package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

type portalContextView struct {
	TenantSlug string
	HouseName  string
	Address    string
	Role       string
	Current    bool
}

// portalShellData is the single assembly point for the shared house shell. It
// keeps cross-house issue reads tenant-bound and separates managed houses from
// the personal portal contexts shown by "Portal wechseln".
func (a *app) portalShellData(ac *authCtx) web.PortalShellData {
	shell := web.PortalShellData{Ready: true}
	if a == nil || ac == nil {
		return shell
	}

	shell.RoleLabel = ac.role
	managed := a.managedTenants(ac)
	organisation, hasOrganisation := a.organisationFor(ac)
	// The Verwaltung layer follows the same rule as before this shell existed
	// (showVerwaltungNav): an organisation, or somebody who administers more
	// than one house. Narrowing it to organisations only took Portfolio and
	// Posteingang away from multi-house admins.
	shell.IsOrganisationMember = a.showVerwaltungNav(*ac)
	if shell.IsOrganisationMember {
		shell.OrganisationName = strings.TrimSpace(organisation.Name)
		if shell.OrganisationName == "" {
			shell.OrganisationName = "Verwaltung"
		}
		shell.ShowInboxNav = hasOrganisation
		if hasOrganisation {
			shell.InboxOpenCount = a.inboxOpenCount(ac)
		}
		shell.CanManageOrganisationSettings = a.isOrganisationAdmin(ac)
		shell.IsVerwaltung = shell.CanManageOrganisationSettings || len(managed) > 1
	}

	contexts := a.portalContextsFor(ac.email, ac.tenant.Slug, ac.role)
	// The picker offers every house the person can switch to: the ones they
	// administer plus the ones they own or rent. Only Home-style portals stay
	// in the separate portal switcher.
	seen := make(map[string]bool)
	for _, tenant := range managed {
		seen[tenant.Config.Slug] = true
		shell.ManagedHouses = append(shell.ManagedHouses, a.portalHouseForContext(ac, portalContextView{
			TenantSlug: tenant.Config.Slug, HouseName: houseDisplayName(tenant.Config), Address: tenant.Config.Address,
			Role: tenant.Role, Current: tenant.Config.Slug == ac.tenant.Slug,
		}))
	}
	for _, context := range contexts {
		if seen[context.TenantSlug] || !a.isCommunityPortal(context.TenantSlug) {
			continue
		}
		seen[context.TenantSlug] = true
		shell.ManagedHouses = append(shell.ManagedHouses, a.portalHouseForContext(ac, context))
	}
	for index := range shell.ManagedHouses {
		if shell.ManagedHouses[index].Current {
			shell.CurrentHousePosition = index + 1
			break
		}
	}

	for _, context := range contexts {
		if a.isCommunityPortal(context.TenantSlug) {
			continue
		}
		shell.PortalContexts = append(shell.PortalContexts, web.PortalContext{
			TenantSlug: context.TenantSlug, HouseName: context.HouseName, Address: context.Address,
			Role: context.Role, Current: context.Current,
		})
	}
	return shell
}

func (a *app) portalHouseForContext(ac *authCtx, context portalContextView) web.PortalHouse {
	house := web.PortalHouse{
		Slug: context.TenantSlug, Name: context.HouseName, Address: context.Address,
		Role: context.Role, Current: context.Current,
	}
	if a == nil || ac == nil || a.issueStore == nil {
		return house
	}
	if identity, ok := a.tenantIdentity(context.TenantSlug); ok {
		house.OpenIssueCount = issueOpenCount(a.visibleIssuesForActor(identity.Ref(), ac.email, context.Role))
	}
	return house
}

// isCommunityPortal identifies a Hausportal/Liegenschaft. Private Home
// portals deliberately stay in "Portal wechseln" rather than joining the
// house picker.
func (a *app) isCommunityPortal(slug string) bool {
	tenant, ok := a.tenantBySlug(slug)
	if !ok {
		return false
	}
	portalType := strings.ToLower(strings.TrimSpace(tenant.PortalType))
	return portalType == config.PortalTypeCommunity
}

func (a *app) portalContextsFor(email string, currentTenant string, currentRole string) []portalContextView {
	email = normalizeEmail(email)
	currentTenant = normalizeSlug(currentTenant)
	currentRole = normalizeRole(currentRole)
	slugs := a.ownTenantSlugs(email)
	if currentTenant != "" {
		found := false
		for _, slug := range slugs {
			found = found || slug == currentTenant
		}
		if !found {
			slugs = append(slugs, currentTenant)
		}
	}
	sort.Slice(slugs, func(i, j int) bool {
		left, leftOK := a.tenantBySlug(slugs[i])
		right, rightOK := a.tenantBySlug(slugs[j])
		if !leftOK || !rightOK {
			return slugs[i] < slugs[j]
		}
		leftName := strings.ToLower(houseDisplayName(left))
		rightName := strings.ToLower(houseDisplayName(right))
		if leftName == rightName {
			return left.Slug < right.Slug
		}
		return leftName < rightName
	})

	contexts := []portalContextView{}
	for _, slug := range slugs {
		if !a.isAllowed(email, slug) {
			continue
		}
		tenant, ok := a.tenantBySlug(slug)
		if !ok {
			continue
		}
		identity, ok := a.tenantIdentity(slug)
		if !ok {
			continue
		}
		for _, role := range a.ownRolesForTenant(email, identity.Ref()) {
			contexts = append(contexts, portalContextView{
				TenantSlug: slug,
				HouseName:  houseDisplayName(tenant),
				Address:    tenant.Address,
				Role:       role,
				Current:    slug == currentTenant && role == currentRole,
			})
		}
	}
	return contexts
}

func (a *app) ownTenantSlugs(email string) []string {
	email = normalizeEmail(email)
	seen := map[string]struct{}{}
	if profile, ok := a.directoryProfile(email); ok {
		for _, slug := range profile.Tenants {
			if slug = normalizeSlug(slug); slug != "" {
				seen[slug] = struct{}{}
			}
		}
		for slug := range profile.TenantMemberships {
			if slug = normalizeSlug(slug); slug != "" {
				seen[slug] = struct{}{}
			}
		}
	}
	if a.homePortals != nil {
		if portals, err := a.homePortals.ListByOwner(email); err == nil {
			for _, portal := range portals {
				if slug := normalizeSlug(portal.Slug); slug != "" {
					seen[slug] = struct{}{}
				}
			}
		}
	}
	for slug := range a.tenants {
		if a.isConfirmedHomePortalOwner(email, slug) {
			seen[normalizeSlug(slug)] = struct{}{}
		}
	}
	if len(seen) == 0 {
		if _, ok := a.allowed[email]; ok {
			seen[a.defaultTenant] = struct{}{}
		}
		if _, ok := a.admins[email]; ok {
			seen[a.defaultTenant] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for slug := range seen {
		if _, ok := a.tenantBySlug(slug); ok {
			out = append(out, slug)
		}
	}
	sort.Strings(out)
	return out
}

func (a *app) ownsPortalTenant(email string, tenantSlug string) bool {
	tenantSlug = normalizeSlug(tenantSlug)
	for _, slug := range a.ownTenantSlugs(email) {
		if slug == tenantSlug {
			return true
		}
	}
	return false
}

// ownRolesForTenant derives selectable roles only from actual assignments. The
// directory membership supplies the primary role; unit assignments can add an
// owner or renter role in the same property. Viewing another person's identity
// remains a separate, audited admin feature.
func (a *app) ownRolesForTenant(email string, tenant store.TenantRef) []string {
	tenantSlug := tenant.Slug
	primary := normalizeRole(a.roleFor(email, tenantSlug))
	roles := make([]string, 0, 2)
	seen := map[string]struct{}{}
	add := func(role string) {
		role = normalizeRole(role)
		if role == "" {
			return
		}
		if _, ok := seen[role]; ok {
			return
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	add(primary)
	if a.unitStore != nil {
		units, _ := store.BindUnitRepository(a.unitStore, tenant)
		if units != nil {
			for _, membership := range units.UnitsForEmail(email) {
				add(membership.Relation)
			}
		}
	}
	return roles
}

func (a *app) ownPortalContextAllowed(email string, tenantSlug string, role string, authMethod string) bool {
	if !a.ownsPortalTenant(email, tenantSlug) || !a.isAllowed(email, tenantSlug) || !a.isAuthMethodAllowed(email, tenantSlug, authMethod) {
		return false
	}
	role = normalizeRole(role)
	identity, ok := a.tenantIdentity(tenantSlug)
	if !ok {
		return false
	}
	for _, allowed := range a.ownRolesForTenant(email, identity.Ref()) {
		if role == allowed {
			return true
		}
	}
	return false
}

func (a *app) switchPortalContext(w http.ResponseWriter, r *http.Request, ac authCtx) {
	cookie, err := r.Cookie("weg_session")
	if err != nil {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	current, ok := a.sessions.GetSession(cookie.Value)
	if !ok || current.Email != ac.email || current.TenantSlug != ac.tenant.Slug {
		http.Error(w, "Anmeldung erforderlich", http.StatusUnauthorized)
		return
	}
	targetSlug := normalizeSlug(r.FormValue("tenant"))
	targetRole := normalizeRole(r.FormValue("role"))
	if !a.ownPortalContextAllowed(current.Email, targetSlug, targetRole, current.AuthMethod) {
		http.Error(w, "Dieser Portal-Kontext ist nicht freigegeben.", http.StatusForbidden)
		return
	}
	target, ok := a.tenantBySlug(targetSlug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	expiresAt := time.Unix(current.ExpiresAt, 0)
	token, _, err := a.sessions.PutSession(current.Email, targetSlug, current.AuthMethod, targetRole, expiresAt)
	if err != nil {
		http.Error(w, "Portal konnte nicht gewechselt werden.", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "weg_session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.sessionSecure,
		SameSite: http.SameSiteLaxMode,
	})
	a.sessions.Delete(cookie.Value)
	a.recordAudit(auditEvent{
		TenantSlug: targetSlug,
		ActorEmail: current.Email,
		ActorRole:  targetRole,
		Action:     auditActionContextSwitch,
		TargetType: "portal_context",
		TargetID:   targetSlug,
		Summary:    "Portal-Kontext gewechselt",
		Details: map[string]string{
			"tenant_from": ac.tenant.Slug,
			"tenant_to":   targetSlug,
			"role_from":   ac.role,
			"role_to":     targetRole,
		},
	})
	http.Redirect(w, r, strings.TrimRight(a.baseURL, "/")+target.PublicURL("/app"), http.StatusSeeOther)
}
