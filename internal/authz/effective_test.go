package authz

import (
	"github.com/inspr-at/hausv-org/internal/store"
	"reflect"
	"testing"
)

func TestEffectivePolicyDefaultsHAUSV699(t *testing.T) {
	policy := &Policy{Rules: store.CapabilityRules{OrgKey: "org"}}
	for _, role := range AllRoles {
		for _, capability := range AllCapabilities {
			actor := Actor{Person: "person@example.com", Tenant: "house", Role: role, Organisation: "org", Policy: policy}
			if got := Can(actor, capability, Resource{Tenant: "house"}); got != RoleHasCapability(role, capability) {
				t.Fatalf("default %s/%s = %v", role, capability, got)
			}
		}
	}
	if got := policy.EffectiveMatrix("org"); !reflect.DeepEqual(got, RoleMatrix) {
		t.Fatal("default presentation changed")
	}
}

func TestEffectivePolicyResolutionHAUSV699(t *testing.T) {
	const cap = CapabilityManageDocuments
	override := store.CapabilityOverride{OrgKey: "org", RoleFamily: "bewohner", Capability: string(cap), Allowed: true}
	global := store.UserCapabilityGrant{OrgKey: "org", Email: "person@example.com", Capability: string(cap), Effect: "grant"}
	deny := global
	deny.Effect = "deny"
	deny.TenantSlug = "house"
	scoped := global
	scoped.TenantSlug = "elsewhere"
	tests := []struct {
		name      string
		overrides []store.CapabilityOverride
		grants    []store.UserCapabilityGrant
		want      bool
	}{
		{name: "default"},
		{name: "family override", overrides: []store.CapabilityOverride{override}, want: true},
		{name: "global grant", grants: []store.UserCapabilityGrant{global}, want: true},
		{name: "scoped grant elsewhere", grants: []store.UserCapabilityGrant{scoped}},
		{name: "deny beats role and grant", overrides: []store.CapabilityOverride{override}, grants: []store.UserCapabilityGrant{global, deny}},
		{name: "deny beats later grant", grants: []store.UserCapabilityGrant{deny, global}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actor := Actor{Person: " Person@Example.com ", Tenant: "house", Role: store.RoleRenter, Organisation: "org", Policy: &Policy{Rules: store.CapabilityRules{OrgKey: "org", Overrides: test.overrides, Grants: test.grants}}}
			if got := Can(actor, cap, Resource{Tenant: "house"}); got != test.want {
				t.Fatalf("got %v want %v", got, test.want)
			}
			if Can(actor, cap, Resource{Tenant: "elsewhere"}) {
				t.Fatal("cross-tenant grant")
			}
			actor.Organisation = "other"
			if Can(actor, cap, Resource{Tenant: "house"}) {
				t.Fatal("cross-organisation grant")
			}
			actor.Organisation = "org"
			actor.Person = "other@example.com"
			actor.Policy.Rules.Overrides = nil
			if Can(actor, cap, Resource{Tenant: "house"}) {
				t.Fatal("cross-person grant")
			}
		})
	}
	// An organisation-wide deny beats a property grant regardless of order.
	global.Effect = "deny"
	scoped.TenantSlug = "house"
	actor := Actor{Person: global.Email, Tenant: "house", Role: store.RoleAdmin, Organisation: "org", Policy: &Policy{Rules: store.CapabilityRules{OrgKey: "org", Grants: []store.UserCapabilityGrant{scoped, global}}}}
	if Can(actor, cap, Resource{Tenant: "house"}) {
		t.Fatal("global deny did not win")
	}
	actor.Policy = &Policy{Unavailable: true}
	if Can(actor, cap, Resource{Tenant: "house"}) {
		t.Fatal("failed policy load granted admin defaults")
	}
}

func TestEffectivePolicyProtectedCapabilitiesHAUSV699(t *testing.T) {
	for _, role := range AllRoles {
		for _, capability := range []Capability{CapabilityManageUsers, CapabilityPlatformAdmin} {
			for _, effect := range []string{"grant", "deny"} {
				policy := &Policy{Rules: store.CapabilityRules{OrgKey: "org", Overrides: []store.CapabilityOverride{{OrgKey: "org", RoleFamily: RoleFamily(role), Capability: string(capability), Allowed: effect == "grant"}}, Grants: []store.UserCapabilityGrant{{OrgKey: "org", Email: "person@example.com", Capability: string(capability), Effect: effect}}}}
				actor := Actor{Person: "person@example.com", Tenant: "house", Organisation: "org", Role: role, Policy: policy}
				if got := Can(actor, capability, Resource{Tenant: "house"}); got != RoleHasCapability(role, capability) {
					t.Fatalf("protected capability changed: %s/%s/%s", role, capability, effect)
				}
			}
		}
	}
}
