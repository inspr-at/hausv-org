package energy

import (
	"math"
	"testing"
	"time"
)

// approxHAUSV428 vergleicht Leistungen mit der Toleranz, die eine
// stückweise Integration über 30 Segmente an Rundungsfehler hinterlässt.
func approxHAUSV428(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

// HAUSV-428: Aus laufend gelesenen Momentanleistungen müssen abgeschlossene
// Viertelstunden entstehen — aber nur dann, wenn die Datenlage sie trägt.

// quarterAnchorHAUSV428 verankert jede Prüfung im laufenden Kalendermonat.
// PeakForMonth filtert nach Kalendermonat; ein fest verdrahtetes Datum stirbt
// am Monatswechsel.
func quarterAnchorHAUSV428() time.Time {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	anchor := QuarterStart(now, time.Local).Add(-QuarterDuration)
	if anchor.Before(monthStart) {
		return monthStart
	}
	return anchor
}

// samplesHAUSV428 erzeugt gleichmäßige Abtastungen konstanter Leistung.
func samplesHAUSV428(start time.Time, interval time.Duration, count int, kw float64) []PowerSample {
	out := make([]PowerSample, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, PowerSample{At: start.Add(time.Duration(i) * interval), KW: kw})
	}
	return out
}

func TestCompleteQuarterHourIsEstimatedNotMeasuredHAUSV428(t *testing.T) {
	start := quarterAnchorHAUSV428()
	interval, complete := QuarterFromPowerSamples(QuarterInput{
		TenantSlug: "jhw22",
		Source:     SourceHomeAssistant,
		Start:      start,
		Now:        start.Add(QuarterDuration),
		Location:   time.Local,
		Samples:    samplesHAUSV428(start, 30*time.Second, 30, 4.8),
		Policy:     DefaultSamplingPolicy(30 * time.Second),
	})
	if !complete {
		t.Fatal("eine vergangene Viertelstunde muss abgeschlossen sein")
	}
	if !approxHAUSV428(interval.AverageKW, 4.8) || !approxHAUSV428(interval.ImportKWh, 1.2) {
		t.Fatalf("Mittelwert/Arbeit = %v kW / %v kWh, erwartet 4,8 kW / 1,2 kWh", interval.AverageKW, interval.ImportKWh)
	}
	// Zwischen zwei Abtastungen bleibt die Leistung angenommen. QualityMeasured
	// wäre die Aussage einer Zählerablesung und damit zu stark.
	if interval.Quality != QualityEstimated {
		t.Fatalf("Qualität = %q, erwartet %q", interval.Quality, QualityEstimated)
	}
	if interval.Source != SourceHomeAssistant || interval.Duration != QuarterDuration {
		t.Fatalf("Quelle/Dauer = %q / %v", interval.Source, interval.Duration)
	}
	if !interval.StartsAt.Equal(start.UTC()) {
		t.Fatalf("Beginn = %s, erwartet %s", interval.StartsAt, start.UTC())
	}
}

func TestRunningQuarterHourIsNeverWrittenHAUSV428(t *testing.T) {
	start := quarterAnchorHAUSV428()
	for _, now := range []time.Time{
		start,
		start.Add(14 * time.Minute),
		start.Add(QuarterDuration - time.Nanosecond),
	} {
		if _, complete := QuarterFromPowerSamples(QuarterInput{
			TenantSlug: "jhw22",
			Start:      start,
			Now:        now,
			Location:   time.Local,
			Samples:    samplesHAUSV428(start, 30*time.Second, 28, 4.8),
			Policy:     DefaultSamplingPolicy(30 * time.Second),
		}); complete {
			t.Fatalf("angefangene Viertelstunde bei %s wurde als abgeschlossen gemeldet", now)
		}
	}
}

func TestQuarterHourWithHoleDegradesInsteadOfInventingHAUSV428(t *testing.T) {
	start := quarterAnchorHAUSV428()
	// Zehn Minuten Loch in der Mitte: fünf Minuten gemessen, dann nichts mehr.
	samples := samplesHAUSV428(start, 30*time.Second, 11, 9.0)
	interval, complete := QuarterFromPowerSamples(QuarterInput{
		TenantSlug: "jhw22",
		Start:      start,
		Now:        start.Add(QuarterDuration),
		Location:   time.Local,
		Samples:    samples,
		Policy:     DefaultSamplingPolicy(30 * time.Second),
	})
	if !complete {
		t.Fatal("die Viertelstunde ist vorbei")
	}
	if interval.Quality != QualityGap {
		t.Fatalf("Qualität = %q, erwartet %q", interval.Quality, QualityGap)
	}
	if interval.AverageKW != 0 || interval.ImportKWh != 0 {
		t.Fatalf("aus einer Messlücke darf kein Wert entstehen: %+v", interval)
	}
	// Und der erfundene Wert wäre hier besonders teuer gewesen: über die
	// gemessenen Minuten lag die Leistung bei 9 kW.
	if PeakForMonth([]Interval{interval}, start, time.Local) != 0 {
		t.Fatal("eine Messlücke darf keine Monatsspitze setzen")
	}
}

func TestFrozenReadingsAreNamedStaleHAUSV428(t *testing.T) {
	start := quarterAnchorHAUSV428()
	interval, complete := QuarterFromPowerSamples(QuarterInput{
		TenantSlug:    "jhw22",
		Start:         start,
		Now:           start.Add(QuarterDuration),
		Location:      time.Local,
		StaleReadings: 30,
		Policy:        DefaultSamplingPolicy(30 * time.Second),
	})
	if !complete || interval.Quality != QualityStale {
		t.Fatalf("Qualität = %q, erwartet %q", interval.Quality, QualityStale)
	}
	if interval.AverageKW != 0 {
		t.Fatalf("eingefrorene Werte dürfen keine Leistung behaupten: %+v", interval)
	}
}

func TestSingleMissedSampleStaysUsableHAUSV428(t *testing.T) {
	start := quarterAnchorHAUSV428()
	samples := samplesHAUSV428(start, 30*time.Second, 30, 6.0)
	// Ein einzelner Aussetzer (eine Abtastung fehlt) darf eine sonst
	// vollständige Viertelstunde nicht verwerfen — sonst bliebe das Cockpit auf
	// einem echten Haus dauerhaft leer.
	dropped := append(append([]PowerSample{}, samples[:10]...), samples[11:]...)
	interval, complete := QuarterFromPowerSamples(QuarterInput{
		TenantSlug: "jhw22",
		Start:      start,
		Now:        start.Add(QuarterDuration),
		Location:   time.Local,
		Samples:    dropped,
		Policy:     DefaultSamplingPolicy(30 * time.Second),
	})
	if !complete || interval.Quality != QualityEstimated || !approxHAUSV428(interval.AverageKW, 6) {
		t.Fatalf("einzelner Aussetzer = %+v", interval)
	}
}

func TestCoverageCountsOnlyBackedTimeHAUSV428(t *testing.T) {
	start := quarterAnchorHAUSV428()
	end := start.Add(QuarterDuration)
	if got := PowerSampleCoverage(nil, start, end, time.Minute); got != 0 {
		t.Fatalf("ohne Messwerte = %v, erwartet 0", got)
	}
	// Ein einzelner Messwert belegt höchstens maxHold, nicht die ganze
	// Viertelstunde. Genau das verhindert, dass ein Messwert je Viertelstunde
	// als vollständige Messung durchgeht.
	single := PowerSampleCoverage([]PowerSample{{At: start, KW: 3}}, start, end, time.Minute)
	if single < 0.066 || single > 0.067 {
		t.Fatalf("einzelner Messwert deckt %v ab, erwartet eine Minute von fünfzehn", single)
	}
	full := PowerSampleCoverage(samplesHAUSV428(start, 30*time.Second, 30, 3), start, end, time.Minute)
	if full != 1 {
		t.Fatalf("lückenlose Abtastung deckt %v ab, erwartet 1", full)
	}
	// Ein Messwert kurz vor Beginn belegt den Anfang mit; ein alter nicht.
	carried := PowerSampleCoverage([]PowerSample{{At: start.Add(-15 * time.Second), KW: 3}}, start, end, time.Minute)
	if carried <= 0 {
		t.Fatal("der Vorhalt vor Beginn muss den Anfang der Viertelstunde belegen")
	}
	old := PowerSampleCoverage([]PowerSample{{At: start.Add(-10 * time.Minute), KW: 3}}, start, end, time.Minute)
	if old != 0 {
		t.Fatalf("ein zehn Minuten alter Messwert deckt %v ab, erwartet 0", old)
	}
}
