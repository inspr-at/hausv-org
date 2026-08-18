package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
)

func TestBuildingSettingsProgressiveSectionsAndIntegratedPayments(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{{
		ID:                    "top-1",
		TenantSlug:            "demo",
		Label:                 "Top 1",
		UnitType:              unitTypeResidential,
		MiteigentumsanteilPPM: 250000,
		BillableWeightPPM:     unitBillableFullPPM,
		OwnerEmails:           []string{"owner@example.com"},
		RenterEmails:          []string{"renter@example.com"},
	}}); err != nil {
		t.Fatalf("seed unit: %v", err)
	}
	if _, err := testUnitPaymentRepository(t, a, "demo").Set(unitPaymentStatus{
		TenantSlug: "demo",
		UnitID:     "top-1",
		Status:     unitPaymentStatusOverdue,
		UpdatedBy:  "manager@example.com",
	}); err != nil {
		t.Fatalf("seed payment: %v", err)
	}
	profile := energy.DefaultProfile("demo", time.Now())
	profile.HouseholdName = "Dachwohnung"
	profile.HomeType = energy.HomeApartment
	profile.UnitID = "top-1"
	profile.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatalf("seed home profile: %v", err)
	}

	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/building?section=units")
	if page.Code != http.StatusOK {
		t.Fatalf("building page status = %d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{
		`aria-label="Bereiche"`,
		`href="/demo/app/settings/building?section=overview"`,
		`href="/demo/app/settings/building?section=contacts"`,
		`href="/demo/app/settings/building?section=units" class="active" aria-current="page"`,
		`href="/demo/app/settings/building?section=appearance"`,
		`unit-dialog-shell" id="unit-add"`,
		`unit-dialog-shell" id="unit-top-1"`,
		`class="unit-card"`,
		`href="#unit-top-1"`,
		`data-confirm="Einheit Top 1 entfernen?"`,
		`<span class="pill dringend">Überfällig</span>`,
		`<legend>Zahlungsstatus</legend>`,
		`Mein Zuhause · Dachwohnung`,
		`role="dialog" aria-modal="true"`,
		`Offizielle Bezeichnung`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("building page should contain %q", want)
		}
	}
	if strings.Contains(body, `class="panel payment-status-panel"`) {
		t.Fatal("payment status must not repeat the unit inventory in a second panel")
	}
	if strings.Contains(body, `id="overview"`) || strings.Contains(body, `id="contacts"`) || strings.Contains(body, `id="appearance"`) {
		t.Fatal("inactive building sections must not render below the active workspace")
	}
}

func TestBuildingSettingsUsesOneEmptyUnitStateAndAnchoredActions(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})

	page := authedRequest(t, a, "manager@example.com", "/demo/app/settings/building?section=units")
	if got := strings.Count(page.Body.String(), "Noch keine Einheiten"); got != 1 {
		t.Fatalf("empty unit state count = %d, want 1", got)
	}

	invalidMeta := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building", url.Values{})
	if got := invalidMeta.Header().Get("Location"); got != "/demo/app/settings/building?section=overview&building=invalid" {
		t.Fatalf("invalid meta redirect = %q", got)
	}
	invalidUnit := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/building/units", url.Values{})
	if got := invalidUnit.Header().Get("Location"); got != "/demo/app/settings/building?section=units&unit=invalid#unit-add" {
		t.Fatalf("invalid unit redirect = %q", got)
	}
}
