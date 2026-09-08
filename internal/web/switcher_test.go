package web

import (
	"regexp"
	"strings"
	"testing"
)

func TestContextBarInBothShellsAndRoles(t *testing.T) {
	for _, role := range []string{"Admin", "Verwalter", "Eigentümer", "Bewohner"} {
		t.Run(role, func(t *testing.T) {
			data := PortalPageData{CanUseResidentAreas: true, HouseName: "Annenstraße 71", Address: "8020 Graz", TenantSlug: "annen", DisplayName: "Vera Beispiel", Initials: "VB", Role: role, Contexts: []PortalContext{{TenantSlug: "annen", HouseName: "Annenstraße 71", Address: "8020 Graz", Role: role, Current: true}, {TenantSlug: "park", HouseName: "Langer Liegenschaftsname im Park", Address: "8010 Graz", Role: role}}}
			portal := renderComponent(t, PortalPage(data))
			management := renderComponent(t, PortalVerwaltungPage(organisationPortalFixture(PortalPageData{Organisation: VerwaltungShell{OrganisationName: "Verwaltung Musterstadt GmbH", RoleLabel: role}, DisplayName: data.DisplayName, Initials: data.Initials, Contexts: data.Contexts}), "Portfolio", VerwaltungPlaceholder()))
			for _, html := range []string{portal, management} {
				for _, marker := range []string{"data-context-bar", "data-context-account", "data-switcher", `role="combobox"`, `aria-autocomplete="list"`, "Annenstraße 71", "8020 Graz", role, ">Profil</a>", ">Einstellungen</a>", " Abmelden</button>", `class="sidebar-release"`, `role="separator"`} {
					if !strings.Contains(html, marker) {
						t.Errorf("missing %q", marker)
					}
				}
				if strings.Contains(html, `class="account"`) || strings.Contains(html, ">Portal wechseln<") || strings.Contains(html, `class="verwaltung-houses"`) {
					t.Fatal("legacy account/switch rendered")
				}
				ids := map[string]bool{}
				for _, match := range regexp.MustCompile(`\bid="([^"]+)"`).FindAllStringSubmatch(html, -1) {
					if ids[match[1]] {
						t.Errorf("duplicate id %s", match[1])
					}
					ids[match[1]] = true
				}
			}
			if !strings.Contains(management, "Alle Liegenschaften") || !strings.Contains(management, "Verwaltung Musterstadt GmbH") {
				t.Fatal("overview breadcrumb missing")
			}
		})
	}
}

func TestScopePreviewMovesIntoContextBar(t *testing.T) {
	html := renderComponent(t, PortalPage(PortalPageData{HouseName: "Annenstraße 71", DisplayName: "Ada Admin", Role: "Bewohner", RolePreview: &RolePreviewState{RoleLabel: "Bewohner", HouseName: "Annenstraße 71", EndPath: "/app/ansicht/ende", EndsInMinutes: 10}}))
	for _, marker := range []string{"context-preview", "Vorschau: Bewohner-Sicht", `<span aria-hidden="true">·</span>`, `aria-label="Vorschau beenden">beenden</button>`, `action="/app/ansicht/ende"`, "schreibgeschützt"} {
		if !strings.Contains(html, marker) {
			t.Errorf("missing %s", marker)
		}
	}
	if strings.Contains(html, `<section class="role-preview-band"`) {
		t.Fatal("preview must not add a second strip under mobile navigation")
	}
}

func TestContextPreviewWithoutPreviewOffersRoleChoice(t *testing.T) {
	scope := ScopeContext{Role: "Admin", PreviewChoices: []RolePreviewChoice{
		{Role: "Bewohner", Label: "Bewohner", StartPath: "/app/ansicht/start"},
	}}
	html := renderComponent(t, ContextPreview(scope, "context-desktop"))
	for _, marker := range []string{`class="context-view-chip">Ansicht als …</span>`, `popovertarget="context-desktop-preview"`, `action="/app/ansicht/start"`, `name="role" value="Bewohner"`} {
		if !strings.Contains(html, marker) {
			t.Errorf("role choice missing %q", marker)
		}
	}
	for _, unexpected := range []string{"Ansicht als Admin", `class="context-preview-chip"`, "Vorschau beenden"} {
		if strings.Contains(html, unexpected) {
			t.Errorf("normal view must not claim an active preview: %q", unexpected)
		}
	}
}
