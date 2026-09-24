package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

// TestAuditOtherOrgUnassignedInboxReadable is the HAUSV-769 probe A01, turned
// into a denial regression. A manager of organisation A who is only a resident
// of house B must not read B's unassigned inbox.
func TestAuditOtherOrgUnassignedInboxReadable(t *testing.T) {
	a, intake, _ := newInboxTestApp(t, roleResident)
	other := a.tenants["demo"]
	other.Slug = "other"
	other.Name = "Other"
	other.Organisation = "other-org"
	addTestTenant(a, other)
	a.organisations["other-org"] = config.OrganisationConfig{Key: "other-org", Name: "Other Org"}
	a.profiles["vera@example.com"] = userProfile{
		Email: "vera@example.com", Role: roleResident, Tenants: []string{"demo", "other"},
		TenantMemberships: map[string]tenantMembership{
			"demo":  {Role: roleResident},
			"other": {Role: roleManager},
		},
		AuthMethods: defaultAuthMethods(),
	}
	if err := intake.Create(context.Background(), store.IntakeItem{
		ID: "audit-unassigned", Source: store.IntakeSourceEmail, Subject: "AUDIT_FOREIGN_ORG_MESSAGE",
		Body: "Synthetic audit data", ReceivedAt: time.Now(), Status: store.IntakeStatusOpen,
	}); err != nil {
		t.Fatal(err)
	}
	if a.roleFor("vera@example.com", "demo") != roleResident || a.roleFor("vera@example.com", "other") != roleManager {
		t.Fatal("invalid mixed-role fixture")
	}

	const secret = "AUDIT_FOREIGN_ORG_MESSAGE"
	paths := []struct {
		method string
		path   string
		form   url.Values
	}{
		{method: http.MethodGet, path: "/demo/app/verwaltung"},
		{method: http.MethodGet, path: "/demo/app/verwaltung/posteingang"},
		{method: http.MethodGet, path: "/demo/app/verwaltung/posteingang/audit-unassigned"},
		{method: http.MethodGet, path: "/demo/app/verwaltung/posteingang/audit-unassigned/vorschlag"},
		{method: http.MethodPost, path: "/demo/app/verwaltung/posteingang/audit-unassigned", form: url.Values{"action": {"assign"}, "house": {"other"}}},
		{method: http.MethodPost, path: "/demo/app/verwaltung/telefonnotiz", form: url.Values{"house": {"demo"}, "subject": {"x"}, "body": {"y"}, "from_name": {"z"}}},
		{method: http.MethodGet, path: "/demo/app/verwaltung/einstellungen"},
		{method: http.MethodPost, path: "/demo/app/verwaltung/einstellungen", form: url.Values{"threshold": {"80"}}},
	}
	for _, path := range paths {
		t.Run(path.method+" "+path.path, func(t *testing.T) {
			response := organisationAccessRequest(t, a, "vera@example.com", "demo", path.method, path.path, path.form)
			if response.Code != http.StatusForbidden || strings.Contains(response.Body.String(), secret) {
				t.Fatalf("status=%d, want 403 without the foreign subject; body=%s", response.Code, response.Body.String())
			}
		})
	}
}

// House-only managers see unassigned items of their own organisation because
// managedIntakeFilter sets IncludeUnassigned for every non-support manager and
// actorCanAccessIntake allows an empty house when that manager has a house in
// the organisation. TestInboxQueueAndCountsAreScopedToManagedHouses locks the
// same rule. Organisation-admin status is not required. Items stay inside the
// selected organisation, and assigned items stay inside managed houses.
func TestOrganisationAccessMatrix(t *testing.T) {
	t.Run("manager of A on A", func(t *testing.T) {
		a := newOrganisationAccessApp(t, userProfile{
			Email: "manager-a@example.com", Role: roleManager, Tenants: []string{"haus-a", "haus-b"},
			TenantMemberships: map[string]tenantMembership{
				"haus-a": {Role: roleManager},
				"haus-b": {Role: roleResident},
			},
			AuthMethods: defaultAuthMethods(),
		})
		page := organisationAccessRequest(t, a, "manager-a@example.com", "haus-a", http.MethodGet, "/haus-a/app/verwaltung/posteingang", nil)
		assertInboxAllows(t, page, "A_HOUSE", "A_UNASSIGNED")
		assertInboxHides(t, page, "B_HOUSE", "B_UNASSIGNED", "B_OTHER_HOUSE")
	})

	t.Run("manager of A resident of B", func(t *testing.T) {
		a := newOrganisationAccessApp(t, userProfile{
			Email: "mixed@example.com", Role: roleResident, Tenants: []string{"haus-a", "haus-b"},
			TenantMemberships: map[string]tenantMembership{
				"haus-a": {Role: roleManager},
				"haus-b": {Role: roleResident},
			},
			AuthMethods: defaultAuthMethods(),
		})
		for _, path := range []string{"/haus-b/app/verwaltung", "/haus-b/app/verwaltung/posteingang", "/haus-b/app/verwaltung/posteingang/b-unassigned", "/haus-b/app/verwaltung/einstellungen"} {
			response := organisationAccessRequest(t, a, "mixed@example.com", "haus-b", http.MethodGet, path, nil)
			if response.Code != http.StatusForbidden || strings.Contains(response.Body.String(), "B_UNASSIGNED") || strings.Contains(response.Body.String(), "B_HOUSE") {
				t.Fatalf("%s status=%d body=%s", path, response.Code, response.Body.String())
			}
		}
		action := organisationAccessRequest(t, a, "mixed@example.com", "haus-b", http.MethodPost, "/haus-b/app/verwaltung/posteingang/b-unassigned", url.Values{"action": {"assign"}, "house": {"haus-a"}})
		if action.Code != http.StatusForbidden {
			t.Fatalf("assign status=%d, want 403", action.Code)
		}
	})

	t.Run("manager of A and B", func(t *testing.T) {
		a := newOrganisationAccessApp(t, userProfile{
			Email: "both@example.com", Role: roleManager, Tenants: []string{"haus-a", "haus-b"},
			TenantMemberships: map[string]tenantMembership{
				"haus-a": {Role: roleManager},
				"haus-b": {Role: roleManager},
			},
			AuthMethods: defaultAuthMethods(),
		})
		onA := organisationAccessRequest(t, a, "both@example.com", "haus-a", http.MethodGet, "/haus-a/app/verwaltung/posteingang", nil)
		assertInboxAllows(t, onA, "A_HOUSE", "A_UNASSIGNED")
		assertInboxHides(t, onA, "B_HOUSE", "B_UNASSIGNED", "B_OTHER_HOUSE")
		onB := organisationAccessRequest(t, a, "both@example.com", "haus-b", http.MethodGet, "/haus-b/app/verwaltung/posteingang", nil)
		assertInboxAllows(t, onB, "B_HOUSE", "B_UNASSIGNED")
		assertInboxHides(t, onB, "A_HOUSE", "A_UNASSIGNED", "B_OTHER_HOUSE")

		portfolioA := organisationAccessRequest(t, a, "both@example.com", "haus-a", http.MethodGet, "/haus-a/app/verwaltung", nil)
		if portfolioA.Code != http.StatusOK || !strings.Contains(portfolioA.Body.String(), `aria-label="Haus A öffnen"`) || strings.Contains(portfolioA.Body.String(), `aria-label="Haus B öffnen"`) {
			t.Fatalf("portfolio A mixed houses: status=%d", portfolioA.Code)
		}
		portfolioB := organisationAccessRequest(t, a, "both@example.com", "haus-b", http.MethodGet, "/haus-b/app/verwaltung", nil)
		if portfolioB.Code != http.StatusOK || !strings.Contains(portfolioB.Body.String(), `aria-label="Haus B öffnen"`) || strings.Contains(portfolioB.Body.String(), `aria-label="Haus A öffnen"`) {
			t.Fatalf("portfolio B mixed houses: status=%d", portfolioB.Code)
		}

		cross := organisationAccessRequest(t, a, "both@example.com", "haus-a", http.MethodPost, "/haus-a/app/verwaltung/posteingang/b-unassigned", url.Values{"action": {"reject"}})
		if cross.Code == http.StatusOK || strings.Contains(cross.Body.String(), "B_UNASSIGNED") {
			t.Fatalf("cross-org action status=%d", cross.Code)
		}
		item, err := a.intake("org-b").Get(context.Background(), "b-unassigned")
		if err != nil || item.Status != store.IntakeStatusOpen || item.TenantSlug != "" {
			t.Fatalf("foreign action changed org B item: %+v err=%v", item, err)
		}
	})

	t.Run("house-only manager in B", func(t *testing.T) {
		a := newOrganisationAccessApp(t, userProfile{
			Email: "house@example.com", Role: roleManager, Tenants: []string{"haus-b"},
			TenantMemberships: map[string]tenantMembership{"haus-b": {Role: roleManager}},
			AuthMethods:       defaultAuthMethods(),
		})
		page := organisationAccessRequest(t, a, "house@example.com", "haus-b", http.MethodGet, "/haus-b/app/verwaltung/posteingang", nil)
		// Existing rule, not organisation-admin status: this manager sees B's
		// unassigned queue and B's own house, and not the sibling house.
		assertInboxAllows(t, page, "B_HOUSE", "B_UNASSIGNED")
		assertInboxHides(t, page, "B_OTHER_HOUSE", "A_HOUSE", "A_UNASSIGNED")
		denied := organisationAccessRequest(t, a, "house@example.com", "haus-b", http.MethodPost, "/haus-b/app/verwaltung/posteingang/b-other", url.Values{"action": {"suggest"}})
		if denied.Code != http.StatusForbidden {
			t.Fatalf("sibling house action status=%d, want 403", denied.Code)
		}
	})

	t.Run("admin", func(t *testing.T) {
		a := newOrganisationAccessApp(t, userProfile{
			Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"haus-b", "haus-b2"},
			TenantMemberships: map[string]tenantMembership{
				"haus-b":  {Role: roleAdmin},
				"haus-b2": {Role: roleAdmin},
			},
			AuthMethods: defaultAuthMethods(),
		})
		page := organisationAccessRequest(t, a, "admin@example.com", "haus-b", http.MethodGet, "/haus-b/app/verwaltung/posteingang", nil)
		assertInboxAllows(t, page, "B_HOUSE", "B_OTHER_HOUSE", "B_UNASSIGNED")
		assertInboxHides(t, page, "A_HOUSE", "A_UNASSIGNED")
		settings := organisationAccessRequest(t, a, "admin@example.com", "haus-b", http.MethodGet, "/haus-b/app/verwaltung/einstellungen", nil)
		if settings.Code != http.StatusOK || !strings.Contains(settings.Body.String(), "Einstellungen speichern") {
			t.Fatalf("admin settings status=%d body=%s", settings.Code, settings.Body.String())
		}
		// A house manager of B still does not become an organisation admin.
		manager := newOrganisationAccessApp(t, userProfile{
			Email: "house@example.com", Role: roleManager, Tenants: []string{"haus-b"},
			TenantMemberships: map[string]tenantMembership{"haus-b": {Role: roleManager}},
			AuthMethods:       defaultAuthMethods(),
		})
		denied := organisationAccessRequest(t, manager, "house@example.com", "haus-b", http.MethodGet, "/haus-b/app/verwaltung/einstellungen", nil)
		if denied.Code != http.StatusForbidden {
			t.Fatalf("house manager settings status=%d, want 403", denied.Code)
		}
	})
}

func newOrganisationAccessApp(t *testing.T, profile userProfile) *app {
	t.Helper()
	a := newTestPortalApp(t, profile)
	a.tenants["demo"] = tenantConfig{Slug: "demo", Name: "Unrelated", Address: "Anderswo 1"}
	addTestTenant(a, tenantConfig{Slug: "haus-a", Name: "Haus A", Address: "A-Gasse 1", Organisation: "org-a"})
	addTestTenant(a, tenantConfig{Slug: "haus-b", Name: "Haus B", Address: "B-Gasse 1", Organisation: "org-b"})
	addTestTenant(a, tenantConfig{Slug: "haus-b2", Name: "Haus B2", Address: "B-Gasse 2", Organisation: "org-b"})
	a.organisations = map[string]config.OrganisationConfig{
		"org-a": {Key: "org-a", Name: "Organisation A"},
		"org-b": {Key: "org-b", Name: "Organisation B"},
	}
	database := dbtest.Open(t)
	a.intake = func(orgKey string) store.IntakeRepository { return store.BindIntakeRepository(database, orgKey) }
	a.orgSettings = func(orgKey string) store.OrgSettingsRepository {
		return store.BindOrgSettingsRepository(database, orgKey)
	}
	now := time.Now().UTC()
	seed := func(org, id, house, subject string) {
		t.Helper()
		item := store.IntakeItem{
			ID: id, TenantSlug: house, Source: store.IntakeSourceEmail, Subject: subject, Body: subject,
			ReceivedAt: now, Status: store.IntakeStatusOpen,
		}
		if err := a.intake(org).Create(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}
	seed("org-a", "a-house", "haus-a", "A_HOUSE")
	seed("org-a", "a-unassigned", "", "A_UNASSIGNED")
	seed("org-b", "b-house", "haus-b", "B_HOUSE")
	seed("org-b", "b-other", "haus-b2", "B_OTHER_HOUSE")
	seed("org-b", "b-unassigned", "", "B_UNASSIGNED")
	return a
}

func organisationAccessRequest(t *testing.T, a *app, email, tenant, method, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, tenant, authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, "http://hausv.org"+path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://hausv.org")
	}
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func assertInboxAllows(t *testing.T, response *httptest.ResponseRecorder, subjects ...string) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%s", response.Code, response.Body.String())
	}
	for _, subject := range subjects {
		if !strings.Contains(response.Body.String(), subject) {
			t.Fatalf("inbox missing %q", subject)
		}
	}
}

func assertInboxHides(t *testing.T, response *httptest.ResponseRecorder, subjects ...string) {
	t.Helper()
	for _, subject := range subjects {
		if strings.Contains(response.Body.String(), subject) {
			t.Fatalf("inbox exposed %q", subject)
		}
	}
}
