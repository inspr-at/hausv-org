package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAnnualStatementManagerFlowAndRoleGate(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{{
		ID: "top-1", Label: "Top 1", UnitType: unitTypeResidential,
		OwnerEmails: []string{"old-owner@example.com"}, RenterEmails: []string{"old-renter@example.com"},
	}}); err != nil {
		t.Fatalf("seed unit: %v", err)
	}

	if got := authedRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement").Code; got != http.StatusForbidden {
		t.Fatalf("resident GET status = %d, want 403", got)
	}
	if got := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/periods", url.Values{
		"year": {"2026"}, "starts_on": {"2026-01-01"}, "ends_on": {"2026-12-31"},
	}).Code; got != http.StatusForbidden {
		t.Fatalf("resident POST status = %d, want 403", got)
	}
	if got := authedFormRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/cost-types", url.Values{
		"key": {"wasser"}, "name": {"Wasser"}, "allocation": {"allocatable"},
	}).Code; got != http.StatusForbidden {
		t.Fatalf("resident cost type POST status = %d, want 403", got)
	}
	if got := authedMultipartFileRequest(t, a, "resident@example.com", "/demo/app/settings/annual-statement/parties/import", nil, "parties_file", "parteien.csv", []byte(
		"Einheit,Rolle,E-Mail\nTop 1,Wohnungseigentümer,resident@example.com\n",
	)).Code; got != http.StatusForbidden {
		t.Fatalf("resident import status = %d, want 403", got)
	}

	hub := authedRequest(t, a, "manager@example.com", "/demo/app/settings")
	if !strings.Contains(hub.Body.String(), `href="/demo/app/settings/annual-statement"`) {
		t.Fatal("manager settings hub does not link annual statement basics")
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement")
	if page.Code != http.StatusOK {
		t.Fatalf("manager page status = %d", page.Code)
	}
	for _, want := range []string{"Jahresabrechnung vorbereiten", "Liegenschaft", "Kostenartenkatalog", "Grundsteuer", "Müllabfuhr", "Hausbetreuung", "Gebäudeversicherung", "Gartenpflege", "Umlagefähig", "Nicht umlagefähig", "Einheiten und Parteien", "Wohnungseigentümer", "Mietverhältnis", "Abrechnungsjahr", "WEG Portal", "Top 1", "old-owner@example.com"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("annual statement page missing %q", want)
		}
	}
	if got := testRepositories(a, "demo").annualStatementCostTypes.List(); len(got) != 5 {
		t.Fatalf("starter cost type catalogue = %+v, want 5 entries", got)
	}

	costTypeSaved := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/cost-types", url.Values{
		"key": {"wasser_abwasser"}, "name": {"Wasser und Abwasser"}, "allocation": {"not_allocatable"},
	})
	if costTypeSaved.Code != http.StatusSeeOther || costTypeSaved.Header().Get("Location") != "/demo/app/settings/annual-statement?cost-type=saved" {
		t.Fatalf("cost type save status=%d location=%q", costTypeSaved.Code, costTypeSaved.Header().Get("Location"))
	}
	costTypes := testRepositories(a, "demo").annualStatementCostTypes.List()
	if len(costTypes) != 6 {
		t.Fatalf("stored cost types = %+v, want 6 entries", costTypes)
	}
	foundCustom := false
	for _, costType := range costTypes {
		if costType.Key == "wasser_abwasser" {
			foundCustom = costType.Name == "Wasser und Abwasser" && !costType.Allocatable
		}
	}
	if !foundCustom {
		t.Fatalf("stored cost types missing non-allocatable custom entry: %+v", costTypes)
	}

	saved := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/periods", url.Values{
		"year": {"2026"}, "starts_on": {"2026-04-01"}, "ends_on": {"2027-03-31"},
	})
	if saved.Code != http.StatusSeeOther || saved.Header().Get("Location") != "/demo/app/settings/annual-statement?year=2026&period=saved" {
		t.Fatalf("period save status=%d location=%q", saved.Code, saved.Header().Get("Location"))
	}
	periods := testRepositories(a, "demo").annualStatementPeriods.List()
	if len(periods) != 1 || periods[0].Year != 2026 || periods[0].StartsOn != "2026-04-01" || periods[0].EndsOn != "2027-03-31" {
		t.Fatalf("stored periods = %+v", periods)
	}

	imported := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/parties/import", nil, "parties_file", "parteien.csv", []byte(
		"Einheit;Rolle;E-Mail\nTop 1;Wohnungseigentümer;owner@example.com\nTop 1;Mietverhältnis;renter@example.com\n",
	))
	if imported.Code != http.StatusSeeOther || imported.Header().Get("Location") != "/demo/app/settings/annual-statement?import=saved&count=2" {
		t.Fatalf("party import status=%d location=%q", imported.Code, imported.Header().Get("Location"))
	}
	units := testUnitRepository(t, a, "demo").List()
	if len(units) != 1 || strings.Join(units[0].OwnerEmails, ",") != "owner@example.com" || strings.Join(units[0].RenterEmails, ",") != "renter@example.com" {
		t.Fatalf("units after party import = %+v", units)
	}
}

func TestAnnualStatementPartyImportRejectsUnknownUnitWithoutInventingOne(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{{ID: "top-1", Label: "Top 1", OwnerEmails: []string{"owner@example.com"}}}); err != nil {
		t.Fatalf("seed unit: %v", err)
	}

	response := authedMultipartFileRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/parties/import", nil, "parties_file", "parteien.csv", []byte(
		"Einheit,Rolle,E-Mail\nTop 1,Wohnungseigentümer,new@example.com\nTop 99,Mietverhältnis,renter@example.com\n",
	))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/demo/app/settings/annual-statement?import=unknown-unit" {
		t.Fatalf("unknown-unit import status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	units := testUnitRepository(t, a, "demo").List()
	if len(units) != 1 || units[0].ID != "top-1" || strings.Join(units[0].OwnerEmails, ",") != "owner@example.com" || len(units[0].RenterEmails) != 0 {
		t.Fatalf("rejected import changed or invented units: %+v", units)
	}
}
