package store

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAnnualStatementVATRoundsOncePerRateGroup(t *testing.T) {
	input := AnnualStatementRunInput{
		Period: AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"},
		Structure: AnnualStatementPeriodStructure{
			Legal: AnnualStatementLegalSettings{Regime: "weg", HeatingConsumptionPercent: 70, ShowVAT: true},
			CostTypes: []AnnualStatementCostType{
				{Key: "wasser", Name: "Wasser", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 10},
				{Key: "muell", Name: "Müll", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 10},
				{Key: "heizung", Name: "Heizung", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 20},
				{Key: "garage", Name: "Garage", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 0},
			},
			UnitBases: []AnnualStatementPeriodUnitBasis{
				{UnitID: "top-2", MiteigentumsanteilPPM: 500000},
				{UnitID: "top-10", MiteigentumsanteilPPM: 500000},
			},
		},
		Units: []AnnualStatementRunUnitIdentity{{ID: "top-10", Label: "Top 10"}, {ID: "top-2", Label: "Top 2"}},
		Receipts: []AnnualStatementReceipt{
			{ID: "r1", PeriodYear: 2025, CostTypeKey: "wasser", AmountCents: 100, InvoiceDate: "2025-03-01", DocumentID: "d1"},
			{ID: "r2", PeriodYear: 2025, CostTypeKey: "muell", AmountCents: 100, InvoiceDate: "2025-03-02", DocumentID: "d2"},
			{ID: "r3", PeriodYear: 2025, CostTypeKey: "heizung", AmountCents: 120, InvoiceDate: "2025-03-03", DocumentID: "d3"},
			{ID: "r4", PeriodYear: 2025, CostTypeKey: "garage", AmountCents: 50, InvoiceDate: "2025-03-04", DocumentID: "d4"},
		},
		Prepayments: []AnnualStatementPrepayment{
			{UnitID: "top-10", PeriodYear: 2025, AmountCents: 0},
			{UnitID: "top-2", PeriodYear: 2025, AmountCents: 0},
		},
		Documents: []AnnualStatementRunDocument{{ID: "d1"}, {ID: "d2"}, {ID: "d3"}, {ID: "d4"}},
	}
	result, issues := CalculateAnnualStatementRun(input)
	if len(issues) != 0 {
		t.Fatalf("issues: %+v", issues)
	}
	var gross, net, vat int64
	for _, unit := range result.Units {
		var unitGross, unitNet, unitVAT int64
		for _, line := range unit.Costs {
			if line.NetCents+line.VATCents != line.AmountCents {
				t.Fatalf("line %s/%s net+vat=%d gross=%d", unit.UnitID, line.CostTypeKey, line.NetCents+line.VATCents, line.AmountCents)
			}
			unitGross += line.AmountCents
			unitNet += line.NetCents
			unitVAT += line.VATCents
		}
		if unitNet+unitVAT != unitGross || unitGross != unit.AllocatedCents {
			t.Fatalf("unit %s net %d vat %d gross %d allocated %d", unit.UnitID, unitNet, unitVAT, unitGross, unit.AllocatedCents)
		}
		gross += unitGross
		net += unitNet
		vat += unitVAT
	}
	if net+vat != gross || gross != result.TotalCents {
		t.Fatalf("house net %d vat %d gross %d total %d", net, vat, gross, result.TotalCents)
	}
	// 10 % group is 200 cents gross → VAT 18, split 9/9. 20 % of 120 is 20. 0 % stays gross.
	if len(result.VATGroups) != 3 || result.VATGroups[0].RatePercent != 0 || result.VATGroups[0].VATCents != 0 || result.VATGroups[0].GrossCents != 50 {
		t.Fatalf("zero rate group: %+v", result.VATGroups)
	}
	if result.VATGroups[1].RatePercent != 10 || result.VATGroups[1].VATCents != 18 || result.VATGroups[1].NetCents != 182 || result.VATGroups[1].GrossCents != 200 {
		t.Fatalf("10%% group: %+v", result.VATGroups[1])
	}
	if result.VATGroups[2].RatePercent != 20 || result.VATGroups[2].VATCents != 20 || result.VATGroups[2].NetCents != 100 || result.VATGroups[2].GrossCents != 120 {
		t.Fatalf("20%% group: %+v", result.VATGroups[2])
	}
	for _, unit := range result.Units {
		for _, group := range unit.VAT {
			if group.RatePercent == 10 && group.VATCents != 9 {
				t.Fatalf("unit %s 10%% vat %d, want 9", unit.UnitID, group.VATCents)
			}
		}
	}
	off := input
	off.Structure.Legal.ShowVAT = false
	plain, issues := CalculateAnnualStatementRun(off)
	if len(issues) != 0 || plain.VATGroups != nil || plain.TotalCents != result.TotalCents {
		t.Fatalf("off changed the gross result: %+v %+v", plain, issues)
	}
	for i := range plain.Units {
		if plain.Units[i].AllocatedCents != result.Units[i].AllocatedCents || plain.Units[i].VAT != nil {
			t.Fatalf("off unit changed: %+v", plain.Units[i])
		}
		for j := range plain.Units[i].Costs {
			if plain.Units[i].Costs[j].AmountCents != result.Units[i].Costs[j].AmountCents || plain.Units[i].Costs[j].VATCents != 0 {
				t.Fatalf("off line changed: %+v", plain.Units[i].Costs[j])
			}
		}
	}
	replay, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: AnnualStatementCalculationVersion, Input: input})
	if len(issues) != 0 || replay.VATGroups != nil || !reflect.DeepEqual(replay, plain) {
		t.Fatalf("version 2 replay applied VAT: %+v %+v", replay, issues)
	}
	again, issues := ReplayAnnualStatementRun(AnnualStatementRun{CalculationVersion: AnnualStatementCalculationVersionVAT, Input: input})
	if len(issues) != 0 || !reflect.DeepEqual(again, result) {
		t.Fatalf("version 3 replay drifted: %+v", again)
	}
}

func TestAnnualStatementVATRejectsUnknownRate(t *testing.T) {
	input := AnnualStatementRunInput{
		Period:    AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"},
		Structure: AnnualStatementPeriodStructure{Legal: AnnualStatementLegalSettings{Regime: "weg", HeatingConsumptionPercent: 70, ShowVAT: true}, CostTypes: []AnnualStatementCostType{{Key: "wasser", Name: "Wasser", Allocatable: true, AllocationKey: AllocationKeyNutzwert, VATRatePercent: 13}}, UnitBases: []AnnualStatementPeriodUnitBasis{{UnitID: "a", MiteigentumsanteilPPM: 1000000}}},
		Units:     []AnnualStatementRunUnitIdentity{{ID: "a"}},
		Receipts:  []AnnualStatementReceipt{{ID: "r", PeriodYear: 2025, CostTypeKey: "wasser", AmountCents: 10, InvoiceDate: "2025-01-02", DocumentID: "d"}},
		Prepayments: []AnnualStatementPrepayment{
			{UnitID: "a", PeriodYear: 2025},
		},
		Documents: []AnnualStatementRunDocument{{ID: "d"}},
	}
	if _, issues := CalculateAnnualStatementRun(input); len(issues) != 1 || issues[0].Code != "vat-rate" {
		t.Fatalf("issues: %+v", issues)
	}
}

func TestAnnualStatementVATPostgresRoundTrip(t *testing.T) {
	if strings.TrimSpace(os.Getenv("HAUSV_TEST_POSTGRES_DSN")) == "" {
		t.Skip("set HAUSV_TEST_POSTGRES_DSN to check the PostgreSQL VAT rate")
	}
	t.Setenv("HAUSV_STORE_TEST_POSTGRES", "1")
	_, lanes := testLanes(t)
	docs := NewSQLDocumentStore(lanes, filepath.Join(t.TempDir(), "docs"))
	sources := AnnualStatementRunSources{Periods: NewSQLAnnualStatementPeriodStore(lanes), Units: NewSQLUnitStore(lanes), Documents: docs, Receipts: NewSQLAnnualStatementReceiptStore(lanes), Prepayments: NewSQLAnnualStatementPrepaymentStore(lanes), Consumption: NewSQLAnnualStatementConsumptionStore(lanes)}
	tenant := testTenantRef("demo")
	seedAnnualRunSources(t, sources, tenant)
	periods, _ := BindAnnualStatementPeriodRepository(sources.Periods, tenant)
	structure, ok := periods.Structure(2025)
	if !ok {
		t.Fatal("missing structure")
	}
	for _, cost := range structure.CostTypes {
		if cost.Key == "tax" && cost.VATRatePercent != 10 {
			t.Fatalf("stored rate %+v", cost)
		}
	}
	legal := structure.Legal
	legal.ShowVAT = true
	if err := periods.SaveLegal(2025, legal); err != nil {
		t.Fatal(err)
	}
	for _, cost := range structure.CostTypes {
		if cost.Key != "tax" {
			continue
		}
		cost.VATRatePercent = 0
		cost.UpdatedBy = "manager@example.com"
		if _, err := periods.SaveStructureCostType(2025, cost); err != nil {
			t.Fatal(err)
		}
	}
	repo, _ := BindAnnualStatementRunRepository(NewSQLAnnualStatementRunStore(lanes, docs), tenant)
	run, err := repo.Create(2025, "manager@example.com", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if run.CalculationVersion != AnnualStatementCalculationVersionVAT {
		t.Fatalf("version %d", run.CalculationVersion)
	}
	var net, vat, gross int64
	for _, group := range run.Result.VATGroups {
		net += group.NetCents
		vat += group.VATCents
		gross += group.GrossCents
	}
	if net+vat != gross || gross != run.Result.TotalCents {
		t.Fatalf("groups %+v total %d", run.Result.VATGroups, run.Result.TotalCents)
	}
	replay, issues := ReplayAnnualStatementRun(run)
	if len(issues) != 0 || !reflect.DeepEqual(replay, run.Result) {
		t.Fatalf("replay %+v %v", replay, issues)
	}
	again, ok, err := repo.Get(run.ID)
	if err != nil || !ok || again.CalculationVersion != run.CalculationVersion || !reflect.DeepEqual(again.Result, run.Result) {
		t.Fatalf("stored run %+v %v", again, err)
	}
}

func TestDefaultAnnualStatementVATPercent(t *testing.T) {
	if DefaultAnnualStatementVATPercent("heizung") != 20 || DefaultAnnualStatementVATPercent("warmwasser") != 20 || DefaultAnnualStatementVATPercent("wasser") != 10 {
		t.Fatal("kind defaults")
	}
}
