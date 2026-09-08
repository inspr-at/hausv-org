package web

import "fmt"

// Standalone component fixtures and legacy callers can still supply the public
// shell fields. Production uses the server's shared ScopeContext directly.
func portalScopeContext(data PortalPageData) ScopeContext {
	if data.Shell.Context.Ready {
		return data.Shell.Context
	}
	shell := portalShellData(data)
	scope := ScopeContext{Ready: true, CanUseSettings: data.CanUseResidentAreas, DisplayName: data.DisplayName, Initials: data.Initials, AvatarURL: shell.AvatarURL, Role: data.Role, Preview: data.RolePreview, PreviewChoices: data.RolePreviewChoices, Segments: []string{data.HouseName}}
	contexts := append([]PortalContext(nil), data.Contexts...)
	if len(contexts) == 0 {
		for _, h := range shell.ManagedHouses {
			contexts = append(contexts, PortalContext{TenantSlug: h.Slug, HouseName: h.Name, Address: h.Address, Role: h.Role, Current: h.Current || h.Slug == data.TenantSlug})
		}
		contexts = append(contexts, shell.PortalContexts...)
	}
	scope.Switcher = fallbackSwitcher(contexts)
	if scope.Switcher.Current == nil {
		scope.Switcher.Current = &LiegenschaftEntry{Key: data.TenantSlug + "|" + data.Role, Tenant: data.TenantSlug, Name: data.HouseName, Address: data.Address, Role: data.Role, Group: "Persönlich", Current: true}
	}
	if shell.IsOrganisationMember {
		scope.Segments = append([]string{shell.OrganisationName}, scope.Segments...)
		scope.Switcher.PortfolioURL = "/app/verwaltung"
	}
	return scope
}

func verwaltungScopeContext(shell VerwaltungShell) ScopeContext {
	if shell.Context.Ready {
		return shell.Context
	}
	contexts := append([]PortalContext(nil), shell.Contexts...)
	if len(contexts) == 0 {
		for _, h := range shell.Houses {
			contexts = append(contexts, PortalContext{TenantSlug: h.Slug, HouseName: h.Name, Address: h.Address, Role: h.Role})
		}
	}
	data := ScopeContext{Ready: true, CanUseSettings: true, DisplayName: shell.DisplayName, Initials: shell.Initials, AvatarURL: shell.AvatarURL, Role: shell.RoleLabel, Segments: []string{shell.OrganisationName, "Alle Liegenschaften"}, Switcher: fallbackSwitcher(contexts)}
	data.Switcher.Current = nil
	data.Switcher.PortfolioURL = "/app/verwaltung"
	return data
}

func fallbackSwitcher(contexts []PortalContext) LiegenschaftSwitcherData {
	data := LiegenschaftSwitcherData{SearchURL: "/app/liegenschaften/suche", PortfolioURL: "/app/liegenschaften"}
	seen := map[string]bool{}
	for _, c := range contexts {
		if !seen[c.TenantSlug] {
			data.Count++
			seen[c.TenantSlug] = true
		}
		group := "Persönlich"
		if c.Role == "Admin" || c.Role == "Verwalter" {
			group = "Verwaltung"
		}
		e := LiegenschaftEntry{Key: c.TenantSlug + "|" + c.Role, Tenant: c.TenantSlug, Name: c.HouseName, Address: c.Address, Role: c.Role, Group: group, Current: c.Current}
		if e.Current {
			data.Current = &e
		}
		if len(data.Entries) < 6 {
			data.Entries = append(data.Entries, e)
		}
	}
	return data
}

func verwaltungSwitcherID(sidebar bool) string {
	if sidebar {
		return "verwaltung-switcher"
	}
	return "verwaltung-switcher-mobile"
}
func switcherCountLabel(count int) string {
	if count == 1 {
		return "1 Liegenschaft"
	}
	return fmt.Sprintf("%d Liegenschaften", count)
}
