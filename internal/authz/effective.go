package authz

import (
	"strings"

	"github.com/inspr-at/hausv-org/internal/store"
)

// Policy is an immutable request snapshot, never a process-global callback.
// A failed load denies access instead of resurrecting revoked default rights.
type Policy struct {
	Rules       store.CapabilityRules
	Unavailable bool
}

func RoleFamily(role string) string {
	for _, family := range RoleFamilies {
		for _, candidate := range family.Roles {
			if store.NormalizeRole(role) == candidate {
				return family.Key
			}
		}
	}
	return ""
}

func NonDelegable(capability Capability) bool {
	return capability == CapabilityManageUsers || capability == CapabilityPlatformAdmin
}

// LockedFor reports capabilities a family can never lose through configuration:
// the administration capabilities everywhere, and Oversight for the Verwaltung
// family, because it gates the Verwaltung area including the rights page itself
// (switching it off locked the administrator out of the page that could undo it).
func LockedFor(family string, capability Capability) bool {
	return NonDelegable(capability) || (family == "verwaltung" && capability == CapabilityOversight)
}

func (p *Policy) RoleAllowed(orgKey, role string, capability Capability) bool {
	if p != nil && (p.Unavailable || p.Rules.OrgKey != orgKey) {
		return false
	}
	allowed := RoleHasCapability(role, capability)
	if p == nil || LockedFor(RoleFamily(role), capability) {
		return allowed
	}
	for _, override := range p.Rules.Overrides {
		if override.OrgKey == orgKey && override.RoleFamily == RoleFamily(role) && override.Capability == string(capability) && store.DelegableCapability(override.Capability) {
			allowed = override.Allowed
		}
	}
	return allowed
}

func (p *Policy) Can(actor Actor, capability Capability) bool {
	allowed := p.RoleAllowed(actor.Organisation, actor.Role, capability)
	if p == nil || LockedFor(RoleFamily(actor.Role), capability) {
		return allowed
	}
	if p.Unavailable || p.Rules.OrgKey != actor.Organisation {
		return false
	}
	for _, grant := range p.Rules.Grants {
		if grant.OrgKey != actor.Organisation || !strings.EqualFold(strings.TrimSpace(grant.Email), strings.TrimSpace(actor.Person)) || grant.Capability != string(capability) || (grant.TenantSlug != "" && grant.TenantSlug != actor.Tenant) || !store.DelegableCapability(grant.Capability) {
			continue
		}
		if grant.Effect == "deny" {
			return false
		}
		if grant.Effect == "grant" {
			allowed = true
		}
	}
	return allowed
}

// MatrixOptions includes the capability-backed operations available in an area
// for any family, retaining explicit product rules of the selected family.
// Operations sharing a capability intentionally move together.
func MatrixOptions(familyKey, areaKey string) []MatrixGrant {
	var out []MatrixGrant
	seen := map[MatrixAction]bool{}
	for _, cell := range RoleMatrix {
		if cell.FamilyKey == familyKey && cell.AreaKey == areaKey {
			for _, grant := range cell.Grants {
				out = append(out, grant)
				seen[grant.Action] = true
			}
		}
	}
	for _, cell := range RoleMatrix {
		if cell.AreaKey == areaKey {
			for _, grant := range cell.Grants {
				if grant.Capability != "" && !seen[grant.Action] {
					out = append(out, grant)
					seen[grant.Action] = true
				}
			}
		}
	}
	return out
}

// EffectiveMatrix resolves the organisation's presentation from the same role
// decisions as Can; resource/assignment restrictions still apply afterwards.
func (p *Policy) EffectiveMatrix(orgKey string) []MatrixCell {
	result := make([]MatrixCell, 0, len(RoleMatrix))
	for _, cell := range RoleMatrix {
		effective := cell
		effective.Grants = nil
		role := ""
		for _, family := range RoleFamilies {
			if family.Key == cell.FamilyKey {
				role = family.RepresentativeRole
			}
		}
		for _, grant := range MatrixOptions(cell.FamilyKey, cell.AreaKey) {
			grantRole := role
			if NonDelegable(grant.Capability) && cell.FamilyKey == "verwaltung" {
				grantRole = grant.ActorRole
			}
			if grant.Capability == "" || p.RoleAllowed(orgKey, grantRole, grant.Capability) {
				effective.Grants = append(effective.Grants, grant)
			}
		}
		result = append(result, effective)
	}
	return result
}

func CapabilityLabel(capability Capability) string {
	switch capability {
	case CapabilityManageUsers:
		return "Benutzer verwalten"
	case CapabilityPlatformAdmin:
		return "Organisationsadministration"
	case CapabilityManageParking:
		return "Stellplätze verwalten"
	case CapabilityManageAnnouncements:
		return "Ankündigungen und Termine verwalten"
	case CapabilityManageDocuments:
		return "Dokumente und Übergaben verwalten"
	case CapabilityManageIssues:
		return "Anliegen und Posteingang verwalten"
	case CapabilityManageVotes:
		return "Abstimmungen verwalten"
	case CapabilityManageBuilding:
		return "Liegenschaft und Einheiten verwalten"
	case CapabilityOwnerDocuments:
		return "Eigentümerdokumente sehen"
	case CapabilityVote:
		return "Stimme abgeben"
	case CapabilityOversight:
		return "Aufsicht über Anliegen"
	case CapabilityManageEnergy:
		return "Energie verwalten"
	case CapabilityControlEnergy:
		return "Energie steuern"
	default:
		return string(capability)
	}
}

// Configured distinguishes an explicit decision from the historical assignment
// policies in areas such as owner documents and energy. Those policies remain
// the fallback only when the organisation has not configured this capability.
func (p *Policy) Configured(actor Actor, capability Capability) (bool, bool) {
	if p == nil || LockedFor(RoleFamily(actor.Role), capability) {
		return false, false
	}
	if p.Unavailable || p.Rules.OrgKey != actor.Organisation {
		return false, true
	}
	for _, override := range p.Rules.Overrides {
		if override.OrgKey == actor.Organisation && override.RoleFamily == RoleFamily(actor.Role) && override.Capability == string(capability) {
			return p.Can(actor, capability), true
		}
	}
	for _, grant := range p.Rules.Grants {
		if grant.OrgKey == actor.Organisation && strings.EqualFold(grant.Email, actor.Person) && grant.Capability == string(capability) && (grant.TenantSlug == "" || grant.TenantSlug == actor.Tenant) {
			return p.Can(actor, capability), true
		}
	}
	return false, false
}
