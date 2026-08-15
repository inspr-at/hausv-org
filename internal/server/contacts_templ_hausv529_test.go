package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestContactsTemplSwitchDefaultsToLegacyRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	body := authedRequest(t, a, "resident@example.com", "/demo/app/kontakte").Body.String()
	if strings.Contains(body, "data-templ-contacts") {
		t.Fatal("contacts templ renderer must remain off by default")
	}
	if !strings.Contains(body, `class="app-main contacts"`) {
		t.Fatal("default contacts response must still use the legacy renderer")
	}
}

func TestContactsTemplUsesSharedPermissionGatedShellForEveryPortalRole(t *testing.T) {
	roles := []string{roleAdmin, roleManager, roleOwner, roleRenter, roleBeirat, roleResident}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			email := strings.ToLower(role) + "@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			a.portalTemplEnabled = true

			response := authedRequest(t, a, email, "/demo/app/kontakte")
			if response.Code != http.StatusOK {
				t.Fatalf("contacts status = %d, want 200", response.Code)
			}
			body := response.Body.String()
			for _, want := range []string{
				"data-templ-contacts",
				`<aside class="sidebar" aria-label="Hausnavigation">`,
				`href="/demo/app/kontakte" class="active" aria-current="page"`,
				`<nav class="nav" aria-label="Bereiche">`,
				"Versionsverlauf",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("templ contacts for %s should contain %q", role, want)
				}
			}
		})
	}
}

func TestContactsTemplKeepsManagementAndContactDetailsReachable(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true
	a.serviceAccessEnabled = true
	if _, _, err := testRepositories(a, "demo").contacts.Upsert(managedContact{
		TenantSlug: "demo", Kind: "Dienstleister", Name: "Liftservice", Email: "lift@example.com", Phone: "+43 316 500",
		Notes: "Nur über die Leitstelle anfordern", ServiceRegion: "Wien", Qualification: "Konzessioniert",
		EnergyCapabilities: []string{"metering", "electrical"}, Active: true,
	}); err != nil {
		t.Fatalf("seed contact: %v", err)
	}

	body := authedRequest(t, a, "manager@example.com", "/demo/app/kontakte").Body.String()
	for _, want := range []string{
		`id="contact-add"`, `action="/demo/app/kontakte"`, `data-dialog="contact-edit-`,
		`class="contact-note"`, "Nur über die Leitstelle anfordern", "Leistungsmessung", "Elektro-Fachnachweis",
		`href="tel:`, `href="mailto:lift@example.com"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("templ contacts missing %q", want)
		}
	}
}
