package web

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestDisclosureChevronUsesUnmodifiedLucideAsset(t *testing.T) {
	asset, err := os.ReadFile("assets/icons/lucide/chevron-down.svg")
	if err != nil {
		t.Fatal(err)
	}
	html := renderComponent(t, DisclosureChevron())
	if !strings.Contains(html, string(asset)) || !strings.Contains(html, `class="disclosure-chevron" aria-hidden="true"`) {
		t.Fatal("shared disclosure must wrap the unchanged, decorative Lucide asset")
	}
	picker := renderComponent(t, LiegenschaftSwitcher(LiegenschaftSwitcherData{}, "test-picker", "Haus", "Graz", "portal", "Liegenschaft", "building-2"))
	if strings.Count(picker, `class="disclosure-chevron"`) != 1 || strings.Contains(picker, "⌄") {
		t.Fatal("house and scope summaries must use exactly one shared chevron")
	}
}

func TestHouseMedallionReflectsRoleAndPortfolio(t *testing.T) {
	for _, tc := range []struct {
		role, icon string
		overview   bool
	}{
		{"Admin", "building-2", false},
		{"Verwalter", "building-2", false},
		{"Bewohner", "house", false},
		{"Eigentümer", "house", false},
		{"Admin", "building-2", true},
	} {
		t.Run(fmt.Sprintf("%s/overview=%t", tc.role, tc.overview), func(t *testing.T) {
			data := PortalPageData{Role: tc.role, Overview: tc.overview, HouseName: "Janischhofweg 22, 8043 Graz", Address: "Janischhofweg 22, 8043 Graz"}
			data.Shell.Context = ScopeContext{Ready: true, Switcher: LiegenschaftSwitcherData{Count: 12}}
			shell := PortalShellData{IsOrganisationMember: true, CurrentHousePosition: 4, ManagedHouses: make([]PortalHouse, 12)}
			asset, err := os.ReadFile("assets/icons/lucide/" + tc.icon + ".svg")
			if err != nil {
				t.Fatal(err)
			}
			for _, sidebar := range []bool{true, false} {
				html := renderComponent(t, PortalHouseHeader(data, shell, sidebar))
				if !strings.Contains(html, string(asset)) {
					t.Fatal("medallion must use the unchanged Lucide asset for the current role")
				}
				want := "Liegenschaft · 4 von 12"
				if tc.overview {
					want = "Alle Liegenschaften · 12"
				} else if !strings.Contains(html, "<strong>Janischhofweg 22</strong><small>8043 Graz</small>") {
					t.Fatal("street and place must remain distinct")
				}
				if !strings.Contains(html, want) || strings.Count(html, `class="disclosure-chevron"`) != 1 {
					t.Fatal("card must preserve the current context count and one disclosure")
				}
			}
		})
	}
}

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
	for _, marker := range []string{"unified-context-bar", "Vorschau: Bewohner-Sicht", `>Ansicht verlassen</button>`, `action="/app/ansicht/ende"`, "Schreibgeschützt"} {
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
	for _, marker := range []string{`class="context-action">Ansicht als …</span>`, `popovertarget="context-desktop-preview"`, `action="/app/ansicht/start"`, `name="role" value="Bewohner"`} {
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
