package server

import (
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/energy"
)

// Das Modell darf nie mehr Flexibilität behaupten, als der Haushalt erklärt
// hat. Jede Abweichung hier verspricht Ersparnis, die es nicht gibt — und
// genau das würde den Leistungstarif-Rechner unglaubwürdig machen.

func honestyScenario(t *testing.T, assets []energy.Asset) []energyScenarioView {
	t.Helper()
	return buildEnergyScenarioViews(energy.HomeProfile{TenantSlug: "haus"}, assets, scenarioIntervals(20))
}

func honestyRated(kw float64) *float64 { return &kw }

// Ein ausdrücklich als "fest" erklärter Speicher darf nicht als Flexibilität
// zählen. Vorher stand der Speicher-Zweig vor der Flexibilitätsprüfung und
// bekam mit 0,70 sogar das höchste Gewicht im Modell.
func TestBatteryDeclaredFixedIsNotCreditedHAUSV422(t *testing.T) {
	views := honestyScenario(t, []energy.Asset{
		{ID: "b1", Kind: "battery", Name: "Notstrom", Source: energyCustomAssetSource,
			Flexibility: energy.FlexFixed, RatedPowerKW: honestyRated(10)},
	})
	if len(views) != 0 {
		t.Fatalf("ein fest erklärter Speicher darf keine Spitzenwirkung ergeben: %+v", views)
	}
}

// Erzeugung verschiebt die Bezugsspitze nicht. Eine PV-Anlage mit erfasster
// Nennleistung wurde bisher wie eine abschaltbare Last verrechnet.
func TestPVWithRatedPowerDoesNotShaveThePeakHAUSV422(t *testing.T) {
	views := honestyScenario(t, []energy.Asset{
		{ID: "p1", Kind: "pv", Name: "PV Dach", Source: energyCustomAssetSource,
			Flexibility: energy.FlexShift, RatedPowerKW: honestyRated(12)},
	})
	if len(views) != 0 {
		t.Fatalf("PV darf keine Spitzenwirkung ergeben: %+v", views)
	}
}

// E-Auto und Wallbox sind als Vorlagen dieselbe Ladelast und zählen einmal.
// Welche der beiden überlebt, darf aber nicht von der Sortierung abhängen:
// erhalten bleibt die Angabe mit der tatsächlichen Nennleistung.
func TestChargingPresetKeepsTheInformativeEntryHAUSV422(t *testing.T) {
	views := honestyScenario(t, []energy.Asset{
		{ID: "a1", Kind: "ev", Name: "E-Auto", Source: "profile-seed"},
		{ID: "a2", Kind: "wallbox", Name: "Wallbox 22 kW", Source: "profile-seed",
			Flexibility: energy.FlexShift, RatedPowerKW: honestyRated(22)},
	})
	if len(views) == 0 {
		t.Fatal("erwartet wurde ein Szenario")
	}
	if !strings.Contains(views[0].Assumptions, "Wallbox 22 kW") {
		t.Fatalf("die erfasste Nennleistung muss die Vorbelegung schlagen: %q", views[0].Assumptions)
	}
	if strings.Contains(views[0].Assumptions, "E-Auto") {
		t.Fatalf("die Ladelast darf nur einmal zählen: %q", views[0].Assumptions)
	}
}

// Vorbelegungen dürfen nicht zweimal gepflegt werden: das Seeding schrieb für
// eine Sauna "unknown", das Modell rechnete sie als verschiebbar.
func TestFlexibilityDefaultsHaveOneSourceOfTruthHAUSV422(t *testing.T) {
	for _, kind := range []string{"ev", "wallbox", "hot-water", "sauna", "heat-pump",
		"battery", "air-conditioning", "pv", "other"} {
		if got, want := defaultAssetFlexibility(kind), energy.DefaultAssetFlexibility(kind); got != want {
			t.Fatalf("%s: Modell rechnet %q, Seeding speichert %q", kind, got, want)
		}
	}
}
