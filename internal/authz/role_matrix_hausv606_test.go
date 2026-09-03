package authz

import (
	"slices"
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestRoleCapabilityMatrixHAUSV606(t *testing.T) {
	all := []Capability{
		CapabilityPlatformAdmin,
		CapabilityManageUsers,
		CapabilityManageParking,
		CapabilityManageAnnouncements,
		CapabilityManageDocuments,
		CapabilityManageIssues,
		CapabilityManageVotes,
		CapabilityManageBuilding,
		CapabilityOwnerDocuments,
		CapabilityVote,
		CapabilityOversight,
		CapabilityManageEnergy,
		CapabilityControlEnergy,
	}
	want := map[string][]Capability{
		store.RoleAdmin:           slices.Clone(all),
		store.RoleManager:         {CapabilityManageUsers, CapabilityManageAnnouncements, CapabilityManageDocuments, CapabilityManageIssues, CapabilityManageVotes, CapabilityManageBuilding, CapabilityOversight, CapabilityManageEnergy, CapabilityControlEnergy},
		store.RoleOwner:           {CapabilityOwnerDocuments, CapabilityVote, CapabilityManageEnergy, CapabilityControlEnergy},
		store.RoleRenter:          {},
		store.RoleBeirat:          {CapabilityOversight},
		store.RoleResident:        {},
		store.RoleServiceProvider: {},
	}
	for role, expected := range want {
		got := make([]Capability, 0, len(all))
		for _, capability := range all {
			if RoleHasCapability(role, capability) {
				got = append(got, capability)
			}
		}
		if !slices.Equal(got, expected) {
			t.Errorf("capabilities for %q = %v, want %v", role, got, expected)
		}
	}
	if len(want) != 7 {
		t.Fatalf("matrix covers %d roles, want all 7 persisted roles", len(want))
	}
}
