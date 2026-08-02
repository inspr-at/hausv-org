package web

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// The first-viewport energy redesign is deliberately bounded: the chart and
// every section following it already have their own interaction and visual QA.
// Keep this fingerprint narrow enough that the mode strip, identity, live
// flow, tariff and next-step surfaces can evolve without making an unrelated
// chart/below rewrite look accidental.
func TestEnergyFirstViewportRedesignLeavesChartAndFollowingSectionsUntouched(t *testing.T) {
	const (
		startMarker = `      <section class="energy-card energy-chart"`
		endMarker   = "\n{{define \"energyData\"}}"
		wantSHA256  = "5374e40cc48a8756d21e6ac0a7e6a0ec5eaf8afa75d037e30d02365928706e5e"
	)

	start := strings.Index(PageTemplates, startMarker)
	if start < 0 {
		t.Fatal("energy chart boundary is missing")
	}
	endOffset := strings.Index(PageTemplates[start:], endMarker)
	if endOffset < 0 {
		t.Fatal("energy-data template boundary is missing")
	}

	protected := PageTemplates[start : start+endOffset]
	got := fmt.Sprintf("%x", sha256.Sum256([]byte(protected)))
	if got != wantSHA256 {
		t.Fatalf("energy chart or a following section changed outside the first-viewport redesign scope: sha256=%s, want %s", got, wantSHA256)
	}
}

func TestEnergyFirstViewportRedesignLeavesSharedSidebarUntouched(t *testing.T) {
	const (
		startMarker = `{{define "sidebar"}}`
		endMarker   = "\n{{define \"releaseHistoryDialog\"}}"
		wantSHA256  = "7e410142b1c0324d5530a396541c9ac9e870420fd53e9e873f419fcf01e2682f"
	)

	start := strings.Index(PageTemplates, startMarker)
	if start < 0 {
		t.Fatal("shared sidebar template is missing")
	}
	endOffset := strings.Index(PageTemplates[start:], endMarker)
	if endOffset < 0 {
		t.Fatal("release-history boundary after the shared sidebar is missing")
	}

	protected := PageTemplates[start : start+endOffset]
	got := fmt.Sprintf("%x", sha256.Sum256([]byte(protected)))
	if got != wantSHA256 {
		t.Fatalf("shared sidebar changed during the right-hand energy redesign: sha256=%s, want %s", got, wantSHA256)
	}
}

func TestEnergyFirstViewportDisclosuresKeepContextAndActionsReachable(t *testing.T) {
	for _, want := range []string{
		`class="energy-info-disclosure" data-energy-disclosure="flow"`,
		`<summary aria-label="Energiefluss verstehen" aria-controls="energy-flow-help">`,
		`Energiefluss verstehen`,
		`aria-label="Zuhause{{if .Live.HasMain}}, {{.Live.Main.Label}} {{.Live.Main.Value}}{{end}}"`,
		`Live aus Home Assistant · nur gelesen`,
		`href="/app/zuhause/onboarding?step=4"`,
		`class="energy-tariff-disclosure" data-energy-disclosure="tariff"`,
		`Mehr erfahren`,
		`Ab 01.01.2027 bemisst der Netzbetreiber die höchste Viertelstunde jedes Kalendermonats.`,
		`Das ist nicht Ihre Stromrechnung.`,
		`Arbeitspreis, Energiekosten, Abgaben und Steuern`,
		`Niedertarif-Fenster (SNAP, WiNAP)`,
		`{{.Tariff.Disclaimer}}`,
		`{{.Tariff.TierHint}}`,
		`action="/app/energie/target"`,
		`action="/app/energie/anschlussleistung"`,
		`Regelprofil {{.Tariff.ID}} · Stand {{.Tariff.Version}}`,
		`href="{{.Tariff.SourceURL}}"`,
		`action="/app/energie/tariff/assessment"`,
		`Festgehaltene Bewertungen`,
		`{{range .TariffAssessments}}`,
		`class="energy-next-why" data-energy-disclosure="recommendation"`,
		`Warum?`,
		`{{.Recommendation.Reason}}`,
		`{{.Recommendation.Benefit}}`,
		`Aufwand: {{.Recommendation.Effort}}`,
		`{{.Recommendation.ImpactRange}}`,
		`{{if .RecommendationURL}}<a class="button primary" href="{{.RecommendationURL}}"`,
		`action="/app/energie/measure"`,
		`name="share" value="inventory"`,
		`name="share" value="measurements"`,
		`name="share" value="contact"`,
		`action="/app/energie/recommendation"`,
		`name="status" value="deferred"`,
		`name="status" value="dismissed"`,
	} {
		if !strings.Contains(PageTemplates, want) {
			t.Errorf("first-viewport progressive disclosure lost %q", want)
		}
	}

	if got := strings.Count(PageTemplates, `name="energy-metric-help"`); got != 2 {
		t.Errorf("exclusive metric-help group members = %d, want the two consolidated tariff explanations", got)
	}
	for _, item := range []struct {
		key   string
		label string
		panel string
	}{
		{key: "tariff", label: "Monatsspitze und Verrechnung erklären", panel: "energy-help-tariff"},
		{key: "annual", label: "Jahreswert erklären", panel: "energy-help-annual"},
	} {
		marker := `data-energy-help="` + item.key + `"`
		start := strings.Index(PageTemplates, marker)
		if start < 0 {
			t.Errorf("metric help %q is missing", item.key)
			continue
		}
		endOffset := strings.Index(PageTemplates[start:], `</details>`)
		if endOffset < 0 {
			t.Errorf("metric help %q has no closing details element", item.key)
			continue
		}
		control := PageTemplates[start : start+endOffset]
		for _, want := range []string{
			`<summary aria-label="` + item.label + `" aria-controls="` + item.panel + `">`,
			`id="` + item.panel + `"`,
		} {
			if !strings.Contains(control, want) {
				t.Errorf("metric help %q lost accessible relationship %q", item.key, want)
			}
		}
	}
	for _, obsolete := range []string{`data-energy-help="consumption"`, `data-energy-help="peak"`, `data-energy-help="billed"`} {
		if strings.Contains(PageTemplates, obsolete) {
			t.Errorf("redundant metric help was not consolidated: %s", obsolete)
		}
	}
}
