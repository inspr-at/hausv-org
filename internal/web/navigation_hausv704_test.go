package web

import (
	"os"
	"strings"
	"testing"
)

// organisationPortalFixture adapts old organisation-only fixtures to the shared
// model. Production assembles these fields once in server.portalBaseData.
func organisationPortalFixture(data PortalPageData) PortalPageData {
	data.Overview = true
	data.Role = data.Organisation.RoleLabel
	data.ActivePage = "organisation-" + data.Organisation.Active
	data.ShowVerwaltungNav = true
	data.CanUseResidentAreas = true
	data.Shell.OrganisationName = data.Organisation.OrganisationName
	data.Shell.IsOrganisationMember = true
	data.Shell.Ready = true
	data.Shell.RoleLabel = data.Role
	data.Shell.CanManageOrganisationSettings = data.Organisation.CanManageSettings
	data.Shell.ShowInboxNav = data.Organisation.ShowInboxNav
	data.Shell.InboxOpenCount = data.Organisation.InboxOpenCount
	if len(data.Contexts) == 0 {
		for _, h := range data.Organisation.Houses {
			data.Contexts = append(data.Contexts, PortalContext{TenantSlug: h.Slug, HouseName: h.Name, Address: h.Address, Role: h.Role})
		}
	}
	scope := portalScopeContext(data)
	scope.Segments = []string{data.Organisation.OrganisationName, "Alle Liegenschaften"}
	scope.Switcher.Current = nil
	data.Shell.Context = scope
	return data
}

func TestSharedSidebarTokensAndHeaderWidthHAUSV704(t *testing.T) {
	sheet, err := os.ReadFile("assets/portal-shell.css")
	if err != nil {
		t.Fatal(err)
	}
	base := renderComponent(t, PortalBaseStyles())
	css := string(sheet) + base
	for _, forbidden := range []string{".portal-page", ".organisation-identity", ".verwaltung-houses", ".verwaltung-portal-switch", ".side-map-card", ".side-place-copy", ".side-brand", "grid-template-columns:210px", "grid-template-columns:250px", "grid-template-columns:240px"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("obsolete shell selector/width %s", forbidden)
		}
	}
	for _, required := range []string{"--sidebar-w:280px", "--sidebar-pad-x:18px", "--context-bar-h:72px", "--context-bar-h:74px", ".portal-section-landing.portal-section-wide{--portal-content-width:1440px}"} {
		if !strings.Contains(css, required) {
			t.Errorf("shared token missing: %s", required)
		}
	}
	for _, rule := range parseCSSRules(string(sheet)) {
		if rule.matches(".shell") && strings.Contains(rule.declarations, "grid-template-columns:") && !strings.Contains(rule.declarations, "grid-template-columns:var(--sidebar-w)minmax(0,1fr)") {
			t.Errorf("sidebar width ignores the shared token: %s", rule.declarations)
		}
	}
}

func TestResumeHouseNavigationUsesValidatedPostHAUSV704(t *testing.T) {
	data := PortalPageData{NavigationTenant: "park", NavigationRole: "Verwalter"}
	html := renderComponent(t, PortalHouseNavItem(data, "document", "Dokumente", "/app/dokumente", false, 0, true))
	for _, required := range []string{`method="post" action="/app/context"`, `name="tenant" value="park"`, `name="role" value="Verwalter"`, `name="next" value="/app/dokumente"`, `class="nav-item" type="submit"`} {
		if !strings.Contains(html, required) {
			t.Errorf("resume navigation missing %s", required)
		}
	}
}
