package server

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

type portalContextView struct {
	TenantSlug string
	HouseName  string
	Address    string
	Role       string
	Current    bool
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
		for _, role := range a.ownRolesForTenant(email, slug) {
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
func (a *app) ownRolesForTenant(email string, tenantSlug string) []string {
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
		for _, membership := range a.unitStore.UnitsForEmail(tenantSlug, email) {
			add(membership.Relation)
		}
	}
	return roles
}

func (a *app) ownPortalContextAllowed(email string, tenantSlug string, role string, authMethod string) bool {
	if !a.ownsPortalTenant(email, tenantSlug) || !a.isAllowed(email, tenantSlug) || !a.isAuthMethodAllowed(email, tenantSlug, authMethod) {
		return false
	}
	role = normalizeRole(role)
	for _, allowed := range a.ownRolesForTenant(email, tenantSlug) {
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
