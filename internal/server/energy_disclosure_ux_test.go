package server

import (
	"strings"
	"testing"

	"github.com/markus-barta/hausv-org/internal/energy"
)

func TestEnergyObservationProgressUsesMeasuredQuarterHours(t *testing.T) {
	intervals := make([]energy.Interval, 80)
	for index := range intervals {
		intervals[index].Quality = energy.QualityMeasured
	}
	intervals[len(intervals)-1].Quality = energy.QualityEstimated

	got := buildEnergyObservationProgressView(intervals)
	if got.Completed != 79 || got.Target != 96 || got.Percent != 82 ||
		got.Label != "79 von 96 Viertelstunden" ||
		got.Title != "Noch 4 Std. 15 Min. beobachten" {
		t.Fatalf("unexpected honest observation progress: %+v", got)
	}

	empty := buildEnergyObservationProgressView(nil)
	if empty.Completed != 0 || empty.Percent != 0 ||
		empty.Title != "Einen vollständigen Tag beobachten" {
		t.Fatalf("unexpected empty observation progress: %+v", empty)
	}
}

func TestEnergyFirstViewportDisclosuresRenderDynamicContextAndRealActions(t *testing.T) {
	a := energyCockpitAppHAUSV425(t, 16, "")
	body := authedRequest(t, a, "owner@example.com", "/app/energie").Body.String()

	for _, want := range []string{
		`Noch keine Live-Werte`,
		`href="/app/zuhause/onboarding?step=4"`,
		`data-energy-disclosure="tariff"`,
		`data-energy-help="peak"`,
		`data-energy-help="billed"`,
		`data-energy-help="annual"`,
		`Höchste mittlere Bezugsleistung einer abgeschlossenen Viertelstunde dieses Kalendermonats.`,
		`16 kW`,
		`Grundlage`,
		`Quelle: test`,
		`Günstigere Stufe`,
		`Höhere Stufe`,
		`10 kW`,
		`6 kW`,
		`33,82 € je kW`,
		`67,64 € je kW`,
		`Das ist nicht Ihre Stromrechnung.`,
		`Arbeitspreis, Energiekosten, Abgaben und Steuern`,
		`Niedertarif-Fenster (SNAP, WiNAP)`,
		`Keine Tarif- oder Einspargarantie.`,
		`Regelprofil at-ne7-draft-2027-v1`,
		`Quelle: E-Control, Begutachtungsentwurf`,
		`action="/app/energie/target"`,
		`action="/app/energie/anschlussleistung"`,
		`action="/app/energie/tariff/assessment"`,
		`Diesen Stand festhalten`,
		`data-energy-disclosure="recommendation"`,
		`Für eine Empfehlung fehlen noch ausreichend abgeschlossene Viertelstunden.`,
		`Normale Schwankungen von echten Spitzen trennen`,
		`Automatisch · keine Steuerung`,
		`Noch keine belastbare Wirkung`,
		`Beobachtung läuft`,
		`action="/app/energie/measure"`,
		`name="share" value="inventory"`,
		`name="share" value="measurements"`,
		`name="share" value="contact"`,
		`name="status" value="deferred"`,
		`name="status" value="dismissed"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered first-viewport disclosure lost %q", want)
		}
	}
}

func TestEnergyBilledHelpExplainsDynamicMinimumReason(t *testing.T) {
	a := energyCockpitAppHAUSV425(t, 3, "40")
	body := authedRequest(t, a, "owner@example.com", "/app/energie").Body.String()

	start := strings.Index(body, `data-energy-help="billed"`)
	if start < 0 {
		t.Fatal("billed-power help is missing")
	}
	endOffset := strings.Index(body[start:], `</details>`)
	if endOffset < 0 {
		t.Fatal("billed-power help is not a complete details element")
	}
	help := body[start : start+endOffset]
	for _, want := range []string{
		`2-kW-Sockel`,
		`20 % der vereinbarten Anschlussleistung`,
		`Mindestbemessung aus der vereinbarten Leistung`,
	} {
		if !strings.Contains(help, want) {
			t.Errorf("billed-power help lost %q", want)
		}
	}
}
