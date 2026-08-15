package server

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestBallotsTemplSwitchDefaultsToLegacyRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	body := authedRequest(t, a, "manager@example.com", "/demo/app/abstimmungen").Body.String()
	if strings.Contains(body, "data-templ-ballots") {
		t.Fatal("ballots templ renderer must remain off by default")
	}
	if !strings.Contains(body, `nav-item active`) {
		t.Fatal("default ballots response must still use the legacy renderer")
	}
}

func TestBallotsTemplKeepsSharedNavigationAndRoleAccess(t *testing.T) {
	for _, persona := range []struct {
		name string
		role string
	}{
		{name: "manager", role: roleManager},
		{name: "owner", role: roleOwner},
		{name: "advisory-board", role: roleBeirat},
		{name: "resident", role: roleResident},
	} {
		t.Run(persona.name, func(t *testing.T) {
			email := persona.name + "@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: persona.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			a.portalTemplEnabled = true
			created, err := testVoteRepository(t, a, "demo").Create(ballot{
				TenantSlug: "demo",
				Title:      "Innenhof begrünen",
				Options:    []string{"Ja", "Nein", "Enthaltung"},
				Type:       ballotTypeCircular,
				Weighting:  ballotWeightingPerShare,
				CreatedBy:  "manager@example.com",
			})
			if err != nil {
				t.Fatalf("create ballot: %v", err)
			}
			if _, _, err := testVoteRepository(t, a, "demo").Open(created.ID, time.Now()); err != nil {
				t.Fatalf("open ballot: %v", err)
			}

			page := authedRequest(t, a, email, "/demo/app/abstimmungen")
			if page.Code != http.StatusOK {
				t.Fatalf("ballots status for %s = %d, want 200", persona.role, page.Code)
			}
			body := page.Body.String()
			for _, want := range []string{
				"data-templ-ballots",
				`href="/demo/app/abstimmungen" class="active" aria-current="page"`,
				"Innenhof begrünen",
				"Details zur Abstimmung",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("templ ballots for %s missing %q:\n%s", persona.role, want, body)
				}
			}
			if strings.Contains(body, "ui-identitaet-1-0") {
				t.Fatalf("templ ballots for %s must not use a coloured identity edge", persona.role)
			}
		})
	}
}
