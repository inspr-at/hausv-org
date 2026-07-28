package energy

import (
	"math"
	"testing"
	"time"
)

func TestQuarterStartUsesCalendarBoundaries(t *testing.T) {
	vienna, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 7, 28, 10, 44, 59, 0, vienna)
	got := QuarterStart(at, vienna)
	want := time.Date(2026, 7, 28, 10, 30, 0, 0, vienna)
	if !got.Equal(want) {
		t.Fatalf("QuarterStart = %v, want %v", got, want)
	}
	if got := AveragePowerKW(4, 15*time.Minute); got != 16 {
		t.Fatalf("AveragePowerKW = %v, want 16", got)
	}
}

func TestClassifyCandidateIsConservativeAndReadOnly(t *testing.T) {
	item, ok := ClassifyCandidate(
		"sensor.grid_import_power",
		"Netzbezug",
		"W",
		"power",
		"measurement",
		"4120",
		time.Now(),
	)
	if !ok || item.Metric != MetricGridImportPower || item.Value == nil || *item.Value != 4120 {
		t.Fatalf("candidate = %+v ok=%v", item, ok)
	}
	if _, ok := ClassifyCandidate("switch.wallbox", "Wallbox", "", "switch", "", "on", time.Now()); ok {
		t.Fatal("a switch must not become a read-only measurement suggestion")
	}
}

func TestMemoryStoreKeepsHousesSeparated(t *testing.T) {
	store := NewMemoryStore()
	for _, tenant := range []string{"home-a", "home-b"} {
		if err := store.SaveProfile(DefaultProfile(tenant, time.Now())); err != nil {
			t.Fatal(err)
		}
		if err := store.UpsertAsset(Asset{ID: "pv", TenantSlug: tenant, Kind: "pv", Confirmed: true}); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := store.ListAssets("home-a")
	b, _ := store.ListAssets("home-b")
	if len(a) != 1 || len(b) != 1 || a[0].TenantSlug == b[0].TenantSlug {
		t.Fatalf("tenant separation failed: a=%+v b=%+v", a, b)
	}
}

func TestPeakForMonthIgnoresOtherMonths(t *testing.T) {
	vienna, _ := time.LoadLocation("Europe/Vienna")
	items := []Interval{
		{StartsAt: time.Date(2026, 7, 1, 10, 0, 0, 0, vienna), AverageKW: 6.8},
		{StartsAt: time.Date(2026, 7, 2, 10, 0, 0, 0, vienna), AverageKW: 4.2},
		{StartsAt: time.Date(2026, 8, 1, 10, 0, 0, 0, vienna), AverageKW: 9.9},
	}
	if got := PeakForMonth(items, time.Date(2026, 7, 28, 0, 0, 0, 0, vienna), vienna); math.Abs(got-6.8) > 0.0001 {
		t.Fatalf("peak = %v, want 6.8", got)
	}
}
