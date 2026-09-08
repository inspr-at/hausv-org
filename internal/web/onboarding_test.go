package web

import (
	"strings"
	"testing"
)

func TestOnboardingModeStripFollowsTheControlCapability(t *testing.T) {
	// The strip is the only place the wizard can start a state change. Without
	// the capability it must degrade to a sentence, never to a hidden button.
	allowed := renderComponent(t, OnboardingPage(OnboardingPageData{
		Portal: PortalPageData{Title: "Einrichtung"}, Step: 1, Progress: 20, CanControlEnergy: true,
	}))
	for _, want := range []string{
		`action="/app/energie/mode"`,
		`name="confirm" value="yes" required`,
		`name="confirmation_text"`,
		`name="mode" value="active"`,
	} {
		if !strings.Contains(allowed, want) {
			t.Errorf("mode strip is missing %q for an allowed actor", want)
		}
	}

	denied := renderComponent(t, OnboardingPage(OnboardingPageData{
		Portal: PortalPageData{Title: "Einrichtung"}, Step: 1, Progress: 20,
	}))
	if strings.Contains(denied, `action="/app/energie/mode"`) {
		t.Fatal("the test run form must not render without the control capability")
	}
	if !strings.Contains(denied, "Freigabe nur für Eigentümer oder Hausadministration") {
		t.Fatal("a denied actor must still be told who may release control")
	}
	// Setup never claims the house is switching anything.
	if !strings.Contains(denied, "<strong>Nur beobachten</strong>") {
		t.Fatal("the wizard must always state the observing mode")
	}
}

func TestOnboardingStepTwoKeepsItsValidationContract(t *testing.T) {
	// Field names, required and maxlength are the wizard's contract with the
	// POST handler; app.js only decorates what the markup already declares.
	html := renderComponent(t, OnboardingPage(OnboardingPageData{
		Portal: PortalPageData{Title: "Einrichtung"}, Step: 2, Progress: 40,
		HouseholdName: "Dachwohnung", HomeType: "community",
		HomeTypeDescription: "Mehrere Parteien und gemeinsam genutzte Anlagen.",
		HasUnitOptions:      true,
		UnitOptions:         []OnboardingUnitOption{{Value: "u1", Label: "Top 1", Selected: true}},
	}))
	for _, want := range []string{
		`name="household_name"`,
		`maxlength="100"`,
		`data-home-type-select`,
		`data-home-unit-field`,
		`<select name="unit_id" required`,
		`aria-describedby="home-onboarding-unit-help"`,
		`id="home-onboarding-unit-help"`,
		`value="community" data-label="Hausgemeinschaft"`,
		`<strong data-home-type-label>Hausgemeinschaft</strong>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("step two is missing %q", want)
		}
	}
	if !strings.Contains(html, `<option value="community" data-label="Hausgemeinschaft" data-description="Mehrere Parteien und gemeinsam genutzte Anlagen. Der Überblick richtet sich an Eigentümergemeinschaft oder Hausverwaltung." selected>`) {
		t.Error("the stored home type must come back selected")
	}
}

func TestOnboardingLockedIdentityStaysReadOnlyButStillPosts(t *testing.T) {
	// A locked type must keep travelling with the form; dropping the hidden
	// input would silently reset the profile on the next POST.
	html := renderComponent(t, OnboardingPage(OnboardingPageData{
		Portal: PortalPageData{Title: "Einrichtung"}, Step: 2, Progress: 40,
		HomeType: "apartment", HomeTypeLabel: "Wohnung", HomeTypeLocked: true,
		HasHomeUnit: true, HomeUnitID: "u1", HomeUnitLabel: "Top 1",
	}))
	for _, want := range []string{
		`<input type="hidden" name="home_type" value="apartment">`,
		`<input type="hidden" name="unit_id" value="u1">`,
		"Die Art ist mit der Einheit verbunden.",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("locked identity is missing %q", want)
		}
	}
	if strings.Contains(html, "data-home-type-select") {
		t.Error("a locked home type must not offer the select")
	}
}

func TestOnboardingMappingStepKeepsBothSubmitPaths(t *testing.T) {
	withCandidates := renderComponent(t, OnboardingPage(OnboardingPageData{
		Portal: PortalPageData{Title: "Einrichtung"}, Step: 4, Progress: 80,
		HasCandidates: true, RecommendedCount: 2,
		Candidates: []OnboardingCandidate{
			{EntityID: "sensor.a", DisplayName: "Netzbezug", SourceName: "HA", Checked: true},
			{EntityID: "sensor.b", DisplayName: "PV", SourceName: "HA"},
		},
		HasAdditional:        true,
		AdditionalCandidates: []OnboardingCandidate{{EntityID: "sensor.c", MetricLabel: "Batterie", SourceName: "HA"}},
		MappingSlots:         []OnboardingMappingSlot{{Key: "load", Label: "Hausverbrauch", Tone: "warning", Status: "Fehlt"}},
		MappingAssetOptions:  []OnboardingOption{{Value: "a1", Label: "Wärmepumpe"}},
	}))
	for _, want := range []string{
		`name="action" value="skip-mappings"`,
		`name="action" value="mappings"`,
		`2 Messwerte übernehmen`,
		`data-mapping-slot="load"`,
		`class="energy-mapping-slot warning"`,
		`name="entities" value="sensor.a" checked`,
		`<code>sensor.c</code>`,
		`<option value="a1">Wärmepumpe</option>`,
	} {
		if !strings.Contains(withCandidates, want) {
			t.Errorf("mapping step is missing %q", want)
		}
	}

	withoutCandidates := renderComponent(t, OnboardingPage(OnboardingPageData{
		Portal: PortalPageData{Title: "Einrichtung"}, Step: 4, Progress: 80,
	}))
	if strings.Contains(withoutCandidates, `name="action" value="mappings"`) {
		t.Error("without candidates there is nothing to adopt")
	}
	if !strings.Contains(withoutCandidates, `name="action" value="skip-mappings"`) {
		t.Error("starting without a connection must always stay reachable")
	}
	for _, want := range []string{
		`name="manual_entity_id"`, `name="manual_metric"`, `name="manual_name"`,
		`name="manual_unit"`, `name="manual_asset_id"`,
	} {
		if !strings.Contains(withoutCandidates, want) {
			t.Errorf("manual mapping field %q must stay available", want)
		}
	}
}

func TestOnboardingProgressStyleStaysInsideTheTrack(t *testing.T) {
	if got := onboardingProgressStyle(40); got != "width:40%" {
		t.Fatalf("progress style = %q", got)
	}
	if got := onboardingProgressStyle(140); got != "width:100%" {
		t.Fatalf("overflowing progress style = %q", got)
	}
	if got := onboardingProgressStyle(-5); got != "width:0%" {
		t.Fatalf("negative progress style = %q", got)
	}
}
