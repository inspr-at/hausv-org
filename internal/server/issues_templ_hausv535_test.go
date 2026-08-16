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

// HAUSV-538 finished what HAUSV-535 left out: the manager board and its triage
// detail were the two /app/anliegen/board routes still on the legacy renderer,
// so a manager with the switch on dropped into the old design mid-session.
func TestIssueBoardTemplRendersTheManagerBoardOnTheSharedShell(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true

	response := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board")
	if response.Code != http.StatusOK {
		t.Fatalf("manager board status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{
		"data-templ-issue-board",
		`<aside class="sidebar" aria-label="Hausnavigation">`,
		`<nav class="nav" aria-label="Bereiche">`,
		"Anliegen bearbeiten",
		`href="/demo/app/anliegen"`,
		// The empty board keeps its two onward actions and the explainer.
		"Der Weg eines Anliegens",
		`href="/demo/app/anliegen?new=1"`,
		`href="/demo/app/announcements"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("templ issue board missing %q", want)
		}
	}
	if strings.Contains(body, `class="app-main"`) {
		t.Fatal("the manager board must not fall back to the legacy shell")
	}
}

func TestIssueBoardTemplKeepsFiltersAndTriageEntryReachable(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Tür schließt nicht",
		Body:         "Die Haustür fällt nicht ins Schloss.",
		LocationType: issueLocationCommon,
		Status:       issueStatusNew,
		Priority:     issuePriorityUrgent,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}

	body := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board").Body.String()
	for _, want := range []string{
		"Tür schließt nicht",
		`action="/demo/app/anliegen/board"`,
		`name="status"`, `name="priority"`, `name="category"`, `name="assignee"`, `name="sort"`,
		`href="/demo/app/anliegen/board?status=Neu"`,
		`href="/demo/app/anliegen/board?assignee=manager@example.com"`,
		`href="/demo/app/anliegen/board/` + issue.ID + `"`,
		"1 dringend",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("templ issue board missing %q", want)
		}
	}

	filtered := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board?status=Erledigt").Body.String()
	if !strings.Contains(filtered, "Kein Anliegen passt zu dieser Auswahl") {
		t.Fatal("an empty filter result must keep its own blank state, not the first-run explainer")
	}
	if strings.Contains(filtered, "Der Weg eines Anliegens") {
		t.Fatal("a filtered-away board must not claim the house has no issues")
	}
}

func TestIssueTriageTemplKeepsEveryStepActionable(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Tür schließt nicht",
		Body:         "Die Haustür fällt nicht ins Schloss.",
		LocationType: issueLocationCommon,
		Status:       issueStatusNew,
		Priority:     issuePriorityUrgent,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	base := "/demo/app/anliegen/board/" + issue.ID

	steps := map[string][]string{
		"1": {
			`name="redirect" value="` + base + `?step=2"`,
			`action="/demo/app/anliegen/workflow"`,
			`value="Dringend"`, `value="Hoch"`, `value="Niedrig"`,
		},
		"2": {
			`name="redirect" value="` + base + `?step=done"`,
			`value="In Bearbeitung"`,
			`name="assignee_email" value="manager@example.com"`,
		},
		"done": {
			`href="` + base + `?step=message"`,
			`href="` + base + `"`,
		},
		"message": {
			`action="/demo/app/anliegen/comment"`,
			`name="redirect" value="` + base + `?step=sent"`,
			`name="message_type" value="information"`,
			`name="message_type" value="question"`,
			`name="attachments"`,
			`name="redirect" value="` + base + `?step=resolution-sent"`,
			"Lösung zur Prüfung senden",
		},
		"sent":            {`href="` + base + `?step=message"`, `href="/demo/app/anliegen/board"`},
		"resolution-sent": {`href="` + base + `?step=message"`, `href="/demo/app/anliegen/board"`},
	}
	for step, wants := range steps {
		t.Run(step, func(t *testing.T) {
			response := authedRequest(t, a, "manager@example.com", base+"?step="+step)
			if response.Code != http.StatusOK {
				t.Fatalf("triage step %s status = %d, want 200", step, response.Code)
			}
			body := response.Body.String()
			if !strings.Contains(body, "data-templ-issue-triage") {
				t.Fatalf("triage step %s did not use the templ renderer", step)
			}
			for _, want := range append(wants, "Stand des Anliegens", "Tür schließt nicht", `href="/demo/app/anliegen/board"`) {
				if !strings.Contains(body, want) {
					t.Fatalf("triage step %s missing %q", step, want)
				}
			}
		})
	}
}

func TestIssueTriageTemplStaysClosedToNonManagers(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true
	issue, err := issueRepositoryForTest(a, "demo").Create(residentIssue{
		TenantSlug:   "demo",
		AuthorEmail:  "resident@example.com",
		AuthorName:   "Resident",
		Category:     "Reparatur",
		Title:        "Tür schließt nicht",
		Body:         "Die Haustür fällt nicht ins Schloss.",
		LocationType: issueLocationCommon,
		Status:       issueStatusNew,
		Priority:     issuePriorityNorm,
	})
	if err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if got := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/board/"+issue.ID).Code; got != http.StatusForbidden {
		t.Fatalf("resident triage status = %d, want 403", got)
	}
	if got := authedRequest(t, a, "resident@example.com", "/demo/app/anliegen/board").Code; got != http.StatusForbidden {
		t.Fatalf("resident board status = %d, want 403", got)
	}
}

func TestIssueBoardTemplSwitchDefaultsToLegacyRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	board := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen/board").Body.String()
	if strings.Contains(board, "data-templ-issue-board") || !strings.Contains(board, `class="app-main"`) {
		t.Fatal("the manager board must stay on the legacy renderer while the switch is off")
	}
}
