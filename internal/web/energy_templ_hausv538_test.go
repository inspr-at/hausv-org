package web

import (
	"encoding/json"
	"strings"
	"testing"
)

// energyFixture carries every branch of the cockpit at once. The conversion
// risk on this page is not layout, it is silent loss: a dropped form, a dropped
// progressive-enhancement hook or a weakened permission gate all render a page
// that still looks right.
func energyFixture() EnergyPageData {
	return EnergyPageData{
		Portal: PortalPageData{
			Title: "Zuhause Test", HouseName: "Haus am Park", Address: "Parkgasse 1",
			DisplayName: "Ada Beispiel", Initials: "AB", Role: "Eigentümer",
			CanUseResidentAreas: true, CanViewEnergy: true,
			Modules: PortalModules{Energy: true},
		},
		FlowConfigJSON:        json.RawMessage(`{"home":{"value":"1,2"}}`),
		LucideIconNamesJSON:   json.RawMessage(`["plug"]`),
		HouseholdName:         "Zuhause Test",
		HomeTypeLabel:         "Einfamilienhaus",
		HomeUnitLabel:         "Top 3",
		HasHomeUnit:           true,
		CanManageEnergy:       true,
		CanControlEnergy:      true,
		CanManageHomeIdentity: true,
		CanManageEnergyData:   true,
		CanGrantEnergyAccess:  true,
		CanInviteEnergyAccess: true,
		MetricCount:           4,
		HasMetrics:            true,
		Live: EnergyLiveView{
			Main: EnergyMetricView{Label: "Hausverbrauch", Value: "1,2 kW", Detail: "live"}, HasMain: true,
			Flows:   []EnergyMetricView{{Label: "Netzbezug", Value: "0,7 kW", Detail: "live"}},
			Battery: EnergyMetricView{Label: "Speicher", Value: "0,3 kW"}, HasBattery: true,
			BatterySOC: EnergyMetricView{Label: "Ladestand", Value: "62 %"}, HasBatterySOC: true,
			Additional: []EnergyMetricView{{Label: "Sauna", Value: "0,0 kW"}}, HasAdditional: true,
			AdditionalCount: 1, AdditionalTopics: "Sauna",
		},
		Chart: EnergyChartView{
			HasData: true, Title: "Letzte 24 Stunden", DialogTitle: "Letzte 24 Stunden",
			Summary: "Spitze um 18:15", Detail: "Details", Status: "96 Werte", Range: "01.08.–02.08.",
			Series: []EnergyChartSeriesView{{
				Key: "load", Label: "Hausverbrauch", Path: "M0 0", MobilePath: "M0 0",
				AreaPath: "M0 0Z", MobileAreaPath: "M0 0Z", Latest: "1,2 kW",
			}},
			XTicks:       []EnergyChartTickView{{Position: "52", MobilePosition: "44", Label: "06:00"}},
			YTicks:       []EnergyChartTickView{{Position: "110", MobilePosition: "110", Label: "0 kW"}},
			HasThreshold: true, ThresholdLabel: "Planungsgrenze", ThresholdValue: "8 kW",
			ThresholdPosition: "60", ThresholdMobilePosition: "60",
			Samples: []EnergyChartSampleView{{
				Index: 0, Time: "18:15", Position: "52", MobilePosition: "44",
				HitPosition: "48", HitWidth: "8", MobileHit: "40", MobileHitWidth: "8",
				Values: []EnergyChartSampleValueView{{Key: "load", Label: "Hausverbrauch", Value: "1,2 kW", Position: "80", MobilePosition: "80"}},
			}},
		},
		Tariff: EnergyTariffView{
			ID: "at-2027", Version: "2026-08", Status: "Entwurf", SourceTitle: "test", SourceURL: "https://example.test/tarif",
			HasEstimate: true, Disclaimer: "Modell", PeakKW: "16 kW", BilledKW: "16 kW", PeakMeterPercent: 100,
			BelowKW: "10 kW", AboveKW: "6 kW", HasTier: true, MinimumReason: "die gemessene Spitze ist maßgeblich",
			HasAgreed: true, AgreedHint: "Hinweis", TierHint: "Staffel", MonthLabel: "August 2026",
			CoverageLabel: "79 von 132 Viertelstunden abgedeckt", Basis: "Grundlage aus Home Assistant",
			PeakTime: "18:15", HasPeakTime: true, AnnualPowerEUR: "676,40 €",
			BelowRateEUR: "33,82 €", AboveRateEUR: "67,64 €", ThresholdKW: "10 kW",
		},
		TargetPeakValue: "8,0", AgreedPowerValue: "14,0",
		TariffAssessments:    []EnergyTariffAssessmentView{{Month: "Juli 2026", Profile: "at-2027", Peak: "12 kW", Annual: "400 €", Quality: "gemessen", Created: "01.08.2026"}},
		HasTariffAssessments: true,
		Quality:              EnergyQualityView{Status: "measured", Label: "Gemessen", Effect: "Belastbar", NextAction: "Weiter beobachten"},
		Coverage:             []EnergyCoverageView{{Label: "Netzbezug", Status: "vollständig", Detail: "Detail", Tone: "good"}},
		CoverageSummary:      "1 von 1",
		Roadmap:              []EnergyRoadmapStep{{Number: 1, Title: "Verstehen", Detail: "Detail", State: "offen", Action: "Weiter", URL: "/app/zuhause/onboarding", Current: true}},
		SystemAssets:         []EnergySystemAsset{{Name: "PV Dach", IsPV: true}, {Name: "Speicher", IsPV: false}},
		HasSystemAssets:      true,
		FreeUntil:            "01.08.2027",
		Assets:               []EnergyAssetOption{{ID: "asset-1", Name: "Wärmepumpe"}},
		HasAssets:            true,
		Maintenance: []EnergyMaintenanceView{{
			ID: "plan-1", AssetID: "asset-1", AssetName: "Wärmepumpe", Title: "Wärmepumpe warten",
			IntervalMonths: 12, NextDueValue: "2026-12-01", DueLabel: "in 4 Monaten", Tone: "warning",
			LastCompleted: "01.12.2025", ContactID: "contact-1", ContactName: "Installateur",
			DocumentID: "doc-1", DocumentTitle: "Wartungsheft", IssueID: "issue-1", IssueTitle: "Wartung 2026",
			EvidenceNote: "Filter tauschen",
		}},
		HasMaintenance:  true,
		ContactOptions:  []EnergyOption{{Value: "contact-1", Label: "Installateur"}},
		DocumentOptions: []EnergyOption{{Value: "doc-1", Label: "Wartungsheft"}},
		IssueOptions:    []EnergyOption{{Value: "issue-1", Label: "Wartung 2026"}},
		Imports:         []EnergyImportView{{Filename: "meter.csv", Date: "01.08.2026 10:00"}},
		HasImports:      true,
		Peaks:           []EnergyPeakView{{Source: "Smart Meter", Value: "16 kW"}},
		HasPeaks:        true,
		Comparison:      EnergyComparisonView{Tone: "good", Title: "Quellen stimmen überein", Details: "Abweichung 2 %"},
		HasComparison:   true,
		Caretakers:      []EnergyCaretakerView{{Email: "helfer@example.com", Name: "Helfer", CanView: true, Editable: true}},
		HasCaretakers:   true,
		Measures: []EnergyMeasureView{{
			ID: "measure-1", IssueID: "issue-2", Title: "Lastverschiebung", Status: "requested", StatusLabel: "Anfrage vorbereitet",
			ContactID: "contact-1", ContactName: "Installateur", Appointment: "05.09.2026", AppointmentValue: "2026-09-05T09:00",
			BeforePeak: "16 kW", AfterPeak: "12 kW", BeforeQuality: "gemessen", AfterQuality: "gemessen", HasComparison: true,
		}},
		HasMeasures: true,
		Recommendation: EnergyRecommendationView{
			ID: "observe", Title: "Einen Tag beobachten", Reason: "Zu wenige Viertelstunden",
			Benefit: "Belastbare Basis", Effort: "Automatisch", ImpactRange: "Keine Aussage",
		},
		RecommendationURL:   "/app/energie",
		ObservationProgress: EnergyObservationProgressView{Completed: 79, Target: 96, Percent: 82, Label: "79 von 96 Viertelstunden", Title: "Noch 4 Std. beobachten"},
		Scenarios: []EnergyScenarioView{{
			Title: "Sauna verschieben", PeakBand: "12–14 kW", EffectBand: "-2 kW", Uncertainty: "mittel",
			Assumptions: "Sauna läuft nachts", BilledBand: "12 kW", HasBilledBand: true,
			BaselineNote: "Ausgangswert 16 kW", FloorNote: "Untergrenze erreicht",
		}},
		HasScenarios:        true,
		ConsumerKindOptions: []EnergyOption{{Value: "sauna", Label: "Sauna"}},
		ConsumerIconOptions: []EnergyOption{{Value: "plug", Label: "Sonstiges"}, {Value: "flame", Label: "Sauna"}},
	}
}

func TestEnergyTemplKeepsEveryLegacyActionAndHook(t *testing.T) {
	html := renderComponent(t, EnergyPage(energyFixture()))

	// Every write path the legacy cockpit offered. A lost form is invisible to
	// link-level reachability checks, so it is named here explicitly.
	for _, action := range []string{
		`action="/app/energie/mode"`,
		`action="/app/energie/verbraucher"`,
		`formaction="/app/energie/verbraucher/entfernen"`,
		`action="/app/energie/target"`,
		`action="/app/energie/anschlussleistung"`,
		`action="/app/energie/tariff/assessment"`,
		`action="/app/energie/maintenance"`,
		`action="/app/energie/maintenance/complete"`,
		`action="/app/energie/smart-meter"`,
		`action="/app/energie/caretaker"`,
		`action="/app/energie/caretaker/invite"`,
		`action="/app/energie/measure"`,
		`action="/app/energie/measure/update"`,
		`action="/app/energie/recommendation"`,
		`enctype="multipart/form-data"`,
		`name="smart_meter_file"`,
		`name="confirmation_text"`,
	} {
		if !strings.Contains(html, action) {
			t.Errorf("energy cockpit lost the action %s", action)
		}
	}

	// Progressive enhancement is a handshake: app.js and energy-flow.js look for
	// these, the markup provides them. A missing hook leaves the script inert.
	for _, hook := range []string{
		"data-home-identity", "data-home-display-name", "data-home-unit-label",
		"data-energy-flow-diagram", "data-energy-reading-count", "data-energy-flow",
		"data-energy-live-status", "data-energy-live-label", "data-energy-live-updated",
		`data-energy-disclosure="flow"`, `data-energy-disclosure="tariff"`, `data-energy-disclosure="recommendation"`,
		`data-energy-help="tariff"`, `data-energy-help="annual"`,
		`data-dialog="energy-recommendation-dialog"`, `data-dialog="energy-chart-dialog"`,
		"data-close-dialog", "data-energy-fullscreen", "data-fullscreen-label", "data-energy-fullscreen-surface",
		"data-energy-chart-interactive", "data-chart-guide", "data-chart-marker-index",
		"data-chart-hit", "data-chart-tooltip", "data-chart-tooltip-time", "data-chart-tooltip-values",
		"data-chart-sample-bank", "data-chart-sample", "data-label",
		"data-consumer-icon-value", "data-consumer-dialog-title", "data-consumer-dialog-context",
		"data-consumer-priority-field", "data-consumer-icon-search", "data-consumer-icon-presets",
		"data-consumer-icon-choice", "data-consumer-icon-results", "data-consumer-icon-result-note",
		"data-consumer-measurement-fields", "data-consumer-measurement-status",
		"data-consumer-recommendations", "data-consumer-delete", "data-consumer-delete-confirm",
		"data-consumer-delete-cancel", "data-consumer-submit",
		`id="energy-lucide-icon-names"`,
	} {
		if !strings.Contains(html, hook) {
			t.Errorf("energy cockpit lost the hook %s", hook)
		}
	}

	// The anchors the page navigates to internally, plus the outbound source.
	for _, link := range []string{
		`href="/app/settings/home?from=energy"`,
		`href="/app/energie?zeitraum=letzte-24h#energieverlauf"`,
		`href="/app/energie?zeitraum=heute#energieverlauf"`,
		`href="#tarif"`,
		`href="/app/zuhause/onboarding?step=3"`, `href="/app/zuhause/onboarding?step=4"`,
		`href="/app/dokumente"`, `href="/app/events"`, `href="/app/anliegen?new=1"`, `href="/app/kontakte"`,
		`href="/app/settings/users"`, `href="/app/settings/energy-data"`,
		`href="/app/dokumente/doc-1/preview"`, `href="/app/anliegen/issue-1"`, `href="/app/anliegen/issue-2"`,
		`href="https://example.test/tarif"`,
		`id="energieverlauf"`, `id="tarif"`, `id="messwerte"`, `id="wartung"`, `id="betreuung"`, `id="fachhilfe"`, `id="fahrplan"`,
	} {
		if !strings.Contains(html, link) {
			t.Errorf("energy cockpit lost %s", link)
		}
	}

	// Readings stay in kilowatt with the non-breaking space the server formats.
	if !strings.Contains(html, "1,2 kW") || strings.Contains(html, " W<") {
		t.Error("live readings must stay in kW exactly as the server formatted them")
	}
	// No coloured left edge anywhere: ui-identitaet-1-0.
	for _, banned := range []string{"border-left:4px", "border-left:3px", "border-left:5px", "border-left:6px"} {
		if strings.Contains(html, banned) {
			t.Errorf("accent rail %q violates the no-coloured-left-edge rule", banned)
		}
	}
}

func TestEnergyTemplHidesEveryManagementControlWithoutThePermission(t *testing.T) {
	data := energyFixture()
	data.CanManageEnergy = false
	data.CanControlEnergy = false
	data.CanManageHomeIdentity = false
	data.CanManageEnergyData = false
	data.CanGrantEnergyAccess = false
	data.CanInviteEnergyAccess = false
	data.Caretakers[0].Editable = true
	html := renderComponent(t, EnergyPage(data))

	for _, gated := range []string{
		`action="/app/energie/mode"`,
		`action="/app/energie/verbraucher"`,
		`action="/app/energie/target"`,
		`action="/app/energie/anschlussleistung"`,
		`action="/app/energie/tariff/assessment"`,
		`action="/app/energie/maintenance"`,
		`action="/app/energie/maintenance/complete"`,
		`action="/app/energie/smart-meter"`,
		`action="/app/energie/caretaker/invite"`,
		`action="/app/energie/measure/update"`,
		`href="/app/settings/home?from=energy"`,
		`href="/app/settings/energy-data"`,
		`href="/app/zuhause/onboarding?step=3"`,
		`href="/app/zuhause/onboarding?step=4"`,
		`id="energy-consumer-dialog"`,
		`id="energy-lucide-icon-names"`,
	} {
		if strings.Contains(html, gated) {
			t.Errorf("%s must stay behind its permission gate", gated)
		}
	}
	if !strings.Contains(html, "Freigabe nur für Eigentümer oder Hausadministration") {
		t.Error("the read-only cockpit must still explain why the mode switch is absent")
	}
	// The caretaker form itself stays visible, but without the save button.
	if !strings.Contains(html, `action="/app/energie/caretaker"`) {
		t.Error("caretaker scopes must stay visible")
	}
	if strings.Contains(html, `<button class="button" type="submit">Zugriff speichern</button>`) {
		t.Error("granting energy access must stay behind CanGrantEnergyAccess")
	}
}

func TestEnergyTemplKeepsTheObservationCallToActionWhenNoRecommendationURLExists(t *testing.T) {
	data := energyFixture()
	data.RecommendationURL = ""
	html := renderComponent(t, EnergyPage(data))
	if !strings.Contains(html, "Beobachtung läuft") {
		t.Error("a recommendation without a target must say that observation is running")
	}
	if strings.Contains(html, "Diesen Schritt öffnen") {
		t.Error("a recommendation without a target must not offer a dead link")
	}
}

func TestEnergyTemplOffersTheMappingShortcutOnlyWhenTheGridImportIsMissing(t *testing.T) {
	// Waiting for the first full quarter hour is not the same as a missing
	// mapping. Offering "Netzbezug zuordnen" while the data is simply not ripe
	// yet sends the household to a screen where there is nothing to do.
	waiting := energyFixture()
	waiting.Tariff.HasEstimate = false
	waiting.Tariff.MissingIsWaiting = true
	waiting.Tariff.MissingReason = "Noch keine volle Viertelstunde erfasst."
	if html := renderComponent(t, EnergyPage(waiting)); strings.Contains(html, `href="#messwerte"`) {
		t.Error("while waiting for data the cockpit must not send anyone to the mapping section")
	}

	unmapped := energyFixture()
	unmapped.Tariff.HasEstimate = false
	unmapped.Tariff.MissingReason = "Der Netzbezug ist noch keinem Messwert zugeordnet."
	html := renderComponent(t, EnergyPage(unmapped))
	if !strings.Contains(html, `href="#messwerte"`) {
		t.Error("an unmapped grid import must keep its shortcut into the measurement section")
	}
	if !strings.Contains(html, "Netzbezug noch nicht zugeordnet") {
		t.Error("the empty tariff state must name the missing mapping")
	}
}
