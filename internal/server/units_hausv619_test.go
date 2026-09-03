package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestHAUSV619SeededUnitsAreVisibleInBuildingAndUserSettings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "vera.verwalter@musterstadt.example", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.profiles["matthias.mieter@musterstadt.example"] = userProfile{
		Email: "matthias.mieter@musterstadt.example", FirstName: "Matthias", LastName: "Dorn",
		Role: roleRenter, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	}
	if err := testUnitRepository(t, a, "demo").SetUnits([]unit{
		{Label: "Top 1", UnitType: unitTypeResidential, OwnerEmails: []string{"alina.eigentuemer@musterstadt.example"}},
		{Label: "Top 3", UnitType: unitTypeResidential, RenterEmails: []string{"matthias.mieter@musterstadt.example"}},
		{Label: "Stellplatz 1", UnitType: unitTypeParking},
	}); err != nil {
		t.Fatalf("seed units: %v", err)
	}
	building := authedRequest(t, a, "vera.verwalter@musterstadt.example", "/demo/app/settings/building?section=units")
	if building.Code != http.StatusOK {
		t.Fatalf("building status = %d", building.Code)
	}
	for _, want := range []string{"Top 1", "Stellplatz 1"} {
		if !strings.Contains(building.Body.String(), want) {
			t.Errorf("building page missing %q", want)
		}
	}
	users := authedRequest(t, a, "vera.verwalter@musterstadt.example", "/demo/app/settings/users")
	if users.Code != http.StatusOK {
		t.Fatalf("users status = %d", users.Code)
	}
	for _, want := range []string{"Matthias Dorn", "Top 3"} {
		if !strings.Contains(users.Body.String(), want) {
			t.Errorf("users page missing %q", want)
		}
	}
}
