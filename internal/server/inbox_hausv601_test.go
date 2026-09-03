package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func newInboxTestApp(t *testing.T, role string) (*app, store.IntakeRepository, store.OrgSettingsRepository) {
	t.Helper()
	const email = "vera@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, FirstName: "Vera", LastName: "Verwalter", Role: role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	tenant := a.tenants["demo"]
	tenant.Organisation = "musterstadt"
	a.tenants["demo"] = tenant
	a.organisations = map[string]config.OrganisationConfig{"musterstadt": {Key: "musterstadt", Name: "Hausverwaltung Musterstadt"}}
	database := dbtest.Open(t)
	a.intake = func(orgKey string) store.IntakeRepository { return store.BindIntakeRepository(database, orgKey) }
	a.orgSettings = func(orgKey string) store.OrgSettingsRepository {
		return store.BindOrgSettingsRepository(database, orgKey)
	}
	a.textbausteine = func(orgKey string) store.TextbausteinRepository {
		return store.BindTextbausteinRepository(database, orgKey)
	}
	return a, a.intake("musterstadt"), a.orgSettings("musterstadt")
}

func seedInboxItem(t *testing.T, repo store.IntakeRepository, id string, status store.IntakeStatus, suggestion *store.IntakeSuggestion) {
	t.Helper()
	now := time.Now().Add(-time.Hour)
	if err := repo.Create(context.Background(), store.IntakeItem{ID: id, TenantSlug: "demo", Unit: "Top 1", Source: store.IntakeSourceEmail, FromName: "Rita", FromEmail: "rita@example.com", Subject: "Wasser im Keller", Body: "Bitte rasch prüfen.", ReceivedAt: now, Status: status, Suggestion: suggestion, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
}

func inboxSuggestion(confidence float64) *store.IntakeSuggestion {
	return &store.IntakeSuggestion{Source: "seed", Category: store.IntakeCategoryRepair, Priority: store.IssuePriorityHigh, TenantSlug: "demo", Unit: "Top 1", Assignee: "vera", Reply: "Wir kümmern uns darum.", Confidence: map[string]float64{"overall": confidence}, CreatedAt: time.Now()}
}

func TestInboxListRendersItemsCountsAndApproveCreatesIssueCommentAudit(t *testing.T) {
	a, intake, settings := newInboxTestApp(t, roleManager)
	seedInboxItem(t, intake, "in-1", store.IntakeStatusProposed, inboxSuggestion(.93))
	page := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang")
	if page.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", page.Code, page.Body.String())
	}
	for _, want := range []string{"Wasser im Keller", "1 offen", "1 mit Vorschlag", "Telefonnotiz", "Vorschlag: Reparatur · Hoch"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("page missing %q", want)
		}
	}

	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/in-1", url.Values{"action": {"approve"}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("approve status=%d body=%s", response.Code, response.Body.String())
	}
	issues := testRequestRepositories(t, a, "demo").issues.List()
	if len(issues) != 1 || issues[0].IntakeID != "in-1" || issues[0].Category != "Reparatur" || len(issues[0].Comments) != 1 || issues[0].Comments[0].Kind != store.IssueCommentKindInformation {
		t.Fatalf("created issue=%#v", issues)
	}
	got, _ := intake.Get(context.Background(), "in-1")
	if got.Status != store.IntakeStatusApproved || got.IssueID != issues[0].ID {
		t.Fatalf("handled intake=%#v", got)
	}
	counters, _ := settings.Get(context.Background())
	if counters.Counters.Approved != 1 {
		t.Fatalf("approved counter=%d", counters.Counters.Approved)
	}
	events := a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionIssueAIAccept})
	if len(events) != 1 || events[0].TargetID != "in-1" {
		t.Fatalf("accept audit=%#v", events)
	}
}

func TestInboxEditRejectAssignSettingsAndOwnerGuard(t *testing.T) {
	a, intake, _ := newInboxTestApp(t, roleManager)
	seedInboxItem(t, intake, "edit", store.IntakeStatusProposed, inboxSuggestion(.9))
	edit := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/edit", url.Values{"action": {"edit"}, "category": {store.IntakeCategoryHouseRules}, "priority": {store.IssuePriorityNorm}, "house": {"demo"}, "unit": {"Top 2"}, "reply": {"Danke."}})
	if edit.Code != http.StatusSeeOther {
		t.Fatalf("edit status=%d body=%s", edit.Code, edit.Body.String())
	}
	issues := testRequestRepositories(t, a, "demo").issues.List()
	if len(issues) != 1 || issues[0].Category != "Sonstiges" || !strings.Contains(issues[0].Body, "Hausordnung/Nachbarschaft") {
		t.Fatalf("edited issue=%#v", issues)
	}

	seedInboxItem(t, intake, "reject", store.IntakeStatusProposed, inboxSuggestion(.9))
	reject := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/reject", url.Values{"action": {"reject"}})
	if reject.Code != http.StatusSeeOther {
		t.Fatalf("reject=%d", reject.Code)
	}
	got, _ := intake.Get(context.Background(), "reject")
	if got.Status != store.IntakeStatusRejected {
		t.Fatalf("reject intake=%#v", got)
	}
	var rejected store.ResidentIssue
	for _, issue := range testRequestRepositories(t, a, "demo").issues.List() {
		if issue.IntakeID == "reject" {
			rejected = issue
		}
	}
	if rejected.ID == "" || rejected.Status != store.IssueStatusNew || len(rejected.Comments) != 0 {
		t.Fatalf("rejected issue=%#v", rejected)
	}

	now := time.Now()
	if err := intake.Create(context.Background(), store.IntakeItem{ID: "unassigned", Source: store.IntakeSourceEmail, Subject: "Zuordnen", Body: "Text", ReceivedAt: now, Status: store.IntakeStatusOpen}); err != nil {
		t.Fatal(err)
	}
	assign := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/unassigned", url.Values{"action": {"assign"}, "house": {"demo"}, "unit": {"Top 3"}})
	if assign.Code != http.StatusSeeOther {
		t.Fatalf("assign=%d", assign.Code)
	}
	assigned, _ := intake.Get(context.Background(), "unassigned")
	if assigned.TenantSlug != "demo" || assigned.Unit != "Top 3" {
		t.Fatalf("assigned=%#v", assigned)
	}

	values := url.Values{"threshold": {"85"}, "auto_enabled": {"1"}, "trust_reparatur": {"auto"}}
	settingsSave := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/einstellungen", values)
	if settingsSave.Code != http.StatusSeeOther {
		t.Fatalf("settings=%d body=%s", settingsSave.Code, settingsSave.Body.String())
	}
	saved, _ := a.orgSettings("musterstadt").Get(context.Background())
	if saved.AutoThreshold != .85 || !saved.AutoEnabled || saved.TrustLevels[store.IntakeCategoryRepair] != "auto" {
		t.Fatalf("settings round trip=%#v", saved)
	}
	if len(a.auditStore.List(store.AuditFilter{TenantSlug: "demo", Action: store.AuditActionVerwaltungSettings})) != 1 {
		t.Fatal("settings audit missing")
	}

	owner, _, _ := newInboxTestApp(t, roleOwner)
	for _, path := range []string{"/demo/app/verwaltung/posteingang", "/demo/app/verwaltung/posteingang/x", "/demo/app/verwaltung/einstellungen"} {
		if rr := authedRequest(t, owner, "vera@example.com", path); rr.Code != http.StatusForbidden {
			t.Fatalf("owner GET %s=%d", path, rr.Code)
		}
	}
	if rr := authedFormRequest(t, owner, "vera@example.com", "/demo/app/verwaltung/telefonnotiz", url.Values{}); rr.Code != http.StatusForbidden {
		t.Fatalf("owner phone=%d", rr.Code)
	}
}
