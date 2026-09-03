package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/config"
)

func TestVerwaltungManagerOfTwoHousesSeesNavigationAndShell(t *testing.T) {
	const email = "manager@example.com"
	a := newTestPortalApp(t, userProfile{
		Email: email, FirstName: "Vera", LastName: "Verwaltung", Role: roleManager,
		Tenants: []string{"demo", "haus-b"},
		TenantMemberships: map[string]tenantMembership{
			"demo":   {Role: roleManager},
			"haus-b": {Role: roleManager},
		},
		AuthMethods: defaultAuthMethods(),
	})
	addTestTenant(a, tenantConfig{Slug: "haus-b", Name: "Haus B", Address: "Nebenweg 2"})

	portal := authedRequest(t, a, email, "/demo/app")
	if portal.Code != http.StatusOK {
		t.Fatalf("portal status = %d: %s", portal.Code, portal.Body.String())
	}
	for _, want := range []string{`href="/demo/app/verwaltung"`, `href="/demo/app/verwaltung/posteingang"`} {
		if !strings.Contains(portal.Body.String(), want) {
			t.Fatalf("portal missing Verwaltung navigation %q", want)
		}
	}

	verwaltung := authedRequest(t, a, email, "/demo/app/verwaltung")
	if verwaltung.Code != http.StatusOK || !strings.Contains(verwaltung.Body.String(), "Dieser Bereich wird gerade gebaut.") {
		t.Fatalf("Verwaltung status=%d body=%s", verwaltung.Code, verwaltung.Body.String())
	}
	for _, house := range []string{"Musterweg 1", "Haus B"} {
		if !strings.Contains(verwaltung.Body.String(), house) {
			t.Fatalf("Verwaltung shell missing house %q", house)
		}
	}
}

func TestVerwaltungOwnerAndResidentAreForbidden(t *testing.T) {
	for _, test := range []struct {
		name string
		role string
	}{
		{name: "Eigentümer", role: roleOwner},
		{name: "Bewohner", role: roleResident},
	} {
		t.Run(test.name, func(t *testing.T) {
			email := strings.ToLower(test.name) + "@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: test.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			portal := authedRequest(t, a, email, "/demo/app")
			if portal.Code != http.StatusOK {
				t.Fatalf("portal status = %d", portal.Code)
			}
			if strings.Contains(portal.Body.String(), `href="/demo/app/verwaltung"`) || strings.Contains(portal.Body.String(), `href="/demo/app/verwaltung/posteingang"`) {
				t.Fatal("resident-facing portal must not expose Verwaltung navigation")
			}
			verwaltung := authedRequest(t, a, email, "/demo/app/verwaltung")
			if verwaltung.Code != http.StatusForbidden {
				t.Fatalf("Verwaltung status = %d, want 403", verwaltung.Code)
			}
		})
	}
}

func TestSingleHouseManagerWithoutOrganisationKeepsExistingNavigation(t *testing.T) {
	const email = "single@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	portal := authedRequest(t, a, email, "/demo/app")
	if portal.Code != http.StatusOK {
		t.Fatalf("portal status = %d: %s", portal.Code, portal.Body.String())
	}
	if strings.Contains(portal.Body.String(), `href="/demo/app/verwaltung"`) || strings.Contains(portal.Body.String(), `href="/demo/app/verwaltung/posteingang"`) {
		t.Fatal("single-house manager without organisation must keep the existing navigation")
	}
}

func TestConfiguredOrganisationShowsVerwaltungNavigationForSingleHouseManager(t *testing.T) {
	const email = "organisation@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	tenant := a.tenants["demo"]
	tenant.Organisation = "musterstadt"
	a.tenants["demo"] = tenant
	a.organisations = map[string]config.OrganisationConfig{"musterstadt": {Key: "musterstadt", Name: "Hausverwaltung Musterstadt GmbH"}}
	portal := authedRequest(t, a, email, "/demo/app")
	if portal.Code != http.StatusOK || !strings.Contains(portal.Body.String(), `href="/demo/app/verwaltung"`) {
		t.Fatalf("configured organisation navigation missing: status=%d", portal.Code)
	}
}
