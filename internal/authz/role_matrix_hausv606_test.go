package authz

import (
	"testing"

	"github.com/inspr-at/hausv-org/internal/store"
)

func TestRoleCapabilityMatrixHAUSV606(t *testing.T) {
	want := map[string]map[Capability]bool{
		store.RoleAdmin: {
			CapabilityPlatformAdmin: true, CapabilityManageUsers: true, CapabilityManageParking: true,
			CapabilityManageAnnouncements: true, CapabilityManageDocuments: true, CapabilityManageIssues: true,
			CapabilityManageVotes: true, CapabilityManageBuilding: true, CapabilityOwnerDocuments: true,
			CapabilityVote: true, CapabilityOversight: true, CapabilityManageEnergy: true, CapabilityControlEnergy: true,
		},
		store.RoleManager: {
			CapabilityPlatformAdmin: false, CapabilityManageUsers: true, CapabilityManageParking: false,
			CapabilityManageAnnouncements: true, CapabilityManageDocuments: true, CapabilityManageIssues: true,
			CapabilityManageVotes: true, CapabilityManageBuilding: true, CapabilityOwnerDocuments: false,
			CapabilityVote: false, CapabilityOversight: true, CapabilityManageEnergy: true, CapabilityControlEnergy: true,
		},
		store.RoleOwner: {
			CapabilityPlatformAdmin: false, CapabilityManageUsers: false, CapabilityManageParking: false,
			CapabilityManageAnnouncements: false, CapabilityManageDocuments: false, CapabilityManageIssues: false,
			CapabilityManageVotes: false, CapabilityManageBuilding: false, CapabilityOwnerDocuments: true,
			CapabilityVote: true, CapabilityOversight: false, CapabilityManageEnergy: true, CapabilityControlEnergy: true,
		},
		store.RoleBeirat: {
			CapabilityPlatformAdmin: false, CapabilityManageUsers: false, CapabilityManageParking: false,
			CapabilityManageAnnouncements: false, CapabilityManageDocuments: false, CapabilityManageIssues: false,
			CapabilityManageVotes: false, CapabilityManageBuilding: false, CapabilityOwnerDocuments: false,
			CapabilityVote: false, CapabilityOversight: true, CapabilityManageEnergy: false, CapabilityControlEnergy: false,
		},
		store.RoleRenter: {
			CapabilityPlatformAdmin: false, CapabilityManageUsers: false, CapabilityManageParking: false,
			CapabilityManageAnnouncements: false, CapabilityManageDocuments: false, CapabilityManageIssues: false,
			CapabilityManageVotes: false, CapabilityManageBuilding: false, CapabilityOwnerDocuments: false,
			CapabilityVote: false, CapabilityOversight: false, CapabilityManageEnergy: false, CapabilityControlEnergy: false,
		},
		store.RoleResident: {
			CapabilityPlatformAdmin: false, CapabilityManageUsers: false, CapabilityManageParking: false,
			CapabilityManageAnnouncements: false, CapabilityManageDocuments: false, CapabilityManageIssues: false,
			CapabilityManageVotes: false, CapabilityManageBuilding: false, CapabilityOwnerDocuments: false,
			CapabilityVote: false, CapabilityOversight: false, CapabilityManageEnergy: false, CapabilityControlEnergy: false,
		},
		store.RoleServiceProvider: {
			CapabilityPlatformAdmin: false, CapabilityManageUsers: false, CapabilityManageParking: false,
			CapabilityManageAnnouncements: false, CapabilityManageDocuments: false, CapabilityManageIssues: false,
			CapabilityManageVotes: false, CapabilityManageBuilding: false, CapabilityOwnerDocuments: false,
			CapabilityVote: false, CapabilityOversight: false, CapabilityManageEnergy: false, CapabilityControlEnergy: false,
		},
	}
	for _, role := range AllRoles {
		expected, ok := want[role]
		if !ok {
			t.Fatalf("missing explicit expectations for role %q", role)
		}
		if len(expected) != len(AllCapabilities) {
			t.Fatalf("role %q has %d expectations, want %d", role, len(expected), len(AllCapabilities))
		}
		for _, capability := range AllCapabilities {
			actor := Actor{Person: "matrix@example.com", Tenant: "haus-1", Role: role}
			if got := Can(actor, capability, Resource{Tenant: "haus-1"}); got != expected[capability] {
				t.Errorf("Can(%q, %q) = %v, want %v", role, capability, got, expected[capability])
			}
		}
	}
	if len(want) != len(AllRoles) {
		t.Fatalf("matrix covers %d roles, want all %d persisted roles", len(want), len(AllRoles))
	}
}

func TestPortalRoleMatrixIsCompleteAndCapabilityBackedHAUSV606(t *testing.T) {
	wantCells := len(RoleFamilies) * len(RoleAreas)
	if len(RoleMatrix) != wantCells {
		t.Fatalf("RoleMatrix has %d cells, want %d", len(RoleMatrix), wantCells)
	}
	seen := make(map[string]bool, wantCells)
	families := make(map[string]bool, len(RoleFamilies))
	areas := make(map[string]bool, len(RoleAreas))
	for _, family := range RoleFamilies {
		families[family.Key] = true
	}
	for _, area := range RoleAreas {
		areas[area.Key] = true
	}
	for _, cell := range RoleMatrix {
		key := cell.FamilyKey + "/" + cell.AreaKey
		if !families[cell.FamilyKey] || !areas[cell.AreaKey] {
			t.Fatalf("matrix cell %q references an unknown family or area", key)
		}
		if seen[key] {
			t.Fatalf("duplicate matrix cell %q", key)
		}
		seen[key] = true
		for _, grant := range cell.Grants {
			if grant.Capability == "" {
				continue
			}
			actor := Actor{Person: "matrix@example.com", Tenant: "haus-1", Role: grant.ActorRole}
			if !Can(actor, grant.Capability, Resource{Tenant: "haus-1"}) {
				t.Errorf("matrix widens %s/%s: %q lacks %q", cell.FamilyKey, cell.AreaKey, grant.ActorRole, grant.Capability)
			}
		}
	}
}
