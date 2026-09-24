package store

import (
	"math"
	"testing"
	"time"
)

type annualStatementPrepaymentTestStores struct {
	prepayments AnnualStatementPrepaymentStorage
	periods     AnnualStatementPeriodStorage
	units       UnitStorage
}

func TestAnnualStatementPrepaymentStorage(t *testing.T) {
	backends := map[string]func(t *testing.T) annualStatementPrepaymentTestStores{
		"memory": func(t *testing.T) annualStatementPrepaymentTestStores {
			periods := NewMemoryAnnualStatementPeriodStore()
			units, err := NewUnitStore(t.TempDir() + "/units.json")
			if err != nil {
				t.Fatalf("unit store: %v", err)
			}
			return annualStatementPrepaymentTestStores{
				prepayments: NewMemoryAnnualStatementPrepaymentStore(periods, units),
				periods:     periods, units: units,
			}
		},
		"sql": func(t *testing.T) annualStatementPrepaymentTestStores {
			_, lanes := testLanes(t)
			return annualStatementPrepaymentTestStores{
				prepayments: NewSQLAnnualStatementPrepaymentStore(lanes),
				periods:     NewSQLAnnualStatementPeriodStore(lanes), units: NewSQLUnitStore(lanes),
			}
		},
	}
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			stores := build(t)
			demoTenant, otherTenant := testTenantRef("demo"), testTenantRef("other")
			demo, ok := BindAnnualStatementPrepaymentRepository(stores.prepayments, demoTenant)
			if !ok {
				t.Fatal("bind demo prepayments")
			}
			other, _ := BindAnnualStatementPrepaymentRepository(stores.prepayments, otherTenant)
			periods, _ := BindAnnualStatementPeriodRepository(stores.periods, demoTenant)
			units, _ := BindUnitRepository(stores.units, demoTenant)
			if _, err := periods.Save(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"}); err != nil {
				t.Fatal(err)
			}
			if err := units.SetUnits([]Unit{{ID: "top-1", Label: "Top 1"}, {ID: "top-2", Label: "Top 2"}}); err != nil {
				t.Fatal(err)
			}
			for label, item := range map[string]AnnualStatementPrepayment{
				"unknown period": {PeriodYear: 2025, UnitID: "top-1", AmountCents: 1, UpdatedBy: "manager@example.com"},
				"unknown unit":   {PeriodYear: 2026, UnitID: "top-9", AmountCents: 1, UpdatedBy: "manager@example.com"},
				"negative":       {PeriodYear: 2026, UnitID: "top-1", AmountCents: -1, UpdatedBy: "manager@example.com"},
			} {
				if _, _, err := demo.Save(item); err == nil {
					t.Fatalf("%s must fail", label)
				}
			}
			created, previous, err := demo.Save(AnnualStatementPrepayment{PeriodYear: 2026, UnitID: " TOP-1 ", AmountCents: 0, UpdatedAt: now, UpdatedBy: "Manager@Example.com"})
			if err != nil || previous != nil || created.UnitID != "top-1" || created.AmountCents != 0 || created.UpdatedBy != "manager@example.com" {
				t.Fatalf("create = %+v previous=%+v err=%v", created, previous, err)
			}
			updated, previous, err := demo.Save(AnnualStatementPrepayment{PeriodYear: 2026, UnitID: "top-1", AmountCents: 12_340, UpdatedBy: "boss@example.com"})
			if err != nil || previous == nil || previous.AmountCents != 0 || updated.AmountCents != 12_340 {
				t.Fatalf("correction = %+v previous=%+v err=%v", updated, previous, err)
			}
			if got := demo.ListByPeriod(2026); len(got) != 1 || got[0].UnitID != "top-1" || got[0].AmountCents != 12_340 {
				t.Fatalf("period list = %+v", got)
			}
			if _, found := other.Get(2026, "top-1"); found || len(other.ListByPeriod(2026)) != 0 {
				t.Fatal("other tenant saw demo prepayment")
			}
		})
	}
}

func TestAnnualStatementSettlementPreviewSplitsReceiptCentsExactly(t *testing.T) {
	costTypes := []AnnualStatementCostType{
		{Key: "steuer", Allocatable: true, AllocationKey: AllocationKeyPersonen},
		{Key: "ruecklage", Allocatable: false},
	}
	units := []Unit{
		{ID: "top-1", Label: "Top 1", Persons: 1, PersonsRecorded: true},
		{ID: "top-2", Label: "Top 2", Persons: 1, PersonsRecorded: true},
		{ID: "top-3", Label: "Top 3", Persons: 1, PersonsRecorded: true},
	}
	receipts := []AnnualStatementReceipt{
		{CostTypeKey: "steuer", AmountCents: 100},
		{CostTypeKey: "ruecklage", AmountCents: 999_999},
	}
	got, ready := AnnualStatementSettlementPreview(costTypes, receipts, units)
	if !ready || len(got) != 3 || got[0].AllocatedCents != 34 || got[1].AllocatedCents != 33 || got[2].AllocatedCents != 33 {
		t.Fatalf("settlement preview = %+v ready=%t", got, ready)
	}
	units[2].PersonsRecorded = false
	if got, ready := AnnualStatementSettlementPreview(costTypes, receipts, units); ready || len(got) != 0 {
		t.Fatalf("blocked allocation must publish no money: %+v ready=%t", got, ready)
	}
	units[2].PersonsRecorded = true
	receipts = []AnnualStatementReceipt{{CostTypeKey: "steuer", AmountCents: math.MaxInt64}, {CostTypeKey: "steuer", AmountCents: 1}}
	if got, ready := AnnualStatementSettlementPreview(costTypes, receipts, units); ready || len(got) != 0 {
		t.Fatalf("overflowing receipt total must fail closed: %+v ready=%t", got, ready)
	}
	costTypes = append(costTypes, AnnualStatementCostType{Key: "reinigung", Allocatable: true, AllocationKey: AllocationKeyFlaeche})
	units = []Unit{{ID: "top-1", Label: "Top 1", Persons: 1, PersonsRecorded: true, UsableAreaM2Hundredths: 1, UsableAreaRecorded: true}}
	receipts = []AnnualStatementReceipt{{CostTypeKey: "steuer", AmountCents: math.MaxInt64}, {CostTypeKey: "reinigung", AmountCents: math.MaxInt64}}
	if got, ready := AnnualStatementSettlementPreview(costTypes, receipts, units); ready || len(got) != 0 {
		t.Fatalf("overflowing cross-key unit total must fail closed: %+v ready=%t", got, ready)
	}
}
