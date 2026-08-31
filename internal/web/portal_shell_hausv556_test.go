package web

import (
	"os"
	"strings"
	"testing"
)

func TestPortalSwitcherIsOneSubduedLineHAUSV556(t *testing.T) {
	portal := PortalPageData{
		Title:       "Test Portal",
		HouseName:   "A deliberately long active portal name",
		Address:     "Test Street 1",
		DisplayName: "Test User",
		Initials:    "TU",
		Role:        "Admin",
		Contexts: []PortalContext{
			{HouseName: "A deliberately long active portal name", TenantSlug: "active", Role: "Admin", Current: true},
			{HouseName: "Another portal", TenantSlug: "other", Role: "Admin"},
		},
	}

	html := renderComponent(t, PortalPage(portal))
	trigger := `<summary aria-label="Portal wechseln">Portal wechseln<span class="context-switch-chevron" aria-hidden="true">⌄</span></summary>`
	if got := strings.Count(html, trigger); got != 2 {
		t.Fatalf("desktop and mobile switchers must use the same one-line trigger; got %d", got)
	}
	if got := strings.Count(html, `action="/app/context"`); got != 2 {
		t.Fatalf("one atomic switch form per surface must remain rendered; got %d", got)
	}

	css, err := os.ReadFile("assets/portal-shell.css")
	if err != nil {
		t.Fatalf("read portal shell CSS: %v", err)
	}
	for _, contract := range []string{
		".mobile-context-switch>summary{min-width:0;min-height:44px",
		"white-space:nowrap",
		"text-overflow:ellipsis",
	} {
		if !strings.Contains(string(css), contract) {
			t.Errorf("mobile switcher CSS is missing %q", contract)
		}
	}
}
