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
	if len(got) != 15 {
		t.Fatalf("baseContext keys = %d, want 15: %#v", len(got), got)
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
	// An unclaimed/deleted energy profile is owner/admin-only. Delegated or
	// resident access begins only after the owner has completed onboarding.
	if got["CanViewEnergy"] != false || got["CanManageEnergy"] != false || got["CanManageHomeIdentity"] != false || got["CanControlEnergy"] != false {
		t.Fatalf("baseContext energy capabilities = %#v", got)
	}
	homeIdentity, ok := got["HomeIdentity"].(homeIdentityView)
	if !ok || homeIdentity.DisplayName != "Mein Zuhause" || homeIdentity.HasDisplayName || homeIdentity.HasUnit {
		t.Fatalf("baseContext default home identity = %#v", got["HomeIdentity"])
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
