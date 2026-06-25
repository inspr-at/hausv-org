package main

import (
	"math"
	"path/filepath"
	"testing"
	"time"
)

func TestSMTPMailerAllowsInternalRelayWithoutAuth(t *testing.T) {
	m := smtpMailer{
		host: "smtp",
		port: "25",
		from: "WEG Portal <noreply@hausv.org>",
	}

	if !m.Configured() {
		t.Fatal("mailer should be configured with host, port, and from")
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("relay mailer should validate without auth: %v", err)
	}
	if auth := m.auth(); auth != nil {
		t.Fatal("relay mailer without user/pass should not create smtp auth")
	}
}

func TestSMTPMailerRequiresPairedCredentials(t *testing.T) {
	m := smtpMailer{
		host: "smtp",
		port: "25",
		user: "user",
		from: "WEG Portal <noreply@hausv.org>",
	}

	if err := m.Validate(); err == nil {
		t.Fatal("mailer should reject partial smtp credentials")
	}
}

func TestSessionSecretRequiredForPublicBaseURL(t *testing.T) {
	t.Setenv("SESSION_KEY", "")

	if _, err := sessionSecret(true); err == nil {
		t.Fatal("public deployment should require SESSION_KEY")
	}
}

func TestSessionSecretCanBeEphemeralForLocalDev(t *testing.T) {
	t.Setenv("SESSION_KEY", "")

	secret, err := sessionSecret(false)
	if err != nil {
		t.Fatalf("local development should allow generated session secret: %v", err)
	}
	if len(secret) != 32 {
		t.Fatalf("generated session secret length = %d, want 32", len(secret))
	}
}

func TestParkingHourlyUsageAppliesHourlyAwattarPrices(t *testing.T) {
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	energy := []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(2 * time.Hour), Value: 102},
	}
	prices := []parkingNumericSample{
		{At: base, Value: 0.20},
		{At: base.Add(time.Hour), Value: 0.40},
	}

	hours := calculateParkingHourlyUsage(energy, prices, 0.10, base.Add(3*time.Hour))
	if len(hours) != 2 {
		t.Fatalf("hour buckets = %d, want 2", len(hours))
	}
	assertClose(t, hours[0].KWh, 1)
	assertClose(t, hours[0].EnergyCost, 0.20)
	assertClose(t, hours[0].GridCost, 0.10)
	assertClose(t, hours[1].KWh, 1)
	assertClose(t, hours[1].EnergyCost, 0.40)
	assertClose(t, hours[1].GridCost, 0.10)
}

func TestParkingMonthsExposeCostsAndPaidFlag(t *testing.T) {
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	data := parkingTenantData{
		Settings: parkingSettings{GridFeeEURPerKWh: 0.10},
		Months: map[string]parkingMonthState{
			"2026-06": {Paid: true},
		},
		EnergySamples: []parkingNumericSample{
			{At: base, Value: 100},
			{At: base.Add(2 * time.Hour), Value: 102},
		},
		PriceSamples: []parkingNumericSample{
			{At: base, Value: 0.20},
			{At: base.Add(time.Hour), Value: 0.40},
		},
	}

	months := calculateParkingMonths(data, base.Add(3*time.Hour), time.UTC)
	if len(months) != 1 {
		t.Fatalf("months = %d, want 1", len(months))
	}
	month := months[0]
	if month.Month != "2026-06" || !month.Paid {
		t.Fatalf("month state = %+v, want paid 2026-06", month)
	}
	if month.KWh != "2,00 kWh" || month.EnergyCost != "0,60 €" || month.GridCost != "0,20 €" || month.TotalCost != "0,80 €" {
		t.Fatalf("unexpected formatted costs: %+v", month)
	}
	if month.HourCount != 2 {
		t.Fatalf("hour count = %d, want 2", month.HourCount)
	}
}

func TestParkingStorePersistsPaidFlagAndGridFee(t *testing.T) {
	path := filepath.Join(t.TempDir(), "parking.json")
	store, err := newParkingStore(path)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := store.SetGridFee("jhw22", 0.123); err != nil {
		t.Fatalf("set grid fee: %v", err)
	}
	if err := store.SetMonthPaid("jhw22", "2026-06", true); err != nil {
		t.Fatalf("set paid flag: %v", err)
	}

	loaded, err := newParkingStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	data := loaded.TenantData("jhw22")
	assertClose(t, data.Settings.GridFeeEURPerKWh, 0.123)
	if !data.Months["2026-06"].Paid {
		t.Fatal("paid flag was not persisted")
	}
}

func assertClose(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("got %.6f, want %.6f", got, want)
	}
}
