// Package authz is the policy layer: who may do what.
//
// It is deliberately PURE — every authorization decision takes an actor and
// resource and returns a decision. No *app, no HTTP, no stores. That is what
// makes the rules testable in isolation and impossible to bypass by accident.
package authz

import (
	"strings"

	"github.com/inspr-at/hausv-org/internal/store"
)

type Capability string

// Actor is the authenticated person and their membership in one tenant.
// Person is deliberately carried even though today's role policy does not use
// it yet: callers must present the complete authorization subject.
type Actor struct {
	Person       string
	Tenant       string
	Role         string
	Organisation string
	Policy       *Policy
}

// Resource identifies the tenant that owns an object being authorized.
type Resource struct {
	Tenant string
}

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

func Can(actor Actor, action Capability, resource Resource) bool {
	if !sameTenant(actor, resource) {
		return false
	}
	return actor.Policy.Can(actor, action)
}

func CanManageContacts(actor Actor, resource Resource) bool {
	return Can(actor, CapabilityManageUsers, resource) || Can(actor, CapabilityManageBuilding, resource) || Can(actor, CapabilityManageIssues, resource)
}

func CanManageAnnouncements(actor Actor, resource Resource) bool {
	return Can(actor, CapabilityManageAnnouncements, resource)
}

func CanManageEvents(actor Actor, resource Resource) bool {
	return Can(actor, CapabilityManageAnnouncements, resource)
}

// RoleHasCapability is the explicitly resource-free policy check. Use Can
// whenever an object is involved so role and tenant ownership are decided
// together.
func RoleHasCapability(role string, action Capability) bool {
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

func RoleCanUseResidentAreas(role string) bool {
	return !IsServiceProviderRole(role)
}

func CanCreateResidentIssue(actor Actor, resource Resource) bool {
	if !sameTenant(actor, resource) {
		return false
	}
	return RoleCanUseResidentAreas(actor.Role) && (!Can(actor, CapabilityOversight, resource) || Can(actor, CapabilityManageIssues, resource))
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

func CanViewAudit(actor Actor, resource Resource) bool {
	if !sameTenant(actor, resource) {
		return false
	}
	switch store.NormalizeRole(actor.Role) {
	case store.RoleAdmin, store.RoleManager, store.RoleOwner, store.RoleRenter, store.RoleBeirat, store.RoleResident, store.RoleServiceProvider:
		return true
	default:
		return false
	}
}

func CanViewFullAudit(actor Actor, resource Resource) bool {
	if !sameTenant(actor, resource) {
		return false
	}
	role := store.NormalizeRole(actor.Role)
	return role == store.RoleAdmin || role == store.RoleManager
}

func CanAssignUserRole(actor Actor, targetRole string, resource Resource) bool {
	targetRole = store.NormalizeRole(targetRole)
	if targetRole == store.RoleAdmin {
		return Can(actor, CapabilityPlatformAdmin, resource)
	}
	return Can(actor, CapabilityManageUsers, resource)
}

func CanManageHandovers(actor Actor, resource Resource) bool {
	return Can(actor, CapabilityManageDocuments, resource) || Can(actor, CapabilityManageBuilding, resource) || Can(actor, CapabilityManageUsers, resource)
}

func sameTenant(actor Actor, resource Resource) bool {
	actorTenant := strings.TrimSpace(actor.Tenant)
	resourceTenant := strings.TrimSpace(resource.Tenant)
	return strings.TrimSpace(actor.Person) != "" && actorTenant != "" && resourceTenant != "" && actorTenant == resourceTenant
}
