package server

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestAnnualInformationHandlersValidateAndPreserveOtherSettings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	repos := testRepositories(a, "demo")
	if err := repos.units.SetUnits([]store.Unit{{ID: "a", Label: "Top 1"}, {ID: "b", Label: "Top 2"}}); err != nil {
		t.Fatal(err)
	}
	units := repos.units.List()
	if len(units) == 0 {
		t.Fatal("no units")
	}
	_, err := repos.annualStatementPeriods.SaveWithStructure(store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31", UpdatedBy: "manager@example.com"}, []store.AnnualStatementCostType{{Key: "lift", Name: "Lift", Allocatable: true, AllocationKey: store.AllocationKeyAgreed}}, units)
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"year": {"2025"}, "cost_type": {"lift"}}
	for i, u := range units {
		share := 0
		if i == 0 {
			share = 1_000_000
		}
		form.Set("share_"+u.ID, strconv.Itoa(share))
	}
	post := func(email, route string, form url.Values) int {
		return authedFormRequest(t, a, email, "/demo/app/settings/annual-statement/"+route, form).Code
	}
	if got := post("resident@example.com", "agreed-shares", form); got != http.StatusForbidden {
		t.Fatal(got)
	}
	if got := post("manager@example.com", "agreed-shares", form); got != http.StatusSeeOther {
		t.Fatal(got)
	}
	form.Set("share_"+units[0].ID, "999999")
	if got := post("manager@example.com", "agreed-shares", form); got != 400 {
		t.Fatal(got)
	}
	energy := url.Values{"year": {"2025"}, "purchase_row": {"0"}, "purchase_0_cost": {"heizung"}, "purchase_0_supplier": {"Versorger"}, "purchase_0_carrier": {"Gas"}, "purchase_0_quantity": {"1000,25"}, "purchase_0_unit": {"kWh"}, "purchase_0_price": {"0,123456"}, "purchase_0_note": {"31.12.2025"}, "remote_meters": {"yes"}}
	if got := post("resident@example.com", "heating-information", energy); got != 403 {
		t.Fatal(got)
	}
	if got := post("manager@example.com", "heating-information", energy); got != 303 {
		t.Fatal(got)
	}
	s, _ := repos.annualStatementPeriods.Structure(2025)
	if s.Legal.AgreedShares["lift"][units[0].ID] != 1_000_000 || s.Legal.HeatingInformation.Purchases[0].PriceMicros != 123456 || s.Legal.HeatingInformation.Purchases[0].QuantityMicros != 1_000_250_000 {
		t.Fatal(s)
	}
	energy.Set("purchase_0_quantity", "-1")
	if got := post("manager@example.com", "heating-information", energy); got != 400 {
		t.Fatal(got)
	}
	if got := post("manager@example.com", "legal", url.Values{"year": {"2025"}, "regime": {"weg"}, "heating_consumption_percent": {"70"}}); got != 303 {
		t.Fatal(got)
	}
	s, _ = repos.annualStatementPeriods.Structure(2025)
	if s.Legal.HeatingInformation == nil || s.Legal.AgreedShares["lift"][units[0].ID] != 1_000_000 {
		t.Fatal("legal save erased information")
	}
}

func TestAnnualInformationDecimalParsing(t *testing.T) {
	for _, raw := range []string{"", "-0,1", "+1", "1e3", "1.1234567", "9223372036855", "1,2,3", "NaN"} {
		if _, ok := parseAnnualInformationMicros(raw); ok {
			t.Fatal(raw)
		}
	}
	for raw, want := range map[string]int64{"0": 0, "0,000001": 1, "12.34": 12340000, "9223372036854.775807": 9223372036854775807} {
		if got, ok := parseAnnualInformationMicros(raw); !ok || got != want {
			t.Fatal(raw, got)
		}
	}
}
