package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestAnnualStatementRunHandlerAllUnitsAndFailClosed(t *testing.T) {
	const manager = "manager@example.com"
	a := newTestPortalApp(t, userProfile{Email: manager, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	repos := testRepositories(a, "demo")
	units := []store.Unit{{ID: "a", Label: "Top 1", MiteigentumsanteilPPM: 250000}, {ID: "b", Label: "Top 2", MiteigentumsanteilPPM: 750000}}
	if err := repos.units.SetUnits(units); err != nil {
		t.Fatal(err)
	}
	period := store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31", UpdatedBy: manager}
	costs := []store.AnnualStatementCostType{{Key: "tax", Name: "Abgabe", AllocationKey: store.AllocationKeyNutzwert, Allocatable: true, UpdatedBy: manager}}
	if _, err := repos.annualStatementPeriods.SaveWithStructure(period, costs, units); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"year": {"2025"}, "amount": {"0,00"}, "unit_id": {"a"}}
	for _, tc := range []struct{ actor, origin string }{{"resident@example.com", "http://hausv.org/demo"}, {manager, "https://foreign.example"}} {
		if response := authedFormRequestWithOrigin(t, a, tc.actor, "/demo/app/settings/annual-statement/runs", form, tc.origin); response.Code != http.StatusForbidden {
			t.Fatalf("unauthorized status=%d", response.Code)
		}
	}
	page := authedRequest(t, a, manager, "/demo/app/settings/annual-statement?year=2025")
	for _, want := range []string{"Abrechnungslauf gesperrt", "Kein bestätigter Beleg erfasst", "Akonto fehlt", "disabled"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	blocked := authedFormRequest(t, a, manager, "/demo/app/settings/annual-statement/runs", form)
	if !strings.Contains(blocked.Header().Get("Location"), "run-status=blocked") {
		t.Fatalf("blocked=%d %s", blocked.Code, blocked.Header().Get("Location"))
	}
	if runs, _ := repos.annualStatementRuns.List(2025); len(runs) != 0 {
		t.Fatal("blocked or unauthorized request wrote run")
	}
	doc, err := repos.documents.CreateGenerated(store.DocumentRecord{Title: "Abgabe", Category: store.DocumentCategoryBilling, Visibility: store.DocumentVisibilityManagerOnly, UploadedBy: manager}, "abgabe.pdf", "application/pdf", []byte("%PDF-1.4 original"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := repos.annualStatementReceipts.Create(store.AnnualStatementReceipt{DocumentID: doc.ID, PeriodYear: 2025, CostTypeKey: "tax", AmountCents: 10000, InvoiceDate: "2025-02-01", CreatedBy: manager})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []store.AnnualStatementPrepayment{{PeriodYear: 2025, UnitID: "a", AmountCents: 3000, UpdatedBy: manager}, {PeriodYear: 2025, UnitID: "b", AmountCents: 0, UpdatedBy: manager}} {
		if _, _, err := repos.annualStatementAkontos.Save(item); err != nil {
			t.Fatal(err)
		}
	}
	ready := authedRequest(t, a, manager, "/demo/app/settings/annual-statement?year=2025")
	if !strings.Contains(ready.Body.String(), "Bereit zur Berechnung") {
		t.Fatal("complete input blocked")
	}
	if runs, _ := repos.annualStatementRuns.List(2025); len(runs) != 0 {
		t.Fatal("GET wrote run")
	}
	response := authedFormRequest(t, a, manager, "/demo/app/settings/annual-statement/runs", form)
	if response.Code != http.StatusSeeOther || !strings.Contains(response.Header().Get("Location"), "run-status=created") {
		t.Fatalf("create=%d %s", response.Code, response.Header().Get("Location"))
	}
	runs, err := repos.annualStatementRuns.List(2025)
	if err != nil || len(runs) != 1 || len(runs[0].Result.Units) != 2 || runs[0].Result.TotalCents != 10000 {
		t.Fatalf("run ignored server data: %+v %v", runs, err)
	}
	redirect, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	page = authedRequest(t, a, manager, redirect.RequestURI())
	for _, want := range []string{"Ergebnis · Lauf 1", "Guthaben 5,00 €", "Nachzahlung 75,00 €", "25,00 %", "75,00 %", "Gespeicherte Arbeitsfassung"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("missing result %q", want)
		}
	}
	if events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: store.AuditActionAnnualRunCreate, Limit: 10}); len(events) != 1 || events[0].TargetID != runs[0].ID {
		t.Fatalf("audit=%+v", events)
	}
	if _, err := repos.annualStatementReceipts.UpdateAmount(receipt.ID, 20000, manager); err != nil {
		t.Fatal(err)
	}
	historical := authedRequest(t, a, manager, "/demo/app/settings/annual-statement?year=2025&run="+runs[0].ID)
	// The historical run remains a separate, immutable result from the work preview.
	body := historical.Body.String()
	start := strings.Index(body, `data-annual-statement-run=`)
	if start < 0 {
		t.Fatal("missing historical run")
	}
	end := strings.Index(body[start:], `</article>`)
	if end < 0 || !strings.Contains(body[start:start+end], "Guthaben 5,00 €") {
		t.Fatal("historical balance changed")
	}
}

type failedAnnualStatementRunRepository struct {
	store.AnnualStatementRunRepository
	err     error
	creates int
}

func (r *failedAnnualStatementRunRepository) Create(int, string, time.Time, ...store.AnnualStatementRunPresentation) (store.AnnualStatementRun, error) {
	r.creates++
	return store.AnnualStatementRun{}, r.err
}

func TestAnnualStatementRunHandlerFormAndStoreErrors(t *testing.T) {
	const manager = "manager@example.com"
	a := newTestPortalApp(t, userProfile{Email: manager, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	repos := testRepositories(a, "demo")
	for _, tc := range []struct {
		name, form, status, message string
		err                         error
		code, creates               int
	}{
		{name: "malformed form", form: "year=%zz", code: http.StatusBadRequest},
		{name: "serialization", form: "year=2025", status: "conflict", message: "Parallel wurde ein weiterer Abrechnungslauf gestartet. Bitte die Seite neu laden", err: fmt.Errorf("create: %w", &pgconn.PgError{Code: "40001"}), code: http.StatusSeeOther, creates: 1},
		{name: "unique revision", form: "year=2025", status: "conflict", message: "Parallel wurde ein weiterer Abrechnungslauf gestartet. Bitte die Seite neu laden", err: &pgconn.PgError{Code: "23505"}, code: http.StatusSeeOther, creates: 1},
		{name: "other store error", form: "year=2025", status: "error", message: "Der Lauf konnte nicht gespeichert werden.", err: errors.New("test store unavailable"), code: http.StatusSeeOther, creates: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failed := &failedAnnualStatementRunRepository{AnnualStatementRunRepository: repos.annualStatementRuns, err: tc.err}
			ac := authCtx{email: manager, role: roleManager, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo"), repositories: repos}
			ac.repositories.annualStatementRuns = failed
			req := httptest.NewRequest(http.MethodPost, "/app/settings/annual-statement/runs", strings.NewReader(tc.form))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			a.createAnnualStatementRun(response, req, ac)
			if response.Code != tc.code || failed.creates != tc.creates {
				t.Fatalf("status=%d creates=%d", response.Code, failed.creates)
			}
			if tc.status != "" {
				target, err := url.Parse(response.Header().Get("Location"))
				if err != nil || target.Query().Get("run-status") != tc.status {
					t.Fatalf("redirect=%s error=%v", response.Header().Get("Location"), err)
				}
				view := annualStatementRunView(failed, repos.documents, 2025, "", target.Query().Get("run-status"), map[string]store.AnnualStatementConsumptionVector{})
				if !strings.Contains(view.Message, tc.message) {
					t.Fatalf("message=%q", view.Message)
				}
			}
		})
	}
}
