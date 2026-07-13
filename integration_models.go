package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

type integrationFormat string

const (
	integrationFormatCAMT053       integrationFormat = "camt.053"
	integrationFormatCAMT054       integrationFormat = "camt.054"
	integrationFormatBMD           integrationFormat = "bmd"
	integrationFormatRZL           integrationFormat = "rzl"
	integrationFormatEBInterface   integrationFormat = "ebinterface"
	integrationFormatManualCSV     integrationFormat = "manual-csv"
	integrationRecordPayment       string            = "payment"
	integrationRecordInvoice       string            = "invoice"
	integrationRecordExportRawData string            = "export-raw-data"
)

type integrationSource struct {
	TenantSlug string
	Format     integrationFormat
	Version    string
	Filename   string
	Imported   time.Time
}

type moneyAmount struct {
	Currency string
	Cents    int64
}

func (m moneyAmount) Validate() error {
	currency := strings.ToUpper(strings.TrimSpace(m.Currency))
	if currency == "" {
		return fmt.Errorf("currency required")
	}
	if len(currency) != 3 {
		return fmt.Errorf("currency must be ISO-4217 alpha-3")
	}
	if m.Cents < 0 {
		return fmt.Errorf("amount must not be negative")
	}
	return nil
}

type canonicalPayment struct {
	TenantSlug      string
	ExternalID      string
	Reference       string
	Amount          moneyAmount
	BookingDate     time.Time
	ValueDate       time.Time
	DebtorName      string
	DebtorIBAN      string
	RemittanceLines []string
	Source          integrationSource
	RawDigest       string
}

func (p canonicalPayment) Validate() []integrationRecordError {
	errors := []integrationRecordError{}
	if normalizeSlug(p.TenantSlug) == "" {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordPayment, RecordID: p.ExternalID, Field: "tenant_slug", Message: "tenant required"})
	}
	if strings.TrimSpace(p.ExternalID) == "" && strings.TrimSpace(p.RawDigest) == "" {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordPayment, RecordID: p.ExternalID, Field: "external_id", Message: "external id or raw digest required"})
	}
	if err := p.Amount.Validate(); err != nil {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordPayment, RecordID: p.ExternalID, Field: "amount", Message: err.Error()})
	}
	if p.BookingDate.IsZero() {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordPayment, RecordID: p.ExternalID, Field: "booking_date", Message: "booking date required"})
	}
	return errors
}

type canonicalInvoice struct {
	TenantSlug         string
	ExternalID         string
	InvoiceNumber      string
	IssuerName         string
	RecipientName      string
	Amount             moneyAmount
	IssueDate          time.Time
	DueDate            time.Time
	ServicePeriodStart time.Time
	ServicePeriodEnd   time.Time
	AttachmentID       string
	Source             integrationSource
	RawDigest          string
}

func (i canonicalInvoice) Validate() []integrationRecordError {
	errors := []integrationRecordError{}
	if normalizeSlug(i.TenantSlug) == "" {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordInvoice, RecordID: i.ExternalID, Field: "tenant_slug", Message: "tenant required"})
	}
	if strings.TrimSpace(i.InvoiceNumber) == "" && strings.TrimSpace(i.ExternalID) == "" {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordInvoice, RecordID: i.ExternalID, Field: "invoice_number", Message: "invoice number or external id required"})
	}
	if err := i.Amount.Validate(); err != nil {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordInvoice, RecordID: i.ExternalID, Field: "amount", Message: err.Error()})
	}
	if i.Amount.Cents <= 0 {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordInvoice, RecordID: i.ExternalID, Field: "amount", Message: "invoice amount required"})
	}
	if i.IssueDate.IsZero() {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordInvoice, RecordID: i.ExternalID, Field: "issue_date", Message: "issue date required"})
	}
	return errors
}

type canonicalExportRecord struct {
	TenantSlug string
	RecordID   string
	Kind       string
	Occurred   time.Time
	UnitID     string
	PersonRef  string
	Reference  string
	Amount     moneyAmount
	Fields     map[string]string
}

func (r canonicalExportRecord) Validate() []integrationRecordError {
	errors := []integrationRecordError{}
	if normalizeSlug(r.TenantSlug) == "" {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordExportRawData, RecordID: r.RecordID, Field: "tenant_slug", Message: "tenant required"})
	}
	if strings.TrimSpace(r.RecordID) == "" {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordExportRawData, RecordID: r.RecordID, Field: "record_id", Message: "record id required"})
	}
	if strings.TrimSpace(r.Kind) == "" {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordExportRawData, RecordID: r.RecordID, Field: "kind", Message: "kind required"})
	}
	if r.Amount.Currency != "" || r.Amount.Cents != 0 {
		if err := r.Amount.Validate(); err != nil {
			errors = append(errors, integrationRecordError{RecordType: integrationRecordExportRawData, RecordID: r.RecordID, Field: "amount", Message: err.Error()})
		}
	}
	return errors
}

type integrationRecordError struct {
	RecordType string
	RecordID   string
	Field      string
	Message    string
}

func (e integrationRecordError) Error() string {
	parts := []string{}
	if e.RecordType != "" {
		parts = append(parts, e.RecordType)
	}
	if e.RecordID != "" {
		parts = append(parts, e.RecordID)
	}
	if e.Field != "" {
		parts = append(parts, e.Field)
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	return strings.Join(parts, ": ")
}

type integrationReport struct {
	Source   integrationSource
	Accepted int
	Rejected int
	Errors   []integrationRecordError
}

func (r integrationReport) HasErrors() bool {
	return len(r.Errors) > 0 || r.Rejected > 0
}

type paymentImportResult struct {
	Payments []canonicalPayment
	Report   integrationReport
}

type invoiceImportResult struct {
	Invoices []canonicalInvoice
	Report   integrationReport
}

type paymentImportAdapter interface {
	ParsePayments(context.Context, integrationSource, io.Reader) (paymentImportResult, error)
}

type invoiceImportAdapter interface {
	ParseInvoices(context.Context, integrationSource, io.Reader) (invoiceImportResult, error)
}

type exportDataAdapter interface {
	WriteExportData(context.Context, io.Writer, []canonicalExportRecord) (integrationReport, error)
}

func buildIntegrationReport(source integrationSource, accepted int, errors []integrationRecordError) integrationReport {
	if accepted < 0 {
		accepted = 0
	}
	report := integrationReport{
		Source:   source,
		Accepted: accepted,
		Rejected: len(errors),
		Errors:   append([]integrationRecordError(nil), errors...),
	}
	return report
}
