package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/markus-barta/hausv-org/internal/energy"
)

func TestBuildingSettingsProgressiveSectionsAndIntegratedPayments(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{{
		ID:                    "top-1",
		TenantSlug:            "jhw22",
		Label:                 "Top 1",
		UnitType:              unitTypeResidential,
		MiteigentumsanteilPPM: 250000,
		BillableWeightPPM:     unitBillableFullPPM,
		OwnerEmails:           []string{"owner@example.com"},
		RenterEmails:          []string{"renter@example.com"},
	}}); err != nil {
		t.Fatalf("seed unit: %v", err)
	}
	if _, err := a.unitPaymentStore.Set(unitPaymentStatus{
		TenantSlug: "jhw22",
		UnitID:     "top-1",
		Status:     unitPaymentStatusOverdue,
		UpdatedBy:  "manager@example.com",
	}); err != nil {
		t.Fatalf("seed payment: %v", err)
	}
	profile := energy.DefaultProfile("jhw22", time.Now())
	profile.HouseholdName = "Penthouse"
	profile.HomeType = energy.HomeApartment
	profile.UnitID = "top-1"
	profile.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatalf("seed home profile: %v", err)
	}

	page := authedRequest(t, a, "manager@example.com", "/app/settings/building")
	if page.Code != http.StatusOK {
		t.Fatalf("building page status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{
		`aria-label="Bereiche"`,
		`href="#overview"`,
		`href="#contacts"`,
		`href="#units"`,
		`href="#appearance"`,
		`<details class="unit-add" id="unit-add">`,
		`<details class="unit-editor" id="unit-top-1">`,
		`data-confirm="Einheit Top 1 entfernen?"`,
		`<span class="pill dringend">Überfällig</span>`,
		`<h3>Zahlungsstatus</h3>`,
		`form="building-meta-form"`,
		`aria-label="Abgrenzung zum Hausprofil"`,
		`<strong>Penthouse</strong>`,
		`Wohnung · Top 1`,
		`Hausprofil der offiziellen Einheit „Top 1“`,
		`href="/app/settings/home?from=building"`,
		`Offizielle Bezeichnung`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("building page should contain %q", want)
		}
	}
	if strings.Contains(body, `class="panel payment-status-panel"`) {
		t.Fatal("payment status must not repeat the unit inventory in a second panel")
	}
	if strings.Index(body, `id="overview"`) > strings.Index(body, `id="contacts"`) ||
		strings.Index(body, `id="contacts"`) > strings.Index(body, `id="units"`) ||
		strings.Index(body, `id="units"`) > strings.Index(body, `id="appearance"`) {
		t.Fatal("building sections are not in the expected task order")
	}
}

func TestBuildingSettingsUsesOneEmptyUnitStateAndAnchoredActions(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})

	page := authedRequest(t, a, "manager@example.com", "/app/settings/building")
	if got := strings.Count(page.Body.String(), "Noch keine Einheiten"); got != 1 {
		t.Fatalf("empty unit state count = %d, want 1", got)
	}

	invalidMeta := authedFormRequest(t, a, "manager@example.com", "/app/settings/building", url.Values{})
	if got := invalidMeta.Header().Get("Location"); got != "/app/settings/building?building=invalid#overview" {
		t.Fatalf("invalid meta redirect = %q", got)
	}
	invalidUnit := authedFormRequest(t, a, "manager@example.com", "/app/settings/building/units", url.Values{})
	if got := invalidUnit.Header().Get("Location"); got != "/app/settings/building?unit=invalid#unit-add" {
		t.Fatalf("invalid unit redirect = %q", got)
	}
}
