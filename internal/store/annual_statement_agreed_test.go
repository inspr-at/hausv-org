package store

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestAgreedSharesBlockInvalidVectorsAndConserve(t *testing.T) {
	in := annualRunFixture()
	for i := range in.Structure.CostTypes {
		in.Structure.CostTypes[i].AllocationKey = AllocationKeyAgreed
	}
	in.Structure.Legal = DefaultAnnualStatementLegalSettings()
	in.Structure.Legal.AgreedShares = map[string]map[string]int{}
	for _, cost := range in.Structure.CostTypes {
		in.Structure.Legal.AgreedShares[cost.Key] = map[string]int{"a": 0, "b": 1_000_000}
	}
	result, issues := CalculateAnnualStatementRun(in)
	if len(issues) > 0 || result.Units[0].AllocatedCents != 0 || result.Units[1].AllocatedCents != result.TotalCents {
		t.Fatal(result, issues)
	}
	key := in.Structure.CostTypes[0].Key
	for name, shares := range map[string]map[string]int{
		"missing zero": {"b": 1_000_000}, "extra": {"a": 0, "b": 1_000_000, "c": 0}, "low": {"a": 0, "b": 999999}, "high": {"a": 1, "b": 1_000_000}, "negative": {"a": -1, "b": 1_000_001}, "overflow": {"a": math.MaxInt, "b": math.MaxInt},
	} {
		t.Run(name, func(t *testing.T) {
			in.Structure.Legal.AgreedShares[key] = shares
			r, issues := CalculateAnnualStatementRun(in)
			if len(issues) == 0 || len(r.Units) > 0 {
				t.Fatal("invalid vector calculated", r)
			}
		})
	}
	in.Structure.Legal.AgreedShares[key] = map[string]int{"a": 500000, "b": 500000}
	result, issues = CalculateAnnualStatementRun(in)
	if len(issues) > 0 {
		t.Fatal(issues)
	}
	// Other cost types keep their own vector; no pooling by key.
	for _, cost := range result.Units[0].Costs {
		if cost.CostTypeKey != key && cost.AmountCents != 0 {
			t.Fatal("vectors mixed", cost)
		}
	}
	run, err := newAnnualStatementRun(in, result, 1, "manager@example.com", time.Now())
	if err != nil || run.CalculationVersion != 5 {
		t.Fatal(run, err)
	}
	replay, issues := ReplayAnnualStatementRun(run)
	if len(issues) > 0 || !reflect.DeepEqual(replay, result) {
		t.Fatal("replay", issues)
	}
	for _, version := range []int{1, 2, 3, 4} {
		run.CalculationVersion = version
		if _, issues := ReplayAnnualStatementRun(run); len(issues) == 0 {
			t.Fatal("old version accepted new key")
		}
	}
}

func TestAgreedAndHeatingInformationPersistence(t *testing.T) {
	_, lanes := testLanes(t)
	for _, storage := range []AnnualStatementPeriodStorage{NewMemoryAnnualStatementPeriodStore(), NewSQLAnnualStatementPeriodStore(lanes)} {
		repo, _ := BindAnnualStatementPeriodRepository(storage, testTenantRef("demo"))
		_, err := repo.Save(AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31", UpdatedBy: "manager@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		legal := DefaultAnnualStatementLegalSettings()
		legal.AgreedShares = map[string]map[string]int{"lift": {"a": 0, "b": 1_000_000}}
		legal.HeatingInformation = &AnnualStatementHeatingInformation{Purchases: []AnnualStatementEnergyPurchase{{CostTypeKey: "heizung", Supplier: "Versorger", Carrier: "Gas", Unit: "kWh", QuantityMicros: 1000000, PriceMicros: 123456, PriceNote: "31.12.2025"}}, RemoteMeters: "yes"}
		if err := repo.SaveLegal(2025, legal); err != nil {
			t.Fatal(err)
		}
		stored, _ := repo.Structure(2025)
		if !reflect.DeepEqual(stored.Legal, legal) {
			t.Fatal(stored)
		}
		stored.Legal.AgreedShares["lift"]["a"] = 1
		stored.Legal.HeatingInformation.Purchases[0].Supplier = "changed"
		if got, _ := repo.Structure(2025); !reflect.DeepEqual(got.Legal, legal) {
			t.Fatal("alias")
		}
		_, _, err = repo.CloneStructure(2025, AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		next, _ := repo.Structure(2026)
		if next.Legal.HeatingInformation != nil || !reflect.DeepEqual(next.Legal.AgreedShares, legal.AgreedShares) {
			t.Fatal("followup carries consumption or loses shares")
		}
		other, _ := BindAnnualStatementPeriodRepository(storage, testTenantRef("other"))
		if _, found := other.Structure(2025); found {
			t.Fatal("foreign read")
		}
		if err := other.SaveLegal(2025, legal); err == nil {
			t.Fatal("foreign write")
		}
		legal.AgreedShares["lift"]["b"] = 999999
		if err := repo.SaveLegal(2025, legal); err == nil {
			t.Fatal("invalid sum persisted")
		}
	}
}

func TestPreviousHeatingSnapshotStorage(t *testing.T) {
	for _, kind := range []string{"memory", "sql"} {
		t.Run(kind, func(t *testing.T) {
			var sources AnnualStatementRunSources
			var storage AnnualStatementRunStorage
			if kind == "memory" {
				periods := NewMemoryAnnualStatementPeriodStore()
				units, _ := NewUnitStore("")
				docs, _ := NewDocumentStore("", filepath.Join(t.TempDir(), "docs"))
				sources = AnnualStatementRunSources{Periods: periods, Units: units, Documents: docs, Receipts: NewMemoryAnnualStatementReceiptStore(periods, NewMemoryAnnualStatementCostTypeStore(), docs), Prepayments: NewMemoryAnnualStatementPrepaymentStore(periods, units), Consumption: NewMemoryAnnualStatementConsumptionStore()}
				storage = NewMemoryAnnualStatementRunStore(sources)
			} else {
				_, lanes := testLanes(t)
				docs := NewSQLDocumentStore(lanes, filepath.Join(t.TempDir(), "docs"))
				sources = AnnualStatementRunSources{Periods: NewSQLAnnualStatementPeriodStore(lanes), Units: NewSQLUnitStore(lanes), Documents: docs, Receipts: NewSQLAnnualStatementReceiptStore(lanes), Prepayments: NewSQLAnnualStatementPrepaymentStore(lanes), Consumption: NewSQLAnnualStatementConsumptionStore(lanes)}
				storage = NewSQLAnnualStatementRunStore(lanes, docs)
			}
			tenant := testTenantRef("demo")
			heating := heatingFixture()
			seedAnnualRunInput(t, sources, tenant, heating)
			periods, _ := BindAnnualStatementPeriodRepository(sources.Periods, tenant)
			if err := periods.SaveLegal(2025, heating.Structure.Legal); err != nil {
				t.Fatal(err)
			}
			consumption, _ := BindAnnualStatementConsumptionRepository(sources.Consumption, tenant)
			start := time.Date(2025, 1, 1, 0, 0, 0, 0, mustViennaLocation(t))
			for _, value := range heating.Consumption["heizung"].Units {
				appendConsumption(t, consumption, consumptionEvidence(value.UnitID, "heizung", "sensor."+value.UnitID, start, 0, "kWh"))
				appendConsumption(t, consumption, consumptionEvidence(value.UnitID, "heizung", "sensor."+value.UnitID, start.AddDate(1, 0, 0), value.ValueMicros, "kWh"))
			}
			repo, _ := BindAnnualStatementRunRepository(storage, tenant)
			prior, err := repo.Create(2025, "manager@example.com", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			in := annualRunFixture()
			in.Period = AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31"}
			for i := range in.Receipts {
				in.Receipts[i].PeriodYear = 2026
				in.Receipts[i].ID = ""
			}
			for i := range in.Prepayments {
				in.Prepayments[i].PeriodYear = 2026
			}
			seedAnnualRunInput(t, sources, tenant, in)
			current, err := repo.Create(2026, "manager@example.com", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if current.Input.PreviousHeating == nil || current.Input.PreviousHeating.RunID != prior.ID {
				t.Fatal("previous run not snapshotted")
			}
			if current.Input.PreviousHeating.Consumption["heizung"].Units[0].ValueMicros != 1_000_000 {
				t.Fatal("previous consumption lost")
			}
			if _, err := repo.Create(2025, "manager@example.com", time.Now()); err != nil {
				t.Fatal(err)
			}
			loaded, _, err := repo.Get(current.ID)
			if err != nil || !reflect.DeepEqual(current, loaded) {
				t.Fatal("later revision rewrote prior comparison", err)
			}
		})
	}
}

func TestPreviousHeatingSnapshotRequiresMatchingDatesAndOwnsValues(t *testing.T) {
	in := heatingFixture()
	prior := AnnualStatementRun{ID: "old", Revision: 1, Input: in}
	period := AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31"}
	got := previousHeatingSnapshot(period, []AnnualStatementRun{prior})
	if got == nil {
		t.Fatal("missing")
	}
	got.Consumption["heizung"].Units[0].ValueMicros = 123
	if prior.Input.Consumption["heizung"].Units[0].ValueMicros == 123 {
		t.Fatal("alias")
	}
	period.StartsOn = "2026-02-01"
	if previousHeatingSnapshot(period, []AnnualStatementRun{prior}) != nil {
		t.Fatal("incomparable period used")
	}
}
