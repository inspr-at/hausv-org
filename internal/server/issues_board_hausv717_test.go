package server

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestIssueBoardTransitionRequirementsAreAtomic(t *testing.T) {
	for _, tc := range []struct {
		name, from, to, assignee, start, end, redirect, result string
	}{
		{"accept requires assignee", issueStatusNew, issueStatusAccepted, "", "", "", "/demo/app/anliegen/board", "zustaendig"},
		{"work requires assignee", issueStatusNew, issueStatusProgress, "", "", "", "", "zustaendig"},
		{"reopen requires assignee", issueStatusDone, issueStatusProgress, "", "", "", "/demo/app/anliegen/board", "zustaendig"},
		{"accept and assign", issueStatusNew, issueStatusAccepted, "manager@example.com", "", "", "/demo/app/anliegen/board", "updated"},
		{"reopen and assign", issueStatusDone, issueStatusProgress, "manager@example.com", "", "", "/demo/app/anliegen/board", "updated"},
		{"back to new", issueStatusDone, issueStatusNew, "", "", "", "/demo/app/anliegen/board", "updated"},
		{"schedule requires date", issueStatusAccepted, issueStatusScheduled, "manager@example.com", "", "", "/demo/app/anliegen/board", "termin"},
		{"schedule with date", issueStatusAccepted, issueStatusScheduled, "manager@example.com", "2026-09-10T10:00", "2026-09-10T11:00", "/demo/app/anliegen/board", "updated"},
		{"schedule rejects backwards end", issueStatusAccepted, issueStatusScheduled, "manager@example.com", "2026-09-10T10:00", "2026-09-10T09:00", "/demo/app/anliegen/board", "invalid"},
		{"external redirect ignored", issueStatusNew, issueStatusAccepted, "", "", "", "https://example.org/", "zustaendig"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const actor = "manager@example.com"
			a := newTestPortalApp(t, userProfile{Email: actor, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			repo := issueRepositoryForTest(a, "demo")
			item, err := repo.Create(residentIssue{TenantSlug: "demo", AuthorEmail: actor, Title: "Prozessregeln", Body: "Unveränderte Beschreibung", Category: "Reparatur", LocationType: issueLocationCommon, Status: tc.from, Priority: issuePriorityHigh})
			if err != nil {
				t.Fatal(err)
			}
			form := url.Values{"id": {item.ID}, "status": {tc.to}, "priority": {item.Priority}, "assignee_email": {tc.assignee}, "redirect": {tc.redirect}}
			if tc.start != "" {
				form.Set("service_start", tc.start)
				form.Set("service_end", tc.end)
			}
			response := authedFormRequest(t, a, actor, "/demo/app/anliegen/workflow", form)
			location, err := url.Parse(response.Header().Get("Location"))
			if err != nil || response.Code != http.StatusSeeOther || location.Query().Get("issue") != tc.result {
				t.Fatalf("response=%d location=%q", response.Code, response.Header().Get("Location"))
			}
			wantPath := "/demo/app/anliegen"
			if tc.redirect == "/demo/app/anliegen/board" {
				wantPath += "/board"
			}
			if location.Path != wantPath {
				t.Fatalf("redirect path=%q, want %q", location.Path, wantPath)
			}
			saved, _ := repo.Get(item.ID)
			if tc.result != "updated" {
				if !reflect.DeepEqual(saved, item) {
					t.Fatal("rejected transition changed stored issue")
				}
				return
			}
			if saved.Status != tc.to || saved.AssigneeEmail != tc.assignee || saved.Priority != item.Priority || saved.Body != item.Body || len(saved.StatusHistory) != 1 {
				t.Fatalf("atomic status/assignment update not persisted: %+v", saved)
			}
			if tc.start != "" && (saved.ServiceProposedStart.IsZero() || !saved.ServiceProposedEnd.After(saved.ServiceProposedStart)) {
				t.Fatal("appointment must persist with status in the same request")
			}
		})
	}
}

func TestIssueBoardPanelPermissionsHistoryAndAssigneeChoices(t *testing.T) {
	const actor = "manager@example.com"
	a := newTestPortalApp(t, userProfile{Email: actor, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["colleague@example.com"] = userProfile{Email: "colleague@example.com", Role: roleManager, Tenants: []string{"demo"}}
	a.profiles["foreign@example.com"] = userProfile{Email: "foreign@example.com", Role: roleManager, Tenants: []string{"other"}}
	repo := issueRepositoryForTest(a, "demo")
	item, err := repo.Create(residentIssue{TenantSlug: "demo", AuthorEmail: actor, Title: "Panelinhalt", Body: "Beschreibung mit Kontext", Category: "Reparatur", LocationType: issueLocationCommon, Status: issueStatusNew, Priority: issuePriorityHigh})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = repo.UpdateWorkflow(item.ID, issueWorkflowUpdate{Status: issueStatusAccepted, Priority: item.Priority, AssigneeEmail: actor, ActorEmail: actor, ActorName: "Mara Manager", ChangedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	path := "/demo/app/anliegen/board/" + item.ID + "/panel"
	response := authedRequest(t, a, actor, path)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("panel status/cache = %d %s", response.Code, response.Header().Get("Cache-Control"))
	}
	body := response.Body.String()
	for _, want := range []string{"Panelinhalt", "Beschreibung mit Kontext", "Neu → Angenommen", "Mara Manager", "Ich übernehme", "colleague@example.com", `action="/demo/app/anliegen/workflow"`, `data-board-move`, `data-board-assign`, `name="priority" value="Hoch"`, `name="redirect" value="/demo/app/anliegen/board"`} {
		if !strings.Contains(body, want) {
			t.Errorf("panel missing %q", want)
		}
	}
	if strings.Contains(body, "foreign@example.com") {
		t.Fatal("panel leaked another tenant's staff")
	}
	for _, role := range []string{roleResident, roleServiceProvider} {
		email := strings.ToLower(role) + "@example.com"
		a.profiles[email] = userProfile{Email: email, Role: role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
		if got := authedRequest(t, a, email, path).Code; got != http.StatusForbidden {
			t.Errorf("%s panel=%d, want forbidden", role, got)
		}
	}
	if got := authedRequest(t, a, actor, "/demo/app/anliegen/board/missing/panel").Code; got != http.StatusNotFound {
		t.Errorf("missing panel=%d", got)
	}
	foreign, err := issueRepositoryForTest(a, "other").Create(residentIssue{TenantSlug: "other", AuthorEmail: "foreign@example.com", Title: "Privat", Body: "Anderes Haus", Category: "Reparatur", LocationType: issueLocationCommon})
	if err != nil {
		t.Fatal(err)
	}
	if got := authedRequest(t, a, actor, "/demo/app/anliegen/board/"+foreign.ID+"/panel").Code; got != http.StatusNotFound {
		t.Errorf("foreign tenant panel=%d, want 404", got)
	}
	triage := authedRequest(t, a, actor, "/demo/app/anliegen/board/"+item.ID+"?step=2").Body.String()
	if strings.Contains(triage, "Noch offen lassen") || !strings.Contains(triage, "colleague@example.com") {
		t.Fatal("triage must require a real staff/service assignee")
	}
}
