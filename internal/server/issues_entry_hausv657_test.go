package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestIssuesEntryByRoleHAUSV657(t *testing.T) {
	for _, tc := range []struct {
		name, role string
		management bool
	}{
		{"admin", roleAdmin, true}, {"manager", roleManager, true},
		{"Sachbearbeiter", houseRoleForOrganisationRole(store.OrganisationRoleClerk), true},
		{"resident", roleResident, false}, {"owner", roleOwner, false}, {"renter", roleRenter, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			email := "viewer@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: tc.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			empty := authedRequest(t, a, email, "/demo/app/anliegen").Body.String()
			if !strings.Contains(empty, "Erstes Anliegen melden") {
				t.Fatal("house without issues must offer first issue")
			}
			repo := issueRepositoryForTest(a, "demo")
			for i := 0; i < 10; i++ {
				_, err := repo.Create(residentIssue{TenantSlug: "demo", AuthorEmail: "other@example.com", Category: "Reparatur", Title: fmt.Sprintf("Fremdes Anliegen %02d", i), Body: "Licht prüfen", LocationType: issueLocationCommon, Status: issueStatusNew, Priority: issuePriorityNorm})
				if err != nil {
					t.Fatal(err)
				}
			}
			response := authedRequest(t, a, email, "/demo/app/anliegen")
			if response.Code != http.StatusOK {
				t.Fatalf("status %d", response.Code)
			}
			body := response.Body.String()
			// The heading follows what the viewer can see: whoever sees the
			// common-area filings gets the plain heading, a role that sees none is
			// still offered a first issue (and learns nothing about the others).
			sees := strings.Contains(body, "Fremdes Anliegen")
			if sees == strings.Contains(body, "Erstes Anliegen melden") {
				t.Fatalf("first-issue heading must follow the viewer's own visibility (sees=%v)", sees)
			}
			if tc.management && !sees {
				t.Fatal("management must see the house's issues")
			}
			if !strings.Contains(body, `id="issue-new"`) || !strings.Contains(body, `data-issue-wizard`) {
				t.Fatal("wizard missing")
			}
			summary := strings.Index(body, `id="issue-open"`)
			if (summary >= 0) != tc.management {
				t.Fatalf("management summary visibility for %s", tc.name)
			}
			if tc.management {
				if got := strings.Count(body, `class="issue-summary-row"`); got != 8 {
					t.Fatalf("summary rows=%d want 8", got)
				}
				if summary > strings.Index(body, `id="issue-new"`) {
					t.Fatal("summary must precede wizard")
				}
				for _, want := range []string{"Alle im Triage-Board", `href="/demo/app/anliegen/board"`, "Zuständig:", "Gemeldet:", "Noch nicht zugewiesen"} {
					if !strings.Contains(body, want) {
						t.Errorf("missing %q", want)
					}
				}
			} else if tc.role != roleOwner && strings.Contains(body, "Fremdes Anliegen") {
				t.Fatal("resident must not receive other residents' issues")
			}
		})
	}
}

func TestNewestOpenIssueSummariesHAUSV657(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	items := []residentIssue{}
	for i := 0; i < 10; i++ {
		items = append(items, residentIssue{ID: fmt.Sprint(i), Title: fmt.Sprintf("Anliegen %d", i), Status: issueStatusNew, CreatedAt: now.Add(-time.Duration(i+1) * 24 * time.Hour)})
	}
	items[1].Status = issueStatusProgress
	items[0].AssigneeEmail = "service@example.com"
	for _, status := range []string{issueStatusDone, issueStatusRejected, issueStatusDuplicate} {
		items = append(items, residentIssue{ID: status, Status: status, CreatedAt: now})
	}
	got := newestOpenIssueSummaries(items, now)
	if len(got) != 8 {
		t.Fatalf("got %d rows, want 8", len(got))
	}
	for i, row := range got {
		if row.Title != fmt.Sprintf("Anliegen %d", i) {
			t.Fatalf("row %d: %q; expected newest first, only open", i, row.Title)
		}
	}
	if got[0].Age != "vor 1 T." || got[0].Assignee != "service@example.com" || got[0].URL != "/app/anliegen/board/0" {
		t.Fatalf("incorrect summary: %+v", got[0])
	}
	if got[1].Status != issueStatusProgress || got[1].Assignee != "Noch nicht zugewiesen" {
		t.Fatalf("incorrect progress summary: %+v", got[1])
	}
	if items[0].ID != "0" || len(items) != 13 {
		t.Fatal("summary mutated source")
	}
}

func TestIssuesEntryWithOnlyClosedHouseIssuesHAUSV657(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	_, err := issueRepositoryForTest(a, "demo").Create(residentIssue{TenantSlug: "demo", AuthorEmail: "other@example.com", Title: "Schon erledigt", Body: "Licht", Status: issueStatusDone, Category: "Reparatur", LocationType: issueLocationCommon})
	if err != nil {
		t.Fatal(err)
	}
	body := authedRequest(t, a, "manager@example.com", "/demo/app/anliegen").Body.String()
	if strings.Contains(body, "Erstes Anliegen melden") || strings.Contains(body, `class="issue-summary-row"`) {
		t.Fatal("closed issue must suppress first title without appearing in open summary")
	}
	if !strings.Contains(body, "Keine offenen Anliegen im Haus.") {
		t.Fatal("missing empty open summary")
	}
}

// Review of HAUSV-657: a resident whose own list is empty keeps the
// first-issue heading even when another tenant filed a unit-private issue —
// the heading must not reveal filings the viewer cannot see.
func TestFirstIssueHeadingKeepsResidentsBlindToPrivateFilingsHAUSV657(t *testing.T) {
	email := "viewer@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := issueRepositoryForTest(a, "demo").Create(residentIssue{TenantSlug: "demo", AuthorEmail: "other@example.com", Category: "Reparatur", Title: "Fremdes privates Anliegen", Body: "Wasserhahn tropft", LocationType: issueLocationUnit, CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	body := authedRequest(t, a, email, "/demo/app/anliegen").Body.String()
	if !strings.Contains(body, "Erstes Anliegen melden") || strings.Contains(body, "Fremdes privates Anliegen") {
		t.Fatal("resident must see the first-issue heading and nothing of the private filing")
	}
	manager := "vera@example.com"
	m := newTestPortalApp(t, userProfile{Email: manager, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := issueRepositoryForTest(m, "demo").Create(residentIssue{TenantSlug: "demo", AuthorEmail: "other@example.com", Category: "Reparatur", Title: "Fremdes privates Anliegen", Body: "Wasserhahn tropft", LocationType: issueLocationUnit, CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if body := authedRequest(t, m, manager, "/demo/app/anliegen").Body.String(); strings.Contains(body, "Erstes Anliegen melden") || !strings.Contains(body, "Fremdes privates Anliegen") {
		t.Fatal("management sees the filing and the plain heading")
	}
}
