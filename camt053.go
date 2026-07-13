package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var camtPaymentReferencePattern = regexp.MustCompile(`HV-[A-Z0-9-]{1,32}`)

type camt053Adapter struct{}

func (camt053Adapter) ParsePayments(ctx context.Context, source integrationSource, r io.Reader) (paymentImportResult, error) {
	select {
	case <-ctx.Done():
		return paymentImportResult{}, ctx.Err()
	default:
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return paymentImportResult{}, err
	}
	var document camtDocument
	if err := xml.Unmarshal(data, &document); err != nil {
		return paymentImportResult{}, err
	}
	version, ok := camt053VersionFromNamespace(document.XMLName.Space)
	if !ok {
		report := buildIntegrationReport(source, 0, []integrationRecordError{{
			RecordType: integrationRecordPayment,
			Field:      "namespace",
			Message:    "unsupported camt.053 namespace",
		}})
		return paymentImportResult{Report: report}, nil
	}
	if source.Format == "" {
		source.Format = integrationFormatCAMT053
	}
	if source.Version == "" {
		source.Version = version
	}

	payments := []canonicalPayment{}
	errors := []integrationRecordError{}
	for statementIndex, statement := range document.CustomerStatement.Statements {
		for entryIndex, entry := range statement.Entries {
			payment, recordErrors := camtPaymentFromEntry(source, statement, entry, statementIndex, entryIndex)
			if len(recordErrors) > 0 {
				errors = append(errors, recordErrors...)
				continue
			}
			payments = append(payments, payment)
		}
	}
	return paymentImportResult{
		Payments: payments,
		Report:   buildIntegrationReport(source, len(payments), errors),
	}, nil
}

func camt053VersionFromNamespace(namespace string) (string, bool) {
	switch strings.TrimSpace(namespace) {
	case "urn:iso:std:iso:20022:tech:xsd:camt.053.001.02":
		return "2009/camt.053.001.02", true
	case "urn:iso:std:iso:20022:tech:xsd:camt.053.001.08":
		return "2019/camt.053.001.08", true
	default:
		return "", false
	}
}

func camtPaymentFromEntry(source integrationSource, statement camtStatement, entry camtEntry, statementIndex int, entryIndex int) (canonicalPayment, []integrationRecordError) {
	recordID := camtEntryRecordID(statement, entry, statementIndex, entryIndex)
	errors := []integrationRecordError{}
	if strings.ToUpper(strings.TrimSpace(entry.CreditDebitIndicator)) != "CRDT" {
		return canonicalPayment{}, []integrationRecordError{{
			RecordType: integrationRecordPayment,
			RecordID:   recordID,
			Field:      "credit_debit_indicator",
			Message:    "only incoming credit entries are imported",
		}}
	}
	amountCents, err := parseDecimalCents(entry.Amount.Value)
	if err != nil {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordPayment, RecordID: recordID, Field: "amount", Message: err.Error()})
	}
	bookingDate, err := parseCAMTDateChoice(entry.BookingDate)
	if err != nil {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordPayment, RecordID: recordID, Field: "booking_date", Message: err.Error()})
	}
	valueDate, _ := parseCAMTDateChoice(entry.ValueDate)
	tx := entry.FirstTransaction()
	payment := canonicalPayment{
		TenantSlug:      normalizeSlug(statement.Account.OwnerName),
		ExternalID:      recordID,
		Reference:       camtPaymentReference(tx),
		Amount:          moneyAmount{Currency: strings.ToUpper(strings.TrimSpace(entry.Amount.Currency)), Cents: amountCents},
		BookingDate:     bookingDate,
		ValueDate:       valueDate,
		DebtorName:      strings.TrimSpace(tx.RelatedParties.Debtor.Name),
		DebtorIBAN:      strings.TrimSpace(tx.RelatedParties.DebtorAccount.ID.IBAN),
		RemittanceLines: cleanStringSlice(tx.Remittance.Unstructured),
		Source:          source,
		RawDigest:       camtEntryDigest(recordID, entry.Amount.Value, entry.BookingDate.Date, entry.BookingDate.DateTime),
	}
	if payment.TenantSlug == "" {
		payment.TenantSlug = normalizeSlug(statement.ID)
	}
	errors = append(errors, payment.Validate()...)
	return payment, errors
}

func camtEntryRecordID(statement camtStatement, entry camtEntry, statementIndex int, entryIndex int) string {
	if entry.Reference != "" {
		return strings.TrimSpace(entry.Reference)
	}
	tx := entry.FirstTransaction()
	for _, candidate := range []string{tx.References.TransactionID, tx.References.EndToEndID, statement.ID} {
		if strings.TrimSpace(candidate) != "" && strings.ToUpper(strings.TrimSpace(candidate)) != "NOTPROVIDED" {
			return strings.TrimSpace(candidate)
		}
	}
	return fmt.Sprintf("stmt-%d-entry-%d", statementIndex+1, entryIndex+1)
}

func camtPaymentReference(tx camtTransactionDetails) string {
	candidates := []string{
		tx.References.EndToEndID,
		tx.References.TransactionID,
		tx.References.InstructionID,
		tx.References.MandateID,
	}
	candidates = append(candidates, tx.Remittance.Unstructured...)
	for _, candidate := range candidates {
		normalized := normalizePaymentReference(candidate)
		if err := validatePaymentReference(normalized); err == nil && strings.HasPrefix(normalized, paymentReferencePrefix+"-") {
			return normalized
		}
		for _, match := range camtPaymentReferencePattern.FindAllString(normalized, -1) {
			match = strings.Trim(match, "-")
			if err := validatePaymentReference(match); err == nil {
				return match
			}
		}
	}
	return ""
}

func parseCAMTDateChoice(choice camtDateChoice) (time.Time, error) {
	if strings.TrimSpace(choice.Date) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(choice.Date))
		if err != nil {
			return time.Time{}, err
		}
		return parsed, nil
	}
	if strings.TrimSpace(choice.DateTime) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(choice.DateTime))
		if err != nil {
			return time.Time{}, err
		}
		return parsed, nil
	}
	return time.Time{}, fmt.Errorf("date required")
}

func parseDecimalCents(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("amount required")
	}
	if strings.HasPrefix(raw, "-") {
		return 0, fmt.Errorf("amount must not be negative")
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("amount has invalid decimal separator")
	}
	whole := parts[0]
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if whole == "" {
		whole = "0"
	}
	if len(fraction) > 2 {
		return 0, fmt.Errorf("amount has more than two decimal places")
	}
	for _, r := range whole + fraction {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("amount must contain digits and decimal point only")
		}
	}
	for len(fraction) < 2 {
		fraction += "0"
	}
	wholeCents, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, err
	}
	fractionCents, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	return wholeCents*100 + fractionCents, nil
}

func cleanStringSlice(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func camtEntryDigest(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

type camtDocument struct {
	XMLName           xml.Name         `xml:"Document"`
	CustomerStatement camtCustomerStmt `xml:"BkToCstmrStmt"`
}

type camtCustomerStmt struct {
	Statements []camtStatement `xml:"Stmt"`
}

type camtStatement struct {
	ID      string      `xml:"Id"`
	Account camtAccount `xml:"Acct"`
	Entries []camtEntry `xml:"Ntry"`
}

type camtAccount struct {
	OwnerName string `xml:"Ownr>Nm"`
}

type camtEntry struct {
	Reference             string           `xml:"NtryRef"`
	Amount                camtAmount       `xml:"Amt"`
	CreditDebitIndicator  string           `xml:"CdtDbtInd"`
	BookingDate           camtDateChoice   `xml:"BookgDt"`
	ValueDate             camtDateChoice   `xml:"ValDt"`
	EntryDetails          camtEntryDetails `xml:"NtryDtls"`
	AdditionalInformation string           `xml:"AddtlNtryInf"`
}

func (e camtEntry) FirstTransaction() camtTransactionDetails {
	if len(e.EntryDetails.TransactionDetails) > 0 {
		return e.EntryDetails.TransactionDetails[0]
	}
	return camtTransactionDetails{}
}

type camtAmount struct {
	Currency string `xml:"Ccy,attr"`
	Value    string `xml:",chardata"`
}

type camtDateChoice struct {
	Date     string `xml:"Dt"`
	DateTime string `xml:"DtTm"`
}

type camtEntryDetails struct {
	TransactionDetails []camtTransactionDetails `xml:"TxDtls"`
}

type camtTransactionDetails struct {
	References     camtReferences     `xml:"Refs"`
	RelatedParties camtRelatedParties `xml:"RltdPties"`
	Remittance     camtRemittance     `xml:"RmtInf"`
}

type camtReferences struct {
	InstructionID string `xml:"InstrId"`
	EndToEndID    string `xml:"EndToEndId"`
	TransactionID string `xml:"TxId"`
	MandateID     string `xml:"MndtId"`
}

type camtRelatedParties struct {
	Debtor        camtParty     `xml:"Dbtr"`
	DebtorAccount camtAccountID `xml:"DbtrAcct"`
}

type camtParty struct {
	Name string `xml:"Nm"`
}

type camtAccountID struct {
	ID camtIBAN `xml:"Id"`
}

type camtIBAN struct {
	IBAN string `xml:"IBAN"`
}

type camtRemittance struct {
	Unstructured []string `xml:"Ustrd"`
}
