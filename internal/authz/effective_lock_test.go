package authz

import (
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

// HAUSV-699: switching off the Verwaltung family's Oversight locked the
// administrator out of the rights page. The lock ignores such overrides and
// personal denies for the Verwaltung family, but stays configurable elsewhere.
func TestOversightStaysLockedForVerwaltung(t *testing.T) {
	policy := &Policy{Rules: store.CapabilityRules{OrgKey: "org",
		Overrides: []store.CapabilityOverride{{OrgKey: "org", RoleFamily: "verwaltung", Capability: string(CapabilityOversight), Allowed: false}},
		Grants:    []store.UserCapabilityGrant{{OrgKey: "org", Email: "vera@example.com", Capability: string(CapabilityOversight), Effect: "deny"}},
	}}
	if !LockedFor("verwaltung", CapabilityOversight) || LockedFor("bewohner", CapabilityOversight) {
		t.Fatal("Oversight must be locked for Verwaltung only")
	}
	if !policy.RoleAllowed("org", "Admin", CapabilityOversight) {
		t.Fatal("override must not remove Oversight from the Verwaltung family")
	}
	if !policy.Can(Actor{Person: "vera@example.com", Role: "Admin", Organisation: "org"}, CapabilityOversight) {
		t.Fatal("personal deny must not remove Oversight from an administrator")
	}
}
