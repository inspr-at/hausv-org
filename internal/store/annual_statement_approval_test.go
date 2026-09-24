package store

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestAnnualStatementRunApprovalTenantIsolation(t *testing.T) {
	for _, kind := range []string{"memory", "sql"} {
		t.Run(kind, func(t *testing.T) {
			var sources AnnualStatementRunSources
			var storage AnnualStatementRunStorage
			if kind == "sql" {
				_, lanes := testLanes(t)
				docs := NewSQLDocumentStore(lanes, filepath.Join(t.TempDir(), "docs"))
				sources = AnnualStatementRunSources{Periods: NewSQLAnnualStatementPeriodStore(lanes), Units: NewSQLUnitStore(lanes), Documents: docs, Receipts: NewSQLAnnualStatementReceiptStore(lanes), Prepayments: NewSQLAnnualStatementPrepaymentStore(lanes), Consumption: NewSQLAnnualStatementConsumptionStore(lanes)}
				storage = NewSQLAnnualStatementRunStore(lanes, docs)
			} else {
				periods := NewMemoryAnnualStatementPeriodStore()
				units, err := NewUnitStore("")
				if err != nil {
					t.Fatal(err)
				}
				docs, err := NewDocumentStore("", filepath.Join(t.TempDir(), "docs"))
				if err != nil {
					t.Fatal(err)
				}
				sources = AnnualStatementRunSources{Periods: periods, Units: units, Documents: docs, Receipts: NewMemoryAnnualStatementReceiptStore(periods, NewMemoryAnnualStatementCostTypeStore(), docs), Prepayments: NewMemoryAnnualStatementPrepaymentStore(periods, units), Consumption: NewMemoryAnnualStatementConsumptionStore()}
				storage = NewMemoryAnnualStatementRunStore(sources)
			}
			tenant := testTenantRef("demo")
			seedAnnualRunSources(t, sources, tenant)
			repo, _ := BindAnnualStatementRunRepository(storage, tenant)
			foreign, _ := BindAnnualStatementRunRepository(storage, testTenantRef("other"))
			run, err := repo.Create(2025, "manager@example.com", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := foreign.Approve(run.ID, "manager@example.com", RoleManager, time.Now()); err == nil {
				t.Fatal("foreign approval")
			}
			final, changed, err := repo.Approve(run.ID, "manager@example.com", RoleManager, time.Now())
			if err != nil || !changed || final.Approval == nil {
				t.Fatal(final, changed, err)
			}
			if !reflect.DeepEqual(final.Input, run.Input) || final.InputHash != run.InputHash {
				t.Fatal("snapshot changed")
			}
			final.Approval.Role = "mutated"
			saved, _, err := repo.Get(run.ID)
			if err != nil || saved.Approval.Role != RoleManager {
				t.Fatal(saved, err)
			}
			again, changed, err := repo.Approve(run.ID, "admin@example.com", RoleAdmin, time.Now())
			if err != nil || changed || !reflect.DeepEqual(again, saved) {
				t.Fatal(again, changed, err)
			}
			if _, found, err := foreign.Get(run.ID); err != nil || found {
				t.Fatal("foreign read", err)
			}
		})
	}
}
