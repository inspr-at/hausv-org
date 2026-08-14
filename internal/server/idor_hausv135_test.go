package server

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

// HAUSV-135: a manager of one tenant must not be able to edit or delete an
// invite that belongs to a different tenant, even though InviteStore.Get is
// keyed by email across all tenants.
func TestEditInviteRejectsForeignTenantTarget(t *testing.T) {
	a := newTestPortalApp(t, userProfile{
		Email: "manager@example.com", Role: roleManager,
		Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods(),
	})
	// A victim invite that belongs ONLY to another building.
	if _, err := a.inviteStore.Add(store.UserProfile{
		Email: "victim@other.example", Role: roleResident,
		Status: "Eingeladen", Tenants: []string{"other-haus"},
		AuthMethods: defaultAuthMethods(),
	}); err != nil {
		t.Fatalf("seed victim: %v", err)
	}

	// Manager of demo tries to rename the other tenant's invite to an address
	// they control — the account-takeover primitive.
	rr := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users/edit", url.Values{
		"orig_email": {"victim@other.example"},
		"email":      {"attacker@evil.example"},
		"role":       {roleResident},
	})
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("edit status = %d, want redirect", rr.Code)
	}

	// The victim invite must be untouched: still present under its original
	// email, still scoped to the other building, never renamed.
	if _, stolen := a.inviteStore.Get("attacker@evil.example"); stolen {
		t.Fatal("SECURITY: cross-tenant rename succeeded — attacker invite exists")
	}
	victim, ok := a.inviteStore.Get("victim@other.example")
	if !ok {
		t.Fatal("SECURITY: victim invite disappeared (cross-tenant mutation)")
	}
	if victim.HasTenant("demo") {
		t.Fatal("SECURITY: victim invite was pulled into the attacker's tenant")
	}

	// And delete must be refused too.
	del := authedFormRequest(t, a, "manager@example.com", "/demo/app/settings/users/delete", url.Values{
		"email": {"victim@other.example"},
	})
	if del.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want redirect", del.Code)
	}
	if _, ok := a.inviteStore.Get("victim@other.example"); !ok {
		t.Fatal("SECURITY: cross-tenant delete succeeded")
	}
}
