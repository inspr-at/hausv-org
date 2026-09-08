package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestOnboardingAlwaysUsesTemplRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	body := authedRequest(t, a, "owner@example.com", "/demo/app/zuhause/onboarding").Body.String()
	if !strings.Contains(body, "data-templ-onboarding") {
		t.Fatal("onboarding response must use the templ renderer")
	}
}

func TestOnboardingTemplUsesSharedPermissionGatedShell(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	response := authedRequest(t, a, "owner@example.com", "/demo/app/zuhause/onboarding")
	if response.Code != http.StatusOK {
		t.Fatalf("onboarding status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{
		"data-templ-onboarding",
		`<aside class="sidebar" aria-label="Navigation der Liegenschaft" data-navigation-surface="sidebar">`,
		`href="/demo/app/energie" class="nav-item active" aria-current="page"`,
		`<nav class="nav" aria-label="Bereiche"`,
		`<header class="context-bar mobile-head" data-context-bar>`,
		"Versionsverlauf",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("templ onboarding page missing %q", want)
		}
	}
	// app.js carries every enhancement this wizard uses; nothing else may load.
	if strings.Count(body, "/assets/app.js?v=") != 1 || strings.Contains(body, "/assets/issues.js") {
		t.Fatal("templ onboarding must load app.js exactly once and no other script")
	}
}

func TestOnboardingTemplKeepsEveryStepActionAndHook(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	steps := map[string][]string{
		"1": {
			`action="/demo/app/zuhause/onboarding"`,
			`name="action" value="understand"`,
			`href="/demo/app"`,
			"Womit möchten Sie beginnen? Mit Ihrem Zuhause.",
		},
		"2": {
			`name="household_name"`, `required`, `maxlength="100"`,
			`data-home-type-select`, `aria-describedby="home-type-explanation"`,
			`data-home-type-explanation`, `data-home-type-label`, `data-home-type-copy`,
			`value="apartment"`, `value="house"`, `value="community"`,
			`data-description="Ein einzelner Haushalt in einem Mehrparteienhaus.`,
			`name="unit_id"`,
			`name="action" value="back"`, `name="action" value="profile"`,
		},
		"3": {
			`name="assets" value="pv"`, `name="assets" value="battery"`,
			`name="action" value="assets"`,
		},
		"4": {
			`data-mapping-slot="load"`, `data-mapping-slot="peaks"`,
			`name="manual_entity_id"`, `name="manual_metric"`, `name="manual_name"`,
			`name="manual_unit"`, `name="manual_asset_id"`,
			`value="grid-import-power"`, `value="battery-soc"`, `value="load-power"`,
			`name="action" value="skip-mappings"`,
		},
		"5": {
			`data-home-identity="onboarding-summary"`, `data-home-display-name`,
			`name="action" value="finish"`, `name="action" value="back"`,
		},
	}
	for step, wants := range steps {
		body := authedRequest(t, a, "owner@example.com", "/demo/app/zuhause/onboarding?step="+step).Body.String()
		for _, want := range wants {
			if !strings.Contains(body, want) {
				t.Errorf("templ onboarding step %s missing %q", step, want)
			}
		}
	}
}

func TestOnboardingTemplKeepsTestRunBehindItsCapabilityGate(t *testing.T) {
	// The deliberate test run is the only state-changing control on this page,
	// and canControlEnergy is what decides it — exactly as in the legacy strip.
	owner := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	ownerBody := authedRequest(t, owner, "owner@example.com", "/demo/app/zuhause/onboarding").Body.String()
	for _, want := range []string{
		`action="/demo/app/energie/mode"`,
		`name="confirmation_text"`,
		`name="mode" value="active"`,
		`name="confirm" value="yes"`,
	} {
		if !strings.Contains(ownerBody, want) {
			t.Errorf("owner must keep the test run control, missing %q", want)
		}
	}
	// A role without the energy-control capability never reaches this page at
	// all: canViewEnergy already refuses it, which is the stronger gate.
	resident := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if response := authedRequest(t, resident, "resident@example.com", "/demo/app/zuhause/onboarding"); response.Code != http.StatusForbidden {
		t.Fatalf("resident onboarding status = %d, want 403", response.Code)
	}
}

func TestOnboardingTemplStillCompletesTheWizard(t *testing.T) {
	// The switch changes the renderer, never the flow: every POST target the
	// legacy page offered must keep working with templ on.
	a := newTestPortalApp(t, userProfile{Email: "owner@example.com", Role: roleOwner, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	for _, form := range []url.Values{
		{"action": {"understand"}},
		{"action": {"profile"}, "household_name": {"Dachwohnung"}, "home_type": {"house"}},
		{"action": {"assets"}, "assets": {"pv"}},
		{"action": {"skip-mappings"}},
		{"action": {"finish"}},
	} {
		response := authedFormRequest(t, a, "owner@example.com", "/demo/app/zuhause/onboarding", form)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("onboarding %v status = %d, want 303", form["action"], response.Code)
		}
	}
	body := authedRequest(t, a, "owner@example.com", "/demo/app/zuhause/onboarding?step=5").Body.String()
	if !strings.Contains(body, "Dachwohnung") {
		t.Fatal("the completed profile name must survive into the templ summary")
	}
}
