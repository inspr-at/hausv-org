package server

import (
	"html"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/inspr-at/hausv-org/internal/config"
)

func TestPortalNavigationByRoleFamilyHAUSV606(t *testing.T) {
	tests := []struct {
		name string
		role string
		want []string
	}{
		{name: "Hausverwaltung", role: roleManager, want: []string{"Portfolio", "Posteingang", "Hausüberblick", "Mein Zuhause", "Aushang", "Termine", "Kontakte", "Dokumente", "Anliegen", "Abstimmungen", "Übergaben", "Benutzer & Rechte", "Verlauf", "Einstellungen", "Hilfe"}},
		{name: "Eigentümer", role: roleOwner, want: []string{"Hausüberblick", "Mein Zuhause", "Aushang", "Termine", "Kontakte", "Dokumente", "Anliegen", "Abstimmungen", "Verlauf", "Einstellungen", "Hilfe"}},
		{name: "Bewohner", role: roleResident, want: []string{"Hausüberblick", "Aushang", "Termine", "Kontakte", "Dokumente", "Anliegen", "Abstimmungen", "Verlauf", "Einstellungen", "Hilfe"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			email := strings.ToLower(test.name) + "@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: test.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			if test.role == roleManager {
				tenant := a.tenants["demo"]
				tenant.Organisation = "test-verwaltung"
				a.tenants["demo"] = tenant
				a.organisations = map[string]config.OrganisationConfig{"test-verwaltung": {Key: "test-verwaltung", Name: "Test-Verwaltung"}}
			}
			page := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app", nil, rolePreviewTestSession(t, a, email, test.role))
			if page.Code != http.StatusOK {
				t.Fatalf("portal status = %d", page.Code)
			}
			if got := roleFamilySidebarLabels(page.Body.String()); !slices.Equal(got, test.want) {
				t.Fatalf("navigation = %v, want %v", got, test.want)
			}

			rights := rolePreviewTestRequest(t, a, http.MethodGet, "/demo/app/verwaltung/rechte", nil, rolePreviewTestSession(t, a, email, test.role))
			wantStatus := http.StatusForbidden
			if test.role == roleManager {
				wantStatus = http.StatusOK
			}
			if rights.Code != wantStatus {
				t.Fatalf("rights route status = %d, want %d", rights.Code, wantStatus)
			}
		})
	}
}

var roleFamilyNavLabelPattern = regexp.MustCompile(`(?s)<span class="nav-label">(.*?)</span>`)
var roleFamilyTagPattern = regexp.MustCompile(`<[^>]+>`)

func roleFamilySidebarLabels(body string) []string {
	start := strings.Index(body, `<nav class="nav"`)
	if start < 0 {
		return nil
	}
	end := strings.Index(body[start:], `</nav>`)
	if end < 0 {
		return nil
	}
	matches := roleFamilyNavLabelPattern.FindAllStringSubmatch(body[start:start+end], -1)
	labels := make([]string, 0, len(matches))
	for _, match := range matches {
		labels = append(labels, strings.TrimSpace(html.UnescapeString(roleFamilyTagPattern.ReplaceAllString(match[1], ""))))
	}
	return labels
}
