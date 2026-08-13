package server

import (
	"testing"
	"time"

	"github.com/markus-barta/hausv-org/internal/energy"
)

func TestEnergyHomeScopeAuthorizationDoesNotCrossUnits(t *testing.T) {
	a := newTestPortalApp(t,
		userProfile{Email: "owner11@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()},
	)
	a.profiles["owner12@example.com"] = userProfile{Email: "owner12@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	if err := a.unitStore.SetTenantUnits("jhw22", []unit{
		{ID: "top-11", TenantSlug: "jhw22", Label: "Top 11", UnitType: unitTypeResidential, OwnerEmails: []string{"owner11@example.com"}},
		{ID: "top-12", TenantSlug: "jhw22", Label: "Top 12", UnitType: unitTypeResidential, OwnerEmails: []string{"owner12@example.com"}},
	}); err != nil {
		t.Fatalf("seed units: %v", err)
	}
	for _, item := range []struct{ home, unit string }{{"top-11", "top-11"}, {"top-12", "top-12"}} {
		profile := energy.DefaultProfileForHome("jhw22", item.home, time.Now())
		profile.HomeType = energy.HomeApartment
		profile.UnitID = item.unit
		profile.HouseholdName = item.home
		profile.OnboardingComplete = true
		if err := a.energyStore.ForHome(item.home).SaveProfile(profile); err != nil {
			t.Fatalf("save %s: %v", item.home, err)
		}
	}

	owner11 := authCtx{email: "owner11@example.com", role: roleResident, tenant: a.tenants["jhw22"]}
	if store, ok := a.energyStoreForHome(owner11, "top-11"); !ok || store == nil {
		t.Fatal("owner of top-11 denied their home")
	}
	if store, ok := a.energyStoreForHome(owner11, "top-12"); ok || store != nil {
		t.Fatal("owner of top-11 gained access to top-12")
	}
}
