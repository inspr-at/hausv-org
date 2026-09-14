package server

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
)

func newHomeIdentityTemplApp(t *testing.T, profileEmail, role, unitID string, owners []string) *app {
	t.Helper()
	a := newTestPortalApp(t, userProfile{
		Email:       profileEmail,
		Role:        role,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	home := energy.DefaultProfile("demo", time.Now())
	home.HouseholdName = "Dachwohnung"
	home.HomeType = energy.HomeApartment
	home.UnitID = unitID
	home.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(home); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{{
		ID:          "einheit-12",
		TenantSlug:  "demo",
		Label:       "Einheit 12",
		UnitType:    unitTypeResidential,
		OwnerEmails: owners,
	}}); err != nil {
		t.Fatalf("seed unit: %v", err)
	}
	return a
}

func TestHomeIdentityAlwaysUsesTemplRenderer(t *testing.T) {
	a := newHomeIdentityTemplApp(t, "owner@example.com", roleOwner, "einheit-12", []string{"owner@example.com"})
	response := authedRequest(t, a, "owner@example.com", "/demo/app/settings/home")
	if response.Code != http.StatusOK {
		t.Fatalf("home settings status = %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "data-templ-home-settings") || !strings.Contains(body, "data-templ-settings") {
		t.Fatal("/app/settings/home must use the templ renderer")
	}
}

func TestHomeIdentityTemplKeepsLockedUnitContract(t *testing.T) {
	a := newHomeIdentityTemplApp(t, "owner@example.com", roleOwner, "einheit-12", []string{"owner@example.com"})

	response := authedRequest(t, a, "owner@example.com", "/demo/app/settings/home?from=energy")
	if response.Code != http.StatusOK {
		t.Fatalf("templ home settings status = %d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		"data-templ-settings",
		"data-templ-home-settings",
		`<aside class="sidebar" aria-label="Navigation der Liegenschaft" data-navigation-surface="sidebar">`,
		`<nav class="nav" aria-label="Bereiche"`,
		// The editor identity hooks other views mirror.
		`data-home-identity="editor-heading"`,
		`data-home-identity="editor-summary"`,
		`aria-label="Dachwohnung, offizielle Einheit Einheit 12"`,
		`data-home-display-name>Dachwohnung<`,
		`data-home-unit-label>Einheit 12<`,
		// Every action of the legacy page.
		`action="/demo/app/settings/home"`,
		`name="from" value="energy"`,
		`name="household_name" value="Dachwohnung"`,
		`name="unit_id" value="einheit-12"`,
		`name="home_type" value="apartment"`,
		`href="/demo/app/energie"`,
		`href="/demo/app/settings"`,
		"Zurück zu Mein Zuhause",
		"Abbrechen",
		"Änderungen speichern",
		`„Dachwohnung“ ist der freundliche Name. „Einheit 12“ bleibt die offizielle Einheit`,
		"Durch die zugeordnete Einheit festgelegt",
		"Diesem Hausprofil zugeordnet",
		// The explanation stays addressable by app.js and by the select.
		`id="home-settings-type-explanation"`,
		"data-home-type-explanation",
		"data-home-type-label",
		"data-home-type-copy",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("templ home settings missing %q", want)
		}
	}
	// A locked type must not offer a way to change it — neither a select nor a
	// unit picker, which is exactly what the server rejects with 403.
	for _, forbidden := range []string{"data-home-type-select", "data-home-unit-field", `<select name="home_type"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("locked home type still renders %q", forbidden)
		}
	}
	// Script parity: the legacy page loaded app.js and nothing else. app.js and
	// switcher.js belong to the shell itself and ship with every portal page.
	var extra []string
	for _, match := range regexp.MustCompile(`/assets/([a-z-]+\.js)\?v=`).FindAllStringSubmatch(body, -1) {
		if match[1] != "app.js" && match[1] != "switcher.js" && match[1] != "support-view.js" && match[1] != "product-version.js" {
			extra = append(extra, match[1])
		}
	}
	if len(extra) != 0 {
		t.Fatalf("templ home settings loaded unexpected scripts %v", extra)
	}
}

func TestHomeIdentityTemplKeepsUnitPickerAndTypeSelect(t *testing.T) {
	a := newHomeIdentityTemplApp(t, "manager@example.com", roleManager, "", nil)

	response := authedRequest(t, a, "manager@example.com", "/demo/app/settings/home")
	if response.Code != http.StatusOK {
		t.Fatalf("templ home settings status = %d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{
		"data-home-type-select",
		`aria-describedby="home-settings-type-explanation"`,
		`<option value="apartment"`,
		`<option value="house"`,
		`<option value="community"`,
		`data-description="Ein einzelner Haushalt in einem Mehrparteienhaus.`,
		// app.js toggles this wrapper, so it must stay a wrapper it can hide.
		"data-home-unit-field",
		`<select name="unit_id" required aria-describedby="home-settings-unit-help">`,
		`id="home-settings-unit-help"`,
		`<option value="einheit-12"`,
		`name="from" value="settings"`,
		"Zurück zu Einstellungen",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("templ home settings missing %q", want)
		}
	}
}

func TestHomeIdentityTemplKeepsAccessGate(t *testing.T) {
	a := newHomeIdentityTemplApp(t, "owner@example.com", roleOwner, "einheit-12", []string{"owner@example.com"})
	a.profiles["resident@example.com"] = userProfile{
		Email:       "resident@example.com",
		Role:        roleResident,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	}
	if response := authedRequest(t, a, "resident@example.com", "/demo/app/settings/home"); response.Code != http.StatusForbidden {
		t.Fatalf("resident home settings status = %d, want 403", response.Code)
	}
}
