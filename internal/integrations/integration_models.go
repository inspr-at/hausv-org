package integrations

import (
	"context"
	"fmt"
	"github.com/inspr-at/hausv-org/internal/textutil"
	"io"
	"strings"
	"time"
)

type Format string

const (
	FormatCAMT053       Format = "camt.053"
	FormatCAMT054       Format = "camt.054"
	FormatBMD           Format = "bmd"
	FormatRZL           Format = "rzl"
	FormatEBInterface   Format = "ebinterface"
	FormatManualCSV     Format = "manual-csv"
	RecordPayment       string = "payment"
	RecordInvoice       string = "invoice"
	RecordExportRawData string = "export-raw-data"
)

type Source struct {
	TenantSlug string
	Format     Format
	Version    string
	Filename   string
	Imported   time.Time
}

type MoneyAmount struct {
	Currency string
	Cents    int64
}

func (m MoneyAmount) Validate() error {
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

type Payment struct {
	TenantSlug      string
	ExternalID      string
	Reference       string
	Amount          MoneyAmount
	BookingDate     time.Time
	ValueDate       time.Time
	DebtorName      string
	DebtorIBAN      string
	RemittanceLines []string
	Source          Source
	RawDigest       string
}

func (p Payment) Validate() []RecordError {
	errors := []RecordError{}
	if textutil.Slug(p.TenantSlug) == "" {
		errors = append(errors, RecordError{RecordType: RecordPayment, RecordID: p.ExternalID, Field: "tenant_slug", Message: "tenant required"})
	}
	if strings.TrimSpace(p.ExternalID) == "" && strings.TrimSpace(p.RawDigest) == "" {
		errors = append(errors, RecordError{RecordType: RecordPayment, RecordID: p.ExternalID, Field: "external_id", Message: "external id or raw digest required"})
	}
	if err := p.Amount.Validate(); err != nil {
		errors = append(errors, RecordError{RecordType: RecordPayment, RecordID: p.ExternalID, Field: "amount", Message: err.Error()})
	}
	if p.BookingDate.IsZero() {
		errors = append(errors, RecordError{RecordType: RecordPayment, RecordID: p.ExternalID, Field: "booking_date", Message: "booking date required"})
	}
	return errors
}

type Invoice struct {
	TenantSlug         string
	ExternalID         string
	InvoiceNumber      string
	IssuerName         string
	RecipientName      string
	Amount             MoneyAmount
	IssueDate          time.Time
	DueDate            time.Time
	ServicePeriodStart time.Time
	ServicePeriodEnd   time.Time
	AttachmentID       string
	Source             Source
	RawDigest          string
}

func (i Invoice) Validate() []RecordError {
	errors := []RecordError{}
	if textutil.Slug(i.TenantSlug) == "" {
		errors = append(errors, RecordError{RecordType: RecordInvoice, RecordID: i.ExternalID, Field: "tenant_slug", Message: "tenant required"})
	}
	if strings.TrimSpace(i.InvoiceNumber) == "" && strings.TrimSpace(i.ExternalID) == "" {
		errors = append(errors, RecordError{RecordType: RecordInvoice, RecordID: i.ExternalID, Field: "invoice_number", Message: "invoice number or external id required"})
	}
	if err := i.Amount.Validate(); err != nil {
		errors = append(errors, RecordError{RecordType: RecordInvoice, RecordID: i.ExternalID, Field: "amount", Message: err.Error()})
	}
	if i.Amount.Cents <= 0 {
		errors = append(errors, RecordError{RecordType: RecordInvoice, RecordID: i.ExternalID, Field: "amount", Message: "invoice amount required"})
	}
	if i.IssueDate.IsZero() {
		errors = append(errors, RecordError{RecordType: RecordInvoice, RecordID: i.ExternalID, Field: "issue_date", Message: "issue date required"})
	}
	return errors
}

type ExportRecord struct {
	TenantSlug string
	RecordID   string
	Kind       string
	Occurred   time.Time
	UnitID     string
	PersonRef  string
	Reference  string
	Amount     MoneyAmount
	Fields     map[string]string
}

func (r ExportRecord) Validate() []RecordError {
	errors := []RecordError{}
	if textutil.Slug(r.TenantSlug) == "" {
		errors = append(errors, RecordError{RecordType: RecordExportRawData, RecordID: r.RecordID, Field: "tenant_slug", Message: "tenant required"})
	}
	if strings.TrimSpace(r.RecordID) == "" {
		errors = append(errors, RecordError{RecordType: RecordExportRawData, RecordID: r.RecordID, Field: "record_id", Message: "record id required"})
	}
	if strings.TrimSpace(r.Kind) == "" {
		errors = append(errors, RecordError{RecordType: RecordExportRawData, RecordID: r.RecordID, Field: "kind", Message: "kind required"})
	}
	if r.Amount.Currency != "" || r.Amount.Cents != 0 {
		if err := r.Amount.Validate(); err != nil {
			errors = append(errors, RecordError{RecordType: RecordExportRawData, RecordID: r.RecordID, Field: "amount", Message: err.Error()})
		}
	}
	return errors
}

type RecordError struct {
	RecordType string
	RecordID   string
	Field      string
	Message    string
}

func (e RecordError) Error() string {
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

type Report struct {
	Source   Source
	Accepted int
	Rejected int
	Errors   []RecordError
}

func (r Report) HasErrors() bool {
	return len(r.Errors) > 0 || r.Rejected > 0
}

type PaymentImportResult struct {
	Payments []Payment
	Report   Report
}

type InvoiceImportResult struct {
	Invoices []Invoice
	Report   Report
}

type PaymentImportAdapter interface {
	ParsePayments(context.Context, Source, io.Reader) (PaymentImportResult, error)
}

type InvoiceImportAdapter interface {
	ParseInvoices(context.Context, Source, io.Reader) (InvoiceImportResult, error)
}

type ExportDataAdapter interface {
	WriteExportData(context.Context, io.Writer, []ExportRecord) (Report, error)
}

func buildReport(source Source, accepted int, errors []RecordError) Report {
	if accepted < 0 {
		accepted = 0
	}
	report := Report{
		Source:   source,
		Accepted: accepted,
		Rejected: len(errors),
		Errors:   append([]RecordError(nil), errors...),
	}
	return report
}
