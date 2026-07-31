package server

import (
	"strings"
	"testing"

	"github.com/markus-barta/hausv-org/internal/energy"
)

func scenarioOfHAUSV422(t *testing.T, assets []energy.Asset) energyScenarioView {
	t.Helper()
	views := buildEnergyScenarioViews(energy.HomeProfile{TenantSlug: "haus"}, assets, scenarioIntervals(20))
	if len(views) == 0 {
		t.Fatal("erwartet wurde ein Szenario")
	}
	return views[0]
}

func ratedHAUSV422(kw float64) *float64 { return &kw }

// Zwei gleichartige Verbraucher müssen doppelt zählen. Vorher verschmolz
// `has[kind]` sie zu einem einzigen Beitrag.
func TestTwoSameKindConsumersCountTwiceHAUSV422(t *testing.T) {
	one := scenarioOfHAUSV422(t, []energy.Asset{
		{ID: "s1", Kind: "sauna", Name: "Sauna A", Source: energyCustomAssetSource,
			Flexibility: energy.FlexShift, RatedPowerKW: ratedHAUSV422(8)},
	})
	two := scenarioOfHAUSV422(t, []energy.Asset{
		{ID: "s1", Kind: "sauna", Name: "Sauna A", Source: energyCustomAssetSource,
			Flexibility: energy.FlexShift, RatedPowerKW: ratedHAUSV422(8)},
		{ID: "s2", Kind: "sauna", Name: "Sauna B", Source: energyCustomAssetSource,
			Flexibility: energy.FlexShift, RatedPowerKW: ratedHAUSV422(8)},
	})
	if one.PeakBand == two.PeakBand {
		t.Fatalf("zwei Verbraucher ergeben dieselbe Wirkung wie einer: %q", one.PeakBand)
	}
	if !strings.Contains(two.Assumptions, "Sauna A") || !strings.Contains(two.Assumptions, "Sauna B") {
		t.Fatalf("beide Verbraucher müssen in den Annahmen stehen: %q", two.Assumptions)
	}
}

// E-Auto und Wallbox sind als Vorlagen dieselbe Ladelast. Beide anzurechnen
// würde Flexibilität erfinden und damit Ersparnis versprechen, die es nicht
// gibt — der häufigste Seed nutzt genau diese Kombination.
func TestEvAndWallboxPresetsCountOnceHAUSV422(t *testing.T) {
	wallboxOnly := scenarioOfHAUSV422(t, []energy.Asset{
		{ID: "a1", Kind: "wallbox", Name: "Wallbox", Source: "onboarding"},
	})
	both := scenarioOfHAUSV422(t, []energy.Asset{
		{ID: "a1", Kind: "wallbox", Name: "Wallbox", Source: "onboarding"},
		{ID: "a2", Kind: "ev", Name: "E-Auto", Source: "onboarding"},
	})
	if wallboxOnly.PeakBand != both.PeakBand {
		t.Fatalf("Vorlage E-Auto + Wallbox wurde doppelt gezählt: %q gegen %q",
			wallboxOnly.PeakBand, both.PeakBand)
	}
}

// Die Nennleistung des Assets schlägt die Vorbelegung der Kategorie.
func TestRatedPowerBeatsKindDefaultHAUSV422(t *testing.T) {
	withDefault := scenarioOfHAUSV422(t, []energy.Asset{
		{ID: "a1", Kind: "heat-pump", Name: "Wärmepumpe", Source: "onboarding"},
	})
	withRated := scenarioOfHAUSV422(t, []energy.Asset{
		{ID: "a1", Kind: "heat-pump", Name: "Wärmepumpe", Source: "onboarding",
			RatedPowerKW: ratedHAUSV422(9)},
	})
	if withDefault.PeakBand == withRated.PeakBand {
		t.Fatalf("die gesetzte Nennleistung wirkte nicht: beide %q", withDefault.PeakBand)
	}
}

// Ein Speicher bleibt eine eigene Rolle mit eigenem Gewicht und darf nicht als
// drosselbare Last verrechnet werden.
func TestBatteryKeepsItsOwnRoleHAUSV422(t *testing.T) {
	asBattery := scenarioOfHAUSV422(t, []energy.Asset{
		{ID: "b1", Kind: "battery", Name: "Speicher", Source: "onboarding"},
	})
	asThrottle := scenarioOfHAUSV422(t, []energy.Asset{
		{ID: "t1", Kind: "heat-pump", Name: "Wärmepumpe", Source: "onboarding",
			RatedPowerKW: ratedHAUSV422(3)},
	})
	if asBattery.PeakBand == asThrottle.PeakBand {
		t.Fatalf("Speicher und drosselbare Last wirken identisch: %q", asBattery.PeakBand)
	}
}

// PV verschiebt die Bezugsspitze nicht und darf keine Flexibilität vortäuschen.
func TestPVAloneYieldsNoScenarioHAUSV422(t *testing.T) {
	views := buildEnergyScenarioViews(energy.HomeProfile{TenantSlug: "haus"},
		[]energy.Asset{{ID: "p1", Kind: "pv", Name: "PV-Anlage", Source: "onboarding"}},
		scenarioIntervals(20))
	if len(views) != 0 {
		t.Fatalf("PV allein darf kein Peak-Szenario ergeben: %+v", views)
	}
}
