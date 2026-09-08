package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestSettingsRoutesAlwaysUseTemplRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	for _, route := range []string{
		"/demo/app/settings",
		"/demo/app/settings/profile",
		"/demo/app/settings/notifications",
		"/demo/app/settings/building",
		"/demo/app/settings/users",
	} {
		t.Run(route, func(t *testing.T) {
			response := authedRequest(t, a, "manager@example.com", route)
			if response.Code != http.StatusOK {
				t.Fatalf("%s status = %d, want 200", route, response.Code)
			}
			if !strings.Contains(response.Body.String(), "data-templ-settings") {
				t.Fatalf("%s must use the templ renderer", route)
			}
		})
	}
}

func TestSettingsTemplRendersAllFiveRoutesWithSharedNavigation(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	cases := map[string][]string{
		"/demo/app/settings":               {"Einstellungen", `href="/demo/app/settings" class="nav-item active"`, `href="/demo/app/settings/building"`, `href="/demo/app/settings/users"`},
		"/demo/app/settings/profile":       {"Profil speichern", `action="/demo/app/settings/profile"`, `name="directory_opt_in"`},
		"/demo/app/settings/notifications": {"Benachrichtigungen speichern", `action="/demo/app/settings/notifications"`, `name="email_enabled"`},
		"/demo/app/settings/building":      {"Gebäude &amp; Einheiten", `data-building-form`, `href="/demo/app/settings/building?section=units"`},
		"/demo/app/settings/users":         {"Benutzer &amp; Rechte", `action="/demo/app/settings/users"`, `action="/demo/app/settings/users/edit"`},
	}
	for route, wants := range cases {
		t.Run(route, func(t *testing.T) {
			response := authedRequest(t, a, "manager@example.com", route)
			if response.Code != http.StatusOK {
				t.Fatalf("templ %s status = %d, want 200", route, response.Code)
			}
			body := response.Body.String()
			for _, want := range append([]string{"data-templ-settings", `<aside class="sidebar" aria-label="Navigation der Liegenschaft">`, `<nav class="nav" aria-label="Bereiche"`}, wants...) {
				if !strings.Contains(body, want) {
					t.Fatalf("templ %s should contain %q", route, want)
				}
			}
		})
	}
}

func TestSettingsTemplKeepsManagementAndRoleAssignmentGates(t *testing.T) {
	resident := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	body := authedRequest(t, resident, "resident@example.com", "/demo/app/settings").Body.String()
	for _, forbidden := range []string{`href="/demo/app/settings/building"`, `href="/demo/app/settings/users"`, `href="/demo/app/settings/modules"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("resident settings hub leaked gated link %q", forbidden)
		}
	}

	manager := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	users := authedRequest(t, manager, "manager@example.com", "/demo/app/settings/users").Body.String()
	if strings.Contains(users, `<option value="Admin"`) {
		t.Fatal("non-admin role editor must not offer the Admin role")
	}
	if strings.Contains(users, `<option value="Dienstleister"`) {
		t.Fatal("role editor must not offer service providers while the feature is disabled")
	}
	manager.serviceAccessEnabled = true
	users = authedRequest(t, manager, "manager@example.com", "/demo/app/settings/users").Body.String()
	if !strings.Contains(users, `<option value="Dienstleister"`) {
		t.Fatal("role editor should offer service providers after the feature is enabled")
	}
}

func TestSettingsTemplStillDeniesServiceProviders(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	for _, route := range []string{"/demo/app/settings", "/demo/app/settings/profile", "/demo/app/settings/notifications"} {
		response := authedRequest(t, a, "service@example.com", route)
		if response.Code != http.StatusForbidden {
			t.Fatalf("service provider %s status = %d, want 403", route, response.Code)
		}
	}
}
