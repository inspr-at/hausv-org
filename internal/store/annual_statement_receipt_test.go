package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type annualStatementReceiptTestStores struct {
	receipts  AnnualStatementReceiptStorage
	periods   AnnualStatementPeriodStorage
	costTypes AnnualStatementCostTypeStorage
	documents DocumentStorage
}

func TestAnnualStatementReceiptStorage(t *testing.T) {
	backends := map[string]func(t *testing.T) annualStatementReceiptTestStores{
		"memory": func(t *testing.T) annualStatementReceiptTestStores {
			periods := NewMemoryAnnualStatementPeriodStore()
			costTypes := NewMemoryAnnualStatementCostTypeStore()
			documents, err := NewDocumentStore("", filepath.Join(t.TempDir(), "documents"))
			if err != nil {
				t.Fatalf("document store: %v", err)
			}
			return annualStatementReceiptTestStores{
				receipts: NewMemoryAnnualStatementReceiptStore(periods, costTypes, documents),
				periods:  periods, costTypes: costTypes, documents: documents,
			}
		},
		"sql": func(t *testing.T) annualStatementReceiptTestStores {
			_, lanes := testLanes(t)
			return annualStatementReceiptTestStores{
				receipts: NewSQLAnnualStatementReceiptStore(lanes),
				periods:  NewSQLAnnualStatementPeriodStore(lanes), costTypes: NewSQLAnnualStatementCostTypeStore(lanes),
				documents: NewSQLDocumentStore(lanes, filepath.Join(t.TempDir(), "documents")),
			}
		},
	}
	now := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	for name, build := range backends {
		t.Run(name, func(t *testing.T) {
			stores := build(t)
			demoTenant, otherTenant := testTenantRef("demo"), testTenantRef("other")
			demo, ok := BindAnnualStatementReceiptRepository(stores.receipts, demoTenant)
			if !ok {
				t.Fatal("bind demo receipts")
			}
			other, ok := BindAnnualStatementReceiptRepository(stores.receipts, otherTenant)
			if !ok {
				t.Fatal("bind other receipts")
			}
			if invalid, ok := BindAnnualStatementReceiptRepository(stores.receipts, TenantRef{}); ok || invalid != nil {
				t.Fatal("invalid tenant must not bind")
			}
			periods, _ := BindAnnualStatementPeriodRepository(stores.periods, demoTenant)
			costTypes, _ := BindAnnualStatementCostTypeRepository(stores.costTypes, demoTenant)
			documents, _ := BindDocumentRepository(stores.documents, demoTenant)
			if _, err := costTypes.Save(AnnualStatementCostType{Key: "wasser", Name: "Wasser", Allocatable: true, AllocationKey: AllocationKeyNutzwert, UpdatedBy: "manager@example.com"}); err != nil {
				t.Fatalf("cost type: %v", err)
			}
			if _, err := periods.SaveWithStructure(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31", UpdatedBy: "manager@example.com"}, costTypes.List(), nil); err != nil {
				t.Fatalf("period: %v", err)
			}
			document, err := documents.CreateGenerated(DocumentRecord{
				Title: "Wasserrechnung", Category: DocumentCategoryBilling, Visibility: DocumentVisibilityManagerOnly, UploadedBy: "manager@example.com",
			}, "wasser.pdf", "application/pdf", []byte("%PDF-1.4 original receipt"), now)
			if err != nil {
				t.Fatalf("document: %v", err)
			}

			base := AnnualStatementReceipt{
				DocumentID: document.ID, PeriodYear: 2026, CostTypeKey: "wasser", AmountCents: 12345,
				InvoiceDate: "2026-04-30", CreatedAt: now, CreatedBy: "Manager@Example.com",
			}
			for label, invalid := range map[string]AnnualStatementReceipt{
				"unknown period":    {DocumentID: document.ID, PeriodYear: 2025, CostTypeKey: "wasser", AmountCents: 1, InvoiceDate: "2026-01-01", CreatedBy: "manager@example.com"},
				"unknown cost type": {DocumentID: document.ID, PeriodYear: 2026, CostTypeKey: "unknown", AmountCents: 1, InvoiceDate: "2026-01-01", CreatedBy: "manager@example.com"},
				"zero amount":       {DocumentID: document.ID, PeriodYear: 2026, CostTypeKey: "wasser", AmountCents: 0, InvoiceDate: "2026-01-01", CreatedBy: "manager@example.com"},
			} {
				if _, err := demo.Create(invalid); err == nil {
					t.Fatalf("%s must fail", label)
				}
			}
			created, err := demo.Create(base)
			if err != nil {
				t.Fatalf("create: %v", err)
			}
			if created.ID == "" || created.CreatedBy != "manager@example.com" || created.UpdatedBy != "manager@example.com" || created.DocumentID != document.ID {
				t.Fatalf("created receipt = %+v", created)
			}
			if _, err := demo.Create(base); err == nil {
				t.Fatal("second receipt for one document must fail")
			}
			if got := demo.ListByPeriod(2026); len(got) != 1 || got[0].ID != created.ID {
				t.Fatalf("period list = %+v", got)
			}
			if _, found := other.Get(created.ID); found || len(other.List()) != 0 {
				t.Fatal("other tenant saw demo receipt")
			}
			if _, err := other.UpdateAmount(created.ID, 777, "other@example.com"); err == nil {
				t.Fatal("other tenant updated demo receipt")
			}
			updated, err := demo.UpdateAmount(created.ID, 54321, "Boss@Example.com")
			if err != nil || updated.AmountCents != 54321 || updated.UpdatedBy != "boss@example.com" || updated.DocumentID != document.ID {
				t.Fatalf("amount update = %+v err=%v", updated, err)
			}
			path, pathOK := documents.FilePath(document)
			if !pathOK {
				t.Fatal("document file path missing before unlink")
			}
			removed, err := demo.Delete(created.ID)
			if err != nil || !removed || len(demo.List()) != 0 {
				t.Fatalf("delete removed=%t err=%v list=%+v", removed, err, demo.List())
			}
			if retained, found := documents.Get(document.ID); !found || retained.ID != document.ID {
				t.Fatalf("original document not retained: %+v found=%t", retained, found)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("original file not retained: %v", err)
			}
		})
	}
}
