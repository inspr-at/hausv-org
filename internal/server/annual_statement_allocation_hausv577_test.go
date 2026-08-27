package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	storepkg "github.com/inspr-at/hausv-org/internal/store"
)

func TestAnnualStatementAllocationKeysPreviewAndBlockedRun(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", Label: "Top 1", UnitType: unitTypeResidential, MiteigentumsanteilPPM: 250_000},
		{ID: "top-2", Label: "Top 2", UnitType: unitTypeResidential, MiteigentumsanteilPPM: 750_000},
	}); err != nil {
		t.Fatalf("seed units: %v", err)
	}

	if got := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/allocation-bases", url.Values{
		"unit_id": {"top-1"}, "usable_area_m2": {"50"}, "persons": {"2"},
	}).Code; got != http.StatusForbidden {
		t.Fatalf("resident bases POST status = %d, want 403", got)
	}

	// Starter catalogue: every allocatable cost type defaults to Nutzwert,
	// which reuses the Miteigentumsanteil already on the units → run possible.
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	if page.Code != http.StatusOK {
		t.Fatalf("page status = %d", page.Code)
	}
	for _, want := range []string{"Verteilerschlüssel", "Vorschau der Anteile", "Lauf möglich", `data-allocation-key="nutzwert"`, "25,00 %", "75,00 %", "erfindet keinen", `name="allocation_key"`} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("page missing %q", want)
		}
	}
	if strings.Contains(page.Body.String(), "Lauf blockiert") {
		t.Fatal("fully mapped Nutzwert catalogue must not block the run")
	}

	// Allocatable without a key is rejected at the store; nothing changes.
	invalid := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/cost-types", url.Values{
		"key": {"wasser"}, "name": {"Wasser"}, "allocation": {"allocatable"}, "allocation_key": {"mea"},
	})
	if invalid.Code != http.StatusSeeOther || invalid.Header().Get("Location") != "/demo/app/settings/annual-statement?cost-type=invalid" {
		t.Fatalf("invalid key status=%d location=%q", invalid.Code, invalid.Header().Get("Location"))
	}
	if got := testRepositories(a, "demo").annualStatementCostTypes.List(); len(got) != 5 {
		t.Fatalf("invalid key must not create a cost type: %+v", got)
	}

	// A Personen cost type with no persons recorded blocks the run and names
	// the unmapped units instead of silently splitting by zero.
	saved := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/cost-types", url.Values{
		"key": {"wasser"}, "name": {"Wasser"}, "allocation": {"allocatable"}, "allocation_key": {"personen"},
	})
	if saved.Code != http.StatusSeeOther || saved.Header().Get("Location") != "/demo/app/settings/annual-statement?cost-type=saved" {
		t.Fatalf("personen cost type status=%d location=%q", saved.Code, saved.Header().Get("Location"))
	}
	page = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	for _, want := range []string{"Lauf blockiert", `data-allocation-key="personen"`, "Ohne Wert:</strong> Top 1, Top 2", "Kostenarten: Wasser"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("blocked page missing %q", want)
		}
	}

	// Invalid basis input is rejected as a whole; unknown unit likewise.
	rejected := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/allocation-bases", url.Values{
		"unit_id": {"top-1", "top-2"}, "usable_area_m2": {"72,505", "80"}, "persons": {"2", "3"},
	})
	if rejected.Header().Get("Location") != "/demo/app/settings/annual-statement?bases=invalid" {
		t.Fatalf("three decimals must be rejected, location=%q", rejected.Header().Get("Location"))
	}
	unknown := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/allocation-bases", url.Values{
		"unit_id": {"top-1", "top-9"}, "usable_area_m2": {"72,5", "80"}, "persons": {"2", "3"},
	})
	if unknown.Header().Get("Location") != "/demo/app/settings/annual-statement?bases=unknown-unit" {
		t.Fatalf("unknown unit location=%q", unknown.Header().Get("Location"))
	}
	for _, item := range testUnitRepository(t, a, "demo").List() {
		if item.UsableAreaM2Hundredths != 0 || item.Persons != 0 {
			t.Fatalf("rejected submits must not write: %+v", item)
		}
	}

	// Recording persons for both units unblocks the run; the blank area stays
	// "not recorded" and Miteigentumsanteil is untouched.
	ok := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/allocation-bases", url.Values{
		"unit_id": {"top-1", "top-2"}, "usable_area_m2": {"72,5", ""}, "persons": {"1", "3"},
	})
	if ok.Code != http.StatusSeeOther || ok.Header().Get("Location") != "/demo/app/settings/annual-statement?bases=saved#verteilerschluessel" {
		t.Fatalf("bases save status=%d location=%q", ok.Code, ok.Header().Get("Location"))
	}
	units := testUnitRepository(t, a, "demo").List()
	if units[0].UsableAreaM2Hundredths != 7_250 || units[0].Persons != 1 || units[0].MiteigentumsanteilPPM != 250_000 || units[1].UsableAreaM2Hundredths != 0 || units[1].Persons != 3 {
		t.Fatalf("units after bases save = %+v", units)
	}
	page = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?bases=saved")
	for _, want := range []string{"Lauf möglich", "Verteilerbasis je Einheit gespeichert.", `value="72,50"`} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("unblocked page missing %q", want)
		}
	}
	if strings.Contains(page.Body.String(), "Lauf blockiert") || strings.Contains(page.Body.String(), "Ohne Wert:") {
		t.Fatal("run must no longer be blocked once every unit has persons")
	}
	previews := storepkg.AnnualStatementAllocationPreviews(testRepositories(a, "demo").annualStatementCostTypes.List(), units)
	if len(previews) != 2 || previews[1].Key != storepkg.AllocationKeyPersonen || previews[1].Blocked || previews[1].Shares[0].SharePPM != 250_000 {
		t.Fatalf("personen preview after save = %+v", previews)
	}

	// Verbrauch has no source until HAUSV-578: selectable, but it blocks.
	authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/cost-types", url.Values{
		"key": {"heizung"}, "name": {"Heizung"}, "allocation": {"allocatable"}, "allocation_key": {"verbrauch"},
	})
	page = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	if !strings.Contains(page.Body.String(), "Noch keine Verbrauchswerte.") || !strings.Contains(page.Body.String(), "Lauf blockiert") {
		t.Fatal("verbrauch key must block the run until measured values exist")
	}
}

func TestParseAnnualStatementArea(t *testing.T) {
	for raw, want := range map[string]struct {
		value int
		ok    bool
	}{"": {0, true}, "72,5": {7250, true}, "72.50": {7250, true}, "100": {10000, true}, ",5": {50, true}, "72,505": {0, false}, "-1": {0, false}, "abc": {0, false}, "1e3": {0, false}} {
		if got, ok := parseAnnualStatementArea(raw); got != want.value || ok != want.ok {
			t.Errorf("parseAnnualStatementArea(%q) = %d,%t want %d,%t", raw, got, ok, want.value, want.ok)
		}
	}
}
