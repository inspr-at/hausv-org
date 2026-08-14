package server

import (
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
)

func TestEnergyObservationProgressUsesMeasuredAndEstimatedQuarterHours(t *testing.T) {
	intervals := make([]energy.Interval, 80)
	for index := range intervals {
		intervals[index].Quality = energy.QualityMeasured
	}
	intervals[len(intervals)-1].Quality = energy.QualityEstimated

	got := buildEnergyObservationProgressView(intervals)
	if got.Completed != 80 || got.Target != 96 || got.Percent != 83 ||
		got.Label != "80 von 96 Viertelstunden" ||
		got.Title != "Noch 4 Std. beobachten" || got.Measured != 79 || got.Estimated != 1 {
		t.Fatalf("unexpected honest observation progress: %+v", got)
	}

	estimated := make([]energy.Interval, 79)
	for index := range estimated {
		estimated[index].Quality = energy.QualityEstimated
	}
	got = buildEnergyObservationProgressView(estimated)
	if got.Completed != 79 || got.Percent != 82 || got.Measured != 0 || got.Estimated != 79 ||
		got.Title != "Noch 4 Std. 15 Min. beobachten" {
		t.Fatalf("Home Assistant observation must advance honestly: %+v", got)
	}

	empty := buildEnergyObservationProgressView(nil)
	if empty.Completed != 0 || empty.Percent != 0 ||
		empty.Title != "Noch 1 Tag beobachten" {
		t.Fatalf("unexpected empty observation progress: %+v", empty)
	}
}

func TestEnergyTariffCoverageKeepsCompactAndDetailedQualityWording(t *testing.T) {
	at := time.Date(2026, time.August, 2, 9, 0, 0, 0, time.Local)
	monthStart := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.Local)
	intervals := make([]energy.Interval, 79)
	for index := range intervals {
		intervals[index] = energy.Interval{
			StartsAt: monthStart.Add(time.Duration(index) * 15 * time.Minute),
			Duration: 15 * time.Minute,
			Quality:  energy.QualityEstimated,
			Source:   energy.SourceHomeAssistant,
		}
	}

	got := buildEnergyTariffCoverageView(intervals, at)
	if got.Label != "79 von 132 Viertelstunden abgedeckt" || got.Measured != 0 || got.Estimated != 79 {
		t.Fatalf("unexpected compact tariff coverage: %+v", got)
	}
	for _, want := range []string{
		"79 von 132 bisherigen Viertelstunden im August 2026",
		"79 aus Momentanwerten geschätzt",
		"Quelle: Home Assistant",
	} {
		if !strings.Contains(got.Basis, want) {
			t.Errorf("detailed tariff basis lost %q: %q", want, got.Basis)
		}
	}
}

func TestEnergyFirstViewportDisclosuresRenderDynamicContextAndRealActions(t *testing.T) {
	a := energyCockpitAppHAUSV425(t, 16, "")
	body := authedRequest(t, a, "owner@example.com", "/demo/app/energie").Body.String()

	for _, want := range []string{
		`Noch keine Live-Werte`,
		`href="/demo/app/zuhause/onboarding?step=4"`,
		`data-energy-disclosure="tariff"`,
		`data-energy-help="tariff"`,
		`data-energy-help="annual"`,
		`höchste mittlere Bezugsleistung einer abgeschlossenen Viertelstunde dieses Kalendermonats.`,
		"16\u00a0kW",
		`Grundlage`,
		`Quelle: test`,
		`Günstigere Stufe`,
		`Höhere Stufe`,
		"10\u00a0kW",
		"6\u00a0kW",
		"33,82\u00a0€ je kW",
		"67,64\u00a0€ je kW",
		`Das ist nicht Ihre Stromrechnung.`,
		`Arbeitspreis, Energiekosten, Abgaben und Steuern`,
		`Niedertarif-Fenster (SNAP, WiNAP)`,
		`Keine Tarif- oder Einspargarantie.`,
		`Regelprofil at-ne7-draft-2027-v1`,
		`Quelle: E-Control, Begutachtungsentwurf`,
		`action="/demo/app/energie/target"`,
		`action="/demo/app/energie/anschlussleistung"`,
		`action="/demo/app/energie/tariff/assessment"`,
		`Diesen Stand festhalten`,
		`data-energy-disclosure="recommendation"`,
		`Für eine Empfehlung fehlen noch ausreichend abgeschlossene Viertelstunden.`,
		`Normale Schwankungen von echten Spitzen trennen`,
		`Automatisch · keine Steuerung`,
		`Noch keine belastbare Wirkung`,
		`Beobachtung läuft`,
		`action="/demo/app/energie/measure"`,
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

func TestEnergyTariffHeaderHelpExplainsDynamicMinimumReason(t *testing.T) {
	a := energyCockpitAppHAUSV425(t, 3, "40")
	body := authedRequest(t, a, "owner@example.com", "/demo/app/energie").Body.String()

	start := strings.Index(body, `data-energy-help="tariff"`)
	if start < 0 {
		t.Fatal("tariff header help is missing")
	}
	endOffset := strings.Index(body[start:], `</details>`)
	if endOffset < 0 {
		t.Fatal("tariff header help is not a complete details element")
	}
	help := body[start : start+endOffset]
	for _, want := range []string{
		`2-kW-Sockel`,
		`20 % der vereinbarten Anschlussleistung`,
		`Mindestbemessung aus der vereinbarten Leistung`,
	} {
		if !strings.Contains(help, want) {
			t.Errorf("tariff header help lost %q", want)
		}
	}
}
