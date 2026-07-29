package server

import "testing"

func TestBaseContextProvidesAuthenticatedPageIdentity(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "resident@example.com",
		FirstName:   "Ada",
		LastName:    "Lovelace",
		Role:        roleResident,
		Permissions: []string{permissionParking},
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	ac := authCtx{
		email:  "resident@example.com",
		role:   roleResident,
		tenant: a.tenants["jhw22"],
	}

	got := a.baseContext(ac)
	if len(got) != 13 {
		t.Fatalf("baseContext keys = %d, want 13: %#v", len(got), got)
	}
	if got["Tenant"] != ac.tenant || got["Email"] != ac.email || got["Role"] != ac.role {
		t.Fatalf("baseContext identity = %#v", got)
	}
	if got["DisplayName"] != "Ada Lovelace" || got["Initials"] != "AL" {
		t.Fatalf("baseContext profile = %#v", got)
	}
	if got["HouseName"] != "Janischhofweg 22" {
		t.Fatalf("baseContext house name = %#v", got["HouseName"])
	}
	if got["MapURL"] != "https://www.openstreetmap.org/search?query=Janischhofweg+22" {
		t.Fatalf("baseContext map URL = %#v", got["MapURL"])
	}
	sidebarMap, ok := got["SidebarMap"].(sidebarMapView)
	if !ok || !sidebarMap.Configured || len(sidebarMap.Tiles) == 0 || len(sidebarMap.Tiles) > 4 {
		t.Fatalf("baseContext sidebar map = %#v", got["SidebarMap"])
	}
	if got["IsAdmin"] != false || got["CanSeeParking"] != true {
		t.Fatalf("baseContext capabilities = %#v", got)
	}
	if got["CanViewEnergy"] != true || got["CanManageEnergy"] != false || got["CanControlEnergy"] != false {
		t.Fatalf("baseContext energy capabilities = %#v", got)
	}
}

func TestWithBaseKeepsPageOverridesExplicit(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email:       "manager@example.com",
		Role:        roleManager,
		Tenants:     []string{"jhw22"},
		AuthMethods: defaultAuthMethods(),
	})
	ac := authCtx{
		email:  "manager@example.com",
		role:   roleManager,
		tenant: a.tenants["jhw22"],
	}

	got := a.withBase(ac, map[string]any{
		"Title":         "Parkplatz-Abrechnung",
		"IsAdmin":       true,
		"CanSeeParking": true,
	})
	if got["Title"] != "Parkplatz-Abrechnung" {
		t.Fatalf("withBase Title = %#v", got["Title"])
	}
	if got["IsAdmin"] != true || got["CanSeeParking"] != true {
		t.Fatalf("withBase overrides = %#v", got)
	}
}
