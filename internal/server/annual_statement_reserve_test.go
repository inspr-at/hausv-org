package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestAnnualStatementReserveIsVisibleOnlyForWEG(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	repos := testRepositories(a, "demo")
	if _, err := repos.annualStatementPeriods.Save(store.AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := repos.annualStatementPeriods.SaveLegal(2026, store.AnnualStatementLegalSettings{Regime: "mrg_voll", HeatingConsumptionPercent: 70}); err != nil {
		t.Fatal(err)
	}
	hidden := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?year=2026")
	if hidden.Code != http.StatusOK || strings.Contains(hidden.Body.String(), `id="ruecklage"`) {
		t.Fatalf("MRG page must hide Rücklage: %d", hidden.Code)
	}
	if err := repos.annualStatementPeriods.SaveLegal(2026, store.AnnualStatementLegalSettings{Regime: "weg", HeatingConsumptionPercent: 70}); err != nil {
		t.Fatal(err)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?year=2026")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), `id="ruecklage"`) || !strings.Contains(page.Body.String(), "Buchung hinzufügen") {
		t.Fatal("WEG page must show Rücklage")
	}
	if got := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/reserve", url.Values{"year": {"2026"}, "kind": {"contribution"}, "entry_date": {"2026-01-01"}, "amount": {"1,00"}}).Code; got != http.StatusForbidden {
		t.Fatalf("resident status=%d", got)
	}
	saved := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/reserve", url.Values{
		"year": {"2026"}, "kind": {"contribution"}, "entry_date": {"2026-04-01"}, "amount": {"10,00"}, "note": {"April"},
	})
	if saved.Code != http.StatusSeeOther || !strings.Contains(saved.Header().Get("Location"), "reserve=saved") {
		t.Fatal(saved.Code, saved.Header().Get("Location"))
	}
	if entries := repos.annualStatementReserve.ListByPeriod(2026); len(entries) != 1 || entries[0].AmountCents != 1000 || entries[0].Kind != store.ReserveKindContribution {
		t.Fatalf("%+v", entries)
	}
	var audited bool
	for _, event := range a.auditStore.List(auditFilter{TenantSlug: "demo", Limit: 20}) {
		if event.Action == store.AuditActionAnnualReserveAdd && event.Details["kind"] == "contribution" && event.Details["amount_cents"] == "1000" && event.ActorEmail == "manager@example.com" {
			audited = true
		}
	}
	if !audited {
		t.Fatal("missing reserve audit")
	}
}
