package energy_test

import (
	"testing"
	"time"

	"github.com/markus-barta/hausv-org/internal/energy"
)

// HAUSV-428: Der Sampler schreibt dieselbe Viertelstunde nach einem Neustart
// erneut. Das muss auf dem echten Speicher idempotent sein und darf Häuser
// nicht vermischen — beides wird hier gegen SQLite und den Speicher geprüft.
func TestSampledQuarterHourIsIdempotentPerHouseHAUSV428(t *testing.T) {
	// Im laufenden Kalendermonat verankert: PeakForMonth filtert nach Monat.
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	start := energy.QuarterStart(now, time.Local).Add(-energy.QuarterDuration)
	if start.Before(monthStart) {
		start = monthStart
	}
	for name, factory := range lifecycleStoreFactories() {
		t.Run(name, func(t *testing.T) {
			storage := factory(t)
			sampled := func(tenant string, kw float64) energy.Interval {
				return energy.Interval{
					TenantSlug: tenant,
					StartsAt:   start.UTC(),
					Duration:   energy.QuarterDuration,
					ImportKWh:  kw / 4,
					AverageKW:  kw,
					Quality:    energy.QualityEstimated,
					Source:     energy.SourceHomeAssistant,
				}
			}
			for _, interval := range []energy.Interval{
				sampled("home-a", 4.8),
				sampled("home-a", 4.8),
				sampled("home-b", 11.2),
			} {
				if err := storage.PutInterval(interval); err != nil {
					t.Fatalf("PutInterval: %v", err)
				}
			}
			// Der Smart-Meter-Export derselben Viertelstunde bleibt eine eigene
			// Zeile: die beiden Quellen werden verglichen, nie verschmolzen.
			reference := sampled("home-a", 5.1)
			reference.Source = energy.SourceSmartMeter
			reference.Quality = energy.QualityMeasured
			if err := storage.PutInterval(reference); err != nil {
				t.Fatalf("PutInterval Referenz: %v", err)
			}

			first, err := storage.ListIntervals("home-a", time.Time{}, time.Time{})
			if err != nil {
				t.Fatalf("ListIntervals: %v", err)
			}
			if len(first) != 2 {
				t.Fatalf("home-a: %d Zeilen, erwartet je eine pro Quelle: %+v", len(first), first)
			}
			second, err := storage.ListIntervals("home-b", time.Time{}, time.Time{})
			if err != nil {
				t.Fatalf("ListIntervals: %v", err)
			}
			if len(second) != 1 || second[0].AverageKW != 11.2 {
				t.Fatalf("home-b: %+v", second)
			}
			if peak := energy.PeakForMonth(second, now, time.Local); peak != 11.2 {
				t.Fatalf("home-b Monatsspitze = %v kW", peak)
			}
			comparison, ok := energy.CompareMonthlyPeaks(first, now, time.Local,
				energy.SourceSmartMeter, energy.SourceHomeAssistant)
			if !ok || comparison.ReferencePeakKW != 5.1 || comparison.ComparedPeakKW != 4.8 {
				t.Fatalf("Quellenvergleich = %+v ok=%v", comparison, ok)
			}
		})
	}
}
