package server

import (
	"math"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

// PP20 surplus billing: surplus-session kWh bill at the flat rate and are
// excluded from spot+grid; everything else must compute exactly as before.

func chargingBillingFixture(t *testing.T) ([]parkingNumericSample, []parkingNumericSample, time.Time) {
	t.Helper()
	base := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	energy := []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(90 * time.Minute), Value: 105},  // 11:30 session start boundary
		{At: base.Add(150 * time.Minute), Value: 112}, // 12:30 session end boundary
		{At: base.Add(180 * time.Minute), Value: 114}, // 13:00
	}
	prices := []parkingNumericSample{{At: base.Add(-time.Hour), Value: 0.20}}
	now := base.Add(6 * time.Hour)
	return energy, prices, now
}

func sumHours(hours []parkingHourUsage) (kWh, surplusKWh, energyCost, gridCost, surplusCost float64) {
	for _, hour := range hours {
		kWh += hour.KWh
		surplusKWh += hour.SurplusKWh
		energyCost += hour.EnergyCost
		gridCost += hour.GridCost
		surplusCost += hour.SurplusCost
	}
	return
}

func approx(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func TestSurplusSplitExactAtBoundarySamples(t *testing.T) {
	energy, prices, now := chargingBillingFixture(t)
	settings := parkingSettings{Tariffs: []parkingTariff{{
		EffectiveFrom:    "2026-01-01",
		GridFeeEURPerKWh: 0.05,
	}}}
	sessions := []chargingSession{{
		ID:       "s1",
		Start:    energy[1].At,
		End:      energy[2].At,
		StartKWh: 105,
		EndKWh:   112,
		Mode:     store.ChargingModeSurplus,
	}}
	hours := calculateParkingHourlyUsageWithSettings(energy, prices, sessions, settings, now, time.UTC)
	kWh, surplusKWh, energyCost, gridCost, surplusCost := sumHours(hours)
	approx(t, "total kWh", kWh, 14)
	// Boundary samples make the surplus share telescope to EndKWh-StartKWh.
	approx(t, "surplus kWh", surplusKWh, 7)
	approx(t, "surplus cost", surplusCost, 7*store.DefaultSurplusRateEURPerKWh)
	approx(t, "energy cost", energyCost, 7*0.20)
	approx(t, "grid cost", gridCost, 7*0.05)
}

func TestNoSessionsKeepsLegacyFormulas(t *testing.T) {
	energy, prices, now := chargingBillingFixture(t)
	settings := parkingSettings{Tariffs: []parkingTariff{{
		EffectiveFrom:    "2026-01-01",
		GridFeeEURPerKWh: 0.05,
	}}}
	hours := calculateParkingHourlyUsageWithSettings(energy, prices, nil, settings, now, time.UTC)
	if len(hours) == 0 {
		t.Fatal("expected hours")
	}
	for _, hour := range hours {
		if hour.SurplusKWh != 0 || hour.SurplusCost != 0 {
			t.Fatalf("no sessions must yield zero surplus, got %+v", hour)
		}
		approx(t, "energy cost", hour.EnergyCost, hour.PriceEUR*hour.KWh)
		approx(t, "grid cost", hour.GridCost, 0.05*hour.KWh)
	}
}

func TestManualSessionsBillAtNormalTariff(t *testing.T) {
	energy, prices, now := chargingBillingFixture(t)
	settings := parkingSettings{GridFeeEURPerKWh: 0.05}
	sessions := []chargingSession{{
		ID:    "m1",
		Start: energy[1].At,
		End:   energy[2].At,
		Mode:  store.ChargingModeManual,
	}}
	hours := calculateParkingHourlyUsageWithSettings(energy, prices, sessions, settings, now, time.UTC)
	_, surplusKWh, _, _, surplusCost := sumHours(hours)
	if surplusKWh != 0 || surplusCost != 0 {
		t.Fatalf("manual sessions must not bill as surplus: kWh=%v cost=%v", surplusKWh, surplusCost)
	}
}

func TestOpenSurplusSessionCountsUntilNow(t *testing.T) {
	energy, prices, _ := chargingBillingFixture(t)
	// Session opened at the second sample and never closed; "now" is after the
	// last sample, so everything from 11:30 on is surplus.
	now := energy[len(energy)-1].At.Add(time.Hour)
	settings := parkingSettings{GridFeeEURPerKWh: 0.05}
	sessions := []chargingSession{{
		ID:    "open",
		Start: energy[1].At,
		Mode:  store.ChargingModeSurplus,
	}}
	hours := calculateParkingHourlyUsageWithSettings(energy, prices, sessions, settings, now, time.UTC)
	_, surplusKWh, _, _, _ := sumHours(hours)
	approx(t, "surplus kWh", surplusKWh, 9) // 105 -> 114
}

func TestCalculateParkingMonthsIncludesSurplusInTotal(t *testing.T) {
	energy, prices, now := chargingBillingFixture(t)
	data := parkingTenantData{
		Settings: parkingSettings{Tariffs: []parkingTariff{{
			EffectiveFrom:    "2026-01-01",
			GridFeeEURPerKWh: 0.05,
			BaseFeeEUR:       3,
		}}},
		EnergySamples: energy,
		PriceSamples:  prices,
		ChargingSessions: []chargingSession{{
			ID:       "s1",
			Start:    energy[1].At,
			End:      energy[2].At,
			StartKWh: 105,
			EndKWh:   112,
			Mode:     store.ChargingModeSurplus,
		}},
	}
	months := calculateParkingMonths(data, now, time.UTC)
	if len(months) != 1 {
		t.Fatalf("expected 1 month, got %d", len(months))
	}
	month := months[0]
	if !month.HasSurplus {
		t.Fatal("month must flag surplus")
	}
	approx(t, "surplus kWh", month.SurplusKWhValue, 7)
	approx(t, "normal kWh", month.NormalKWhValue, 7)
	wantTotal := 7*0.20 + 7*0.05 + 7*store.DefaultSurplusRateEURPerKWh + 3
	approx(t, "total", month.TotalCostValue, wantTotal)
	// Average aWATTar refers to normally billed energy only.
	approx(t, "avg awattar", month.EnergyCostValue/month.NormalKWhValue, 0.20)
}
