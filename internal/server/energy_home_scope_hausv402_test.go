package server

import (
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
)

func TestEnergyHomeScopeAuthorizationDoesNotCrossUnits(t *testing.T) {
	a := newTestPortalApp(t,
		userProfile{Email: "owner11@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()},
	)
	a.profiles["owner12@example.com"] = userProfile{Email: "owner12@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{ID: "einheit-12", TenantSlug: "demo", Label: "Einheit 12", UnitType: unitTypeResidential, OwnerEmails: []string{"owner11@example.com"}},
		{ID: "top-12", TenantSlug: "demo", Label: "Top 12", UnitType: unitTypeResidential, OwnerEmails: []string{"owner12@example.com"}},
	}); err != nil {
		t.Fatalf("seed units: %v", err)
	}
	for _, item := range []struct{ home, unit string }{{"einheit-12", "einheit-12"}, {"top-12", "top-12"}} {
		profile := energy.DefaultProfileForHome("demo", item.home, time.Now())
		profile.HomeType = energy.HomeApartment
		profile.UnitID = item.unit
		profile.HouseholdName = item.home
		profile.OnboardingComplete = true
		if err := a.energyStore.ForHome(item.home).SaveProfile(profile); err != nil {
			t.Fatalf("save %s: %v", item.home, err)
		}
	}

	owner11 := authCtx{email: "owner11@example.com", role: roleResident, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo")}
	if store, ok := a.energyStoreForHome(owner11, "einheit-12"); !ok || store == nil {
		t.Fatal("owner of einheit-12 denied their home")
	}
	if store, ok := a.energyStoreForHome(owner11, "top-12"); ok || store != nil {
		t.Fatal("owner of einheit-12 gained access to top-12")
	}
}
