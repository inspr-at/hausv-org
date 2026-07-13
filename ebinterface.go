package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

type ebInterfaceAdapter struct{}

func (ebInterfaceAdapter) ParseInvoices(ctx context.Context, source integrationSource, r io.Reader) (invoiceImportResult, error) {
	select {
	case <-ctx.Done():
		return invoiceImportResult{}, ctx.Err()
	default:
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return invoiceImportResult{}, err
	}
	invoice, report := parseEBInterfaceInvoice(source, data)
	if report.HasErrors() {
		return invoiceImportResult{Report: report}, nil
	}
	return invoiceImportResult{
		Invoices: []canonicalInvoice{invoice},
		Report:   report,
	}, nil
}

func parseEBInterfaceInvoice(source integrationSource, data []byte) (canonicalInvoice, integrationReport) {
	var document ebInterfaceDocument
	if err := xml.Unmarshal(data, &document); err != nil {
		recordErr := integrationRecordError{RecordType: integrationRecordInvoice, Field: "xml", Message: err.Error()}
		return canonicalInvoice{}, buildIntegrationReport(source, 0, []integrationRecordError{recordErr})
	}
	version, ok := ebInterfaceVersionFromNamespace(document.XMLName.Space)
	if !ok {
		recordErr := integrationRecordError{RecordType: integrationRecordInvoice, Field: "namespace", Message: "unsupported ebInterface namespace"}
		return canonicalInvoice{}, buildIntegrationReport(source, 0, []integrationRecordError{recordErr})
	}
	if source.Format == "" {
		source.Format = integrationFormatEBInterface
	}
	if source.Version == "" {
		source.Version = version
	}
	invoice := ebInterfaceCanonicalInvoice(source, document, data)
	errors := invoice.Validate()
	if len(errors) > 0 {
		return canonicalInvoice{}, buildIntegrationReport(source, 0, errors)
	}
	return invoice, buildIntegrationReport(source, 1, nil)
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

func ebInterfaceCanonicalInvoice(source integrationSource, document ebInterfaceDocument, data []byte) canonicalInvoice {
	issueDate, _ := parseISODate(document.InvoiceDate)
	dueDate, _ := parseISODate(firstNonEmpty(document.PaymentMethod.DueDate, document.PaymentConditions.DueDate))
	periodStart, _ := parseISODate(document.Delivery.Period.FromDate)
	periodEnd, _ := parseISODate(document.Delivery.Period.ToDate)
	amount := document.TotalGrossAmount
	if strings.TrimSpace(amount.Value) == "" {
		amount = document.InvoiceTotals.TotalGrossAmount
	}
	amountCents, _ := parseDecimalCents(amount.Value)
	currency := firstNonEmpty(amount.Currency, document.Currency, "EUR")
	invoiceNumber := strings.TrimSpace(document.InvoiceNumber)
	externalID := invoiceNumber
	if externalID == "" {
		externalID = camtEntryDigest(document.XMLName.Space, string(data))
	}
	return canonicalInvoice{
		TenantSlug:         firstNonEmpty(normalizeSlug(source.TenantSlug), normalizeSlug(document.InvoiceRecipient.DisplayName()), normalizeSlug(document.Biller.DisplayName())),
		ExternalID:         externalID,
		InvoiceNumber:      invoiceNumber,
		IssuerName:         document.Biller.DisplayName(),
		RecipientName:      document.InvoiceRecipient.DisplayName(),
		Amount:             moneyAmount{Currency: currency, Cents: amountCents},
		IssueDate:          issueDate,
		DueDate:            dueDate,
		ServicePeriodStart: periodStart,
		ServicePeriodEnd:   periodEnd,
		Source:             source,
		RawDigest:          ebInterfaceDigest(data),
	}
}

func storeEBInterfaceInvoiceDocument(store *documentStore, invoice canonicalInvoice, uploadedBy string, data []byte, now time.Time) (documentRecord, error) {
	if store == nil {
		return documentRecord{}, fmt.Errorf("document store unavailable")
	}
	title := "E-Rechnung"
	if strings.TrimSpace(invoice.InvoiceNumber) != "" {
		title += " " + strings.TrimSpace(invoice.InvoiceNumber)
	}
	if strings.TrimSpace(invoice.IssuerName) != "" {
		title += " - " + strings.TrimSpace(invoice.IssuerName)
	}
	filenameToken := sanitizeFilenameToken(firstNonEmpty(invoice.InvoiceNumber, invoice.ExternalID, "rechnung"))
	return store.CreateGenerated(documentRecord{
		TenantSlug: invoice.TenantSlug,
		Title:      title,
		Category:   documentCategoryBilling,
		Visibility: documentVisibilityManagerOnly,
		UploadedBy: uploadedBy,
	}, "ebinterface-"+filenameToken+".xml", "application/xml", data, now)
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

func sanitizeFilenameToken(raw string) string {
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
	return strings.TrimSpace(firstNonEmpty(p.Name, p.Address.Name))
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
