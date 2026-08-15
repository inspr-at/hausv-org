package server

import (
	"strings"
	"testing"
)

func TestPortalTemplSwitchDefaultsToLegacyRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	body := authedRequest(t, a, "manager@example.com", "/demo/app").Body.String()
	if strings.Contains(body, "data-templ-portal") {
		t.Fatal("portal templ renderer must remain off by default")
	}
	if !strings.Contains(body, `class="home-hero"`) {
		t.Fatal("default portal response must still use the legacy renderer")
	}
}

func TestPortalTemplCombinesModuleAndCapabilityGates(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true
	if err := a.tenantOverrides.SetDisabledModules("demo", []string{"energy", "events", "contacts", "documents", "issues", "votes", "parking", "handovers", "users", "audit", "help"}); err != nil {
		t.Fatalf("disable portal modules: %v", err)
	}

	body := authedRequest(t, a, "admin@example.com", "/demo/app").Body.String()
	if !strings.Contains(body, `href="/demo/app/announcements"`) || !strings.Contains(body, `href="/demo/app/settings"`) {
		t.Fatal("enabled announcement module and always-available settings must remain reachable")
	}
	for _, forbidden := range []string{
		`href="/demo/app/energie"`, `href="/demo/app/events"`, `href="/demo/app/kontakte"`,
		`href="/demo/app/dokumente"`, `href="/demo/app/anliegen`, `href="/demo/app/abstimmungen"`,
		`href="/demo/app/parking"`, `href="/demo/app/uebergaben"`, `href="/demo/app/settings/users"`,
		`href="/demo/app/audit"`, `href="/demo/app/hilfe"`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("disabled module link leaked into templ portal: %s", forbidden)
		}
	}
}

func TestPortalTemplSelectsDensityByRoleAndKeepsRoleScopedNavigation(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		wantClass  string
		issuesPath string
	}{
		{name: "admin", role: roleAdmin, wantClass: "dense", issuesPath: "/demo/app/anliegen/board"},
		{name: "manager", role: roleManager, wantClass: "dense", issuesPath: "/demo/app/anliegen/board"},
		{name: "owner", role: roleOwner, wantClass: "calm", issuesPath: "/demo/app/anliegen"},
		{name: "advisory board", role: roleBeirat, wantClass: "calm", issuesPath: "/demo/app/anliegen"},
		{name: "resident", role: roleResident, wantClass: "calm", issuesPath: "/demo/app/anliegen"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			email := strings.ReplaceAll(test.name, " ", "-") + "@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, FirstName: "Ada", Role: test.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			a.portalTemplEnabled = true

			body := authedRequest(t, a, email, "/demo/app").Body.String()
			for _, want := range []string{
				"data-templ-portal", `class="portal-page ` + test.wantClass + `"`,
				`href="` + test.issuesPath + `"`, "Hausüberblick", "Aushang", "Termine", "Kontakte", "Dokumente", "Abstimmungen", "Hilfe", "Versionsverlauf",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("templ portal for %s should contain %q", test.role, want)
				}
			}
			if test.wantClass == "calm" && strings.Contains(body, `href="/demo/app/anliegen/board"`) {
				t.Fatalf("calm role %s must not receive the issue board route", test.role)
			}
			if test.wantClass == "calm" && strings.Contains(body, `href="/demo/app/settings/users"`) {
				t.Fatalf("calm role %s must not receive user-management navigation", test.role)
			}
		})
	}
}
