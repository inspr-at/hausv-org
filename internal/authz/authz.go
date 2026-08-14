// Package authz is the policy layer: who may do what.
//
// It is deliberately PURE — every function takes a role (and sometimes a
// record) and returns a decision. No *app, no HTTP, no stores. That is what
// makes the rules testable in isolation and impossible to bypass by accident.
package authz

import (
	"github.com/inspr-at/hausv-org/internal/store"
)

type Capability string

const (
	CapabilityPlatformAdmin       Capability = "platform-admin"
	CapabilityManageUsers         Capability = "manage-users"
	CapabilityManageParking       Capability = "manage-parking"
	CapabilityManageAnnouncements Capability = "manage-announcements"
	CapabilityManageDocuments     Capability = "manage-documents"
	CapabilityManageIssues        Capability = "manage-issues"
	CapabilityManageVotes         Capability = "manage-votes"
	CapabilityManageBuilding      Capability = "manage-building"
	CapabilityOwnerDocuments      Capability = "owner-documents"
	CapabilityVote                Capability = "vote"
	CapabilityOversight           Capability = "oversight"
	CapabilityManageEnergy        Capability = "manage-energy"
	CapabilityControlEnergy       Capability = "control-energy"
)

func CanManageContacts(role string) bool {
	return HasCapability(role, CapabilityManageUsers) || HasCapability(role, CapabilityManageBuilding) || HasCapability(role, CapabilityManageIssues)
}

func CanManageAnnouncements(role string) bool {
	return HasCapability(role, CapabilityManageAnnouncements)
}

func CanManageEvents(role string) bool {
	return HasCapability(role, CapabilityManageAnnouncements)
}

func HasCapability(role string, action Capability) bool {
	role = store.NormalizeRole(role)
	if role == store.RoleAdmin {
		return true
	}
	switch action {
	case CapabilityPlatformAdmin, CapabilityManageParking:
		return false
	case CapabilityManageUsers, CapabilityManageAnnouncements, CapabilityManageDocuments, CapabilityManageIssues, CapabilityManageVotes, CapabilityManageBuilding:
		return role == store.RoleManager
	case CapabilityManageEnergy, CapabilityControlEnergy:
		return role == store.RoleManager || role == store.RoleOwner
	case CapabilityOwnerDocuments, CapabilityVote:
		return role == store.RoleOwner
	case CapabilityOversight:
		return role == store.RoleBeirat || role == store.RoleManager
	default:
		return false
	}
}

func IsServiceProviderRole(role string) bool {
	return store.NormalizeRole(role) == store.RoleServiceProvider
}

func CanUseResidentAreas(role string) bool {
	return !IsServiceProviderRole(role)
}

func CanCreateResidentIssue(role string) bool {
	return CanUseResidentAreas(role) && (!HasCapability(role, CapabilityOversight) || HasCapability(role, CapabilityManageIssues))
}

func CanResidentTransition(from string, to string) bool {
	from = store.NormalizeIssueStatus(from)
	to = store.NormalizeIssueStatus(to)
	switch to {
	case store.IssueStatusDone:
		return from == store.IssueStatusNew || from == store.IssueStatusProgress
	case store.IssueStatusNew:
		return from == store.IssueStatusDone || from == store.IssueStatusRejected || from == store.IssueStatusDuplicate
	default:
		return false
	}
}

// serviceProviderStatusRank orders the states a provider may move an assigned
// issue through: Neu → Angenommen → Termin vereinbart → In Bearbeitung →
// Erledigt. Off-chain states (Abgelehnt, Duplikat) return ok=false.
func serviceProviderStatusRank(status string) (int, bool) {
	switch store.NormalizeIssueStatus(status) {
	case store.IssueStatusNew:
		return 0, true
	case store.IssueStatusAccepted:
		return 1, true
	case store.IssueStatusScheduled:
		return 2, true
	case store.IssueStatusProgress:
		return 3, true
	case store.IssueStatusDone:
		return 4, true
	default:
		return 0, false
	}
}

// CanServiceProviderTransition allows a provider to advance an issue forward
// along the chain only. Reopening to Neu and the off-chain states (Abgelehnt,
// Duplikat) stay manager-only.
func CanServiceProviderTransition(from string, to string) bool {
	toRank, toOK := serviceProviderStatusRank(to)
	if !toOK || store.NormalizeIssueStatus(to) == store.IssueStatusNew {
		return false
	}
	fromRank, fromOK := serviceProviderStatusRank(from)
	if !fromOK {
		return false
	}
	return toRank >= fromRank
}

func CanViewAudit(role string) bool {
	switch store.NormalizeRole(role) {
	case store.RoleAdmin, store.RoleManager, store.RoleOwner, store.RoleRenter, store.RoleBeirat, store.RoleResident, store.RoleServiceProvider:
		return true
	default:
		return false
	}
}

func CanViewFullAudit(role string) bool {
	role = store.NormalizeRole(role)
	return role == store.RoleAdmin || role == store.RoleManager
}

func CanAssignUserRole(actorRole string, targetRole string) bool {
	targetRole = store.NormalizeRole(targetRole)
	if targetRole == store.RoleAdmin {
		return HasCapability(actorRole, CapabilityPlatformAdmin)
	}
	return HasCapability(actorRole, CapabilityManageUsers)
}

func CanManageHandovers(role string) bool {
	return HasCapability(role, CapabilityManageDocuments) || HasCapability(role, CapabilityManageBuilding) || HasCapability(role, CapabilityManageUsers)
}
