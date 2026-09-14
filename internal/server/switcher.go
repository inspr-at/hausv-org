package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/web"
)

// switcherDirectory contains metadata only. No cross-tenant issue reads are
// performed until a bounded set of entries is actually displayed.
func (a *app) switcherDirectory(ac *authCtx) []web.LiegenschaftEntry {
	if ac.preview != nil || ac.supportView != nil {
		return []web.LiegenschaftEntry{{Key: ac.tenant.Slug + "|" + ac.role, Tenant: ac.tenant.Slug, Name: houseDisplayName(ac.tenant), Address: ac.tenant.Address, Role: ac.role, Group: "Persönlich", Current: true}}
	}
	contexts := a.portalContextsFor(ac.email, ac.tenant.Slug, ac.role)
	entries := make([]web.LiegenschaftEntry, 0, len(contexts))
	for _, c := range contexts {
		group := "Persönlich"
		if a.isSwitcherProperty(c.TenantSlug) && (c.Role == roleAdmin || c.Role == roleManager) {
			group = "Verwaltung"
		}
		entries = append(entries, web.LiegenschaftEntry{Key: c.TenantSlug + "|" + c.Role, Tenant: c.TenantSlug, Name: c.HouseName, Address: c.Address, Role: c.Role, Group: group, Current: c.Current})
	}
	return entries
}

func (a *app) switcherCounts(ac *authCtx, entries []web.LiegenschaftEntry) {
	for i := range entries {
		e := &entries[i]
		house := a.portalHouseForContext(ac, portalContextView{TenantSlug: e.Tenant, HouseName: e.Name, Address: e.Address, Role: e.Role, Current: e.Current})
		e.Open = house.OpenIssueCount
	}
}

// scopeContext is the one assembly point for identity, role preview and scope
// in both shells. Overview pages deliberately have no active property segment.
func (a *app) scopeContext(ac *authCtx, organisation string, overview bool) web.ScopeContext {
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	data := web.ScopeContext{Ready: true, CanUseSettings: roleCanUseResidentAreas(ac.role), DisplayName: profile.DisplayName(), Initials: profile.Initials(), AvatarURL: a.profilePictureURL(ac.email), Role: ac.role, SupportView: supportViewPortalData(ac), Preview: rolePreviewPortalData(ac), PreviewChoices: a.rolePreviewChoices(ac)}
	if a.canStartSupportView(ac) {
		data.SupportURL = ac.tenant.PublicURL("/app/support-view")
	}
	if ac.preview != nil && ac.realRole != "" {
		data.Role = ac.realRole
	}
	entries := a.switcherDirectory(ac)
	key := sha256.Sum256([]byte(normalizeEmail(ac.email)))
	data.Switcher = web.LiegenschaftSwitcherData{StorageKey: fmt.Sprintf("hausv:recent:%x", key[:12]), SearchURL: ac.tenant.PublicURL("/app/liegenschaften/suche"), PortfolioURL: "/app/liegenschaften"}
	seen := map[string]bool{}
	for _, entry := range entries {
		if a.isSwitcherProperty(entry.Tenant) && !seen[entry.Tenant] {
			data.Switcher.Count++
			seen[entry.Tenant] = true
		}
		if entry.Current && !overview {
			current := entry
			data.Switcher.Current = &current
		}
	}
	if data.Switcher.Current != nil {
		current := []web.LiegenschaftEntry{*data.Switcher.Current}
		a.switcherCounts(ac, current)
		data.Switcher.Current = &current[0]
	}
	// Up to three per group makes the distinction discoverable without serialising
	// an unbounded personal directory into every shell and breadcrumb.
	groupCounts := map[string]int{}
	for _, e := range entries {
		if e.Current && !overview || groupCounts[e.Group] >= 3 {
			continue
		}
		groupCounts[e.Group]++
		data.Switcher.Entries = append(data.Switcher.Entries, e)
	}
	a.switcherCounts(ac, data.Switcher.Entries)
	if organisation != "" && ac.preview == nil {
		data.Segments = append(data.Segments, organisation)
		data.Switcher.PortfolioURL = "/app/verwaltung"
	}
	label := houseDisplayName(ac.tenant)
	if overview {
		label = "Alle Liegenschaften"
	}
	data.Segments = append(data.Segments, label)
	return data
}

func switcherMatches(e web.LiegenschaftEntry, query string) bool {
	text := strings.ToLower(e.Name + " " + e.Address + " " + e.Tenant + " " + e.Role + " " + e.Group)
	for _, word := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(text, word) {
			return false
		}
	}
	return true
}

func (a *app) liegenschaftenSearch(w http.ResponseWriter, r *http.Request, ac authCtx) {
	w.Header().Set("Cache-Control", "private, no-store")
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	entries := a.switcherDirectory(&ac)
	// Match the same authentication-method restriction as POST /app/context.
	if cookie, err := r.Cookie("weg_session"); err == nil {
		if session, ok := a.sessions.GetSession(cookie.Value); ok {
			allowed := entries[:0]
			for _, e := range entries {
				if a.isAuthMethodAllowed(ac.email, e.Tenant, session.AuthMethod) {
					allowed = append(allowed, e)
				}
			}
			entries = allowed
		}
	}
	matches := make([]web.LiegenschaftEntry, 0)
	if r.URL.Query().Has("recent") {
		byKey := make(map[string]web.LiegenschaftEntry, len(entries))
		for _, e := range entries {
			byKey[e.Key] = e
		}
		seen := map[string]bool{}
		for _, key := range r.URL.Query()["recent"] {
			if e, ok := byKey[key]; ok && !e.Current && !seen[key] {
				matches = append(matches, e)
				seen[key] = true
				if len(matches) == 6 {
					break
				}
			}
		}
	} else {
		for _, e := range entries {
			if switcherMatches(e, query) {
				matches = append(matches, e)
			}
		}
	}
	if r.URL.Query().Get("format") == "json" {
		matches = matches[:min(8, len(matches))]
		a.switcherCounts(&ac, matches)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(struct {
			Entries []web.LiegenschaftEntry `json:"entries"`
		}{matches})
		return
	}
	total := len(matches)
	page := boundedPage(r.URL.Query().Get("page"), total, 50)
	start := (page - 1) * 50
	matches = matches[start:min(start+50, total)]
	a.switcherCounts(&ac, matches)
	previous, next := "", ""
	link := func(p int) string {
		return "/app/liegenschaften?" + url.Values{"q": {query}, "page": {strconv.Itoa(p)}}.Encode()
	}
	if page > 1 {
		previous = link(page - 1)
	}
	if start+50 < total {
		next = link(page + 1)
	}
	// Reuse the existing permission-gated navigation context for a utility page;
	// only its title and active navigation entry differ from the help surface.
	portal := a.helpPortalContext(ac)
	portal.Title = "Meine Liegenschaften"
	portal.ActivePage = "liegenschaften"
	var rendered bytes.Buffer
	if err := web.LiegenschaftenPage(portal, matches, query, previous, next, total, page).Render(r.Context(), &rendered); err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(prefixTenantHTMLPaths(rendered.String(), ac.tenant.Slug)))
}

func boundedPage(raw string, total, size int) int {
	page, err := strconv.Atoi(raw)
	if err != nil {
		page = 1
	}
	return max(1, min(page, max(1, (total+size-1)/size)))
}

// Configuration defaults an omitted portal type to community. Lightweight
// fixtures and legacy dynamic records may reach the shell before that default.
func (a *app) isSwitcherProperty(slug string) bool {
	tenant, ok := a.tenantBySlug(slug)
	return ok && (tenant.PortalType == "" || tenant.PortalType == config.PortalTypeCommunity)
}
