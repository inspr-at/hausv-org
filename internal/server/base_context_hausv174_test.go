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
	if len(got) != 7 {
		t.Fatalf("baseContext keys = %d, want 7: %#v", len(got), got)
	}
	if got["Tenant"] != ac.tenant || got["Email"] != ac.email || got["Role"] != ac.role {
		t.Fatalf("baseContext identity = %#v", got)
	}
	if got["DisplayName"] != "Ada Lovelace" || got["Initials"] != "AL" {
		t.Fatalf("baseContext profile = %#v", got)
	}
	if got["IsAdmin"] != false || got["CanSeeParking"] != true {
		t.Fatalf("baseContext capabilities = %#v", got)
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
