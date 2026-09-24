package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestRechtePageAccessAndMatrixHAUSV606(t *testing.T) {
	tests := []struct {
		role string
		want int
	}{
		{role: roleAdmin, want: http.StatusOK},
		{role: roleManager, want: http.StatusOK},
		{role: roleOwner, want: http.StatusForbidden},
		{role: roleBeirat, want: http.StatusForbidden},
		{role: roleRenter, want: http.StatusForbidden},
		{role: roleResident, want: http.StatusForbidden},
		{role: roleServiceProvider, want: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.role, func(t *testing.T) {
			email := strings.ToLower(test.role) + "@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: test.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			response := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app/verwaltung/rechte", nil, rolePreviewTestSession(t, a, email, test.role))
			if response.Code != test.want {
				t.Fatalf("GET rechte as %s = %d, want %d", test.role, response.Code, test.want)
			}
			if test.want != http.StatusOK {
				return
			}
			body := response.Body.String()
			for _, want := range []string{
				"Rechtematrix der Rollenfamilien",
				"Hausverwaltung",
				"Eigentümer",
				"Bewohner",
				"Technische Vertrauensperson",
				"Dienstleister",
				`data-family="verwaltung" data-area="portfolio"`,
				`data-family="bewohner" data-area="account"`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("rights page missing %q", want)
				}
			}
			if count := strings.Count(body, "data-family="); count != 48 {
				t.Errorf("rendered matrix cells = %d, want 48", count)
			}
		})
	}
}

func TestRechtePageFailsClosedAcrossTenantHAUSV606(t *testing.T) {
	const email = "manager@example.com"
	a := newTestPortalApp(t, userProfile{Email: email, Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	response := rolePreviewTestRequest(t, a, http.MethodGet, "/other/app/verwaltung/rechte", nil, rolePreviewTestSession(t, a, email, roleManager))
	if response.Code == http.StatusOK {
		t.Fatal("manager received rights matrix for an unassigned tenant")
	}
}
