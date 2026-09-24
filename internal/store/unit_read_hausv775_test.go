package store

import (
	"bytes"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListCheckedReportsUnreadableUnit(t *testing.T) {
	pool, lanes := testLanes(t)
	demoRef := testTenantRef("demo")
	otherRef := testTenantRef("other")
	store := NewSQLUnitStore(lanes)
	demo, ok := BindUnitRepository(store, demoRef)
	if !ok {
		t.Fatal("bind demo")
	}
	other, ok := BindUnitRepository(store, otherRef)
	if !ok {
		t.Fatal("bind other")
	}
	if err := demo.SetUnits([]Unit{
		{ID: "top-1", Label: "Top 1", UnitType: UnitTypeResidential, MiteigentumsanteilPPM: 400000},
		{ID: "top-2", Label: "Top 2", UnitType: UnitTypeResidential, MiteigentumsanteilPPM: 600000},
	}); err != nil {
		t.Fatal(err)
	}
	if err := other.SetUnits([]Unit{
		{ID: "haus-b", Label: "Haus B", UnitType: UnitTypeResidential, MiteigentumsanteilPPM: 1000000},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(`UPDATE units SET data='invalid-json' WHERE tenant_id=$1 AND id='top-2'`, demoRef.ID); err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	if got := demo.List(); len(got) != 0 {
		t.Fatalf("fail-closed list = %d units", len(got))
	}
	listed, err := demo.ListChecked()
	var dataErr *UnitDataError
	if !errors.As(err, &dataErr) || dataErr.UnitID != "top-2" || dataErr.Op != "decode" || len(listed) != 0 {
		t.Fatalf("ListChecked = %d, %v", len(listed), err)
	}
	if got := demo.UnitCount(); got != 0 {
		t.Fatalf("fail-closed count = %d", got)
	}
	if count, err := demo.UnitCountChecked(); err == nil || count != 0 {
		t.Fatalf("UnitCountChecked = %d, %v", count, err)
	}
	if got := demo.BillableUnitWeight(); got != 0 {
		t.Fatalf("billable weight = %d, want 0", got)
	}
	if !strings.Contains(logs.String(), demoRef.Slug) {
		t.Fatalf("billable read did not log the house: %s", logs.String())
	}
	healthy, err := other.ListChecked()
	if err != nil || len(healthy) != 1 || healthy[0].Label != "Haus B" {
		t.Fatalf("healthy house = %+v, %v", healthy, err)
	}
	if got := other.BillableUnitWeight(); got != UnitBillableFullPPM {
		t.Fatalf("healthy weight = %d", got)
	}
}

func TestAnnualStatementCalculationBlocksOnUnreadableUnits(t *testing.T) {
	pool, lanes := testLanes(t)
	docs := NewSQLDocumentStore(lanes, filepath.Join(t.TempDir(), "docs"))
	sources := AnnualStatementRunSources{
		Periods:     NewSQLAnnualStatementPeriodStore(lanes),
		Units:       NewSQLUnitStore(lanes),
		Documents:   docs,
		Receipts:    NewSQLAnnualStatementReceiptStore(lanes),
		Prepayments: NewSQLAnnualStatementPrepaymentStore(lanes),
		Consumption: NewSQLAnnualStatementConsumptionStore(lanes),
	}
	tenant := testTenantRef("demo")
	seedAnnualRunSources(t, sources, tenant)
	if _, err := pool.Exec(`UPDATE units SET data='invalid-json' WHERE tenant_id=$1 AND id='b'`, tenant.ID); err != nil {
		t.Fatal(err)
	}
	repo, ok := BindAnnualStatementRunRepository(NewSQLAnnualStatementRunStore(lanes, docs), tenant)
	if !ok {
		t.Fatal("bind runs")
	}
	_, _, err := repo.Preview(2025, nil)
	assertUnitDataBlock(t, err)
	_, err = repo.Create(2025, "manager@example.com", time.Now())
	assertUnitDataBlock(t, err)
	runs, err := repo.List(2025)
	if err != nil || len(runs) != 0 {
		t.Fatalf("blocked calculation stored a run: %v %+v", err, runs)
	}
}

func assertUnitDataBlock(t *testing.T, err error) {
	t.Helper()
	var blocked *AnnualStatementRunBlockedError
	if !errors.As(err, &blocked) || len(blocked.Issues) != 1 || blocked.Issues[0].Code != "unit-data" || blocked.Issues[0].UnitID != "b" {
		t.Fatalf("calculation error = %v, want unit-data for b", err)
	}
}
