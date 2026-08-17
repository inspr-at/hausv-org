package server

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestParkingTemplSwitchDefaultsToLegacyRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	body := authedRequest(t, a, "admin@example.com", "/demo/app/parking").Body.String()
	if strings.Contains(body, "data-templ-parking") {
		t.Fatal("parking templ renderer must remain off by default")
	}
	if !strings.Contains(body, `class="page wide parking-page"`) {
		t.Fatal("default parking response must still use the legacy renderer")
	}
}

func TestParkingTemplKeepsEveryActionForManagers(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true
	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	if err := a.parkingStore.AppendReadings("demo", []parkingNumericSample{
		{At: base, Value: 100},
		{At: base.Add(2 * time.Hour), Value: 102},
	}, []parkingNumericSample{
		{At: base, Value: 0.20},
		{At: base.Add(time.Hour), Value: 0.40},
	}); err != nil {
		t.Fatalf("AppendReadings: %v", err)
	}

	response := authedRequest(t, a, "admin@example.com", "/demo/app/parking")
	if response.Code != http.StatusOK {
		t.Fatalf("parking status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{
		"data-templ-parking",
		`href="/demo/app/parking" class="nav-item active" aria-current="page"`,
		// Every action the legacy overview carried, including the two that no
		// reachability check can see: the reminder POST and the export.
		`action="/demo/app/parking/reminders"`,
		"Erinnerungen senden",
		`href="/demo/app/parking/export/`,
		`href="/demo/app/parking/settings"`,
		"parking-months-panel",
		"parking-current-month",
		"Neuester Monat",
		`href="/demo/app/parking/month/2026-06"`,
		`id="parking-month-2026-06"`,
		"Wie wird gerechnet?",
		"Letzter Messpunkt:",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("templ parking overview should contain %q", want)
		}
	}
	if !strings.Contains(body, "/assets/attachments.js?v=") {
		t.Fatal("templ parking must keep the script set of the legacy route")
	}
}

func TestParkingTemplEmptyStateKeepsItsPermissionGates(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true
	if _, err := a.inviteStore.Add(userProfile{Email: "parker@example.com", FirstName: "Pat", LastName: "Parker", Role: roleRenter, Tenants: []string{"demo"}, Permissions: []string{permissionParking}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("Add invite: %v", err)
	}

	adminBody := authedRequest(t, a, "admin@example.com", "/demo/app/parking").Body.String()
	for _, want := range []string{
		"data-templ-parking",
		"parking-empty",
		"Bereit für die erste Abrechnung",
		"Noch keine Monatswerte",
		"Abrechnung konfigurieren",
		`href="/demo/app/settings/parking-access"`,
		"Zugriff verwalten",
		"Nur Nachweis",
	} {
		if !strings.Contains(adminBody, want) {
			t.Fatalf("templ parking empty state for an admin should contain %q", want)
		}
	}

	// A resident reaches this page through the per-user parking permission
	// alone. Losing either gate here would hand them the configuration routes.
	residentBody := authedRequest(t, a, "parker@example.com", "/demo/app/parking").Body.String()
	if !strings.Contains(residentBody, "Hausüberblick öffnen") {
		t.Fatal("a resident without management rights needs the way back to the portal")
	}
	for _, forbidden := range []string{
		`href="/demo/app/parking/settings"`,
		`href="/demo/app/settings/parking-access"`,
		"Zugriff verwalten",
		`action="/demo/app/parking/reminders"`,
	} {
		if strings.Contains(residentBody, forbidden) {
			t.Fatalf("templ parking must not expose %q to a resident", forbidden)
		}
	}
}
