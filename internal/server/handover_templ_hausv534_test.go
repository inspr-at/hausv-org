package server

import (
	"strings"
	"testing"
)

func TestHandoversTemplSwitchDefaultsToLegacyRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	body := authedRequest(t, a, "manager@example.com", "/demo/app/uebergaben").Body.String()
	if strings.Contains(body, "data-templ-handovers") {
		t.Fatal("handover templ renderer must remain off by default")
	}
	if !strings.Contains(body, `class="page wide handover-page"`) {
		t.Fatal("default handover response must still use the legacy renderer")
	}
}

func TestHandoversTemplUsesSharedPermissionGatedPortalShell(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true

	body := authedRequest(t, a, "manager@example.com", "/demo/app/uebergaben").Body.String()
	for _, want := range []string{
		"data-templ-handovers",
		`href="/demo/app/uebergaben" class="nav-item active" aria-current="page"`,
		`aria-label="Hausnavigation"`,
		`aria-label="Navigation öffnen"`,
		`id="handover-create"`,
		`name="rooms_text"`,
		`name="meters_text"`,
		`name="keys_text"`,
		`name="attachments"`,
		"ohne Kautions- oder Schadenabrechnung",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("templ handover page missing %q", want)
		}
	}
	if strings.Contains(body, "ui-identitaet-1-0") {
		t.Fatal("templ handover page must not use a coloured identity edge")
	}
}
