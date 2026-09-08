package web

import (
	"os"
	"strings"
	"testing"
)

// HAUSV-697 supersedes the one-line third dropdown. The same quiet, readable
// trigger now belongs to the unified panel in the sidebar, drawer and scope.
func TestPortalSwitcherUsesOneSharedPanelHAUSV556(t *testing.T) {
	portal := PortalPageData{HouseName: "A deliberately long active portal name", DisplayName: "Test User", Role: "Admin", Contexts: []PortalContext{{TenantSlug: "active", HouseName: "Active portal", Role: "Admin", Current: true}, {TenantSlug: "other", HouseName: "Another portal", Role: "Admin"}}}
	html := renderComponent(t, PortalPage(portal))
	if got := strings.Count(html, `id="portal-house-picker"`) + strings.Count(html, `id="portal-house-picker-mobile"`); got != 2 {
		t.Fatalf("sidebar/drawer picker count: %d", got)
	}
	if strings.Contains(html, ">Portal wechseln<") {
		t.Fatal("third dropdown remains")
	}
	if !strings.Contains(html, `name="tenant" value="other"`) || !strings.Contains(html, `action="/app/context"`) {
		t.Fatal("atomic POST switch missing")
	}
	css, err := os.ReadFile("assets/portal-shell.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, contract := range []string{".context-scope .house-header-card", ".switcher-row-copy strong", "-webkit-line-clamp:2", "minmax(0,1fr) auto 28px"} {
		if !strings.Contains(string(css), contract) {
			t.Errorf("missing %s", contract)
		}
	}
}
