package integrations

import (
	"context"
	"os"
	"strings"
	"testing"
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
			result, err := EBInterfaceAdapter{}.ParseInvoices(context.Background(), Source{TenantSlug: "demo", Filename: tc.fixture}, f)
			if err != nil {
				t.Fatalf("ParseInvoices: %v", err)
			}
			if result.Report.Accepted != 1 || result.Report.Rejected != 0 || len(result.Invoices) != 1 {
				t.Fatalf("result = %+v", result)
			}
			invoice := result.Invoices[0]
			if invoice.Source.Format != FormatEBInterface || invoice.Source.Version != tc.version {
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

func TestEBInterfaceAdapterReportsInvalidRecords(t *testing.T) {
	result, err := EBInterfaceAdapter{}.ParseInvoices(context.Background(), Source{TenantSlug: "demo"}, strings.NewReader(`<?xml version="1.0"?><Invoice xmlns="http://www.ebinterface.at/schema/6p0/"><InvoiceNumber>BAD</InvoiceNumber></Invoice>`))
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

	unsupported, err := EBInterfaceAdapter{}.ParseInvoices(context.Background(), Source{}, strings.NewReader(`<Invoice xmlns="http://www.ebinterface.at/schema/4p3/"></Invoice>`))
	if err != nil {
		t.Fatalf("ParseInvoices unsupported: %v", err)
	}
	if unsupported.Report.Rejected != 1 || unsupported.Report.Errors[0].Field != "namespace" {
		t.Fatalf("unsupported = %+v", unsupported)
	}
}
