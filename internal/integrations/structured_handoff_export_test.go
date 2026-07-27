package integrations

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStructuredHandoffAdapterWritesCSV(t *testing.T) {
	record := ExportRecord{
		TenantSlug: "JHW22",
		RecordID:   "parking-2026-06-top-11",
		Kind:       "payment-status",
		Occurred:   time.Date(2026, 7, 8, 0, 0, 0, 0, time.Local),
		UnitID:     "top-11",
		PersonRef:  "person:max",
		Reference:  "HV-JHW22-202606-A1B2C3",
		Amount:     MoneyAmount{Currency: "EUR", Cents: 13304},
		Fields: map[string]string{
			"period":             "2026-06",
			"status":             "paid",
			"source":             "parking",
			"document_reference": "doc-abc",
			"description":        "Parkplatzabrechnung Juni 2026",
			"verification_note":  "Neutral handoff without target-system compatibility claim",
		},
	}

	var out strings.Builder
	report, err := StructuredHandoffAdapter{}.WriteExportData(context.Background(), &out, []ExportRecord{record})
	if err != nil {
		t.Fatalf("WriteExportData: %v", err)
	}
	if report.Source.Format != FormatManualCSV || report.Source.Version != StructuredHandoffVersion || report.Accepted != 1 || report.Rejected != 0 {
		t.Fatalf("report = %+v", report)
	}
	want, err := os.ReadFile("testdata/structured-handoff-raw-v0.csv")
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if out.String() != string(want) {
		t.Fatalf("structured handoff CSV mismatch\nwant:\n%s\ngot:\n%s", string(want), out.String())
	}
}

func TestStructuredHandoffAdapterRejectsAccountingFields(t *testing.T) {
	record := ExportRecord{
		TenantSlug: "jhw22",
		RecordID:   "raw-1",
		Kind:       "payment-status",
		Occurred:   time.Date(2026, 7, 8, 0, 0, 0, 0, time.Local),
		Amount:     MoneyAmount{Currency: "EUR", Cents: 100},
		Fields: map[string]string{
			"debit_account": "4000",
			"tax_code":      "20",
			"status":        "paid",
		},
	}

	var out strings.Builder
	report, err := StructuredHandoffAdapter{}.WriteExportData(context.Background(), &out, []ExportRecord{record})
	if err != nil {
		t.Fatalf("WriteExportData: %v", err)
	}
	if report.Accepted != 0 || report.Rejected != 2 {
		t.Fatalf("report = %+v", report)
	}
	for _, want := range []string{"debit_account", "tax_code"} {
		found := false
		for _, recordErr := range report.Errors {
			if recordErr.Field == want && strings.Contains(recordErr.Message, "out of scope") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing rejection for %q in %+v", want, report.Errors)
		}
	}
	if strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("invalid records should only write header, got:\n%s", out.String())
	}
}

func TestStructuredHandoffAdapterRejectsUnmappedFieldsAndMissingDate(t *testing.T) {
	record := ExportRecord{
		TenantSlug: "jhw22",
		RecordID:   "raw-2",
		Kind:       "payment-status",
		Amount:     MoneyAmount{Currency: "EUR", Cents: 100},
		Fields: map[string]string{
			"custom_note": "not mapped yet",
		},
	}

	var out strings.Builder
	report, err := StructuredHandoffAdapter{}.WriteExportData(context.Background(), &out, []ExportRecord{record})
	if err != nil {
		t.Fatalf("WriteExportData: %v", err)
	}
	if report.Accepted != 0 || report.Rejected != 2 {
		t.Fatalf("report = %+v", report)
	}
	for _, want := range []string{"occurred", "custom_note"} {
		found := false
		for _, recordErr := range report.Errors {
			if recordErr.Field == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing error for %q in %+v", want, report.Errors)
		}
	}
}
