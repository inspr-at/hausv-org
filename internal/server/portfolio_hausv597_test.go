package server

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestBuildPortfolioRanksHousesAndCalculatesKPIs(t *testing.T) {
	now := time.Date(2026, time.September, 9, 10, 0, 0, 0, time.Local)
	dueToday := time.Date(2026, time.September, 9, 16, 0, 0, 0, time.Local)
	houses := []portfolioHouseInput{
		{
			Slug: "alpha", Name: "Zederhaus", UnitCount: 12,
			Issues: []store.ResidentIssue{
				{Status: store.IssueStatusNew, Priority: store.IssuePriorityHigh, CreatedAt: now.Add(-8 * 24 * time.Hour), AssigneeEmail: "vera@example.com"},
				{Status: store.IssueStatusProgress, Priority: store.IssuePriorityNorm, CreatedAt: now.Add(-2 * time.Hour), DueAt: dueToday, AssigneeEmail: "vera@example.com"},
				{Status: store.IssueStatusDone, Priority: store.IssuePriorityUrgent, CreatedAt: now.Add(-30 * 24 * time.Hour), DueAt: now.Add(-time.Hour)},
			},
			PeopleNames: map[string]string{"vera@example.com": "Vera Verwalter"},
		},
		{
			Slug: "beta", Name: "Ahornhaus", UnitCount: 8,
			Issues: []store.ResidentIssue{
				{Status: store.IssueStatusAccepted, Priority: store.IssuePriorityUrgent, CreatedAt: now.Add(-30 * time.Hour), DueAt: now.Add(-time.Hour)},
			},
		},
		{
			Slug: "quiet", Name: "Birkenhaus", UnitCount: 4,
			Issues: []store.ResidentIssue{{Status: store.IssueStatusDone, CreatedAt: now.Add(-time.Hour)}},
		},
	}

	tests := []struct {
		name      string
		sort      string
		wantOrder []string
	}{
		{name: "handlungsbedarf", sort: "need", wantOrder: []string{"Zederhaus", "Ahornhaus"}},
		{name: "name", sort: "name", wantOrder: []string{"Ahornhaus", "Zederhaus"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := buildPortfolio(now, "Hausverwaltung Musterstadt", "Vera", test.sort, houses)
			if got.HouseCount != 3 || got.UnitCount != 24 || got.ActionHouseCount != 2 || got.QuietHouseCount != 1 {
				t.Fatalf("house totals = houses:%d units:%d action:%d quiet:%d", got.HouseCount, got.UnitCount, got.ActionHouseCount, got.QuietHouseCount)
			}
			if got.OpenIssues != 3 || got.Overdue != 2 || got.DueToday != 2 || got.NewSinceYesterday != 1 {
				t.Fatalf("KPIs = open:%d overdue:%d today:%d new:%d", got.OpenIssues, got.Overdue, got.DueToday, got.NewSinceYesterday)
			}
			if len(got.Houses) != 2 || got.Houses[0].Name != test.wantOrder[0] || got.Houses[1].Name != test.wantOrder[1] {
				t.Fatalf("ranking = %+v, want %v", got.Houses, test.wantOrder)
			}
			zeder := got.Houses[0]
			if test.sort == "name" {
				zeder = got.Houses[1]
			}
			if zeder.Score != 9 || zeder.Oldest != "8 Tage" || zeder.Assignee != "Vera Verwalter" {
				t.Fatalf("Zederhaus aggregate = %+v", zeder)
			}
		})
	}
}

func TestPortfolioIssueOverdueFallbackThresholds(t *testing.T) {
	now := time.Date(2026, time.September, 9, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		issue   store.ResidentIssue
		overdue bool
	}{
		{name: "high at seven days is not overdue", issue: store.ResidentIssue{Priority: store.IssuePriorityHigh, CreatedAt: now.Add(-7 * 24 * time.Hour)}},
		{name: "high beyond seven days is overdue", issue: store.ResidentIssue{Priority: store.IssuePriorityHigh, CreatedAt: now.Add(-7*24*time.Hour - time.Second)}, overdue: true},
		{name: "normal at fourteen days is not overdue", issue: store.ResidentIssue{Priority: store.IssuePriorityNorm, CreatedAt: now.Add(-14 * 24 * time.Hour)}},
		{name: "normal beyond fourteen days is overdue", issue: store.ResidentIssue{Priority: store.IssuePriorityNorm, CreatedAt: now.Add(-14*24*time.Hour - time.Second)}, overdue: true},
		{name: "explicit past due date wins", issue: store.ResidentIssue{Priority: store.IssuePriorityLow, CreatedAt: now.Add(-time.Hour), DueAt: now.Add(-time.Second)}, overdue: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := portfolioIssueOverdue(now, test.issue); got != test.overdue {
				t.Fatalf("portfolioIssueOverdue() = %v, want %v", got, test.overdue)
			}
		})
	}
}

func TestPortfolioHTTPShowsOnlyManagedHousesAndForbidsOwner(t *testing.T) {
	const email = "multi@example.com"
	a := newTestPortalApp(t, userProfile{
		Email: email, FirstName: "Mara", LastName: "Mehrhaus", Role: roleAdmin,
		Tenants: []string{"demo", "haus-b"},
		TenantMemberships: map[string]tenantMembership{
			"demo":   {Role: roleAdmin},
			"haus-b": {Role: roleAdmin},
		},
		AuthMethods: defaultAuthMethods(),
	})
	demo := a.tenants["demo"]
	demo.Name = "Demohaus Portfolio"
	a.tenants["demo"] = demo
	addTestTenant(a, tenantConfig{Slug: "haus-b", Name: "Haus B Portfolio", Address: "Nebenweg 2"})
	addTestTenant(a, tenantConfig{Slug: "foreign", Name: "Fremdhaus Geheim", Address: "Nicht sichtbar 3"})

	for _, slug := range []string{"demo", "haus-b", "foreign"} {
		repository := testRequestRepositories(t, a, slug).issues
		if _, err := repository.Create(store.ResidentIssue{
			AuthorEmail: "resident@example.com", Category: "Reparatur", Title: "Offenes Anliegen " + slug,
			Body: "Bitte bearbeiten", LocationType: store.IssueLocationCommon, Status: store.IssueStatusNew,
			CreatedAt: time.Now().Add(-time.Hour),
		}); err != nil {
			t.Fatalf("seed issue for %s: %v", slug, err)
		}
	}

	page := authedRequest(t, a, email, "/demo/app/verwaltung")
	if page.Code != http.StatusOK {
		t.Fatalf("portfolio status = %d: %s", page.Code, page.Body.String())
	}
	for _, want := range []string{"Demohaus Portfolio", "Haus B Portfolio"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("portfolio missing managed house %q", want)
		}
	}
	if strings.Contains(page.Body.String(), "Fremdhaus Geheim") || strings.Contains(page.Body.String(), "Offenes Anliegen foreign") {
		t.Fatal("portfolio exposed a house the actor does not manage")
	}

	const ownerEmail = "owner-portfolio@example.com"
	owner := newTestPortalApp(t, userProfile{Email: ownerEmail, Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	denied := authedRequest(t, owner, ownerEmail, "/demo/app/verwaltung")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("owner portfolio status = %d, want 403", denied.Code)
	}
}

func TestPortfolioActivityFiltersBeforeLimitAndKeepsAuditLog(t *testing.T) {
	const email = "manager@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	now := time.Now()
	appendEvent := func(tenant, action string, at time.Time) {
		t.Helper()
		if err := a.auditStore.Append(store.AuditEvent{TenantSlug: tenant, Action: action, ActorEmail: email, At: at}); err != nil {
			t.Fatal(err)
		}
	}
	// Session activity alone leaves a calm empty state.
	appendEvent("demo", store.AuditActionLogin, now.Add(-time.Hour))
	page := authedRequest(t, a, email, "/demo/app/verwaltung")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Noch keine Aktivitäten.") {
		t.Fatal("portfolio with only a login must show an empty activity feed")
	}
	for i := range 8 {
		appendEvent("demo", store.AuditActionIssueWorkflow, now.Add(time.Duration(i-60)*time.Minute))
	}
	appendEvent("demo", store.AuditActionAnnualRunApprove, now.Add(-20*time.Minute))
	// More than the store's default page of reads and session changes must not
	// crowd older domain events out of the six visible activity slots.
	for i := range 210 {
		for _, action := range []string{store.AuditActionLogin, store.AuditActionContextSwitch, store.AuditActionDocumentDownload} {
			appendEvent("demo", action, now.Add(time.Duration(i-1000)*time.Second))
		}
	}
	appendEvent("foreign", store.AuditActionEventCreate, now)
	page = authedRequest(t, a, email, "/demo/app/verwaltung")
	body := page.Body.String()
	if page.Code != http.StatusOK || strings.Count(body, `class="portfolio-audit-item"`) != 6 || !strings.Contains(body, "Jahresabrechnung freigegeben") {
		t.Fatal("portfolio must show the six latest domain events, including the approval")
	}
	_, recent, _ := strings.Cut(body, `id="portfolio-recent-title"`)
	recent, _, _ = strings.Cut(recent, "</section>")
	for _, forbidden := range []string{" · Anmeldung", " · Portal gewechselt", " · Dokument heruntergeladen", " · Termin angelegt"} {
		if strings.Contains(recent, forbidden) {
			t.Fatalf("portfolio activity exposed %q", forbidden)
		}
	}
	if len(a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionLogin, Limit: 500})) != 211 {
		t.Fatal("filtering the portfolio must preserve login audit records")
	}
}
