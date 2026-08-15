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
	// HAUSV-527: managing roles get the dense composition only when the house has
	// something open. With an empty house they get the calm one, so this table
	// seeds an issue for the roles that should come out dense.
	tests := []struct {
		name       string
		role       string
		wantClass  string
		issuesPath string
		seedIssue  bool
	}{
		{name: "admin", role: roleAdmin, wantClass: "dense", issuesPath: "/demo/app/anliegen/board", seedIssue: true},
		{name: "manager", role: roleManager, wantClass: "dense", issuesPath: "/demo/app/anliegen/board", seedIssue: true},
		{name: "admin with an empty house", role: roleAdmin, wantClass: "calm", issuesPath: "/demo/app/anliegen/board"},
		{name: "manager with an empty house", role: roleManager, wantClass: "calm", issuesPath: "/demo/app/anliegen/board"},
		{name: "owner", role: roleOwner, wantClass: "calm", issuesPath: "/demo/app/anliegen"},
		{name: "advisory board", role: roleBeirat, wantClass: "calm", issuesPath: "/demo/app/anliegen"},
		{name: "resident", role: roleResident, wantClass: "calm", issuesPath: "/demo/app/anliegen"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			email := strings.ReplaceAll(test.name, " ", "-") + "@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, FirstName: "Ada", Role: test.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			a.portalTemplEnabled = true
			if test.seedIssue {
				_, _ = testRepositories(a, "demo").issues.Create(residentIssue{
					TenantSlug: "demo", AuthorEmail: email, AuthorName: "Ada",
					Category: "Reparatur", Title: "Heizung Stiege 2 kalt", Body: "Offen",
					LocationType: issueLocationUnit, Status: issueStatusNew, Priority: issuePriorityNorm,
				})
			}

			body := authedRequest(t, a, email, "/demo/app").Body.String()
			for _, want := range []string{
				"data-templ-portal", `class="portal-page ` + test.wantClass + `"`,
				`href="` + test.issuesPath + `"`, "Hausüberblick", "Aushang", "Termine", "Kontakte", "Dokumente", "Abstimmungen", "Hilfe", "Versionsverlauf",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("templ portal for %s should contain %q", test.role, want)
				}
			}
			// Navigation follows the ROLE, never the density. A manager looking at an
			// empty house renders calm (HAUSV-527) and must still keep every
			// managing route — conflating the two would turn a layout decision into
			// a permissions bug.
			managing := test.role == roleAdmin || test.role == roleManager
			if !managing && strings.Contains(body, `href="/demo/app/anliegen/board"`) {
				t.Fatalf("non-managing role %s must not receive the issue board route", test.role)
			}
			if !managing && strings.Contains(body, `href="/demo/app/settings/users"`) {
				t.Fatalf("non-managing role %s must not receive user-management navigation", test.role)
			}
			if managing && !strings.Contains(body, `href="/demo/app/settings/users"`) {
				t.Fatalf("managing role %s must keep user-management navigation regardless of density", test.role)
			}
		})
	}
}

// HAUSV-527: an empty house gives a manager nothing to be dense about. The dense
// composition renders tall empty cards in that state, which reads worse than the
// calm one, so density follows content as well as role.
func TestPortalDensityFollowsContentNotOnlyRole(t *testing.T) {
	for _, tc := range []struct {
		name      string
		role      string
		issues    int
		events    int
		unread    int
		wantDense bool
	}{
		{"manager with open work is dense", roleManager, 3, 0, 0, true},
		{"admin with upcoming events is dense", roleAdmin, 0, 2, 0, true},
		{"manager with only unread notices is dense", roleManager, 0, 0, 1, true},
		{"manager with an empty house is calm", roleManager, 0, 0, 0, false},
		{"admin with an empty house is calm", roleAdmin, 0, 0, 0, false},
		{"resident with open work is still calm", roleResident, 5, 3, 2, false},
		{"owner with open work is still calm", roleOwner, 5, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := portalIsDense(tc.role, tc.issues, tc.events, tc.unread)
			if got != tc.wantDense {
				t.Fatalf("role=%s issues=%d events=%d unread=%d: dense=%v, want %v",
					tc.role, tc.issues, tc.events, tc.unread, got, tc.wantDense)
			}
		})
	}
}
