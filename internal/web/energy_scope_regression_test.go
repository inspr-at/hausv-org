package web

import (
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/version"
)

func legacyShellTestPortal() PortalPageData {
	brand, _ := Assets.ReadFile("assets/icons/lucide/house.svg")
	return PortalPageData{Map: PortalMap{Configured: true}, BrandMarkSVG: string(brand), Title: "Test", HouseName: "Testhaus", Address: "Testgasse 1", MapURL: "https://www.openstreetmap.org/", DisplayVersion: "test", ReleaseNotes: version.Notes(), CanUseResidentAreas: true}
}

// HAUSV-705 retires the hash-protected legacy footer. Guard its removal and
// require the real shared sidebar on migrated pages instead of hashing dead HTML.
func TestRetiredSidebarCannotReturn(t *testing.T) {
	for _, forbidden := range []string{`{{define "sidebar"}}`, `{{define "appOpen"}}`, `{{define "appClose"}}`, `.app-shell`, `.side-foot`, `.side-map-card`, `.portal-context-switch`} {
		if strings.Contains(PageTemplates, forbidden) {
			t.Fatalf("retired shell returned: %s", forbidden)
		}
	}
	portal := legacyShellTestPortal()
	side := renderComponent(t, PortalSidebar(portal))
	html := renderComponent(t, EnergyDataPage(EnergyDataPageData{Portal: portal}))
	if !strings.Contains(html, side) || strings.Count(html, `<aside class="sidebar"`) != 1 {
		t.Fatal("energy data must render exactly the shared sidebar")
	}
	if strings.Contains(html, `class="side-foot"`) {
		t.Fatal("retired account footer returned")
	}
}
