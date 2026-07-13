package integrations

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCanonicalIntegrationValidationReportsRecordLevelErrors(t *testing.T) {
	valid := Payment{
		TenantSlug:  "JHW22",
		ExternalID:  "txn-1",
		Amount:      MoneyAmount{Currency: "EUR", Cents: 1234},
		BookingDate: time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		Source:      Source{Format: FormatCAMT053, Version: "2019"},
	}
	if errs := valid.Validate(); len(errs) != 0 {
		t.Fatalf("valid payment errors = %+v", errs)
	}

	invalid := Payment{ExternalID: "txn-2", Amount: MoneyAmount{Currency: "EURO", Cents: -1}}
	errs := invalid.Validate()
	if len(errs) != 3 {
		t.Fatalf("invalid payment errors = %+v, want tenant/amount/date", errs)
	}
	for _, want := range []string{"tenant_slug", "amount", "booking_date"} {
		found := false
		for _, err := range errs {
			if err.Field == want && err.RecordID == "txn-2" && err.RecordType == RecordPayment {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing record-level error for %q in %+v", want, errs)
		}
	}
	report := buildReport(Source{Format: FormatCAMT053}, 1, errs)
	if report.Accepted != 1 || report.Rejected != 3 || !report.HasErrors() {
		t.Fatalf("report = %+v", report)
	}
}

func TestCanonicalIntegrationAdaptersAreFormatNeutral(t *testing.T) {
	var _ PaymentImportAdapter = fakePaymentAdapter{}
	var _ PaymentImportAdapter = CAMT054Adapter{}
	var _ InvoiceImportAdapter = fakeInvoiceAdapter{}
	var _ InvoiceImportAdapter = EBInterfaceAdapter{}
	var _ ExportDataAdapter = fakeExportAdapter{}
	var _ ExportDataAdapter = BMDRawDataAdapter{}

	payments, err := fakePaymentAdapter{}.ParsePayments(context.Background(), Source{Format: FormatCAMT053}, strings.NewReader("fixture"))
	if err != nil {
		t.Fatalf("ParsePayments: %v", err)
	}
	if len(payments.Payments) != 1 || payments.Payments[0].Source.Format != FormatCAMT053 {
		t.Fatalf("payments = %+v", payments)
	}
	if payments.Report.Accepted != 1 || payments.Report.Rejected != 0 {
		t.Fatalf("payment report = %+v", payments.Report)
	}

	var out strings.Builder
	report, err := fakeExportAdapter{}.WriteExportData(context.Background(), &out, []ExportRecord{{
		TenantSlug: "jhw22",
		RecordID:   "raw-1",
		Kind:       "parking-payment-status",
		Amount:     MoneyAmount{Currency: "EUR", Cents: 99},
	}})
	if err != nil {
		t.Fatalf("WriteExportData: %v", err)
	}
	if report.Accepted != 1 || strings.TrimSpace(out.String()) != "raw-1" {
		t.Fatalf("export result report=%+v out=%q", report, out.String())
	}
}

func TestInterfaceDocumentationCoversQAGatesAndAustrianFormats(t *testing.T) {
	for _, path := range []string{
		"../../docs/interface-qa.md",
		"../../docs/camt054-evaluation.md",
		"../../docs/bmd-rawdata-verification.md",
		"../../docs/austrian-interface-decisions.md",
		"../../docs/integration-architecture.md",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing interface documentation %s: %v", path, err)
		}
	}
	qa, err := os.ReadFile("../../docs/interface-qa.md")
	if err != nil {
		t.Fatalf("read interface QA doc: %v", err)
	}
	text := string(qa)
	for _, want := range []string{
		"XSD-Gate",
		"Golden Files",
		"camt.053",
		"camt.054",
		"BMD/RZL",
		"BMD-NTCS",
		"ebInterface",
		"Keine Buchung",
		"Zahlungsreferenz maximal 35 Zeichen",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface QA doc missing %q:\n%s", want, text)
		}
	}
}

func TestCanonicalModelsKeepAccountingOutOfProductScope(t *testing.T) {
	record := ExportRecord{
		TenantSlug: "jhw22",
		RecordID:   "parking-2026-06",
		Kind:       "payment-status",
		Reference:  "HVP-JHW22-202606-A1B2C3",
		Amount:     MoneyAmount{Currency: "EUR", Cents: 13304},
		Fields: map[string]string{
			"status": "paid",
			"scope":  "transparency",
		},
	}
	if errs := record.Validate(); len(errs) != 0 {
		t.Fatalf("export record errors = %+v", errs)
	}
	if _, hasDebit := record.Fields["debit_account"]; hasDebit {
		t.Fatal("canonical export record should not require bookkeeping account fields")
	}
	if _, hasCredit := record.Fields["credit_account"]; hasCredit {
		t.Fatal("canonical export record should not require bookkeeping account fields")
	}
}

type fakePaymentAdapter struct{}

func (fakePaymentAdapter) ParsePayments(_ context.Context, source Source, _ io.Reader) (PaymentImportResult, error) {
	payment := Payment{
		TenantSlug:  "jhw22",
		ExternalID:  "fixture-1",
		Amount:      MoneyAmount{Currency: "EUR", Cents: 100},
		BookingDate: time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		Source:      source,
	}
	return PaymentImportResult{Payments: []Payment{payment}, Report: buildReport(source, 1, payment.Validate())}, nil
}

type fakeInvoiceAdapter struct{}

func (fakeInvoiceAdapter) ParseInvoices(_ context.Context, source Source, _ io.Reader) (InvoiceImportResult, error) {
	invoice := Invoice{
		TenantSlug:    "jhw22",
		ExternalID:    "invoice-1",
		InvoiceNumber: "RE-1",
		Amount:        MoneyAmount{Currency: "EUR", Cents: 100},
		IssueDate:     time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		Source:        source,
	}
	return InvoiceImportResult{Invoices: []Invoice{invoice}, Report: buildReport(source, 1, invoice.Validate())}, nil
}

type fakeExportAdapter struct{}

func (fakeExportAdapter) WriteExportData(_ context.Context, w io.Writer, records []ExportRecord) (Report, error) {
	errors := []RecordError{}
	accepted := 0
	for _, record := range records {
		if errs := record.Validate(); len(errs) > 0 {
			errors = append(errors, errs...)
			continue
		}
		accepted++
		if _, err := io.WriteString(w, record.RecordID+"\n"); err != nil {
			return Report{}, err
		}
	}
	return buildReport(Source{Format: FormatManualCSV}, accepted, errors), nil
}
