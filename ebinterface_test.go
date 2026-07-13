package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestEBInterfaceAdapterParses5And6GoldenFiles(t *testing.T) {
	for _, tc := range []struct {
		name          string
		fixture       string
		version       string
		invoiceNumber string
		issuer        string
		cents         int64
	}{
		{"5.0", "testdata/ebinterface-5p0.xml", "5.0", "RE-2026-0005", "Elektro Beispiel GmbH", 24050},
		{"6.0", "testdata/ebinterface-6p0.xml", "6.0", "RE-2026-0006", "Hausservice Beispiel e.U.", 9990},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(tc.fixture)
			if err != nil {
				t.Fatalf("open fixture: %v", err)
			}
			defer f.Close()
			result, err := ebInterfaceAdapter{}.ParseInvoices(context.Background(), integrationSource{TenantSlug: "jhw22", Filename: tc.fixture}, f)
			if err != nil {
				t.Fatalf("ParseInvoices: %v", err)
			}
			if result.Report.Accepted != 1 || result.Report.Rejected != 0 || len(result.Invoices) != 1 {
				t.Fatalf("result = %+v", result)
			}
			invoice := result.Invoices[0]
			if invoice.Source.Format != integrationFormatEBInterface || invoice.Source.Version != tc.version {
				t.Fatalf("source = %+v", invoice.Source)
			}
			if invoice.InvoiceNumber != tc.invoiceNumber || invoice.IssuerName != tc.issuer || invoice.Amount.Cents != tc.cents || invoice.Amount.Currency != "EUR" {
				t.Fatalf("invoice = %+v", invoice)
			}
			if invoice.IssueDate.IsZero() || invoice.DueDate.IsZero() || invoice.RawDigest == "" {
				t.Fatalf("invoice missing expected fields: %+v", invoice)
			}
		})
	}
}

func TestEBInterfaceInvoiceCanBeStoredAsProtectedDocument(t *testing.T) {
	data, err := os.ReadFile("testdata/ebinterface-6p0.xml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	result, err := ebInterfaceAdapter{}.ParseInvoices(context.Background(), integrationSource{TenantSlug: "jhw22", Filename: "ebinterface-6p0.xml"}, strings.NewReader(string(data)))
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

func TestEBInterfaceAdapterReportsInvalidRecords(t *testing.T) {
	result, err := ebInterfaceAdapter{}.ParseInvoices(context.Background(), integrationSource{TenantSlug: "jhw22"}, strings.NewReader(`<?xml version="1.0"?><Invoice xmlns="http://www.ebinterface.at/schema/6p0/"><InvoiceNumber>BAD</InvoiceNumber></Invoice>`))
	if err != nil {
		t.Fatalf("ParseInvoices: %v", err)
	}
	if len(result.Invoices) != 0 || result.Report.Accepted != 0 || result.Report.Rejected == 0 {
		t.Fatalf("result = %+v", result)
	}
	fields := map[string]bool{}
	for _, item := range result.Report.Errors {
		fields[item.Field] = true
	}
	for _, want := range []string{"amount", "issue_date"} {
		if !fields[want] {
			t.Fatalf("missing field error %q in %+v", want, result.Report.Errors)
		}
	}

	unsupported, err := ebInterfaceAdapter{}.ParseInvoices(context.Background(), integrationSource{}, strings.NewReader(`<Invoice xmlns="http://www.ebinterface.at/schema/4p3/"></Invoice>`))
	if err != nil {
		t.Fatalf("ParseInvoices unsupported: %v", err)
	}
	if unsupported.Report.Rejected != 1 || unsupported.Report.Errors[0].Field != "namespace" {
		t.Fatalf("unsupported = %+v", unsupported)
	}
}
