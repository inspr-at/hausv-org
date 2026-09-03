package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestInboxQueueAndCountsAreScopedToManagedHouses(t *testing.T) {
	const email = "manager@example.com"
	a := newTestPortalApp(t, userProfile{
		Email: email, Role: roleManager, Tenants: []string{"demo", "house-b"},
		TenantMemberships: map[string]tenantMembership{"demo": {Role: roleManager}, "house-b": {Role: roleManager}},
		AuthMethods:       defaultAuthMethods(),
	})
	for _, tenant := range []tenantConfig{
		{Slug: "house-b", Name: "Haus B", Address: "B-Gasse 2", Organisation: "org"},
		{Slug: "house-c", Name: "Haus C", Address: "C-Gasse 3", Organisation: "org"},
	} {
		addTestTenant(a, tenant)
	}
	demo := a.tenants["demo"]
	demo.Organisation = "org"
	a.tenants["demo"] = demo
	a.organisations = map[string]config.OrganisationConfig{"org": {Key: "org", Name: "Organisation"}}
	database := dbtest.Open(t)
	repo := store.BindIntakeRepository(database, "org")
	a.intake = func(string) store.IntakeRepository { return repo }
	now := time.Now().UTC()
	for _, item := range []store.IntakeItem{
		{ID: "a", TenantSlug: "demo", Subject: "Managed A", Body: "A"},
		{ID: "b", TenantSlug: "house-b", Subject: "Managed B", Body: "B"},
		{ID: "c", TenantSlug: "house-c", Subject: "Secret C", Body: "C"},
		{ID: "u", Subject: "Unassigned U", Body: "U"},
	} {
		item.Source = store.IntakeSourceEmail
		item.Status = store.IntakeStatusOpen
		item.ReceivedAt = now
		if err := repo.Create(t.Context(), item); err != nil {
			t.Fatal(err)
		}
	}

	page := authedRequest(t, a, email, "/demo/app/verwaltung/posteingang")
	body := page.Body.String()
	if page.Code != http.StatusOK || !strings.Contains(body, "Managed A") || !strings.Contains(body, "Managed B") || !strings.Contains(body, "Unassigned U") {
		t.Fatalf("managed queue missing expected items: status=%d body=%s", page.Code, body)
	}
	if strings.Contains(body, "Secret C") || !strings.Contains(body, "3 offen") || !strings.Contains(body, "1 unzugeordnet") {
		t.Fatalf("queue or counts leaked unmanaged house: %s", body)
	}
}

func TestInboxAssigneeLimitFiltersBeforePagination(t *testing.T) {
	a, repo, _ := newInboxTestApp(t, roleManager)
	now := time.Now().UTC()
	for index, item := range []store.IntakeItem{
		{ID: "other", Subject: "Other newest", Suggestion: &store.IntakeSuggestion{Assignee: "other"}},
		{ID: "vera-first", Subject: "Vera first", Suggestion: &store.IntakeSuggestion{Assignee: "vera"}},
		{ID: "vera-second", Subject: "Vera second", Suggestion: &store.IntakeSuggestion{Assignee: "vera"}},
	} {
		item.TenantSlug = "demo"
		item.Source = store.IntakeSourceEmail
		item.Status = store.IntakeStatusProposed
		item.Body = item.Subject
		item.ReceivedAt = now.Add(-time.Duration(index) * time.Minute)
		if err := repo.Create(t.Context(), item); err != nil {
			t.Fatal(err)
		}
	}

	page := authedRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang?assignee=vera&limit=1")
	body := page.Body.String()
	if page.Code != http.StatusOK || !strings.Contains(body, "2 offen") || !strings.Contains(body, "2 mit Vorschlag") || !strings.Contains(body, "Vera first") {
		t.Fatalf("assignee queue/count mismatch: status=%d body=%s", page.Code, body)
	}
	if strings.Contains(body, "Other newest") || strings.Contains(body, "Vera second") {
		t.Fatalf("assignee limit returned wrong rows: %s", body)
	}
}

func TestVerwaltungSettingsRequireOrganisationAdmin(t *testing.T) {
	manager, _, _ := newInboxTestApp(t, roleManager)
	page := authedRequest(t, manager, "vera@example.com", "/demo/app/verwaltung")
	if page.Code != http.StatusOK || strings.Contains(page.Body.String(), `href="/demo/app/verwaltung/einstellungen"`) {
		t.Fatalf("manager shell exposed settings: status=%d", page.Code)
	}
	if denied := authedRequest(t, manager, "vera@example.com", "/demo/app/verwaltung/einstellungen"); denied.Code != http.StatusForbidden {
		t.Fatalf("manager settings status=%d, want 403", denied.Code)
	}

	admin, _, _ := newInboxTestApp(t, roleAdmin)
	allowed := authedRequest(t, admin, "vera@example.com", "/demo/app/verwaltung/einstellungen")
	if allowed.Code != http.StatusOK || !strings.Contains(allowed.Body.String(), "Einstellungen speichern") {
		t.Fatalf("organisation admin settings status=%d body=%s", allowed.Code, allowed.Body.String())
	}
	saved := authedFormRequest(t, admin, "vera@example.com", "/demo/app/verwaltung/einstellungen", url.Values{"threshold": {"85"}})
	if saved.Code != http.StatusSeeOther {
		t.Fatalf("organisation admin settings save status=%d body=%s", saved.Code, saved.Body.String())
	}
}

func TestDemoLoginWrongCodeIsNonEnumeratingAndOverridesLocalDev(t *testing.T) {
	const known = "manager@example.com"
	a := newTestPortalApp(t, userProfile{Email: known, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.localDevLogin = true
	a.demoLogin = true
	a.demoLoginCode = "correct-code"
	post := func(email string) *httptest.ResponseRecorder {
		values := url.Values{"email": {email}}
		req := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/auth/request", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://hausv.org")
		rr := httptest.NewRecorder()
		a.handler().ServeHTTP(rr, req)
		return rr
	}
	knownResponse := post(known)
	unknownResponse := post("unknown@example.com")
	if knownResponse.Code != http.StatusOK || unknownResponse.Code != http.StatusOK || knownResponse.Body.String() != unknownResponse.Body.String() {
		t.Fatalf("wrong-code responses differ: known=%d/%d bytes unknown=%d/%d bytes", knownResponse.Code, knownResponse.Body.Len(), unknownResponse.Code, unknownResponse.Body.Len())
	}
	if strings.Contains(knownResponse.Body.String(), `class="dev-link"`) || !strings.Contains(knownResponse.Body.String(), "Der Zugangscode war falsch") {
		t.Fatal("missing demo code must render only the shared error page")
	}
}

type failGetAfterAssignRepository struct {
	store.IntakeRepository
	assigned bool
}

func (r *failGetAfterAssignRepository) Get(ctx context.Context, id string) (store.IntakeItem, error) {
	if r.assigned {
		return store.IntakeItem{}, errors.New("get after assign failed")
	}
	return r.IntakeRepository.Get(ctx, id)
}

func (r *failGetAfterAssignRepository) Assign(ctx context.Context, id, tenantSlug, unit string) error {
	if err := r.IntakeRepository.Assign(ctx, id, tenantSlug, unit); err != nil {
		return err
	}
	r.assigned = true
	return nil
}

func TestInboxAssignStopsWhenReloadFails(t *testing.T) {
	a, base, _ := newInboxTestApp(t, roleManager)
	item := store.IntakeItem{ID: "assign-failure", Source: store.IntakeSourceEmail, Status: store.IntakeStatusOpen, Subject: "Assign", Body: "Assign", ReceivedAt: time.Now().UTC()}
	if err := base.Create(t.Context(), item); err != nil {
		t.Fatal(err)
	}
	failing := &failGetAfterAssignRepository{IntakeRepository: base}
	a.intake = func(string) store.IntakeRepository { return failing }
	response := authedFormRequest(t, a, "vera@example.com", "/demo/app/verwaltung/posteingang/assign-failure", url.Values{"action": {"assign"}, "house": {"demo"}})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("assign status=%d body=%s", response.Code, response.Body.String())
	}
	if events := a.auditStore.List(store.AuditFilter{Action: store.AuditActionIntakeAssign}); len(events) != 0 {
		t.Fatalf("failed assign created audit events: %#v", events)
	}
}

func TestInboxLocalDayAndEditedDueDate(t *testing.T) {
	location, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		t.Fatal(err)
	}
	previous := time.Local
	time.Local = location
	t.Cleanup(func() { time.Local = previous })

	handled := time.Date(2026, 9, 3, 0, 30, 0, 0, location)
	item := store.IntakeItem{ReceivedAt: handled.UTC(), Handling: &store.IntakeHandling{At: handled.UTC()}}
	if !intakeHandledToday(item, time.Date(2026, 9, 3, 3, 0, 0, 0, location)) {
		t.Fatal("00:30 Vienna handling must count on its Vienna calendar day")
	}

	item.Suggestion = &store.IntakeSuggestion{}
	form := url.Values{"due": {"2026-09-03"}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	applySuggestionForm(&item, req)
	threeAM := time.Date(2026, 9, 3, 3, 0, 0, 0, location)
	if !item.DueAt.After(threeAM) || item.DueAt.In(location).Hour() != 17 {
		t.Fatalf("edited due date=%s, want 17:00 local after 03:00", item.DueAt.In(location))
	}
}

func TestOrglessBreakGlassAdminSeesPortfolioWithoutInbox(t *testing.T) {
	const email = "breakglass@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	addTestTenant(a, tenantConfig{Slug: "house-b", Name: "Haus B", Address: "B-Gasse 2"})
	a.admins[email] = struct{}{}
	page := authedRequest(t, a, email, "/demo/app")
	body := page.Body.String()
	if page.Code != http.StatusOK || !strings.Contains(body, `href="/demo/app/verwaltung"`) || strings.Contains(body, `href="/demo/app/verwaltung/posteingang"`) {
		t.Fatalf("org-less navigation status=%d body=%s", page.Code, body)
	}
	inbox := authedRequest(t, a, email, "/demo/app/verwaltung/posteingang")
	if inbox.Code != http.StatusSeeOther || !strings.Contains(inbox.Header().Get("Location"), "/app/verwaltung?flash=Posteingang+braucht+eine+Organisation") {
		t.Fatalf("org-less inbox status=%d location=%q", inbox.Code, inbox.Header().Get("Location"))
	}
}

func TestRolePreviewRejectsEveryRegisteredAppPOST(t *testing.T) {
	source, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	registered := regexp.MustCompile(`mux\.HandleFunc\("POST (/app/[^" ]+)"`).FindAllSubmatch(source, -1)
	placeholder := regexp.MustCompile(`\{[^}]+\}`)
	if len(registered) == 0 {
		t.Fatal("no POST /app routes discovered")
	}

	const email = "admin@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	adminCookie := rolePreviewTestSession(t, a, email, roleAdmin)
	start := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/start", url.Values{"role": {roleOwner}}, adminCookie)
	previewCookie := rolePreviewResponseCookie(t, start)
	for _, match := range registered {
		pattern := string(match[1])
		if pattern == "/app/ansicht/ende" {
			continue
		}
		path := "/demo" + placeholder.ReplaceAllString(pattern, "test")
		response := rolePreviewTestRequest(t, a, http.MethodPost, path, nil, previewCookie)
		if response.Code != http.StatusForbidden {
			t.Fatalf("preview POST %s status=%d, want 403", pattern, response.Code)
		}
	}

	end := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/ende", nil, previewCookie)
	if end.Code != http.StatusSeeOther {
		t.Fatalf("preview end status=%d, want redirect", end.Code)
	}
	restoredCookie := rolePreviewResponseCookie(t, end)
	secondStart := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/start", url.Values{"role": {roleOwner}}, restoredCookie)
	logoutCookie := rolePreviewResponseCookie(t, secondStart)
	logout := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/auth/logout", nil, logoutCookie)
	if logout.Code != http.StatusSeeOther {
		t.Fatalf("preview logout status=%d, want redirect", logout.Code)
	}
}
