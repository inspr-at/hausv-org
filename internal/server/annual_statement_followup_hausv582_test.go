package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	storepkg "github.com/inspr-at/hausv-org/internal/store"
)

func TestAnnualStatementFollowupClonesStructureWithoutAmountsHAUSV582(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{{
		ID: "top-1", Label: "Top 1", UnitType: unitTypeResidential, MiteigentumsanteilPPM: 1_000_000,
	}}); err != nil {
		t.Fatal(err)
	}
	repositories := testRepositories(a, "demo")
	if _, err := repositories.annualStatementPeriods.Save(storepkg.AnnualStatementPeriod{
		Year: 2026, StartsOn: "2026-04-01", EndsOn: "2027-03-31", UpdatedBy: "manager@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if err := repositories.annualStatementCostTypes.EnsureDefaults("manager@example.com"); err != nil {
		t.Fatal(err)
	}
	costTypesBefore := repositories.annualStatementCostTypes.List()
	if _, _, err := repositories.annualStatementAkontos.Save(storepkg.AnnualStatementPrepayment{
		PeriodYear: 2026, UnitID: "top-1", AmountCents: 12_345, UpdatedBy: "manager@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	document, err := repositories.documents.CreateGenerated(storepkg.DocumentRecord{
		Title: "Grundsteuer 2026", Category: storepkg.DocumentCategoryBilling, Visibility: storepkg.DocumentVisibilityManagerOnly, UploadedBy: "manager@example.com",
	}, "grundsteuer-2026.pdf", "application/pdf", []byte("%PDF-1.4 fixture"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repositories.annualStatementReceipts.Create(storepkg.AnnualStatementReceipt{
		DocumentID: document.ID, PeriodYear: 2026, CostTypeKey: "grundsteuer", AmountCents: 50_000,
		InvoiceDate: "2026-06-30", CreatedBy: "manager@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	form := url.Values{"source_year": {"2026"}}
	if got := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/periods/next", form).Code; got != http.StatusForbidden {
		t.Fatalf("resident POST = %d, want 403", got)
	}
	response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/periods/next", form)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/demo/app/settings/annual-statement?year=2027&period=cloned#perioden" {
		t.Fatalf("clone status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	periods := repositories.annualStatementPeriods.List()
	if len(periods) != 2 || periods[0].Year != 2027 || periods[0].StartsOn != "2027-04-01" || periods[0].EndsOn != "2028-03-31" {
		t.Fatalf("periods after clone = %+v", periods)
	}
	if got := repositories.annualStatementCostTypes.List(); len(got) != len(costTypesBefore) {
		t.Fatalf("cost type catalogue was copied instead of reused: before=%d after=%d", len(costTypesBefore), len(got))
	}
	if got := repositories.annualStatementAkontos.ListByPeriod(2027); len(got) != 0 {
		t.Fatalf("new period inherited prepayments: %+v", got)
	}
	if got := repositories.annualStatementReceipts.ListByPeriod(2027); len(got) != 0 {
		t.Fatalf("new period inherited receipts: %+v", got)
	}
	events := a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAnnualPeriodSave, Limit: 20})
	if len(events) != 1 || events[0].TargetID != "2027" || events[0].Details["source_year"] != "2026" || events[0].Details["copied_amounts"] != "false" {
		t.Fatalf("follow-up audit = %+v", events)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?year=2027&period=cloned")
	for _, want := range []string{
		"Folgejahr 2028 vorbereiten", "01.04.2028 – 31.03.2029", "Abrechnungsfrist als Erinnerung", "30.09.2028", "keine Rechtsberatung",
		"Vorlage übernommen. Belege, Beträge und Akonto wurden nicht kopiert.",
	} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("follow-up page missing %q", want)
		}
	}
}

func TestAnnualStatementFollowupNeverOverwritesExistingPeriodHAUSV582(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	repository := testRepositories(a, "demo").annualStatementPeriods
	for _, period := range []storepkg.AnnualStatementPeriod{
		{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"},
		{Year: 2027, StartsOn: "2027-02-01", EndsOn: "2028-01-31", UpdatedBy: "manager@example.com"},
	} {
		if _, err := repository.Save(period); err != nil {
			t.Fatal(err)
		}
	}
	response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/periods/next", url.Values{"source_year": {"2026"}})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/demo/app/settings/annual-statement?year=2027&period=exists#perioden" {
		t.Fatalf("duplicate clone status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	periods := repository.List()
	if len(periods) != 2 || periods[0].StartsOn != "2027-02-01" || periods[0].EndsOn != "2028-01-31" {
		t.Fatalf("existing target was overwritten: %+v", periods)
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAnnualPeriodSave, Limit: 20})); got != 0 {
		t.Fatalf("rejected duplicate produced %d audit events", got)
	}
}

func TestAnnualStatementFollowupDatesClampMonthEndsHAUSV582(t *testing.T) {
	next, ok := annualStatementFollowupPeriod(storepkg.AnnualStatementPeriod{
		Year: 2024, StartsOn: "2024-02-29", EndsOn: "2024-08-31",
	}, "manager@example.com")
	if !ok || next.Year != 2025 || next.StartsOn != "2025-02-28" || next.EndsOn != "2025-08-31" {
		t.Fatalf("clamped follow-up = %+v ok=%t", next, ok)
	}
	if got := annualStatementDeadline("2024-12-31"); got != "30.06.2025" {
		t.Fatalf("calendar-year deadline = %q", got)
	}
}
