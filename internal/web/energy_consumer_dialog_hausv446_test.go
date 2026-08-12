package web

import (
	"strings"
	"testing"
)

func TestEnergyConsumerCardsUseLocalLucideIconsHAUSV446(t *testing.T) {
	scriptBytes, err := Assets.ReadFile("assets/energy-flow.js")
	if err != nil {
		t.Fatalf("energy-flow.js lesen: %v", err)
	}
	script := string(scriptBytes)
	for _, want := range []string{
		`"solar-panel", "battery", "utility-pole", "house"`,
		`"car-front"`, `"plug-zap"`, `"grip-vertical"`, `"pencil"`,
		`energy-ui-icon-`, `openConsumerDialog(c`, `Klicken zum Bearbeiten`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("Lucide-/Dialog-Vertrag fehlt %q", want)
		}
	}
	for _, obsolete := range []string{`var ICONS =`, `ICONS.grip`, `location.hash = "anlagen"`, `className = "tile-actions"`} {
		if strings.Contains(script, obsolete) {
			t.Errorf("alte Inline-SVG-/Sprungnavigation ist noch vorhanden: %q", obsolete)
		}
	}
}

func TestCuratedConsumerLucideAssetsAreVendoredHAUSV446(t *testing.T) {
	for _, name := range []string{
		"car-front", "plug-zap", "heater", "fan", "washing-machine", "flame",
		"drill", "waves-ladder", "square-parking", "snowflake", "shower-head",
		"plug", "grip-vertical", "plus", "search", "x",
	} {
		contents, err := Assets.ReadFile("assets/icons/lucide/" + name + ".svg")
		if err != nil {
			t.Errorf("lokales Lucide-Asset %s fehlt: %v", name, err)
			continue
		}
		if !strings.Contains(string(contents), `<svg`) {
			t.Errorf("Lucide-Asset %s ist kein SVG", name)
		}
	}
}

func TestEnergyConsumerCardHoverCannotChangeGeometryHAUSV446(t *testing.T) {
	for _, want := range []string{
		`.energy-flow-big { height: 60px; box-sizing: border-box;`,
		`.energy-flow-subtitles > span { grid-area: 1 / 1;`,
		`.energy-flow-big.editable:hover .energy-flow-icon-default`,
		`.energy-flow-big.editable:hover .energy-flow-icon-edit`,
		`id="energy-consumer-dialog"`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Errorf("sprungfreier Verbraucher-Vertrag fehlt %q", want)
		}
	}
}
