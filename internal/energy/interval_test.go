package energy

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestAggregatePowerIntervalsUsesQuarterBoundaries(t *testing.T) {
	location := mustVienna(t)
	from := time.Date(2026, time.July, 28, 10, 0, 0, 0, location)
	samples := []PowerSample{
		{At: from, KW: 4},
		{At: from.Add(5 * time.Minute), KW: 8},
		{At: from.Add(20 * time.Minute), KW: 2},
	}
	intervals := AggregatePowerIntervals("home", samples, from, from.Add(30*time.Minute), location, "ha")
	if len(intervals) != 2 {
		t.Fatalf("intervals = %+v", intervals)
	}
	// First quarter: 5 min × 4 kW + 10 min × 8 kW = 6.666… kW.
	if math.Abs(intervals[0].AverageKW-6.666667) > 0.00001 {
		t.Fatalf("first average = %f", intervals[0].AverageKW)
	}
	if intervals[0].StartsAt.In(location).Minute() != 0 || intervals[1].StartsAt.In(location).Minute() != 15 {
		t.Fatalf("quarter boundaries = %+v", intervals)
	}
}

func TestAggregateIntervalsPreservesAutumnRepeatedHour(t *testing.T) {
	location := mustVienna(t)
	// 2026-10-25 00:00Z through 02:00Z spans both local 02:00 hours.
	from := time.Date(2026, time.October, 25, 0, 0, 0, 0, time.UTC)
	to := from.Add(2 * time.Hour)
	intervals := AggregatePowerIntervals("home", []PowerSample{{At: from, KW: 1}}, from, to, location, "ha")
	if len(intervals) != 8 {
		t.Fatalf("repeated-hour intervals = %d, want 8", len(intervals))
	}
	labels := map[string]int{}
	for _, interval := range intervals {
		labels[interval.StartsAt.In(location).Format("15:04 -07:00")]++
	}
	if labels["02:00 +02:00"] != 1 || labels["02:00 +01:00"] != 1 {
		t.Fatalf("DST labels = %#v", labels)
	}
}

func TestAggregateEnergyIntervalsDetectsResetAndInterpolates(t *testing.T) {
	location := mustVienna(t)
	from := time.Date(2026, time.July, 28, 10, 0, 0, 0, location)
	ok := AggregateEnergyIntervals("home", []EnergySample{
		{At: from.Add(-5 * time.Minute), KWh: 100},
		{At: from.Add(20 * time.Minute), KWh: 101},
	}, from, from.Add(15*time.Minute), location, "meter")
	if len(ok) != 1 || ok[0].Quality != QualityMeasured || math.Abs(ok[0].ImportKWh-0.6) > 0.00001 {
		t.Fatalf("interpolated interval = %+v", ok)
	}
	reset := AggregateEnergyIntervals("home", []EnergySample{
		{At: from, KWh: 100},
		{At: from.Add(15 * time.Minute), KWh: 1},
	}, from, from.Add(15*time.Minute), location, "meter")
	if len(reset) != 1 || reset[0].Quality != QualityConflict {
		t.Fatalf("reset interval = %+v", reset)
	}
}

func TestSmartMeterCSVIsIdempotentAndTenantScoped(t *testing.T) {
	location := mustVienna(t)
	input := "timestamp;import_kwh\n2026-07-28T10:00:00+02:00;0,50\n2026-07-28T10:15:00+02:00;0,25\n"
	record, intervals, err := ParseSmartMeterCSV(strings.NewReader(input), "home-a", "export.csv", location)
	if err != nil {
		t.Fatalf("ParseSmartMeterCSV: %v", err)
	}
	if len(intervals) != 2 || intervals[0].AverageKW != 2 {
		t.Fatalf("intervals = %+v", intervals)
	}
	store := NewMemoryStore()
	inserted, err := store.PutImport(record, intervals)
	if err != nil || !inserted {
		t.Fatalf("first PutImport inserted=%v err=%v", inserted, err)
	}
	inserted, err = store.PutImport(record, intervals)
	if err != nil || inserted {
		t.Fatalf("second PutImport inserted=%v err=%v", inserted, err)
	}
	if other, _ := store.ListIntervals("home-b", time.Time{}, time.Time{}); len(other) != 0 {
		t.Fatalf("cross-tenant intervals = %+v", other)
	}
}

func TestSmartMeterCSVRejectsUnknownAndNonQuarterFormats(t *testing.T) {
	location := mustVienna(t)
	if _, _, err := ParseSmartMeterCSV(strings.NewReader("foo;bar\nx;y\n"), "home", "bad.csv", location); err == nil {
		t.Fatal("expected unknown format error")
	}
	if _, _, err := ParseSmartMeterCSV(strings.NewReader("timestamp;import_kwh\n2026-07-28 10:07;1\n"), "home", "bad.csv", location); err == nil {
		t.Fatal("expected quarter alignment error")
	}
}

func TestCompareMonthlyPeaksKeepsSourcesSeparateAndFlagsDifference(t *testing.T) {
	location := mustVienna(t)
	at := time.Date(2026, 7, 28, 12, 0, 0, 0, location)
	intervals := []Interval{
		{StartsAt: time.Date(2026, 7, 10, 8, 0, 0, 0, location), AverageKW: 10, Source: "smart-meter", Quality: QualityMeasured},
		{StartsAt: time.Date(2026, 7, 10, 8, 0, 0, 0, location), AverageKW: 12, Source: "home-assistant", Quality: QualityMeasured},
	}
	comparison, ok := CompareMonthlyPeaks(intervals, at, location, "smart-meter", "home-assistant")
	if !ok {
		t.Fatal("comparison unavailable")
	}
	if comparison.DeltaKW != 2 || comparison.DeltaPercent != 20 || comparison.Status != "check" {
		t.Fatalf("comparison = %+v", comparison)
	}
}

func TestCountUsableQuartersKeepsMeasuredAndEstimatedDistinct(t *testing.T) {
	counts := CountUsableQuarters([]Interval{
		{Quality: QualityMeasured},
		{Quality: QualityEstimated},
		{Quality: QualityEstimated},
		{Quality: QualityGap},
		{Quality: QualityConflict},
		{Quality: QualityStale},
	})
	if counts.Total != 3 || counts.Measured != 1 || counts.Estimated != 2 {
		t.Fatalf("usable quarter counts = %+v", counts)
	}
}

func mustVienna(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	return location
}
