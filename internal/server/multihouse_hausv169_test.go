package server

import (
	"net/http"
	"net/url"
	"testing"
)

// HAUSV-169 / HAUSV-135 at the HANDLER level. The store-level parity tests prove
// the scoped operations are correct; these prove the admin handlers actually use
// them. The defect lived here: editInvite wrote the whole profile back, so a
// manager of one house changed a person's standing in another.

func multiHouseResident(t *testing.T, a *app) {
	t.Helper()
	if _, err := a.inviteStore.Add(userProfile{
		Email: "anna@example.com", FirstName: "Anna", LastName: "Muster",
		// Deliberately NO explicit per-house memberships: this is the legacy shape
		// where every house falls back to the top-level role. It is the shape that
		// actually leaks — writing the whole profile changes the default, and with
		// it the person's role in every house that has no explicit entry.
		Role: roleRenter, Tenants: []string{"jhw22", "haus-b"},
		AuthMethods: defaultAuthMethods(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestEditInviteDoesNotChangeRoleInAnotherHouse(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin,
		Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	multiHouseResident(t, a)

	// The jhw22 admin promotes Anna to Verwalter — in jhw22.
	edit := authedFormRequest(t, a, "admin@example.com", "/app/settings/users/edit", url.Values{
		"orig_email": {"anna@example.com"},
		"email":      {"anna@example.com"},
		"first_name": {"Anna"},
		"last_name":  {"Muster"},
		"role":       {roleManager},
	})
	if edit.Code != http.StatusSeeOther {
		t.Fatalf("edit status = %d, want redirect", edit.Code)
	}

	profile, ok := a.inviteStore.Get("anna@example.com")
	if !ok {
		t.Fatal("profile gone")
	}
	if got := normalizeRole(profile.ForTenant("jhw22").Role); got != roleManager {
		t.Fatalf("jhw22 role = %q, want %q", got, roleManager)
	}
	// The whole point: haus-b keeps the role it had before the jhw22 edit.
	if got := normalizeRole(profile.ForTenant("haus-b").Role); got != roleRenter {
		t.Fatalf("haus-b role changed to %q — a house edit leaked into another house", got)
	}
}

func TestDeleteInviteOnlyRemovesTheCurrentHouse(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin,
		Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	multiHouseResident(t, a)

	del := authedFormRequest(t, a, "admin@example.com", "/app/settings/users/delete", url.Values{
		"email": {"anna@example.com"},
	})
	if del.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want redirect", del.Code)
	}

	profile, ok := a.inviteStore.Get("anna@example.com")
	if !ok {
		t.Fatal("removing one house must not delete the person globally")
	}
	if profile.HasTenant("jhw22") {
		t.Fatal("jhw22 membership should be gone")
	}
	if !profile.HasTenant("haus-b") {
		t.Fatal("haus-b membership must survive")
	}
	if got := normalizeRole(profile.ForTenant("haus-b").Role); got != roleRenter {
		t.Fatalf("haus-b role damaged: %q", got)
	}
}

// Single-house behaviour must be unchanged: removing the only house still
// removes the record entirely, exactly as the previous delete did.
func TestDeleteInviteRemovesRecordWhenNoHouseRemains(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "admin@example.com", Role: roleAdmin,
		Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	})
	if _, err := a.inviteStore.Add(userProfile{
		Email: "solo@example.com", Role: roleRenter,
		Tenants: []string{"jhw22"}, AuthMethods: defaultAuthMethods(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	del := authedFormRequest(t, a, "admin@example.com", "/app/settings/users/delete", url.Values{
		"email": {"solo@example.com"},
	})
	if del.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d", del.Code)
	}
	if _, ok := a.inviteStore.Get("solo@example.com"); ok {
		t.Fatal("the record should be gone once no house is left")
	}
}
