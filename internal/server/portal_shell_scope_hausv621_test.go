package server

import (
	"net/http"
	"strings"
	"testing"
)

// The Verwaltung layer belongs to people who administer houses. An owner whose
// house belongs to a Hausverwaltung must not see Portfolio or Posteingang, and
// somebody who administers two houses without an organisation must.
func TestOrganisationBlockFollowsAdministeredHouses(t *testing.T) {
	t.Run("owner in an organisation house sees no Verwaltung layer", func(t *testing.T) {
		const email = "alina@example.com"
		a := newTestPortalApp(t, userProfile{
			Email: email, FirstName: "Alina", LastName: "Auer", Role: roleOwner,
			Tenants: []string{"demo"}, TenantMemberships: map[string]tenantMembership{"demo": {Role: roleOwner}},
			AuthMethods: defaultAuthMethods(),
		})
		tenant := a.tenants["demo"]
		tenant.Organisation = "musterstadt"
		a.tenants["demo"] = tenant
		body := authedRequest(t, a, email, "/demo/app").Body.String()
		for _, forbidden := range []string{`href="/demo/app/verwaltung"`, `href="/demo/app/verwaltung/posteingang"`} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("owner sees %s", forbidden)
			}
		}
	})

	t.Run("admin of two houses without an organisation sees it", func(t *testing.T) {
		const email = "admin@example.com"
		a := newTestPortalApp(t, userProfile{
			Email: email, FirstName: "Vera", LastName: "Verwaltung", Role: roleAdmin,
			Tenants:           []string{"demo", "haus-b"},
			TenantMemberships: map[string]tenantMembership{"demo": {Role: roleAdmin}, "haus-b": {Role: roleAdmin}},
			AuthMethods:       defaultAuthMethods(),
		})
		addTestTenant(a, tenantConfig{Slug: "haus-b", Name: "Haus B", Address: "Nebenweg 2"})
		page := authedRequest(t, a, email, "/demo/app")
		if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), `href="/demo/app/verwaltung"`) {
			t.Fatalf("multi-house admin lost the Verwaltung navigation (status %d)", page.Code)
		}
	})
}
