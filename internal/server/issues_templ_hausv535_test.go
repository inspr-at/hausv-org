package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestIssuesTemplSwitchDefaultsToLegacyRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	body := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen").Body.String()
	if strings.Contains(body, "data-templ-issues") {
		t.Fatal("issues templ renderer must remain off by default")
	}
	if !strings.Contains(body, `class="app-main"`) {
		t.Fatal("default issues response must still use the legacy renderer")
	}
}

func TestIssuesTemplUsesSharedPermissionGatedShellWithoutNewRoleDenials(t *testing.T) {
	roles := []string{roleAdmin, roleManager, roleOwner, roleRenter, roleBeirat, roleResident, roleServiceProvider}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			email := strings.ToLower(role) + "@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			a.portalTemplEnabled = true
			if role == roleServiceProvider {
				a.serviceAccessEnabled = true
			}

			response := authedRequest(t, a, email, "/demo/app/anliegen")
			if response.Code != http.StatusOK {
				t.Fatalf("issues status for %s = %d, want 200", role, response.Code)
			}
			body := response.Body.String()
			issuesNavURL := "/demo/app/anliegen"
			if role == roleAdmin || role == roleManager {
				issuesNavURL = "/demo/app/anliegen/board"
			}
			for _, want := range []string{
				"data-templ-issues",
				`<aside class="sidebar" aria-label="Hausnavigation">`,
				`<nav class="nav" aria-label="Bereiche">`,
				`href="` + issuesNavURL + `" class="active" aria-current="page"`,
				"Versionsverlauf",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("templ issues for %s should contain %q", role, want)
				}
			}
			if strings.Contains(body, "ui-identitaet-1-0") {
				t.Fatalf("templ issues for %s must not use a coloured identity edge", role)
			}
			canCreate := role != roleServiceProvider && role != roleBeirat
			if strings.Contains(body, `id="issue-create-form"`) != canCreate {
				t.Fatalf("issue creation gate for %s does not match the existing capability", role)
			}
			canManage := role == roleAdmin || role == roleManager
			if strings.Contains(body, `href="/demo/app/anliegen/board"`) != canManage {
				t.Fatalf("board gate for %s does not match issue-management capability", role)
			}
		})
	}
}

func TestIssuesTemplKeepsResidentCreationAndDetailReachable(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Kellerlicht defekt",
		Body:         "Das Licht beim Kellerabgang bleibt dunkel.",
		LocationType: issueLocationCommon,
		Status:       issueStatusNew,
		Priority:     issuePriorityNorm,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	body := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen?new=1").Body.String()
	for _, want := range []string{
		"Kellerlicht defekt",
		`href="/demo/app/anliegen/` + issue.ID + `"`,
		`id="issue-new" open`,
		`id="issue-create-form"`,
		`action="/demo/app/anliegen"`,
		`name="attachments"`,
		"Keine Gesundheitsdaten",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("resident issues templ missing %q", want)
		}
	}
	if strings.Contains(body, `action="/demo/app/anliegen/workflow"`) {
		t.Fatal("resident list must not expose the inline service workflow")
	}
}

func TestIssuesTemplKeepsAssignedServiceProviderToolsReachable(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "service@example.com", Role: roleServiceProvider, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true
	a.serviceAccessEnabled = true
	if _, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:    "demo",
		AuthorEmail:   "resident@example.com",
		AuthorName:    "Resident",
		Category:      "Reparatur",
		Title:         "Heizung prüfen",
		Body:          "Der Heizkörper bleibt kalt.",
		LocationType:  issueLocationCommon,
		Status:        issueStatusNew,
		Priority:      issuePriorityUrgent,
		AssigneeEmail: "service@example.com",
	}); err != nil {
		t.Fatalf("create assigned issue: %v", err)
	}
	if _, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Nicht zugewiesen",
		Body:         "Darf nicht sichtbar sein.",
		LocationType: issueLocationCommon,
		Status:       issueStatusNew,
		Priority:     issuePriorityNorm,
	}); err != nil {
		t.Fatalf("create unassigned issue: %v", err)
	}

	body := authedRequest(t, a, "service@example.com", "/demo/app/anliegen").Body.String()
	for _, want := range []string{
		"Heizung prüfen",
		`action="/demo/app/anliegen/comment"`,
		`action="/demo/app/anliegen/workflow"`,
		`name="service_start"`,
		`name="service_end"`,
		`name="service_proposal"`,
		"Kalender abonnieren",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("service-provider issues templ missing %q", want)
		}
	}
	for _, hidden := range []string{"Nicht zugewiesen", `id="issue-create-form"`, `href="/demo/app/anliegen/board"`} {
		if strings.Contains(body, hidden) {
			t.Fatalf("service-provider issues templ must not expose %q", hidden)
		}
	}
}

func TestIssuesTemplSwitchDoesNotConvertManagerBoard(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true

	response := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board")
	if response.Code != http.StatusOK {
		t.Fatalf("manager board status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if strings.Contains(body, "data-templ-issues") {
		t.Fatal("HAUSV-535 must leave the separate manager board on its existing renderer")
	}
	if !strings.Contains(body, "Anliegen bearbeiten") {
		t.Fatal("manager board content must remain reachable")
	}
}
