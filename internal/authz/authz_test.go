package authz

import (
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

// The service-provider (Dienstleister) role is the only externally-invited role
// and the most security-sensitive one. These tests pin its policy directly at
// the authz layer, rather than only transitively through the server handlers
// (HAUSV-128).

func TestServiceProviderRoleNormalizationAndPredicate(t *testing.T) {
	for _, alias := range []string{
		"Dienstleister", "dienstleister", "Handwerker", "service-provider",
		"service_provider", "service provider", "contractor", "vendor", "external",
	} {
		if !IsServiceProviderRole(alias) {
			t.Errorf("IsServiceProviderRole(%q) = false, want true", alias)
		}
	}
	for _, other := range []string{
		store.RoleAdmin, store.RoleManager, store.RoleOwner, store.RoleRenter,
		store.RoleBeirat, store.RoleResident, "",
	} {
		if IsServiceProviderRole(other) {
			t.Errorf("IsServiceProviderRole(%q) = true, want false", other)
		}
	}
}

func TestServiceProviderHasNoCapabilities(t *testing.T) {
	caps := []Capability{
		CapabilityPlatformAdmin, CapabilityManageUsers, CapabilityManageParking,
		CapabilityManageAnnouncements, CapabilityManageDocuments, CapabilityManageIssues,
		CapabilityManageVotes, CapabilityManageBuilding, CapabilityOwnerDocuments,
		CapabilityVote, CapabilityOversight, CapabilityManageLeases, CapabilityApproveValorisation,
	}
	for _, c := range caps {
		if RoleHasCapability(store.RoleServiceProvider, c) {
			t.Errorf("service provider unexpectedly has capability %q", c)
		}
	}
}

func TestServiceProviderExcludedFromResidentAreasAndIssueCreation(t *testing.T) {
	resource := Resource{Tenant: "demo"}
	provider := Actor{Person: "provider@example.invalid", Tenant: "demo", Role: store.RoleServiceProvider}
	resident := Actor{Person: "resident@example.invalid", Tenant: "demo", Role: store.RoleResident}
	if RoleCanUseResidentAreas(store.RoleServiceProvider) {
		t.Error("service provider must not be able to use resident areas")
	}
	if CanCreateResidentIssue(provider, resource) {
		t.Error("service provider must not be able to create issues")
	}
	// Sanity: a resident is not excluded.
	if !RoleCanUseResidentAreas(store.RoleResident) {
		t.Error("resident should be able to use resident areas")
	}
	if !CanCreateResidentIssue(resident, resource) {
		t.Error("resident should be able to create issues")
	}
}

func TestCanRefusesResourceFromAnotherTenant(t *testing.T) {
	actor := Actor{Person: "manager@example.invalid", Tenant: "demo", Role: store.RoleManager}
	if !Can(actor, CapabilityManageDocuments, Resource{Tenant: "demo"}) {
		t.Fatal("manager should be allowed to manage documents in their tenant")
	}
	if Can(actor, CapabilityManageDocuments, Resource{Tenant: "other-house"}) {
		t.Fatal("manager must not be allowed to manage a resource from another tenant")
	}
	if CanCreateResidentIssue(actor, Resource{Tenant: "other-house"}) {
		t.Fatal("composite authorization helpers must also refuse another tenant")
	}
}

func TestCanPreservesRoleCapabilityMatrixWithinTenant(t *testing.T) {
	roles := []string{
		store.RoleAdmin, store.RoleManager, store.RoleOwner, store.RoleRenter,
		store.RoleBeirat, store.RoleResident, store.RoleServiceProvider, "", "unknown",
	}
	capabilities := []Capability{
		CapabilityPlatformAdmin, CapabilityManageUsers, CapabilityManageParking,
		CapabilityManageAnnouncements, CapabilityManageDocuments, CapabilityManageIssues,
		CapabilityManageVotes, CapabilityManageBuilding, CapabilityOwnerDocuments,
		CapabilityVote, CapabilityOversight, CapabilityManageEnergy, CapabilityControlEnergy,
		CapabilityManageLeases, CapabilityApproveValorisation,
	}
	resource := Resource{Tenant: "demo"}
	for _, role := range roles {
		actor := Actor{Person: "person@example.invalid", Tenant: "demo", Role: role}
		for _, capability := range capabilities {
			if got, want := Can(actor, capability, resource), RoleHasCapability(role, capability); got != want {
				t.Errorf("Can(role=%q, capability=%q) = %v, want existing role decision %v", role, capability, got, want)
			}
		}
	}
}

// Invariants that hold regardless of how many intermediate states exist: a
// provider may advance an open issue toward completion, but may NEVER reopen,
// reject, or mark it duplicate.
func TestServiceProviderTransitionInvariants(t *testing.T) {
	allowed := []struct{ from, to string }{
		{store.IssueStatusNew, store.IssueStatusProgress},
		{store.IssueStatusNew, store.IssueStatusDone},
		{store.IssueStatusProgress, store.IssueStatusDone},
	}
	for _, tc := range allowed {
		if !CanServiceProviderTransition(tc.from, tc.to) {
			t.Errorf("CanServiceProviderTransition(%q, %q) = false, want true", tc.from, tc.to)
		}
	}

	blocked := []struct{ from, to string }{
		{store.IssueStatusProgress, store.IssueStatusNew},      // reopen
		{store.IssueStatusDone, store.IssueStatusProgress},     // reopen a closed issue
		{store.IssueStatusProgress, store.IssueStatusRejected}, // reject
		{store.IssueStatusProgress, store.IssueStatusDuplicate},
		{store.IssueStatusNew, store.IssueStatusRejected},
	}
	for _, tc := range blocked {
		if CanServiceProviderTransition(tc.from, tc.to) {
			t.Errorf("CanServiceProviderTransition(%q, %q) = true, want false", tc.from, tc.to)
		}
	}
}

// The full forward chain: any forward step is allowed, any backward step is not,
// and reopening to Neu is never allowed.
func TestServiceProviderTransitionFullChain(t *testing.T) {
	chain := []string{
		store.IssueStatusNew, store.IssueStatusAccepted, store.IssueStatusScheduled,
		store.IssueStatusProgress, store.IssueStatusDone,
	}
	for i := range chain {
		for j := range chain {
			from, to := chain[i], chain[j]
			// Forward (or same) is allowed, except a move TO Neu is never allowed.
			want := j >= i && to != store.IssueStatusNew
			if got := CanServiceProviderTransition(from, to); got != want {
				t.Errorf("CanServiceProviderTransition(%q, %q) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestAuditVisibilitySeparatesAccessFromFullTenantHistory(t *testing.T) {
	resource := Resource{Tenant: "demo"}
	for _, role := range []string{
		store.RoleAdmin, store.RoleManager, store.RoleOwner, store.RoleRenter,
		store.RoleBeirat, store.RoleResident, store.RoleServiceProvider,
	} {
		if !CanViewAudit(Actor{Person: "person@example.invalid", Tenant: "demo", Role: role}, resource) {
			t.Errorf("CanViewAudit(%q) = false, want true", role)
		}
	}
	for _, role := range []string{store.RoleAdmin, store.RoleManager} {
		if !CanViewFullAudit(Actor{Person: "person@example.invalid", Tenant: "demo", Role: role}, resource) {
			t.Errorf("CanViewFullAudit(%q) = false, want true", role)
		}
	}
	for _, role := range []string{store.RoleOwner, store.RoleRenter, store.RoleBeirat, store.RoleResident, store.RoleServiceProvider, "", "unknown"} {
		if CanViewFullAudit(Actor{Person: "person@example.invalid", Tenant: "demo", Role: role}, resource) {
			t.Errorf("CanViewFullAudit(%q) = true, want false", role)
		}
	}
}
