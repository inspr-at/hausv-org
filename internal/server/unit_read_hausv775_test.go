package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/dbtest"
	"github.com/inspr-at/hausv-org/internal/store"
)

func TestUnreadableUnitsAreExplainedOnVisiblePages(t *testing.T) {
	const email = "vera.verwalter@musterstadt.example"
	a, refs := newSQLTestApp(t, []sqlTestTenant{
		{Slug: "demo", Name: "Demohaus"},
		{Slug: "haus-b", Name: "Gesundes Haus"},
	}, userProfile{
		Email: email, FirstName: "Vera", LastName: "Verwalter", Role: roleAdmin,
		Tenants: []string{"demo", "haus-b"},
		TenantMemberships: map[string]tenantMembership{
			"demo":   {Role: roleAdmin},
			"haus-b": {Role: roleAdmin},
		},
		AuthMethods: defaultAuthMethods(),
	})
	a.annualStatementPeriods = store.NewSQLAnnualStatementPeriodStore(a.tenantDB)
	a.annualStatementCostTypes = store.NewSQLAnnualStatementCostTypeStore(a.tenantDB)
	a.annualStatementReceipts = store.NewSQLAnnualStatementReceiptStore(a.tenantDB)
	a.annualStatementAkontos = store.NewSQLAnnualStatementPrepaymentStore(a.tenantDB)
	a.annualConsumption = store.NewSQLAnnualStatementConsumptionStore(a.tenantDB)
	a.annualStatementRuns = store.NewSQLAnnualStatementRunStore(a.tenantDB, a.documentStore)

	demoUnits, ok := store.BindUnitRepository(a.unitStore, refs["demo"])
	if !ok {
		t.Fatal("bind demo units")
	}
	if err := demoUnits.SetUnits([]store.Unit{
		{ID: "top-1", Label: "Top Kaputt", UnitType: store.UnitTypeResidential},
		{ID: "top-2", Label: "Top Zwei", UnitType: store.UnitTypeResidential},
	}); err != nil {
		t.Fatal(err)
	}
	healthyUnits, ok := store.BindUnitRepository(a.unitStore, refs["haus-b"])
	if !ok {
		t.Fatal("bind healthy units")
	}
	if err := healthyUnits.SetUnits([]store.Unit{{ID: "ok-1", Label: "Top Lesbar", UnitType: store.UnitTypeResidential}}); err != nil {
		t.Fatal(err)
	}
	periods, ok := store.BindAnnualStatementPeriodRepository(a.annualStatementPeriods, refs["demo"])
	if !ok {
		t.Fatal("bind periods")
	}
	if _, err := periods.SaveWithStructure(store.AnnualStatementPeriod{
		Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: email, UpdatedAt: time.Now(),
	}, store.AnnualStatementDefaultCostTypes(email), []store.Unit{{ID: "top-1", Label: "Top Kaputt"}, {ID: "top-2", Label: "Top Zwei"}}); err != nil {
		t.Fatal(err)
	}
	pool := dbtest.MaintenanceView(t, a.scopedDB)
	if _, err := pool.Exec(`UPDATE units SET data='invalid-json' WHERE tenant_id=$1 AND id='top-2'`, refs["demo"].ID); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"/demo/app/settings/building?section=units",
		"/demo/app/settings/annual-statement?year=2026",
		"/demo/app/kontakte",
		"/demo/app/verwaltung",
	} {
		page := tenantRequest(t, a, email, "demo", path)
		if page.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, page.Code)
		}
		body := page.Body.String()
		if !strings.Contains(body, unitDataUnreadableNotice) {
			t.Fatalf("%s missing the unreadable-unit notice", path)
		}
		if strings.Contains(body, "Noch keine Einheiten") {
			t.Fatalf("%s presented an empty house", path)
		}
	}

	healthy := tenantRequest(t, a, email, "haus-b", "/haus-b/app/settings/building?section=units")
	if healthy.Code != http.StatusOK || !strings.Contains(healthy.Body.String(), "Top Lesbar") || strings.Contains(healthy.Body.String(), unitDataUnreadableNotice) {
		t.Fatalf("healthy house status=%d body missing the unit or showing the notice", healthy.Code)
	}

	created := tenantFormRequest(t, a, email, "demo", "/demo/app/settings/annual-statement/runs", "year=2026")
	if created.Code != http.StatusSeeOther || !strings.Contains(created.Header().Get("Location"), "run-status=blocked") {
		t.Fatalf("calculation status=%d location=%s", created.Code, created.Header().Get("Location"))
	}
	blocked := tenantRequest(t, a, email, "demo", created.Header().Get("Location"))
	if blocked.Code != http.StatusOK || !strings.Contains(blocked.Body.String(), unitDataUnreadableNotice) {
		t.Fatalf("blocked calculation page status=%d", blocked.Code)
	}
}

func tenantRequest(t *testing.T, a *app, email, tenant, path string) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, tenant, authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://hausv.org"+path, nil)
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}

func tenantFormRequest(t *testing.T, a *app, email, tenant, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	token, _, err := a.sessions.Put(email, tenant, authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://hausv.org"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://hausv.org")
	req.AddCookie(&http.Cookie{Name: "weg_session", Value: token})
	rr := httptest.NewRecorder()
	a.handler().ServeHTTP(rr, req)
	return rr
}
