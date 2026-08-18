package server

import (
	"html"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestConnectorHelpGuidesNewUsersAndKeepsSetupReachable(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})

	page := authedRequest(t, a, "admin@example.com", "/demo/app/hilfe")
	if page.Code != http.StatusOK {
		t.Fatalf("help status = %d, want 200", page.Code)
	}
	for _, want := range []string{
		`href="/demo/app/hilfe"`, `class="nav-item active"`,
		"So läuft es in HAUSV ab", "Technisch und datensparsam",
		"Home-Assistant-Adresse und Token werden nicht an HAUSV übertragen",
		"GET /api/config", "GET /api/states", "Connector einrichten",
	} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("help page missing %q", want)
		}
	}

	pairing := authedFormRequest(t, a, "admin@example.com", "/demo/app/hilfe/connector/pairing", nil)
	if pairing.Code != http.StatusOK {
		t.Fatalf("pairing status = %d, want 200", pairing.Code)
	}
	match := regexp.MustCompile(`data-pairing-code>([^<]+)</code>`).FindStringSubmatch(pairing.Body.String())
	if len(match) != 2 {
		t.Fatal("new pairing code is not shown exactly on the pairing response")
	}
	pairingCode := html.UnescapeString(match[1])
	stored, found, err := a.homeConnectors.Get("demo")
	if err != nil || !found || stored.Status != store.HomeConnectorPairing {
		t.Fatalf("stored pairing = %#v, found=%v, err=%v", stored, found, err)
	}
	if strings.Contains(string(stored.PairingHash), pairingCode) {
		t.Fatal("raw pairing code must not be persisted")
	}

	now := time.Now()
	_, exchanged, err := a.homeConnectors.ExchangePairing(
		homeConnectorHash(a.homeConnectorHashKey, pairingCode),
		homeConnectorHash(a.homeConnectorHashKey, strings.Repeat("c", 43)),
		store.HomeConnectorHeartbeat{ConnectorVersion: "0.1.0", HomeAssistantVersion: "2026.8.1", EntityCount: 12}, now)
	if err != nil || !exchanged {
		t.Fatalf("exchange pairing: exchanged=%v err=%v", exchanged, err)
	}
	connected := authedRequest(t, a, "admin@example.com", "/demo/app/hilfe")
	for _, want := range []string{"Verbunden und aktuell", "Home Assistant 2026.8.1", "12 erkannte Messwerte", "Widerrufen"} {
		if !strings.Contains(connected.Body.String(), want) {
			t.Errorf("connected help page missing %q", want)
		}
	}
}

func TestConnectorHelpIsReadableButSetupNeedsEnergyPermission(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "resident@example.com", Role: roleResident,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	page := authedRequest(t, a, "resident@example.com", "/demo/app/hilfe")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Hilfe zur Energieverbindung") {
		t.Fatalf("help must stay readable: status=%d", page.Code)
	}
	if strings.Contains(page.Body.String(), "Connector einrichten</button>") {
		t.Fatal("setup action must not be shown without energy-management permission")
	}
	post := authedFormRequest(t, a, "resident@example.com", "/demo/app/hilfe/connector/pairing", nil)
	if post.Code != http.StatusForbidden {
		t.Fatalf("setup without permission = %d, want 403", post.Code)
	}
}

func TestConnectorHelpTemplUsesSharedPortalForEveryResidentRole(t *testing.T) {
	roles := []string{roleAdmin, roleManager, roleOwner, roleResident}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			email := strings.ToLower(role) + "@example.com"
			a := newTestPortalApp(t, userProfile{
				Email: email, Role: role,
				Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
			})

			page := authedRequest(t, a, email, "/demo/app/hilfe")
			if page.Code != http.StatusOK {
				t.Fatalf("templ help status = %d, want 200", page.Code)
			}
			body := page.Body.String()
			for _, want := range []string{
				`data-templ-help`,
				`href="/demo/app/hilfe" class="nav-item active" aria-current="page"`,
				`Hilfe zur Energieverbindung`,
				`So läuft es in HAUSV ab`,
				`Technisch und datensparsam`,
				`Versionsverlauf`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("templ help page missing %q", want)
				}
			}
			if strings.Contains(body, `class="app-shell"`) {
				t.Error("templ help rendered the legacy navigation instead of the shared templ navigation")
			}
		})
	}
}
