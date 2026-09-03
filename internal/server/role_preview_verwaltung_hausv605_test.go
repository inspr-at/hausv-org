package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// A role preview narrows the effective role; the Verwaltung layer (portfolio,
// inbox, settings) must vanish with it, both in navigation and on the routes.
func TestRolePreviewHidesAndDeniesVerwaltungLayer(t *testing.T) {
	const email = "admin@example.com"
	a := newTestPortalApp(t, userProfile{
		Email: email, FirstName: "Vera", LastName: "Verwaltung", Role: roleAdmin,
		Tenants:           []string{"demo", "haus-b"},
		TenantMemberships: map[string]tenantMembership{"demo": {Role: roleAdmin}, "haus-b": {Role: roleAdmin}},
		AuthMethods:       defaultAuthMethods(),
	})
	addTestTenant(a, tenantConfig{Slug: "haus-b", Name: "Haus B", Address: "Nebenweg 2"})
	adminCookie := rolePreviewTestSession(t, a, email, roleAdmin)

	before := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app", nil, adminCookie)
	if !strings.Contains(before.Body.String(), `href="/demo/app/verwaltung"`) {
		t.Fatalf("admin of two houses should see the Verwaltung navigation before the preview")
	}

	start := rolePreviewTestRequest(t, a, http.MethodPost, "/demo/app/ansicht/start", url.Values{"role": {roleOwner}}, adminCookie)
	previewCookie := rolePreviewResponseCookie(t, start)

	page := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app", nil, previewCookie)
	if page.Code != http.StatusOK {
		t.Fatalf("preview portal status = %d", page.Code)
	}
	if strings.Contains(page.Body.String(), `href="/demo/app/verwaltung"`) || strings.Contains(page.Body.String(), `href="/demo/app/verwaltung/posteingang"`) {
		t.Fatalf("preview as Eigentümer must not show Portfolio or Posteingang navigation")
	}
	for _, path := range []string{"/demo/app/verwaltung", "/demo/app/verwaltung/posteingang", "/demo/app/verwaltung/einstellungen"} {
		response := rolePreviewTestRequest(t, a, http.MethodGet, path, nil, previewCookie)
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s during preview: status = %d, want 403", path, response.Code)
		}
	}
}
