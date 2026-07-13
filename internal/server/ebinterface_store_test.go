package server

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/markus-barta/hausv-org/internal/integrations"
)

func TestEBInterfaceInvoiceCanBeStoredAsProtectedDocument(t *testing.T) {
	data, err := os.ReadFile("../integrations/testdata/ebinterface-6p0.xml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	result, err := integrations.EBInterfaceAdapter{}.ParseInvoices(context.Background(), integrations.Source{TenantSlug: "jhw22", Filename: "ebinterface-6p0.xml"}, strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("ParseInvoices: %v", err)
	}
	store, err := newDocumentStore("", t.TempDir())
	if err != nil {
		t.Fatalf("document store: %v", err)
	}
	created, err := storeEBInterfaceInvoiceDocument(store, result.Invoices[0], "manager@example.com", data, time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("storeEBInterfaceInvoiceDocument: %v", err)
	}
	if created.Category != documentCategoryBilling || created.Visibility != documentVisibilityManagerOnly || created.ContentType != "application/xml" {
		t.Fatalf("document metadata = %+v", created)
	}
	if !strings.Contains(created.Title, "RE-2026-0006") || !strings.Contains(created.Title, "Hausservice Beispiel") {
		t.Fatalf("document title = %q", created.Title)
	}
	storedPath, ok := store.FilePath(created)
	if !ok {
		t.Fatal("created document missing file path")
	}
	storedData, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("read stored document: %v", err)
	}
	if string(storedData) != string(data) {
		t.Fatal("stored ebInterface XML changed")
	}
}
