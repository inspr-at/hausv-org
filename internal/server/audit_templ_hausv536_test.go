package server

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAuditTemplSwitchDefaultsToLegacyRenderer(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})

	body := authedRequest(t, a, "manager@example.com", "/demo/app/audit").Body.String()
	if strings.Contains(body, "data-templ-audit") {
		t.Fatal("audit templ renderer must remain off by default")
	}
	if !strings.Contains(body, `class="app-main audit-screen"`) {
		t.Fatal("default audit response must still use the legacy renderer")
	}
}

func TestAuditTemplUsesSharedPortalNavigationForFourRoles(t *testing.T) {
	roles := []string{roleAdmin, roleManager, roleOwner, roleResident}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			email := role + "@example.com"
			a := newTestPortalApp(t, userProfile{Email: email, Role: role, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			a.portalTemplEnabled = true

			response := authedRequest(t, a, email, "/demo/app/audit")
			if response.Code != http.StatusOK {
				t.Fatalf("audit status = %d, want 200", response.Code)
			}
			body := response.Body.String()
			for _, want := range []string{
				"data-templ-audit",
				`href="/demo/app/audit" class="active" aria-current="page"`,
				"Noch nichts im Verlauf",
				"Was festgehalten wird",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("templ audit for %s should contain %q", role, want)
				}
			}
		})
	}
}

func TestAuditTemplKeepsFiltersDetailsAndManagementLinksReachable(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.portalTemplEnabled = true
	if err := a.auditStore.Append(auditEvent{
		TenantSlug: "demo",
		At:         time.Date(2026, 8, 15, 12, 30, 0, 0, time.Local),
		ActorEmail: "manager@example.com",
		ActorRole:  roleManager,
		Action:     auditActionLogin,
		TargetType: "session",
		TargetID:   "manager@example.com",
		Summary:    "Anmeldung erfolgreich",
		Details:    map[string]string{"auth_method": "E-Mail-Link"},
	}); err != nil {
		t.Fatalf("append audit event: %v", err)
	}

	body := authedRequest(t, a, "manager@example.com", "/demo/app/audit?action=login&q=E-Mail-Link").Body.String()
	for _, want := range []string{
		`method="get" action="/demo/app/audit"`,
		`name="action"`,
		`name="q" value="E-Mail-Link"`,
		"Anmeldung erfolgreich",
		"E-Mail-Link",
		`href="/demo/app/settings/users"`,
		`href="/demo/app/settings"`,
		"Einträge verstehen",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("templ audit should contain %q", want)
		}
	}
}
