package main

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCanonicalIntegrationValidationReportsRecordLevelErrors(t *testing.T) {
	valid := canonicalPayment{
		TenantSlug:  "JHW22",
		ExternalID:  "txn-1",
		Amount:      moneyAmount{Currency: "EUR", Cents: 1234},
		BookingDate: time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		Source:      integrationSource{Format: integrationFormatCAMT053, Version: "2019"},
	}
	if errs := valid.Validate(); len(errs) != 0 {
		t.Fatalf("valid payment errors = %+v", errs)
	}

	invalid := canonicalPayment{ExternalID: "txn-2", Amount: moneyAmount{Currency: "EURO", Cents: -1}}
	errs := invalid.Validate()
	if len(errs) != 3 {
		t.Fatalf("invalid payment errors = %+v, want tenant/amount/date", errs)
	}
	for _, want := range []string{"tenant_slug", "amount", "booking_date"} {
		found := false
		for _, err := range errs {
			if err.Field == want && err.RecordID == "txn-2" && err.RecordType == integrationRecordPayment {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing record-level error for %q in %+v", want, errs)
		}
	}
	report := buildIntegrationReport(integrationSource{Format: integrationFormatCAMT053}, 1, errs)
	if report.Accepted != 1 || report.Rejected != 3 || !report.HasErrors() {
		t.Fatalf("report = %+v", report)
	}
}

func TestCanonicalIntegrationAdaptersAreFormatNeutral(t *testing.T) {
	var _ paymentImportAdapter = fakePaymentAdapter{}
	var _ paymentImportAdapter = camt054Adapter{}
	var _ invoiceImportAdapter = fakeInvoiceAdapter{}
	var _ invoiceImportAdapter = ebInterfaceAdapter{}
	var _ exportDataAdapter = fakeExportAdapter{}
	var _ exportDataAdapter = bmdRawDataAdapter{}

	payments, err := fakePaymentAdapter{}.ParsePayments(context.Background(), integrationSource{Format: integrationFormatCAMT053}, strings.NewReader("fixture"))
	if err != nil {
		t.Fatalf("ParsePayments: %v", err)
	}
	if len(payments.Payments) != 1 || payments.Payments[0].Source.Format != integrationFormatCAMT053 {
		t.Fatalf("payments = %+v", payments)
	}
	if payments.Report.Accepted != 1 || payments.Report.Rejected != 0 {
		t.Fatalf("payment report = %+v", payments.Report)
	}

	var out strings.Builder
	report, err := fakeExportAdapter{}.WriteExportData(context.Background(), &out, []canonicalExportRecord{{
		TenantSlug: "jhw22",
		RecordID:   "raw-1",
		Kind:       "parking-payment-status",
		Amount:     moneyAmount{Currency: "EUR", Cents: 99},
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
		"docs/interface-qa.md",
		"docs/camt054-evaluation.md",
		"docs/bmd-rawdata-verification.md",
		"docs/austrian-interface-decisions.md",
		"docs/integration-architecture.md",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing interface documentation %s: %v", path, err)
		}
	}
	qa, err := os.ReadFile("docs/interface-qa.md")
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
	record := canonicalExportRecord{
		TenantSlug: "jhw22",
		RecordID:   "parking-2026-06",
		Kind:       "payment-status",
		Reference:  "HVP-JHW22-202606-A1B2C3",
		Amount:     moneyAmount{Currency: "EUR", Cents: 13304},
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

func (fakePaymentAdapter) ParsePayments(_ context.Context, source integrationSource, _ io.Reader) (paymentImportResult, error) {
	payment := canonicalPayment{
		TenantSlug:  "jhw22",
		ExternalID:  "fixture-1",
		Amount:      moneyAmount{Currency: "EUR", Cents: 100},
		BookingDate: time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		Source:      source,
	}
	return paymentImportResult{Payments: []canonicalPayment{payment}, Report: buildIntegrationReport(source, 1, payment.Validate())}, nil
}

type fakeInvoiceAdapter struct{}

func (fakeInvoiceAdapter) ParseInvoices(_ context.Context, source integrationSource, _ io.Reader) (invoiceImportResult, error) {
	invoice := canonicalInvoice{
		TenantSlug:    "jhw22",
		ExternalID:    "invoice-1",
		InvoiceNumber: "RE-1",
		Amount:        moneyAmount{Currency: "EUR", Cents: 100},
		IssueDate:     time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		Source:        source,
	}
	return invoiceImportResult{Invoices: []canonicalInvoice{invoice}, Report: buildIntegrationReport(source, 1, invoice.Validate())}, nil
}

type fakeExportAdapter struct{}

func (fakeExportAdapter) WriteExportData(_ context.Context, w io.Writer, records []canonicalExportRecord) (integrationReport, error) {
	errors := []integrationRecordError{}
	accepted := 0
	for _, record := range records {
		if errs := record.Validate(); len(errs) > 0 {
			errors = append(errors, errs...)
			continue
		}
		accepted++
		if _, err := io.WriteString(w, record.RecordID+"\n"); err != nil {
			return integrationReport{}, err
		}
	}
	return buildIntegrationReport(integrationSource{Format: integrationFormatManualCSV}, accepted, errors), nil
}
