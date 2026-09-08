package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// The DnD client posts exactly the same form as the keyboard/no-JS menu.
// Assert persisted state as well as the redirect; validation redirects are 303
// too and must never be mistaken for successful status changes by the client.
func TestIssueBoardWorkflowAcceptsAndRejectsMoves(t *testing.T) {
	for _, tc := range []struct {
		name, role, target, result string
		appointment                bool
		code                       int
	}{
		{"accept", roleManager, issueStatusAccepted, "updated", false, http.StatusSeeOther},
		{"start work", roleManager, issueStatusProgress, "updated", false, http.StatusSeeOther},
		{"scheduled with appointment", roleManager, issueStatusScheduled, "updated", true, http.StatusSeeOther},
		{"scheduled without appointment", roleManager, issueStatusScheduled, "termin", false, http.StatusSeeOther},
		{"unknown status", roleManager, "Invented", "invalid", false, http.StatusSeeOther},
		{"resident cannot accept", roleResident, issueStatusAccepted, "", false, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const actor = "board@example.com"
			a := newTestPortalApp(t, userProfile{Email: actor, Role: tc.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			original := residentIssue{
				TenantSlug: "demo", AuthorEmail: actor, AuthorName: "Board Probe",
				Category: "Reparatur", Title: "Triage-Board Probe", Body: "Die Tür schließt nicht.",
				LocationType: issueLocationCommon, Status: issueStatusNew,
				Priority: issuePriorityHigh, AssigneeEmail: actor,
			}
			if tc.appointment {
				original.ServiceProposedStart = time.Now().Add(24 * time.Hour)
				original.ServiceProposedEnd = original.ServiceProposedStart.Add(time.Hour)
			}
			repo := issueRepositoryForTest(a, "demo")
			issue, err := repo.Create(original)
			if err != nil {
				t.Fatal(err)
			}
			response := authedFormRequest(t, a, actor, "/demo/app/anliegen/workflow", url.Values{
				"id": {issue.ID}, "status": {tc.target}, "priority": {issue.Priority}, "assignee_email": {issue.AssigneeEmail},
			})
			if response.Code != tc.code {
				t.Fatalf("POST status = %d, want %d", response.Code, tc.code)
			}
			location, err := url.Parse(response.Header().Get("Location"))
			if err != nil || location.Query().Get("issue") != tc.result {
				t.Fatalf("redirect = %q, want issue=%s", response.Header().Get("Location"), tc.result)
			}
			wantStatus := issueStatusNew
			if tc.result == "updated" {
				wantStatus = tc.target
				if location.Path != "/demo/app/anliegen/board" {
					t.Fatalf("success must return the tenant board: %s", location.Path)
				}
			}
			saved, found := repo.Get(issue.ID)
			if !found || saved.Status != wantStatus || saved.Priority != original.Priority || saved.AssigneeEmail != original.AssigneeEmail || saved.Body != original.Body {
				t.Fatalf("move must only change the accepted status: status=%q, found=%t", saved.Status, found)
			}
			if tc.result == "updated" {
				body := authedRequest(t, a, actor, location.RequestURI()).Body.String()
				for _, want := range []string{
					`id="issue-` + issue.ID + `" draggable="true" data-board-card data-issue-status="` + tc.target + `"`,
					`data-board-panel`,
					`data-assignee="` + actor + `"`,
					`/assets/issue-board.js?v=`,
				} {
					if !strings.Contains(body, want) {
						t.Errorf("persisted board missing %q", want)
					}
				}
			}
		})
	}
}

func TestIssueBoardWorkflowRejectsForeignOrigin(t *testing.T) {
	const actor = "board@example.com"
	a := newTestPortalApp(t, userProfile{Email: actor, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	repo := issueRepositoryForTest(a, "demo")
	issue, err := repo.Create(residentIssue{TenantSlug: "demo", AuthorEmail: actor, Category: "Reparatur", Title: "Tür prüfen", Body: "Bitte prüfen.", LocationType: issueLocationCommon, Status: issueStatusNew, Priority: issuePriorityNorm})
	if err != nil {
		t.Fatal(err)
	}
	response := authedFormRequestWithOrigin(t, a, actor, "/demo/app/anliegen/workflow", url.Values{
		"id": {issue.ID}, "status": {issueStatusAccepted}, "priority": {issue.Priority},
	}, "https://foreign.example")
	if response.Code != http.StatusForbidden {
		t.Fatalf("foreign-origin move = %d, want 403", response.Code)
	}
	saved, _ := repo.Get(issue.ID)
	if saved.Status != issueStatusNew {
		t.Fatal("rejected request changed the issue")
	}
}

// HAUSV-717 moves workflow editors into the fetched detail panel. Cards remain
// compact and keyboard reachable; the board does not embed hidden full editors.
func assertIssueBoardStatusMenus(t *testing.T, body string) {
	t.Helper()
	cards := strings.Count(body, ` data-board-card `)
	if cards == 0 || strings.Count(body, `tabindex="0" role="button"`) != cards {
		t.Fatal("each board card must be keyboard reachable")
	}
	if !strings.Contains(body, `data-board-panel`) {
		t.Fatal("board must contain the detail panel host")
	}
	for _, forbidden := range []string{`data-board-move`, `name="assignee_email"`, `name="service_start"`, `name="service_end"`, `name="service_proposal"`, `issue-description-preview`} {
		if strings.Contains(body, forbidden) {
			t.Errorf("compact cards must not embed %s", forbidden)
		}
	}
}
