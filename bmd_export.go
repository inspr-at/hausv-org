package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const bmdRawDataVersion = "raw-v0"

var bmdRawDataHeader = []string{
	"tenant_slug",
	"record_id",
	"kind",
	"occurred_at",
	"unit_id",
	"person_ref",
	"reference",
	"amount_currency",
	"amount_cents",
	"amount_decimal",
	"period",
	"status",
	"source",
	"document_reference",
	"description",
	"verification_note",
}

var bmdRawDataAllowedFields = map[string]struct{}{
	"period":             {},
	"status":             {},
	"source":             {},
	"document_reference": {},
	"description":        {},
	"verification_note":  {},
}

var bmdRawDataAccountingFields = map[string]struct{}{
	"account":        {},
	"contra_account": {},
	"credit_account": {},
	"debit_account":  {},
	"gegenkonto":     {},
	"konto":          {},
	"tax_code":       {},
	"vat_code":       {},
}

type bmdRawDataAdapter struct{}

func (bmdRawDataAdapter) WriteExportData(ctx context.Context, w io.Writer, records []canonicalExportRecord) (integrationReport, error) {
	source := integrationSource{Format: integrationFormatBMD, Version: bmdRawDataVersion}
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	if err := writer.Write(bmdRawDataHeader); err != nil {
		return integrationReport{}, err
	}

	accepted := 0
	errors := []integrationRecordError{}
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return integrationReport{}, err
		}
		recordErrors := bmdRawDataRecordErrors(record)
		if len(recordErrors) > 0 {
			errors = append(errors, recordErrors...)
			continue
		}
		if err := writer.Write(bmdRawDataRow(record)); err != nil {
			return integrationReport{}, err
		}
		accepted++
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return integrationReport{}, err
	}
	return buildIntegrationReport(source, accepted, errors), nil
}

func bmdRawDataRecordErrors(record canonicalExportRecord) []integrationRecordError {
	errors := record.Validate()
	if record.Occurred.IsZero() {
		errors = append(errors, integrationRecordError{RecordType: integrationRecordExportRawData, RecordID: record.RecordID, Field: "occurred", Message: "event date required for BMD raw data candidate"})
	}
	for key := range record.Fields {
		normalized := normalizeBMDRawDataField(key)
		if _, reserved := bmdRawDataAccountingFields[normalized]; reserved {
			errors = append(errors, integrationRecordError{RecordType: integrationRecordExportRawData, RecordID: record.RecordID, Field: key, Message: "accounting account fields are out of scope before Steuerberater verification"})
			continue
		}
		if _, allowed := bmdRawDataAllowedFields[normalized]; !allowed {
			errors = append(errors, integrationRecordError{RecordType: integrationRecordExportRawData, RecordID: record.RecordID, Field: key, Message: "field is not part of BMD raw data candidate v0"})
		}
	}
	return errors
}

func bmdRawDataRow(record canonicalExportRecord) []string {
	fields := normalizeBMDRawDataFields(record.Fields)
	return []string{
		normalizeSlug(record.TenantSlug),
		strings.TrimSpace(record.RecordID),
		strings.TrimSpace(record.Kind),
		formatBMDRawDataDate(record.Occurred),
		strings.TrimSpace(record.UnitID),
		strings.TrimSpace(record.PersonRef),
		strings.TrimSpace(record.Reference),
		strings.ToUpper(strings.TrimSpace(record.Amount.Currency)),
		fmt.Sprintf("%d", record.Amount.Cents),
		formatCentsDecimal(record.Amount.Cents),
		fields["period"],
		fields["status"],
		fields["source"],
		fields["document_reference"],
		fields["description"],
		fields["verification_note"],
	}
}

func normalizeBMDRawDataFields(fields map[string]string) map[string]string {
	normalized := map[string]string{}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		normalized[normalizeBMDRawDataField(key)] = strings.TrimSpace(fields[key])
	}
	return normalized
}

func normalizeBMDRawDataField(field string) string {
	return strings.ToLower(strings.TrimSpace(field))
}

func formatBMDRawDataDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(time.Local).Format("2006-01-02")
}

func formatCentsDecimal(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d,%02d", sign, cents/100, cents%100)
}
