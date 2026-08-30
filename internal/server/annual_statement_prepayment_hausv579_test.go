package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	storepkg "github.com/inspr-at/hausv-org/internal/store"
)

func TestAnnualStatementPrepaymentsArePeriodBoundAuditedAndCompared(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", Label: "Top 1", UnitType: unitTypeResidential, MiteigentumsanteilPPM: 250_000},
		{ID: "top-2", Label: "Top 2", UnitType: unitTypeResidential, MiteigentumsanteilPPM: 750_000},
	}); err != nil {
		t.Fatal(err)
	}
	repositories := testRepositories(a, "demo")
	if _, err := repositories.annualStatementPeriods.Save(storepkg.AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := repositories.annualStatementCostTypes.EnsureDefaults("manager@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := repositories.annualStatementPeriods.EnsureStructure(2026, repositories.annualStatementCostTypes.List(), repositories.units.List(), "manager@example.com"); err != nil {
		t.Fatal(err)
	}
	document, err := repositories.documents.CreateGenerated(storepkg.DocumentRecord{
		Title: "Grundsteuer", Category: storepkg.DocumentCategoryBilling, Visibility: storepkg.DocumentVisibilityManagerOnly, UploadedBy: "manager@example.com",
	}, "grundsteuer.pdf", "application/pdf", []byte("%PDF-1.4 receipt"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repositories.annualStatementReceipts.Create(storepkg.AnnualStatementReceipt{
		DocumentID: document.ID, PeriodYear: 2026, CostTypeKey: "grundsteuer", AmountCents: 10_000,
		InvoiceDate: "2026-06-30", CreatedBy: "manager@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if got := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/prepayments", url.Values{
		"year": {"2026"}, "unit_id": {"top-1"}, "amount": {"100,00"},
	}).Code; got != http.StatusForbidden {
		t.Fatalf("resident POST = %d, want 403", got)
	}
	for name, form := range map[string]url.Values{
		"unknown period": {"year": {"2025"}, "unit_id": {"top-1"}, "amount": {"100,00"}},
		"unknown unit":   {"year": {"2026"}, "unit_id": {"top-9"}, "amount": {"100,00"}},
		"negative":       {"year": {"2026"}, "unit_id": {"top-1"}, "amount": {"-1,00"}},
		"too precise":    {"year": {"2026"}, "unit_id": {"top-1"}, "amount": {"1,001"}},
	} {
		response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/prepayments", form)
		if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/demo/app/settings/annual-statement?year=2026&prepayment=invalid#vorauszahlungen" {
			t.Fatalf("%s: status=%d location=%q", name, response.Code, response.Header().Get("Location"))
		}
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAnnualPrepaymentSave, Limit: 20})); got != 0 {
		t.Fatalf("rejected writes produced %d audit events", got)
	}
	for _, amount := range []string{"100,00", "125,50"} {
		response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/prepayments", url.Values{
			"year": {"2026"}, "unit_id": {"top-1"}, "amount": {amount},
		})
		if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/demo/app/settings/annual-statement?year=2026&prepayment=saved#vorauszahlungen" {
			t.Fatalf("save %s status=%d location=%q", amount, response.Code, response.Header().Get("Location"))
		}
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAnnualPrepaymentSave, Limit: 20})
	if len(events) != 2 || events[0].Details["previous_amount_cents"] != "10000" || events[0].Details["new_amount_cents"] != "12550" {
		t.Fatalf("correction audit = %+v", events)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?year=2026")
	for _, want := range []string{"Vorauszahlungen &amp; Akonto", "Top 1", `value="125,50"`, "Vorausgezahlt", "Zugeteilt", "25,00 €", "Saldo", "Guthaben 100,50 €", "Arbeitsvorschau"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("page missing %q", want)
		}
	}
	top2 := strings.Index(page.Body.String(), "Top 2")
	if top2 < 0 || !strings.Contains(page.Body.String()[top2:], "nicht erfasst") {
		t.Fatal("unit without a recorded Akonto must not be shown as 0,00 € paid")
	}
	if _, err := repositories.annualStatementPeriods.SaveStructureCostType(2026, storepkg.AnnualStatementCostType{
		Key: "wasser", Name: "Wasser", Allocatable: true, AllocationKey: storepkg.AllocationKeyPersonen, UpdatedBy: "manager@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	blocked := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?year=2026")
	if !strings.Contains(blocked.Body.String(), "noch nicht berechenbar") {
		t.Fatal("blocked allocation must not publish a unit-level money split")
	}
}
