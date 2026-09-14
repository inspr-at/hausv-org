package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/config"
)

func TestOrganisationPortalShowsTwoLevelHouseShellHAUSV621(t *testing.T) {
	const email = "vera@example.com"
	slugs := make([]string, 0, 12)
	memberships := make(map[string]tenantMembership, 12)
	for index := 0; index < 12; index++ {
		slug := "demo"
		if index > 0 {
			slug = fmt.Sprintf("haus-%02d", index+1)
		}
		slugs = append(slugs, slug)
		memberships[slug] = tenantMembership{Role: roleAdmin}
	}
	a := newTestPortalApp(t, userProfile{
		Email: email, FirstName: "Vera", LastName: "Verwalter", Role: roleAdmin,
		Tenants: slugs, TenantMemberships: memberships, AuthMethods: defaultAuthMethods(),
	})
	current := a.tenants["demo"]
	current.Name = "Janusbergweg 123"
	current.Address = "8010 Graz"
	current.Organisation = "musterstadt"
	a.tenants["demo"] = current
	for index, slug := range slugs[1:] {
		addTestTenant(a, tenantConfig{
			Slug: slug, Name: fmt.Sprintf("Z-Haus %02d", index+2),
			Address: fmt.Sprintf("Testgasse %d, 8010 Graz", index+2), Organisation: "musterstadt",
		})
	}
	a.organisations = map[string]config.OrganisationConfig{
		"musterstadt": {Key: "musterstadt", Name: "Hausverwaltung Musterstadt GmbH"},
	}
	if _, err := issueRepositoryForTest(a, "haus-02").Create(residentIssue{
		ID: "offen", TenantSlug: "haus-02", AuthorEmail: email, AuthorName: "Vera Verwalter",
		Category: "Reparatur", Title: "Lift", Body: "Der Lift steckt fest.", LocationType: issueLocationCommon,
		Status: issueStatusOpen, Priority: issuePriorityNorm, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("create issue: %v", err)
	}

	page := authedRequest(t, a, email, "/demo/app")
	if page.Code != http.StatusOK {
		t.Fatalf("portal status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{
		"Hausverwaltung Musterstadt GmbH", "Organisation", "Liegenschaft · 1 von 12",
		"Liegenschaft wechseln", "12 Liegenschaften", "Liegenschaft suchen", "1 offen",
		"Alle Liegenschaften (12)", "Textbausteine", "Rechte", "Posteingang",
		"Hausüberblick · Janusbergweg 123", "<h1>Janusbergweg 123</h1>",
		`class="nav-item active"`, `role="combobox"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("organisation portal missing %q", want)
		}
	}
	if got := strings.Count(body, `data-house-picker-shell=`); got != 1 {
		t.Errorf("desktop and mobile house picker count = %d, want 1", got)
	}
	if got := strings.Count(body, "Hausüberblick"); got < 2 {
		t.Errorf("desktop/mobile navigation diverged: Hausüberblick count = %d", got)
	}
}

func TestOwnerPortalKeepsGreetingAndFlatHouseNavigationHAUSV620(t *testing.T) {
	const email = "sophie@example.com"
	a := newTestPortalApp(t, userProfile{
		Email: email, FirstName: "Sophie", LastName: "Beispiel", Role: roleOwner,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	page := authedRequest(t, a, email, "/demo/app")
	if page.Code != http.StatusOK {
		t.Fatalf("portal status = %d", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, "<h1>Hallo Sophie.</h1>") || !strings.Contains(body, "Hausüberblick · Musterweg 1") {
		t.Fatalf("resident heading lost house context or greeting")
	}
	for _, forbidden := range []string{`<header class="portal-organisation-identity"`, `data-two-level="true"`} {
		if strings.Contains(body, forbidden) {
			t.Errorf("resident shell unexpectedly contains %q", forbidden)
		}
	}
	if got := strings.Count(body, `data-house-picker-shell=`); got != 1 {
		t.Errorf("desktop/mobile shared house switcher count = %d, want 1", got)
	}
}

func TestMultiHousePortalUsesPickerWithoutOrganisationHAUSV621(t *testing.T) {
	a := newPortalContextTestApp(t)
	for _, slug := range []string{"demo", "haus-b"} {
		tenant := a.tenants[slug]
		tenant.PortalType = config.PortalTypeCommunity
		a.tenants[slug] = tenant
	}
	page := authedRequest(t, a, "multi@example.com", "/demo/app")
	if page.Code != http.StatusOK {
		t.Fatalf("portal status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{
		"Meine Liegenschaften", "Musterweg 1", "Haus B", "Liegenschaft wechseln",
		`name="tenant" value="demo"`, `name="tenant" value="haus-b"`,
		`name="role" value="Eigentümer"`, `name="role" value="Mieter"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("multi-house picker missing %q", want)
		}
	}
	for _, forbidden := range []string{
		`<header class="portal-organisation-identity"`, `data-two-level="true"`, "Portal wechseln",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("unaffiliated multi-house shell unexpectedly contains %q", forbidden)
		}
	}
	if got := strings.Count(body, `data-house-picker-shell=`); got != 1 {
		t.Errorf("desktop and mobile house picker count = %d, want 1", got)
	}
}
