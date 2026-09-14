package server

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

type portalContextView struct {
	TenantSlug string
	HouseName  string
	Address    string
	Role       string
	Current    bool
}

// portalBaseData supplies the same identity, permissions, badges and context on every page.
func (a *app) portalBaseData(ac authCtx, activePage, title string) web.PortalPageData {
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	modules := a.portalModulesFor(ac.tenant.Slug)
	unreadAnnouncements := 0
	if ac.repositories.announcements != nil && ac.repositories.announcementReads != nil && strings.TrimSpace(ac.email) != "" {
		now := time.Now()
		unreadAnnouncements = unreadAnnouncementCount(ac.repositories.announcements.Visible(now), ac.repositories.announcementReads.LastSeen(ac.email), now)
	}
	openIssues := 0
	if a.issueStore != nil {
		openIssues = issueOpenCount(a.visibleIssuesForActor(ac.tenantRef, ac.email, ac.role))
	}
	shell := a.portalShellData(&ac)
	data := web.PortalPageData{
		Title:               title + " · " + houseDisplayName(ac.tenant) + " · " + ac.role,
		TenantSlug:          ac.tenant.Slug,
		HouseName:           houseDisplayName(ac.tenant),
		Address:             ac.tenant.Address,
		MapURL:              tenantMapURL(ac.tenant.Address),
		HeroImageURL:        ac.tenant.HeroImageURL,
		BrandIcon:           ac.tenant.BrandIcon,
		BrandMarkSVG:        tenantBrandMarkSVG(ac.tenant.BrandIcon),
		Map:                 portalMapForTenant(ac.tenant),
		DisplayName:         profile.DisplayName(),
		Initials:            profile.Initials(),
		Role:                ac.role,
		DisplayVersion:      version.DisplayVersion(version.Version),
		ActivePage:          activePage,
		Modules:             web.PortalModules{Energy: modules.Energy, Announcements: modules.Announcements, Events: modules.Events, Contacts: modules.Contacts, Documents: modules.Documents, Issues: modules.Issues, Votes: modules.Votes, Parking: modules.Parking, Handovers: modules.Handovers, Users: modules.Users, Audit: modules.Audit, Help: modules.Help},
		CanUseResidentAreas: roleCanUseResidentAreas(ac.role),
		CanViewEnergy:       modules.Energy && a.canViewEnergy(ac),
		HomeIdentity:        a.homeIdentityForActor(ac, modules.Energy && a.canViewEnergy(ac)),
		CanManageIssues:     ac.can(capabilityManageIssues),
		CanSeeParking:       modules.Parking && (ac.can(capabilityManageParking) || ac.can(capabilityPlatformAdmin) || profile.HasPermission(permissionParking)),
		CanManageHandovers:  modules.Handovers && canManageHandovers(ac.actor(), ac.resource()),
		CanManageUsers:      modules.Users && ac.can(capabilityManageUsers),
		CanViewAudit:        modules.Audit && canViewAudit(ac.actor(), ac.resource()),
		Issues:              make([]view.IssueView, openIssues),
		UnreadAnnouncements: unreadAnnouncements,
		Shell:               shell,
		ReleaseNotes:        version.Notes(),
	}
	data.CanCreateResidentIssue = canCreateResidentIssue(ac.actor(), ac.resource())
	data.RolePreview = rolePreviewPortalData(&ac)
	data.RolePreviewChoices = shell.Context.PreviewChoices
	data.Contexts = shell.PortalContexts
	data.ShowVerwaltungNav = shell.IsOrganisationMember
	data.ShowInboxNav = shell.ShowInboxNav
	data.InboxOpenCount = shell.InboxOpenCount
	data.Organisation = web.VerwaltungShell{OrganisationName: shell.OrganisationName, RoleLabel: shell.RoleLabel, ShowInboxNav: shell.ShowInboxNav, CanManageSettings: shell.CanManageOrganisationSettings, InboxOpenCount: shell.InboxOpenCount}
	for _, house := range shell.ManagedHouses {
		data.Organisation.Houses = append(data.Organisation.Houses, web.VerwaltungHouse{Slug: house.Slug, Name: house.Name, Address: house.Address, Role: house.Role})
	}
	return data
}

// portalShellData is the single assembly point for the shared house shell. It
// keeps cross-house issue reads tenant-bound and separates managed houses from
// the personal portal contexts used by the shared switcher.
func (a *app) portalShellData(ac *authCtx) web.PortalShellData {
	shell := web.PortalShellData{Ready: true}
	if a == nil || ac == nil {
		return shell
	}

	shell.RoleLabel = ac.role
	// One assembly point for the account tile, so the sidebar, the mobile header
	// and every section page show the same picture (HAUSV-675).
	shell.AvatarURL = a.profilePictureURL(ac.email)
	managed := a.organisationManagedTenants(context.Background(), ac)
	organisation, hasOrganisation := a.organisationRecordFor(context.Background(), ac)
	// The Verwaltung layer needs someone who actually administers houses: a
	// member of an organisation, or somebody who administers more than one
	// house without one. Residents and owners never see it, even though their
	// house belongs to an organisation.
	shell.IsOrganisationMember = len(managed) > 0 && (hasOrganisation || len(managed) > 1)
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

	contexts := a.portalContextsForActor(ac)
	// The picker offers every house the person can switch to: the ones they
	// administer plus the ones they own or rent. ScopeContext also preserves
	// every personal role and private Home portal in the shared panel.
	seen := make(map[string]bool)
	for _, tenant := range managed {
		seen[tenant.Config.Slug] = true
		shell.ManagedHouses = append(shell.ManagedHouses, portalHouseMetadata(portalContextView{
			TenantSlug: tenant.Config.Slug, HouseName: houseDisplayName(tenant.Config), Address: tenant.Config.Address,
			Role: tenant.Role, Current: tenant.Config.Slug == ac.tenant.Slug,
		}))
	}
	for _, context := range contexts {
		if seen[context.TenantSlug] || !a.isCommunityPortal(context.TenantSlug) {
			continue
		}
		seen[context.TenantSlug] = true
		shell.ManagedHouses = append(shell.ManagedHouses, portalHouseMetadata(context))
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
	shell.Context = a.scopeContext(ac, shell.OrganisationName, false)
	return shell
}

func portalHouseMetadata(context portalContextView) web.PortalHouse {
	return web.PortalHouse{Slug: context.TenantSlug, Name: context.HouseName, Address: context.Address, Role: context.Role, Current: context.Current}
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
// portals remain personal contexts in the shared panel, outside the
// Liegenschaften count.
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
	next := portalContextNext(r.FormValue("next"))
	http.Redirect(w, r, strings.TrimRight(a.baseURL, "/")+target.PublicURL(next), http.StatusSeeOther)
}

// Only known house menu destinations can continue a context switch.
func portalContextNext(path string) string {
	switch path {
	case "/app", "/app/energie", "/app/announcements", "/app/events", "/app/kontakte", "/app/dokumente", "/app/anliegen", "/app/anliegen/board", "/app/abstimmungen", "/app/parking", "/app/uebergaben", "/app/settings/users", "/app/audit", "/app/settings", "/app/hilfe":
		return path
	default:
		return "/app"
	}
}
