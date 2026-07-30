package server

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strings"
	"testing"
	"time"
)

// HAUSV-164: per-month CSV export with hour-level detail, a print/PDF-friendly
// settlement view, and a data-quality warning for incomplete months.

func readSemicolonCSV(t *testing.T, s string) [][]string {
	t.Helper()
	r := csv.NewReader(strings.NewReader(s))
	r.Comma = ';'
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	return rows
}

func csvRowAfter(t *testing.T, rows [][]string, label string) []string {
	t.Helper()
	for i, row := range rows {
		if len(row) > 0 && row[0] == label && i+1 < len(rows) {
			return rows[i+1]
		}
	}
	t.Fatalf("row after %q not found", label)
	return nil
}

func csvRowsAfter(t *testing.T, rows [][]string, label string) [][]string {
	t.Helper()
	for i, row := range rows {
		if len(row) > 0 && row[0] == label {
			return rows[i+1:]
		}
	}
	t.Fatalf("rows after %q not found", label)
	return nil
}

func TestParkingMonthCSVTotalsMatchView(t *testing.T) {
	energy, prices, now := chargingBillingFixture(t)
	data := parkingTenantData{
		Settings:      parkingSettings{Tariffs: []parkingTariff{{EffectiveFrom: "2026-01-01", GridFeeEURPerKWh: 0.05, BaseFeeEUR: 3}}},
		EnergySamples: energy,
		PriceSamples:  prices,
	}
	month := energy[0].At.In(time.UTC).Format("2006-01")
	view := calculateParkingMonthDetails(data, month, now, time.UTC)
	if !view.HasHours || view.Summary.Month == "" {
		t.Fatalf("fixture should produce a month with hours: %+v", view.Summary)
	}

	var buf bytes.Buffer
	tenant := tenantConfig{Slug: "jhw22", Name: "Janischhofweg 22", Address: "Janischhofweg 22"}
	if err := writeParkingMonthCSV(&buf, tenant, view); err != nil {
		t.Fatalf("csv: %v", err)
	}
	rows := readSemicolonCSV(t, buf.String())
	if len(rows) == 0 || len(rows[0]) != 1 || rows[0][0] != parkingMonthCSVTitle {
		t.Fatalf("CSV title row = %v, want %q", rows, parkingMonthCSVTitle)
	}
	if strings.Contains(buf.String(), "WEG Portal") {
		t.Fatalf("CSV contains obsolete product wording:\n%s", buf.String())
	}

	// Monthly summary total in the CSV must equal the on-screen month total.
	sumRow := csvRowAfter(t, rows, "Monatssumme")
	if got := sumRow[len(sumRow)-1]; got != view.Summary.TotalCost {
		t.Fatalf("CSV month total %q != view total %q", got, view.Summary.TotalCost)
	}

	// Hourly rows must match the on-screen hourly detail one-for-one.
	hourRows := csvRowsAfter(t, rows, "Stunde")
	if len(hourRows) != len(view.Hours) {
		t.Fatalf("CSV hourly rows %d != view hours %d", len(hourRows), len(view.Hours))
	}
	first := hourRows[0]
	if first[0] != view.Hours[0].AtLabel || first[len(first)-1] != view.Hours[0].TotalCost {
		t.Fatalf("first hourly row mismatch: %v vs at=%q total=%q", first, view.Hours[0].AtLabel, view.Hours[0].TotalCost)
	}
}

func TestParkingMonthCSVWarnsOnPartialData(t *testing.T) {
	energy, prices, now := chargingBillingFixture(t) // coverage starts mid-month
	data := parkingTenantData{
		Settings:      parkingSettings{Tariffs: []parkingTariff{{EffectiveFrom: "2026-01-01", GridFeeEURPerKWh: 0.05, BaseFeeEUR: 3}}},
		EnergySamples: energy,
		PriceSamples:  prices,
	}
	month := energy[0].At.In(time.UTC).Format("2006-01")
	view := calculateParkingMonthDetails(data, month, now, time.UTC)
	if !view.Summary.Partial {
		t.Fatal("mid-month coverage should flag Partial")
	}
	var buf bytes.Buffer
	if err := writeParkingMonthCSV(&buf, tenantConfig{Name: "Janischhofweg 22", Address: "Janischhofweg 22"}, view); err != nil {
		t.Fatalf("csv: %v", err)
	}
	if !strings.Contains(buf.String(), "unvollständig") {
		t.Fatal("CSV should carry the incomplete-data warning")
	}
}

func TestParkingMonthExportRouteServesCSV(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err := a.parkingStore.UpsertTariff("jhw22", parkingTariff{EffectiveFrom: "2026-01-01", GridFeeEURPerKWh: 0.05, BaseFeeEUR: 3}); err != nil {
		t.Fatalf("tariff: %v", err)
	}
	energy, prices, _ := chargingBillingFixture(t)
	if err := a.parkingStore.AppendReadings("jhw22", energy, prices); err != nil {
		t.Fatalf("seed: %v", err)
	}
	month := energy[0].At.In(time.Local).Format("2006-01")

	res := authedRequest(t, a, "admin@example.com", "/app/parking/month/"+month+"/export")
	if res.Code != http.StatusOK {
		t.Fatalf("export status = %d, want 200", res.Code)
	}
	if ct := res.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("content-type = %q", ct)
	}
	if cd := res.Header().Get("Content-Disposition"); !strings.Contains(cd, "parkplatzabrechnung-"+month+".csv") {
		t.Fatalf("content-disposition = %q", cd)
	}
	if !strings.Contains(res.Body.String(), "Stunde") {
		t.Fatal("CSV should contain the hourly header")
	}
}

func TestParkingMonthExportDeniedForNonParkingUser(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	res := authedRequest(t, a, "resident@example.com", "/app/parking/month/2026-07/export")
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.Code)
	}
}

func TestParkingMonthPageShowsExportPrintAndWarning(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if err := a.parkingStore.UpsertTariff("jhw22", parkingTariff{EffectiveFrom: "2026-01-01", GridFeeEURPerKWh: 0.05, BaseFeeEUR: 3}); err != nil {
		t.Fatalf("tariff: %v", err)
	}
	energy, prices, _ := chargingBillingFixture(t)
	if err := a.parkingStore.AppendReadings("jhw22", energy, prices); err != nil {
		t.Fatalf("seed: %v", err)
	}
	month := energy[0].At.In(time.Local).Format("2006-01")

	page := authedRequest(t, a, "admin@example.com", "/app/parking/month/"+month)
	if page.Code != http.StatusOK {
		t.Fatalf("status = %d", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, "/app/parking/month/"+month+"/export") {
		t.Fatal("month page should link to the CSV export")
	}
	if !strings.Contains(body, "data-print") {
		t.Fatal("month page should offer a print/PDF button")
	}
	if !strings.Contains(body, "Messdaten unvollständig") {
		t.Fatal("partial month should show the data-quality warning")
	}
}
