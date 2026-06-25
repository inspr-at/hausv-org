package main

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"strconv"
	"strings"
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

func TestSignedSessionRoundTripSurvivesNewStore(t *testing.T) {
	secret := []byte(strings.Repeat("s", 32))
	first := newSessionStore(secret)

	token, _, err := first.Put("Markus@Barta.com", "JHW22", authMethodOIDC, time.Hour)
	if err != nil {
		t.Fatalf("put session: %v", err)
	}

	second := newSessionStore(secret)
	email, tenantSlug, authMethod, ok := second.Get(token)
	if !ok {
		t.Fatal("session should verify in a new store with the same secret")
	}
	if email != "markus@barta.com" || tenantSlug != "jhw22" || authMethod != authMethodOIDC {
		t.Fatalf("unexpected session claims: %s %s %s", email, tenantSlug, authMethod)
	}

	other := newSessionStore([]byte(strings.Repeat("x", 32)))
	if _, _, _, ok := other.Get(token); ok {
		t.Fatal("session should not verify with a different secret")
	}
}

func TestParseUserProfilesNormalizesAuthMethods(t *testing.T) {
	raw := `[{"email":"joerg.lehner@gmx.at","first_name":"Jörg","last_name":"Lehner","tenants":["jhw22"],"auth_methods":["zitadel"]}]`

	profiles, err := parseUserProfiles(raw, map[string]struct{}{}, map[string]struct{}{}, "jhw22")
	if err != nil {
		t.Fatalf("parse profiles: %v", err)
	}
	profile := profiles["joerg.lehner@gmx.at"]
	if !profile.AllowsAuthMethod(authMethodOIDC) {
		t.Fatal("profile should allow OIDC")
	}
	if profile.AllowsAuthMethod(authMethodEmail) {
		t.Fatal("profile should not allow email login")
	}
	if got := profile.UserRow().AuthLabel; got != "Zitadel SSO" {
		t.Fatalf("auth label = %q", got)
	}
}

func TestOIDCLoginDefersUnavailableDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	login, err := newOIDCLogin(ctx, "https://auth.invalid.example", "client-id", "", "", "Zitadel")
	if err != nil {
		t.Fatalf("new OIDC login should not fail hard when discovery is unavailable: %v", err)
	}
	if !login.Configured() {
		t.Fatal("OIDC should remain configured so discovery can be retried later")
	}
	if login.provider != nil || login.verifier != nil {
		t.Fatal("provider should not be initialized after canceled discovery")
	}
	if err := login.EnsureProvider(ctx); err == nil {
		t.Fatal("retry with canceled context should still report discovery failure")
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
	if month.AverageAwattar != "0,300 €/kWh" || month.EffectivePrice != "0,400 €/kWh" {
		t.Fatalf("unexpected average prices: %+v", month)
	}
	if month.DetailPath != "/app/parking/month/2026-06" {
		t.Fatalf("detail path = %q", month.DetailPath)
	}
	if month.HourCount != 2 {
		t.Fatalf("hour count = %d, want 2", month.HourCount)
	}
}

func TestParkingMonthDetailsExposeHourlyRows(t *testing.T) {
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	data := parkingTenantData{
		Settings: parkingSettings{GridFeeEURPerKWh: 0.10},
		EnergySamples: []parkingNumericSample{
			{At: base, Value: 100},
			{At: base.Add(2 * time.Hour), Value: 102},
		},
		PriceSamples: []parkingNumericSample{
			{At: base, Value: 0.20},
			{At: base.Add(time.Hour), Value: 0.40},
		},
	}

	detail := calculateParkingMonthDetails(data, "2026-06", base.Add(3*time.Hour), time.UTC)
	if !detail.HasHours || len(detail.Hours) != 2 {
		t.Fatalf("detail hours = %d, has=%v; want 2 true", len(detail.Hours), detail.HasHours)
	}
	if detail.Summary.TotalCost != "0,80 €" || detail.Summary.AverageAwattar != "0,300 €/kWh" {
		t.Fatalf("unexpected detail summary: %+v", detail.Summary)
	}
	if detail.Hours[0].AtLabel != "25.06. 10:00" || detail.Hours[0].TotalCost != "0,30 €" {
		t.Fatalf("unexpected first hour: %+v", detail.Hours[0])
	}
	if detail.Hours[0].KWhTitle != "Verbrauch: 1,000000 kWh" {
		t.Fatalf("unexpected kWh title: %q", detail.Hours[0].KWhTitle)
	}
	if detail.Hours[0].EnergyCostTitle != "Stromkosten: 0,200000 € = 1,000000 kWh × 0,200000 €/kWh" {
		t.Fatalf("unexpected energy title: %q", detail.Hours[0].EnergyCostTitle)
	}
	if detail.Hours[0].TotalCostTitle != "Summe: 0,300000 € = Strom 0,200000 € + Delta 0,100000 €" {
		t.Fatalf("unexpected total title: %q", detail.Hours[0].TotalCostTitle)
	}
	if detail.Hours[1].AtLabel != "25.06. 11:00" || detail.Hours[1].TotalCost != "0,50 €" {
		t.Fatalf("unexpected second hour: %+v", detail.Hours[1])
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

func TestSamplesFromStatisticsUsesMillisecondsAndPreferredFields(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	state := 123.45
	sum := 456.78
	mean := 0.31
	stats := []haStatistic{
		{Start: json.RawMessage(strconvFormatInt(start.UnixMilli())), State: &state, Sum: &sum, Mean: &mean},
	}

	energy := samplesFromStatistics(stats, "state", "sum")
	if len(energy) != 1 {
		t.Fatalf("energy samples = %d, want 1", len(energy))
	}
	if !energy[0].At.Equal(start) {
		t.Fatalf("sample time = %s, want %s", energy[0].At, start)
	}
	assertClose(t, energy[0].Value, state)

	price := samplesFromStatistics(stats, "mean")
	if len(price) != 1 {
		t.Fatalf("price samples = %d, want 1", len(price))
	}
	assertClose(t, price[0].Value, mean)
}

func TestSamplesFromStatisticsFallsBackWhenColumnsAreOmitted(t *testing.T) {
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	sum := 44.5
	stats := []haStatistic{
		{Start: json.RawMessage(`"` + start.Format(time.RFC3339) + `"`), Sum: &sum},
	}

	samples := samplesFromStatistics(stats, "state", "sum")
	if len(samples) != 1 {
		t.Fatalf("samples = %d, want 1", len(samples))
	}
	assertClose(t, samples[0].Value, sum)
}

func TestParseHistoryStartDefaultsToCurrentYear(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	start, err := parseHistoryStart("", now)
	if err != nil {
		t.Fatalf("parse history start: %v", err)
	}
	if start.Year() != 2026 || start.Month() != time.January || start.Day() != 1 {
		t.Fatalf("history start = %s, want first day of 2026", start)
	}
}

func assertClose(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("got %.6f, want %.6f", got, want)
	}
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
