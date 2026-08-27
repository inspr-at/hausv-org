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

	// Malformed rows reject the whole submit before the store is touched:
	// a blank unit_id, a duplicate unit_id, or parallel arrays that do not
	// line up. None of them writes or records a success audit.
	for name, form := range map[string]url.Values{
		"blank unit id":    {"unit_id": {"", "top-2"}, "usable_area_m2": {"72,5", "80"}, "persons": {"2", "3"}},
		"only blank ids":   {"unit_id": {"  "}, "usable_area_m2": {"72,5"}, "persons": {"2"}},
		"duplicate id":     {"unit_id": {"top-1", "top-1"}, "usable_area_m2": {"72,5", "80"}, "persons": {"2", "3"}},
		"missing persons":  {"unit_id": {"top-1", "top-2"}, "usable_area_m2": {"72,5", "80"}, "persons": {"2"}},
		"missing area":     {"unit_id": {"top-1", "top-2"}, "usable_area_m2": {"72,5"}, "persons": {"2", "3"}},
		"no rows":          {"usable_area_m2": {"72,5"}, "persons": {"2"}},
		"negative persons": {"unit_id": {"top-1", "top-2"}, "usable_area_m2": {"72,5", "80"}, "persons": {"-1", "3"}},
		"extra area":       {"unit_id": {"top-1", "top-2"}, "usable_area_m2": {"72,5", "80", "90"}, "persons": {"2", "3"}},
		"extra persons":    {"unit_id": {"top-1", "top-2"}, "usable_area_m2": {"72,5", "80"}, "persons": {"2", "3", "4"}},
		"negative area":    {"unit_id": {"top-1", "top-2"}, "usable_area_m2": {"-1", "80"}, "persons": {"2", "3"}},
	} {
		response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/allocation-bases", form)
		if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/demo/app/settings/annual-statement?bases=invalid" {
			t.Fatalf("%s: status=%d location=%q, want bases=invalid", name, response.Code, response.Header().Get("Location"))
		}
	}
	// The page submits every unit; a subset or an unknown unit means a stale
	// page and is rejected as a whole as well.
	for name, form := range map[string]url.Values{
		"subset":       {"unit_id": {"top-1"}, "usable_area_m2": {"72,5"}, "persons": {"2"}},
		"unknown unit": {"unit_id": {"top-1", "top-9"}, "usable_area_m2": {"72,5", "80"}, "persons": {"2", "3"}},
	} {
		response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/allocation-bases", form)
		if response.Header().Get("Location") != "/demo/app/settings/annual-statement?bases=stale" {
			t.Fatalf("%s: location=%q, want bases=stale", name, response.Header().Get("Location"))
		}
	}
	for _, item := range testUnitRepository(t, a, "demo").List() {
		if item.UsableAreaM2Hundredths != 0 || item.UsableAreaRecorded || item.Persons != 0 || item.PersonsRecorded {
			t.Fatalf("rejected submits must not write: %+v", item)
		}
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAnnualBasesSave, Limit: 20})); got != 0 {
		t.Fatalf("rejected submits must not record a success audit, got %d events", got)
	}

	// Recording persons for both units unblocks the run; the blank area stays
	// "not recorded" and Miteigentumsanteil is untouched. "0" is a recorded
	// value (Stellplatz), distinct from blank — for persons and for area.
	ok := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/allocation-bases", url.Values{
		"unit_id": {"top-1", "top-2"}, "usable_area_m2": {"72,5", ""}, "persons": {"1", "0"},
	})
	if ok.Code != http.StatusSeeOther || ok.Header().Get("Location") != "/demo/app/settings/annual-statement?bases=saved#verteilerschluessel" {
		t.Fatalf("bases save status=%d location=%q", ok.Code, ok.Header().Get("Location"))
	}
	units := testUnitRepository(t, a, "demo").List()
	if units[0].UsableAreaM2Hundredths != 7_250 || !units[0].UsableAreaRecorded || units[0].Persons != 1 || !units[0].PersonsRecorded || units[0].MiteigentumsanteilPPM != 250_000 ||
		units[1].UsableAreaM2Hundredths != 0 || units[1].UsableAreaRecorded || units[1].Persons != 0 || !units[1].PersonsRecorded {
		t.Fatalf("units after bases save = %+v", units)
	}
	if got := len(a.auditStore.List(auditFilter{TenantSlug: "demo", Action: auditActionAnnualBasesSave, Limit: 20})); got != 1 {
		t.Fatalf("successful save must record exactly one audit event, got %d", got)
	}
	page = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?bases=saved")
	for _, want := range []string{"Lauf möglich", "Verteilerbasis je Einheit gespeichert.", `value="72,50"`, `name="persons" value="0"`, "100,00 %", "0,00 %"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("unblocked page missing %q", want)
		}
	}
	if strings.Contains(page.Body.String(), "Lauf blockiert") || strings.Contains(page.Body.String(), "Ohne Wert:") {
		t.Fatal("run must no longer be blocked once every unit has recorded persons")
	}
	previews := storepkg.AnnualStatementAllocationPreviews(testRepositories(a, "demo").annualStatementCostTypes.List(), units)
	if len(previews) != 2 || previews[1].Key != storepkg.AllocationKeyPersonen || previews[1].Blocked || previews[1].Shares[0].SharePPM != 1_000_000 || previews[1].Shares[1].SharePPM != 0 {
		t.Fatalf("personen preview after save = %+v", previews)
	}

	// A Flächen key blocks on the blank area of Top 2; recording "0" m² maps it
	// with a zero share and unblocks, and the form echoes "0,00" back.
	authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/cost-types", url.Values{
		"key": {"reinigung"}, "name": {"Reinigung"}, "allocation": {"allocatable"}, "allocation_key": {"flaeche"},
	})
	page = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	if !strings.Contains(page.Body.String(), "Ohne Wert:</strong> Top 2") || !strings.Contains(page.Body.String(), "Lauf blockiert") {
		t.Fatal("blank area must block the Flächen key")
	}
	zeroArea := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/allocation-bases", url.Values{
		"unit_id": {"top-1", "top-2"}, "usable_area_m2": {"72,5", "0"}, "persons": {"1", "0"},
	})
	if zeroArea.Header().Get("Location") != "/demo/app/settings/annual-statement?bases=saved#verteilerschluessel" {
		t.Fatalf("zero area save location=%q", zeroArea.Header().Get("Location"))
	}
	units = testUnitRepository(t, a, "demo").List()
	if units[1].UsableAreaM2Hundredths != 0 || !units[1].UsableAreaRecorded {
		t.Fatalf("explicit 0 m² must be recorded: %+v", units[1])
	}
	page = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	for _, want := range []string{"Lauf möglich", `name="usable_area_m2" value="0,00"`, `data-allocation-key="flaeche"`, "0,00 m²"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("zero-area page missing %q", want)
		}
	}
	if strings.Contains(page.Body.String(), "Ohne Wert:") {
		t.Fatal("a recorded 0 m² must not count as unmapped")
	}

	// An ordinary unit edit in the building settings rebuilds the record from
	// a form that knows nothing about the bases; they must survive it.
	edited := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building/units", url.Values{
		"orig_id": {"top-1"}, "id": {"top-1"}, "label": {"Top 1 (Dachgeschoß)"}, "unit_type": {"residential"}, "miteigentumsanteil": {"250000"},
	})
	if edited.Code != http.StatusSeeOther || strings.Contains(edited.Header().Get("Location"), "unit=invalid") {
		t.Fatalf("unit edit status=%d location=%q", edited.Code, edited.Header().Get("Location"))
	}
	units = testUnitRepository(t, a, "demo").List()
	if units[0].Label != "Top 1 (Dachgeschoß)" || units[0].UsableAreaM2Hundredths != 7_250 || !units[0].UsableAreaRecorded || units[0].Persons != 1 || !units[0].PersonsRecorded {
		t.Fatalf("ordinary unit edit wiped the allocation bases: %+v", units[0])
	}
	page = authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	if strings.Contains(page.Body.String(), "Lauf blockiert") {
		t.Fatal("run must stay possible after an unrelated unit edit")
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
		value    int
		recorded bool
		ok       bool
	}{
		"": {0, false, true}, "0": {0, true, true}, "0,00": {0, true, true}, "72,5": {7250, true, true}, "72.50": {7250, true, true}, "100": {10000, true, true}, ",5": {50, true, true}, "99999,99": {9_999_999, true, true},
		"72,505": {0, false, false}, "-1": {0, false, false}, "abc": {0, false, false}, "1e3": {0, false, false}, "100000": {0, false, false},
	} {
		if got, recorded, ok := parseAnnualStatementArea(raw); got != want.value || recorded != want.recorded || ok != want.ok {
			t.Errorf("parseAnnualStatementArea(%q) = %d,%t,%t want %d,%t,%t", raw, got, recorded, ok, want.value, want.recorded, want.ok)
		}
	}
}

func TestAnnualStatementNutzwertTotalNoticeWhenSharesDoNotSumToOneMillion(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "top-1", Label: "Top 1", UnitType: unitTypeResidential, MiteigentumsanteilPPM: 250_000},
		{ID: "top-2", Label: "Top 2", UnitType: unitTypeResidential, MiteigentumsanteilPPM: 550_000},
	}); err != nil {
		t.Fatalf("seed units: %v", err)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	if !strings.Contains(page.Body.String(), "summieren sich auf 800.000 statt 1.000.000") || !strings.Contains(page.Body.String(), "31,25 %") {
		t.Fatal("a Nutzwert total other than 1.000.000 must be surfaced next to the normalised shares")
	}
}
