package server

import "testing"

// VerwaltungSidebar and VerwaltungMobileHeader receive the same shell value.
// This locks their role source to the effective tenant role so neither shell
// may derive a different label from the organisation or directory profile.
func TestRoleDisplayUsesEffectiveRoleInEveryVerwaltungShellHAUSV606(t *testing.T) {
	tests := []struct {
		role string
	}{
		{role: roleAdmin},
		{role: roleManager},
	}
	for _, test := range tests {
		t.Run(test.role, func(t *testing.T) {
			const email = "verwaltung@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: test.role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			ac := authCtx{email: email, role: test.role, tenant: a.tenants["demo"], tenantRef: testTenantRef("demo")}
			shell := a.verwaltungShell(t.Context(), &ac, "rechte")
			if shell.Organisation.RoleLabel != test.role {
				t.Fatalf("shared desktop/mobile role label = %q, want %q", shell.Organisation.RoleLabel, test.role)
			}
			if shell.Organisation.Active != "rechte" {
				t.Fatalf("rights shell active key = %q", shell.Organisation.Active)
			}
		})
	}
}
