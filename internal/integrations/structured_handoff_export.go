package integrations

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/markus-barta/hausv-org/internal/textutil"
	"io"
	"sort"
	"strings"
	"time"
)

const StructuredHandoffVersion = "raw-v0"

var StructuredHandoffHeader = []string{
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

var structuredHandoffAllowedFields = map[string]struct{}{
	"period":             {},
	"status":             {},
	"source":             {},
	"document_reference": {},
	"description":        {},
	"verification_note":  {},
}

var structuredHandoffAccountingFields = map[string]struct{}{
	"account":        {},
	"contra_account": {},
	"credit_account": {},
	"debit_account":  {},
	"gegenkonto":     {},
	"konto":          {},
	"tax_code":       {},
	"vat_code":       {},
}

// StructuredHandoffAdapter writes a format-neutral CSV for controlled manual
// handoff. It deliberately does not claim compatibility with BMD, RZL, or any
// other accounting system.
type StructuredHandoffAdapter struct{}

func (StructuredHandoffAdapter) WriteExportData(ctx context.Context, w io.Writer, records []ExportRecord) (Report, error) {
	source := Source{Format: FormatManualCSV, Version: StructuredHandoffVersion}
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	if err := writer.Write(StructuredHandoffHeader); err != nil {
		return Report{}, err
	}

	accepted := 0
	errors := []RecordError{}
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		recordErrors := structuredHandoffRecordErrors(record)
		if len(recordErrors) > 0 {
			errors = append(errors, recordErrors...)
			continue
		}
		if err := writer.Write(structuredHandoffRow(record)); err != nil {
			return Report{}, err
		}
		accepted++
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return Report{}, err
	}
	return buildReport(source, accepted, errors), nil
}

func structuredHandoffRecordErrors(record ExportRecord) []RecordError {
	errors := record.Validate()
	if record.Occurred.IsZero() {
		errors = append(errors, RecordError{RecordType: RecordExportRawData, RecordID: record.RecordID, Field: "occurred", Message: "event date required for structured handoff"})
	}
	for key := range record.Fields {
		normalized := normalizeStructuredHandoffField(key)
		if _, reserved := structuredHandoffAccountingFields[normalized]; reserved {
			errors = append(errors, RecordError{RecordType: RecordExportRawData, RecordID: record.RecordID, Field: key, Message: "accounting account and tax fields are out of scope for this product"})
			continue
		}
		if _, allowed := structuredHandoffAllowedFields[normalized]; !allowed {
			errors = append(errors, RecordError{RecordType: RecordExportRawData, RecordID: record.RecordID, Field: key, Message: "field is not part of structured handoff raw-v0"})
		}
	}
	return errors
}

func structuredHandoffRow(record ExportRecord) []string {
	fields := normalizeStructuredHandoffFields(record.Fields)
	return []string{
		textutil.Slug(record.TenantSlug),
		strings.TrimSpace(record.RecordID),
		strings.TrimSpace(record.Kind),
		formatStructuredHandoffDate(record.Occurred),
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

func normalizeStructuredHandoffFields(fields map[string]string) map[string]string {
	normalized := map[string]string{}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		normalized[normalizeStructuredHandoffField(key)] = strings.TrimSpace(fields[key])
	}
	return normalized
}

func normalizeStructuredHandoffField(field string) string {
	return strings.ToLower(strings.TrimSpace(field))
}

func formatStructuredHandoffDate(t time.Time) string {
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
