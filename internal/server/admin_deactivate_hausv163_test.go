package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// HAUSV-163 (deactivate slice): admins can suspend a user without deleting them.
// Deactivation blocks every login method, is reversible, is guarded against
// self-lockout and removing the last active admin, and never touches the
// break-glass bootstrap admin.

func TestDeactivatedUserCannotLogIn(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "invitee@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if !a.isAllowed("invitee@example.com", "demo") {
		t.Fatal("precondition: invitee should be allowed")
	}

	res := authedFormRequest(t, a, "admin@example.com", "/demo/app/settings/users/edit", url.Values{
		"orig_email":   {"invitee@example.com"},
		"email":        {"invitee@example.com"},
		"role":         {roleResident},
		"auth_methods": {"email", "oidc"},
		"deactivated":  {"1"},
	})
	if res.Code != http.StatusSeeOther || !strings.Contains(res.Header().Get("Location"), "updated") {
		t.Fatalf("deactivate redirect = %d %q", res.Code, res.Header().Get("Location"))
	}
	if a.isAllowed("invitee@example.com", "demo") {
		t.Fatal("deactivated user must not be allowed")
	}
	if a.isAuthMethodAllowed("invitee@example.com", "demo", authMethodEmail) || a.isAuthMethodAllowed("invitee@example.com", "demo", authMethodOIDC) {
		t.Fatal("deactivated user must not pass any auth-method check")
	}
	row := userRowForEmail(t, a.userRows(testTenantRef("demo")), "invitee@example.com")
	if !row.Deactivated || row.Status != "Deaktiviert" {
		t.Fatalf("roster should show Deaktiviert: %+v", row)
	}
}

func TestReactivateRestoresLogin(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "invitee@example.com", Role: roleResident, Deactivated: true, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if a.isAllowed("invitee@example.com", "demo") {
		t.Fatal("precondition: deactivated user should be blocked")
	}

	res := authedFormRequest(t, a, "admin@example.com", "/demo/app/settings/users/edit", url.Values{
		"orig_email":   {"invitee@example.com"},
		"email":        {"invitee@example.com"},
		"role":         {roleResident},
		"auth_methods": {"email", "oidc"},
		// deactivated unchecked -> reactivate
	})
	if res.Code != http.StatusSeeOther {
		t.Fatalf("reactivate redirect = %d", res.Code)
	}
	if !a.isAllowed("invitee@example.com", "demo") {
		t.Fatal("reactivated user should be allowed again")
	}
}

func TestAdminCannotDeactivateSelf(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	res := authedFormRequest(t, a, "admin@example.com", "/demo/app/settings/users/edit", url.Values{
		"orig_email":   {"admin@example.com"},
		"email":        {"admin@example.com"},
		"role":         {roleAdmin},
		"auth_methods": {"email"},
		"deactivated":  {"1"},
	})
	if res.Code != http.StatusSeeOther || !strings.Contains(res.Header().Get("Location"), "self_lockout") {
		t.Fatalf("self-deactivate redirect = %d %q, want self_lockout", res.Code, res.Header().Get("Location"))
	}
	if a.profileForTenant("admin@example.com", "demo").Deactivated {
		t.Fatal("admin must not be deactivated after refused self-deactivation")
	}
}

func TestSecondAdminCanBeDeactivated(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "second-admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	res := authedFormRequest(t, a, "admin@example.com", "/demo/app/settings/users/edit", url.Values{
		"orig_email":   {"second-admin@example.com"},
		"email":        {"second-admin@example.com"},
		"role":         {roleAdmin},
		"auth_methods": {"email", "oidc"},
		"deactivated":  {"1"},
	})
	if res.Code != http.StatusSeeOther || !strings.Contains(res.Header().Get("Location"), "updated") {
		t.Fatalf("deactivate second admin redirect = %d %q", res.Code, res.Header().Get("Location"))
	}
	if a.isAllowed("second-admin@example.com", "demo") {
		t.Fatal("deactivated second admin must not be allowed")
	}
}

// Break-glass admins are exempt from deactivation even if a record somehow
// carries the flag (defence-in-depth for the documented recovery path).
func TestBreakGlassAdminExemptFromDeactivation(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "other-admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.admins["breakglass@example.com"] = struct{}{}
	a.profiles["breakglass@example.com"] = userProfile{Email: "breakglass@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}
	if _, err := a.inviteStore.Add(userProfile{Email: "breakglass@example.com", Role: roleAdmin, Adopted: true, Deactivated: true, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if !a.isAllowed("breakglass@example.com", "demo") {
		t.Fatal("break-glass admin must stay allowed despite a deactivation override")
	}
	if !a.isAuthMethodAllowed("breakglass@example.com", "demo", authMethodEmail) {
		t.Fatal("break-glass auth method must be exempt from deactivation")
	}
}

// The edit dialog renders the deactivate toggle.
func TestUserSettingsPageRendersDeactivateToggle(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "invitee@example.com", Role: roleResident, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	page := authedRequest(t, a, "admin@example.com", "/demo/app/settings/users")
	if page.Code != http.StatusOK {
		t.Fatalf("status = %d", page.Code)
	}
	if !strings.Contains(page.Body.String(), `name="deactivated"`) {
		t.Fatal("edit dialog should render the deactivate toggle")
	}
}
