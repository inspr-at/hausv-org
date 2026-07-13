package authz

import (
	"testing"

	"github.com/markus-barta/hausv-org/internal/store"
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
		CapabilityVote, CapabilityOversight,
	}
	for _, c := range caps {
		if HasCapability(store.RoleServiceProvider, c) {
			t.Errorf("service provider unexpectedly has capability %q", c)
		}
	}
}

func TestServiceProviderExcludedFromResidentAreasAndIssueCreation(t *testing.T) {
	if CanUseResidentAreas(store.RoleServiceProvider) {
		t.Error("service provider must not be able to use resident areas")
	}
	if CanCreateResidentIssue(store.RoleServiceProvider) {
		t.Error("service provider must not be able to create issues")
	}
	// Sanity: a resident is not excluded.
	if !CanUseResidentAreas(store.RoleResident) {
		t.Error("resident should be able to use resident areas")
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
