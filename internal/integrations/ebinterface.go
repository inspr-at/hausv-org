package integrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"github.com/inspr-at/hausv-org/internal/textutil"
	"io"
	"strings"
	"time"
)

type EBInterfaceAdapter struct{}

func (EBInterfaceAdapter) ParseInvoices(ctx context.Context, source Source, r io.Reader) (InvoiceImportResult, error) {
	select {
	case <-ctx.Done():
		return InvoiceImportResult{}, ctx.Err()
	default:
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return InvoiceImportResult{}, err
	}
	invoice, report := parseEBInterfaceInvoice(source, data)
	if report.HasErrors() {
		return InvoiceImportResult{Report: report}, nil
	}
	return InvoiceImportResult{
		Invoices: []Invoice{invoice},
		Report:   report,
	}, nil
}

func parseEBInterfaceInvoice(source Source, data []byte) (Invoice, Report) {
	var document ebInterfaceDocument
	if err := xml.Unmarshal(data, &document); err != nil {
		recordErr := RecordError{RecordType: RecordInvoice, Field: "xml", Message: err.Error()}
		return Invoice{}, buildReport(source, 0, []RecordError{recordErr})
	}
	version, ok := ebInterfaceVersionFromNamespace(document.XMLName.Space)
	if !ok {
		recordErr := RecordError{RecordType: RecordInvoice, Field: "namespace", Message: "unsupported ebInterface namespace"}
		return Invoice{}, buildReport(source, 0, []RecordError{recordErr})
	}
	if source.Format == "" {
		source.Format = FormatEBInterface
	}
	if source.Version == "" {
		source.Version = version
	}
	invoice := ebInterfaceCanonicalInvoice(source, document, data)
	errors := invoice.Validate()
	if len(errors) > 0 {
		return Invoice{}, buildReport(source, 0, errors)
	}
	return invoice, buildReport(source, 1, nil)
}

func ebInterfaceVersionFromNamespace(namespace string) (string, bool) {
	switch strings.TrimSpace(namespace) {
	case "http://www.ebinterface.at/schema/5p0/":
		return "5.0", true
	case "http://www.ebinterface.at/schema/6p0/":
		return "6.0", true
	default:
		return "", false
	}
}

func ebInterfaceCanonicalInvoice(source Source, document ebInterfaceDocument, data []byte) Invoice {
	issueDate, _ := parseISODate(document.InvoiceDate)
	dueDate, _ := parseISODate(textutil.FirstNonEmpty(document.PaymentMethod.DueDate, document.PaymentConditions.DueDate))
	periodStart, _ := parseISODate(document.Delivery.Period.FromDate)
	periodEnd, _ := parseISODate(document.Delivery.Period.ToDate)
	amount := document.TotalGrossAmount
	if strings.TrimSpace(amount.Value) == "" {
		amount = document.InvoiceTotals.TotalGrossAmount
	}
	amountCents, _ := ParseDecimalCents(amount.Value)
	currency := textutil.FirstNonEmpty(amount.Currency, document.InvoiceCurrency, document.Currency, "EUR")
	invoiceNumber := strings.TrimSpace(document.InvoiceNumber)
	externalID := invoiceNumber
	if externalID == "" {
		externalID = camtEntryDigest(document.XMLName.Space, string(data))
	}
	return Invoice{
		TenantSlug:         textutil.FirstNonEmpty(textutil.Slug(source.TenantSlug), textutil.Slug(document.InvoiceRecipient.DisplayName()), textutil.Slug(document.Biller.DisplayName())),
		ExternalID:         externalID,
		InvoiceNumber:      invoiceNumber,
		IssuerName:         document.Biller.DisplayName(),
		RecipientName:      document.InvoiceRecipient.DisplayName(),
		Amount:             MoneyAmount{Currency: currency, Cents: amountCents},
		IssueDate:          issueDate,
		DueDate:            dueDate,
		ServicePeriodStart: periodStart,
		ServicePeriodEnd:   periodEnd,
		Source:             source,
		RawDigest:          ebInterfaceDigest(data),
	}
}

func parseISODate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("date required")
	}
	return time.Parse("2006-01-02", raw)
}

func ebInterfaceDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func SanitizeFilenameToken(raw string) string {
	raw = paymentReferenceASCII(raw)
	if raw == "" {
		return "rechnung"
	}
	if len(raw) > 48 {
		raw = raw[:48]
	}
	return strings.ToLower(raw)
}

type ebInterfaceDocument struct {
	XMLName           xml.Name                     `xml:"Invoice"`
	Currency          string                       `xml:"Currency,attr"`
	InvoiceCurrency   string                       `xml:"InvoiceCurrency,attr"`
	InvoiceNumber     string                       `xml:"InvoiceNumber"`
	InvoiceDate       string                       `xml:"InvoiceDate"`
	Biller            ebInterfaceParty             `xml:"Biller"`
	InvoiceRecipient  ebInterfaceParty             `xml:"InvoiceRecipient"`
	Delivery          ebInterfaceDelivery          `xml:"Delivery"`
	PaymentMethod     ebInterfacePaymentMethod     `xml:"PaymentMethod"`
	PaymentConditions ebInterfacePaymentConditions `xml:"PaymentConditions"`
	TotalGrossAmount  ebInterfaceAmount            `xml:"TotalGrossAmount"`
	InvoiceTotals     ebInterfaceTotals            `xml:"InvoiceTotals"`
}

type ebInterfaceParty struct {
	Name    string             `xml:"Name"`
	Address ebInterfaceAddress `xml:"Address"`
}

func (p ebInterfaceParty) DisplayName() string {
	return strings.TrimSpace(textutil.FirstNonEmpty(p.Name, p.Address.Name))
}

type ebInterfaceAddress struct {
	Name string `xml:"Name"`
}

type ebInterfaceDelivery struct {
	Period ebInterfacePeriod `xml:"Period"`
}

type ebInterfacePeriod struct {
	FromDate string `xml:"FromDate"`
	ToDate   string `xml:"ToDate"`
}

type ebInterfacePaymentMethod struct {
	DueDate string `xml:"DueDate"`
}

type ebInterfacePaymentConditions struct {
	DueDate string `xml:"DueDate"`
}

type ebInterfaceTotals struct {
	TotalGrossAmount ebInterfaceAmount `xml:"TotalGrossAmount"`
}

type ebInterfaceAmount struct {
	Currency string `xml:"Currency,attr"`
	Value    string `xml:",chardata"`
}
