package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// HAUSV-163: config-sourced (env) users become editable in-app via adopt-on-edit,
// per-user login methods are editable, and admin management is guarded against
// self-lockout and against removing the break-glass bootstrap admin.

// adopt-on-edit: editing a config user creates an app override that wins at
// runtime (no redeploy) while the env record stays untouched.
func TestAdoptOnEditGrantsConfigUserPermission(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", FirstName: "Res", LastName: "Ident", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}

	if a.profileForTenant("resident@example.com", "jhw22").HasPermission(permissionParking) {
		t.Fatal("precondition: config resident should not have parking")
	}

	res := authedFormRequest(t, a, "admin@example.com", "/app/settings/users/edit", url.Values{
		"orig_email":   {"resident@example.com"},
		"email":        {"resident@example.com"},
		"first_name":   {"Res"},
		"last_name":    {"Ident"},
		"role":         {roleRenter},
		"permissions":  {"parking"},
		"auth_methods": {"email", "oidc"},
	})
	if res.Code != http.StatusSeeOther || !strings.Contains(res.Header().Get("Location"), "updated") {
		t.Fatalf("adopt-edit redirect = %d %q", res.Code, res.Header().Get("Location"))
	}

	if !a.profileForTenant("resident@example.com", "jhw22").HasPermission(permissionParking) {
		t.Fatal("adopted override should grant parking without a redeploy")
	}
	if a.profiles["resident@example.com"].HasPermission(permissionParking) {
		t.Fatal("env profile must not be mutated by adopt-on-edit")
	}
	rec, ok := a.inviteStore.Get("resident@example.com")
	if !ok || !rec.Adopted {
		t.Fatalf("store record should be marked adopted: %+v ok=%v", rec, ok)
	}
}

// A plain (non-adopted) store record must never escalate a config user — the
// override only wins when explicitly adopted (HAUSV-135 anti-escalation).
func TestPlainStoreRecordDoesNotEscalateConfigUser(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}
	// A store record claiming Admin but NOT marked adopted.
	if _, err := a.inviteStore.Add(userProfile{Email: "resident@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if got := a.profileForTenant("resident@example.com", "jhw22").Role; normalizeRole(got) != roleResident {
		t.Fatalf("non-adopted record must not escalate: role = %q", got)
	}
}

// Login methods are editable per user and enforced at sign-in.
func TestEditUserLoginMethods(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "invitee@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("seed invite: %v", err)
	}

	res := authedFormRequest(t, a, "admin@example.com", "/app/settings/users/edit", url.Values{
		"orig_email":   {"invitee@example.com"},
		"email":        {"invitee@example.com"},
		"role":         {roleResident},
		"auth_methods": {"oidc"},
	})
	if res.Code != http.StatusSeeOther || !strings.Contains(res.Header().Get("Location"), "updated") {
		t.Fatalf("redirect = %d %q", res.Code, res.Header().Get("Location"))
	}
	if a.isAuthMethodAllowed("invitee@example.com", "jhw22", authMethodEmail) {
		t.Fatal("email login should be disabled after restricting to OIDC")
	}
	if !a.isAuthMethodAllowed("invitee@example.com", "jhw22", authMethodOIDC) {
		t.Fatal("OIDC login should remain allowed")
	}
}

// An empty login-method selection falls back to both methods rather than
// persisting zero methods (which would lock the user out).
func TestEditUserEmptyLoginMethodsFallsBackToBoth(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "invitee@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: []string{authMethodOIDC}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	res := authedFormRequest(t, a, "admin@example.com", "/app/settings/users/edit", url.Values{
		"orig_email": {"invitee@example.com"},
		"email":      {"invitee@example.com"},
		"role":       {roleResident},
		// no auth_methods checked
	})
	if res.Code != http.StatusSeeOther {
		t.Fatalf("redirect = %d", res.Code)
	}
	if !a.isAuthMethodAllowed("invitee@example.com", "jhw22", authMethodEmail) || !a.isAuthMethodAllowed("invitee@example.com", "jhw22", authMethodOIDC) {
		t.Fatal("empty selection should fall back to both login methods")
	}
}

// Self-lockout: an admin cannot demote themselves out of the Admin role.
func TestAdminCannotDemoteSelf(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	res := authedFormRequest(t, a, "admin@example.com", "/app/settings/users/edit", url.Values{
		"orig_email":   {"admin@example.com"},
		"email":        {"admin@example.com"},
		"role":         {roleResident},
		"auth_methods": {"email"},
	})
	if res.Code != http.StatusSeeOther || !strings.Contains(res.Header().Get("Location"), "self_lockout") {
		t.Fatalf("self-demote redirect = %d %q, want self_lockout", res.Code, res.Header().Get("Location"))
	}
	if normalizeRole(a.roleFor("admin@example.com", "jhw22")) != roleAdmin {
		t.Fatal("admin must remain admin after refused self-demotion")
	}
}

// Self-lockout: a store-backed admin cannot delete their own account.
func TestAdminCannotDeleteSelf(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "config-admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "boss@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	res := authedFormRequest(t, a, "boss@example.com", "/app/settings/users/delete", url.Values{
		"email": {"boss@example.com"},
	})
	if res.Code != http.StatusSeeOther || !strings.Contains(res.Header().Get("Location"), "self_lockout") {
		t.Fatalf("self-delete redirect = %d %q, want self_lockout", res.Code, res.Header().Get("Location"))
	}
	if _, ok := a.inviteStore.Get("boss@example.com"); !ok {
		t.Fatal("admin must survive refused self-deletion")
	}
}

// A different admin can still be removed (guard does not over-block).
func TestSecondAdminCanBeDeleted(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "config-admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "boss@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	res := authedFormRequest(t, a, "config-admin@example.com", "/app/settings/users/delete", url.Values{
		"email": {"boss@example.com"},
	})
	if res.Code != http.StatusSeeOther || !strings.Contains(res.Header().Get("Location"), "deleted") {
		t.Fatalf("delete redirect = %d %q, want deleted", res.Code, res.Header().Get("Location"))
	}
	if _, ok := a.inviteStore.Get("boss@example.com"); ok {
		t.Fatal("second admin should have been deleted")
	}
}

// Break-glass: an ADMIN_EMAILS bootstrap admin is protected from in-app edits and
// always resolves to an allowed admin (the documented recovery path).
func TestBreakGlassAdminProtectedAndAllowed(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "other-admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.admins["breakglass@example.com"] = struct{}{}
	a.profiles["breakglass@example.com"] = userProfile{Email: "breakglass@example.com", Role: roleAdmin, Status: "Aktiv", Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}

	// clean recovery scenario: no override -> admin + allowed
	if !a.isAllowed("breakglass@example.com", "jhw22") {
		t.Fatal("break-glass admin must always be allowed to sign in")
	}
	if normalizeRole(a.roleFor("breakglass@example.com", "jhw22")) != roleAdmin {
		t.Fatal("break-glass admin must resolve to Admin")
	}

	// editing a break-glass admin is refused.
	res := authedFormRequest(t, a, "other-admin@example.com", "/app/settings/users/edit", url.Values{
		"orig_email":   {"breakglass@example.com"},
		"email":        {"breakglass@example.com"},
		"role":         {roleResident},
		"auth_methods": {"email"},
	})
	if res.Code != http.StatusSeeOther || !strings.Contains(res.Header().Get("Location"), "protected") {
		t.Fatalf("edit break-glass redirect = %d %q, want protected", res.Code, res.Header().Get("Location"))
	}

	// defence-in-depth: even a manually-injected demoting override cannot block
	// the break-glass login floor.
	if _, err := a.inviteStore.Add(userProfile{Email: "breakglass@example.com", Role: roleResident, Adopted: true, Tenants: []string{"jhw22"}, AuthMethods: []string{authMethodOIDC}}); err != nil {
		t.Fatalf("inject override: %v", err)
	}
	if !a.isAllowed("breakglass@example.com", "jhw22") {
		t.Fatal("break-glass admin must remain allowed to sign in even with a demoting override")
	}
}

// The Benutzer & Rechte page renders the login-method controls and opens an
// edit affordance for config-sourced users (adopt-on-edit).
func TestUserSettingsPageRendersLoginMethodsAndConfigEdit(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	a.profiles["resident@example.com"] = userProfile{Email: "resident@example.com", FirstName: "Res", LastName: "Ident", Role: roleRenter, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}

	page := authedRequest(t, a, "admin@example.com", "/app/settings/users")
	if page.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", page.Code)
	}
	body := page.Body.String()
	if !strings.Contains(body, `name="auth_methods"`) {
		t.Fatal("login-method checkboxes should render")
	}
	if !strings.Contains(body, `id="edit-resident@example.com"`) {
		t.Fatal("config resident should have an edit dialog (adopt-on-edit)")
	}
	if !strings.Contains(body, "aus Konfiguration") {
		t.Fatal("config-sourced hint should render")
	}
}

// Permission check: a resident cannot reach the user-management handlers.
func TestNonAdminCannotManageUsers(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	res := authedFormRequest(t, a, "resident@example.com", "/app/settings/users/edit", url.Values{
		"orig_email": {"resident@example.com"},
		"email":      {"resident@example.com"},
		"role":       {roleAdmin},
	})
	if res.Code != http.StatusForbidden {
		t.Fatalf("resident edit status = %d, want 403", res.Code)
	}
}

// Permission check: a manager (manage-users but not platform-admin) cannot mint
// an admin by editing another user.
func TestManagerCannotEscalateToAdmin(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()})
	if _, err := a.inviteStore.Add(userProfile{Email: "resident@example.com", Role: roleResident, Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods()}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	res := authedFormRequest(t, a, "manager@example.com", "/app/settings/users/edit", url.Values{
		"orig_email":   {"resident@example.com"},
		"email":        {"resident@example.com"},
		"role":         {roleAdmin},
		"auth_methods": {"email"},
	})
	if res.Code != http.StatusSeeOther || !strings.Contains(res.Header().Get("Location"), "forbidden_role") {
		t.Fatalf("manager escalate redirect = %d %q, want forbidden_role", res.Code, res.Header().Get("Location"))
	}
	if normalizeRole(a.profileForTenant("resident@example.com", "jhw22").Role) == roleAdmin {
		t.Fatal("resident must not have been escalated to admin")
	}
}
