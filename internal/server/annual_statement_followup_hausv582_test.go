package server

import (
	"net/http"
	"net/url"
	"reflect"
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
		OwnerEmails: []string{"owner@example.com"}, UsableAreaM2Hundredths: 7_500, UsableAreaRecorded: true, Persons: 2, PersonsRecorded: true,
	}}); err != nil {
		t.Fatal(err)
	}
	repositories := testRepositories(a, "demo")
	if err := repositories.annualStatementCostTypes.EnsureDefaults("manager@example.com"); err != nil {
		t.Fatal(err)
	}
	costTypesBefore := repositories.annualStatementCostTypes.List()
	if _, err := repositories.annualStatementPeriods.SaveWithStructure(storepkg.AnnualStatementPeriod{
		Year: 2026, StartsOn: "2026-04-01", EndsOn: "2027-03-31", UpdatedBy: "manager@example.com",
	}, costTypesBefore, repositories.units.List()); err != nil {
		t.Fatal(err)
	}
	structureBeforeGET, ok := repositories.annualStatementPeriods.Structure(2026)
	if !ok {
		t.Fatal("source structure missing before GET")
	}
	if page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?year=2026"); page.Code != http.StatusOK {
		t.Fatalf("source page status=%d", page.Code)
	}
	structureAfterGET, ok := repositories.annualStatementPeriods.Structure(2026)
	if !ok || !reflect.DeepEqual(structureBeforeGET, structureAfterGET) || !reflect.DeepEqual(costTypesBefore, repositories.annualStatementCostTypes.List()) {
		t.Fatalf("GET mutated structure/catalog: before=%+v after=%+v", structureBeforeGET, structureAfterGET)
	}
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
	if response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/cost-types", url.Values{
		"period_year": {"2027"}, "key": {"grundsteuer"}, "name": {"Grundsteuer Zieljahr"},
		"allocation": {"allocatable"}, "allocation_key": {storepkg.AllocationKeyPersonen},
	}); response.Code != http.StatusSeeOther {
		t.Fatalf("target cost type edit status=%d", response.Code)
	}
	if response := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement/allocation-bases", url.Values{
		"period_year": {"2027"}, "unit_id": {"top-1"}, "usable_area_m2": {"90,00"}, "persons": {"4"},
	}); response.Code != http.StatusSeeOther {
		t.Fatalf("target basis edit status=%d", response.Code)
	}
	sourceStructure, sourceOK := repositories.annualStatementPeriods.Structure(2026)
	targetStructure, targetOK := repositories.annualStatementPeriods.Structure(2027)
	if !sourceOK || !targetOK {
		t.Fatalf("period structures missing: source=%t target=%t", sourceOK, targetOK)
	}
	sourceCostTypes, targetCostTypes := map[string]storepkg.AnnualStatementCostType{}, map[string]storepkg.AnnualStatementCostType{}
	for _, item := range sourceStructure.CostTypes {
		sourceCostTypes[item.Key] = item
	}
	for _, item := range targetStructure.CostTypes {
		targetCostTypes[item.Key] = item
	}
	if sourceCostTypes["grundsteuer"].Name == targetCostTypes["grundsteuer"].Name || sourceCostTypes["grundsteuer"].AllocationKey == targetCostTypes["grundsteuer"].AllocationKey {
		t.Fatalf("target cost-type edit leaked into source: source=%+v target=%+v", sourceStructure.CostTypes, targetStructure.CostTypes)
	}
	if len(sourceStructure.UnitBases) != 1 || len(targetStructure.UnitBases) != 1 ||
		sourceStructure.UnitBases[0].UsableAreaM2Hundredths != 7_500 || sourceStructure.UnitBases[0].Persons != 2 ||
		targetStructure.UnitBases[0].UsableAreaM2Hundredths != 9_000 || targetStructure.UnitBases[0].Persons != 4 {
		t.Fatalf("target basis edit leaked into source: source=%+v target=%+v", sourceStructure.UnitBases, targetStructure.UnitBases)
	}
	canonical := repositories.units.List()[0]
	if canonical.OwnerEmails[0] != "owner@example.com" || canonical.UsableAreaM2Hundredths != 7_500 || canonical.Persons != 2 {
		t.Fatalf("period edit mutated canonical unit/party data: %+v", canonical)
	}
	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?year=2027&period=cloned")
	for _, want := range []string{
		"Folgejahr 2028 vorbereiten", "01.04.2028 – 31.03.2029", "Abrechnungsfrist als Erinnerung", "30.09.2028", "Vereinbarungen und Einzelfall bitte prüfen",
		"Vorlage übernommen. Belege, Beträge und Akonto wurden nicht kopiert.",
	} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("follow-up page missing %q", want)
		}
	}
}

func TestAnnualStatementLegacyMemoryPeriodGETUsesPreinstalledDefaultsWithoutWrites(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	repositories := testRepositories(a, "demo")
	if _, err := repositories.annualStatementPeriods.Save(storepkg.AnnualStatementPeriod{
		Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	structureBefore, found := repositories.annualStatementPeriods.Structure(2026)
	if !found || len(structureBefore.CostTypes) != len(storepkg.AnnualStatementDefaultCostTypes("manager@example.com")) {
		t.Fatalf("compatibility write did not preinstall the legacy defaults: found=%t structure=%+v", found, structureBefore)
	}
	if got := repositories.annualStatementCostTypes.List(); len(got) != 0 {
		t.Fatalf("compatibility write mutated the empty global catalog: %+v", got)
	}
	for attempt := 0; attempt < 2; attempt++ {
		page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/annual-statement?year=2026")
		if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Grundsteuer") || !strings.Contains(page.Body.String(), "Müllabfuhr") {
			t.Fatalf("legacy period GET %d: status=%d body=%q", attempt+1, page.Code, page.Body.String())
		}
	}
	structureAfter, found := repositories.annualStatementPeriods.Structure(2026)
	if !found || !reflect.DeepEqual(structureAfter, structureBefore) || len(repositories.annualStatementCostTypes.List()) != 0 {
		t.Fatalf("GET lazily mutated period/global structure: before=%+v after=%+v", structureBefore, structureAfter)
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
