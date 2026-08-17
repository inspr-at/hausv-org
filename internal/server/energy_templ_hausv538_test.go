package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestEnergyCockpitTemplSwitchDefaultsToLegacyRenderer(t *testing.T) {
	a := energyCockpitAppHAUSV425(t, 16, "")

	body := authedRequest(t, a, "owner@example.com", "/demo/app/energie").Body.String()
	if strings.Contains(body, "data-templ-energy") {
		t.Fatal("energy templ renderer must remain off by default")
	}
	if !strings.Contains(body, `class="app-main energy-cockpit-main"`) {
		t.Fatal("default energy response must still use the legacy renderer")
	}
}

func TestEnergyCockpitTemplUsesSharedShellAndKeepsItsScriptAndWritePaths(t *testing.T) {
	a := energyCockpitAppHAUSV425(t, 16, "14,0")
	a.portalTemplEnabled = true
	a.energyTemplEnabled = true

	response := authedRequest(t, a, "owner@example.com", "/demo/app/energie")
	if response.Code != http.StatusOK {
		t.Fatalf("energy cockpit status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{
		"data-templ-energy",
		`<aside class="sidebar" aria-label="Hausnavigation">`,
		`href="/demo/app/energie" class="nav-item active" aria-current="page"`,
		`<nav class="nav" aria-label="Bereiche"`,
		"Versionsverlauf",
		// Tenant prefixing must reach the cockpit's own write paths and the
		// icon masks in the page stylesheet.
		`action="/demo/app/energie/target"`,
		`action="/demo/app/energie/anschlussleistung"`,
		// versioned exactly as the legacy stylesheet did — an unversioned mask is
		// served from cache forever after the glyph changes
		`url("/demo/assets/icons/lucide/plug.svg?v=`,
		`/demo/assets/energy-flow.js?v=`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("templ energy cockpit should contain %q", want)
		}
	}
	if strings.Count(body, "/demo/assets/app.js?v=") != 1 {
		t.Error("app.js must be loaded exactly once")
	}
	// The cockpit is the only converted page that ships energy-flow.js; loading
	// any other page script here would bind listeners to markup written for a
	// different route.
	for _, foreign := range []string{"issues.js", "attachments.js", "announcements.js", "building-settings.js", "users.js"} {
		if strings.Contains(body, "/demo/assets/"+foreign) {
			t.Errorf("energy cockpit must not load %s", foreign)
		}
	}
}

func TestEnergyCockpitTemplKeepsTheHouseholdNameAsItsDocumentTitle(t *testing.T) {
	a := energyCockpitAppHAUSV425(t, 16, "")
	a.portalTemplEnabled = true
	a.energyTemplEnabled = true

	body := authedRequest(t, a, "owner@example.com", "/demo/app/energie").Body.String()
	if !strings.Contains(body, "<title>Zuhause Test</title>") {
		t.Error("the cockpit tab must keep naming the household, exactly as the legacy page did")
	}
}
