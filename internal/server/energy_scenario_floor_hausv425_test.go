package server

import (
	"strings"
	"testing"
	"time"

	"github.com/markus-barta/hausv-org/internal/energy"
)

// Ein Szenario darf keine Ersparnis nahelegen, die unterhalb der
// Mindestbemessung gar nicht mehr eintritt: die Spitze sinkt dort real weiter,
// der verrechnete Betrag nicht.

func scenarioIntervals(peakKW float64) []energy.Interval {
	start := time.Now().Truncate(15 * time.Minute).Add(-time.Hour)
	return []energy.Interval{{
		StartsAt:  start,
		Duration:  15 * time.Minute,
		AverageKW: peakKW,
		Quality:   energy.QualityMeasured,
		Source:    "test",
	}}
}

func scenarioAssets() []energy.Asset {
	return []energy.Asset{
		{ID: "a1", Kind: "wallbox", Name: "Wallbox", Confirmed: true},
		{ID: "a2", Kind: "battery", Name: "Speicher", Confirmed: true},
	}
}

func TestScenarioWarnsWhenItReachesTheBillingFloorHAUSV425(t *testing.T) {
	agreed := 40.0 // 20 % davon sind 8 kW und damit eine hohe Untergrenze
	profile := energy.HomeProfile{TenantSlug: "haus", AgreedPowerKW: &agreed}

	views := buildEnergyScenarioViews(profile, scenarioAssets(), scenarioIntervals(9))
	if len(views) == 0 {
		t.Fatal("erwartet wurde ein Szenario")
	}
	if views[0].FloorNote == "" {
		t.Fatal("unterhalb der Mindestbemessung muss das Szenario den Hinweis tragen")
	}
	if !strings.Contains(views[0].FloorNote, "8") {
		t.Fatalf("der Hinweis muss die Untergrenze nennen, war %q", views[0].FloorNote)
	}
}

func TestScenarioStaysQuietAboveTheBillingFloorHAUSV425(t *testing.T) {
	// Ohne erfasste Anschlussleistung liegt die Untergrenze bei 2 kW; ein
	// Szenario, das dort nicht hinkommt, darf nicht warnen.
	views := buildEnergyScenarioViews(energy.HomeProfile{TenantSlug: "haus"}, scenarioAssets(), scenarioIntervals(30))
	if len(views) == 0 {
		t.Fatal("erwartet wurde ein Szenario")
	}
	if views[0].FloorNote != "" {
		t.Fatalf("oberhalb der Mindestbemessung darf kein Hinweis erscheinen, war %q", views[0].FloorNote)
	}
}
