package server

import (
	"html/template"
	"testing"
)

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
	if len(got) != 16 {
		t.Fatalf("baseContext keys = %d, want 16: %#v", len(got), got)
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
	sidebarAddress, ok := got["SidebarAddress"].(sidebarAddressView)
	if !ok || sidebarAddress.Full != "Janischhofweg 22" || sidebarAddress.Primary != "Janischhofweg 22" || sidebarAddress.HasLocality {
		t.Fatalf("baseContext sidebar address = %#v", got["SidebarAddress"])
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

func TestSidebarAddressKeepsAStableMobileHouseIdentity(t *testing.T) {
	tests := []struct {
		name     string
		tenant   tenantConfig
		primary  string
		locality string
	}{
		{
			name:     "portal abbreviation and postcode",
			tenant:   tenantConfig{Name: "JHW22-Portal", Address: "Janischhofweg 22, 8043 Graz"},
			primary:  "Janischhofweg 22",
			locality: "Graz",
		},
		{
			name:     "named house",
			tenant:   tenantConfig{Name: "Haus Kirchweg", Address: "Kirchweg 8, 8043 Graz, Österreich"},
			primary:  "Kirchweg 8",
			locality: "Graz",
		},
		{
			name:    "generic portal falls back to street",
			tenant:  tenantConfig{Name: "WEG Portal", Address: "Sonnenweg 4"},
			primary: "Sonnenweg 4",
		},
		{
			name:    "missing address falls back to portal name",
			tenant:  tenantConfig{Name: "Haus am Park"},
			primary: "Haus am Park",
		},
		{
			name:    "generic pilot address falls back to house name",
			tenant:  tenantConfig{Name: "Haus Eltern", Address: "Pilot Eltern"},
			primary: "Haus Eltern",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sidebarAddressForTenant(tt.tenant)
			if got.Primary != tt.primary || got.Locality != tt.locality || got.HasLocality != (tt.locality != "") {
				t.Fatalf("sidebarAddressForTenant(%#v) = %#v", tt.tenant, got)
			}
		})
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

func TestTenantTemplateViewTrustsOnlyServerOwnedHeroRoute(t *testing.T) {
	for _, raw := range []string{"/tenant-hero/jhw22", "/assets/jhw22-hero.jpg"} {
		internal := tenantTemplateViewFrom(tenantConfig{Slug: "jhw22", HeroImageURL: raw})
		if got, ok := internal.HeroImageURL.(template.URL); !ok || got != template.URL(raw) {
			t.Fatalf("internal hero URL = %#v, want trusted same-origin route %q", internal.HeroImageURL, raw)
		}
	}

	for _, raw := range []string{"javascript:alert(1)", "https://example.com/hero.jpg", "/assets/../private", "/assets/hero.jpg?variant=external"} {
		configured := tenantTemplateViewFrom(tenantConfig{Slug: "jhw22", HeroImageURL: raw})
		if got, ok := configured.HeroImageURL.(string); !ok || got != raw {
			t.Fatalf("configured hero URL = %#v, want untrusted string %q", configured.HeroImageURL, raw)
		}
	}
}
