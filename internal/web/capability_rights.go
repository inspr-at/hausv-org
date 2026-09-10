package web

import (
	"github.com/inspr-at/hausv-org/internal/authz"
	"github.com/inspr-at/hausv-org/internal/store"
)

type RechteData struct {
	Editable   bool
	Policy     *authz.Policy
	OrgKey     string
	Saved      bool
	UserCounts map[string]int
}

func (data RechteData) cell(family, area string) authz.MatrixCell {
	if data.Policy == nil {
		return rechteMatrixCell(family, area)
	}
	for _, cell := range data.Policy.EffectiveMatrix(data.OrgKey) {
		if cell.FamilyKey == family && cell.AreaKey == area {
			return cell
		}
	}
	return authz.MatrixCell{FamilyKey: family, AreaKey: area}
}

func (data RechteData) allowed(family string, capability authz.Capability) bool {
	role := ""
	for _, candidate := range authz.RoleFamilies {
		if candidate.Key == family {
			role = candidate.RepresentativeRole
		}
	}
	if authz.LockedFor(family, capability) {
		return family == "verwaltung"
	}
	return data.Policy.RoleAllowed(data.OrgKey, role, capability)
}

// lockNote explains why a switch cannot be changed.
func lockNote(family string, capability authz.Capability) string {
	if authz.NonDelegable(capability) {
		return "Nur per Administrationsrolle, nicht delegierbar"
	}
	return "Grundlage der Verwaltungssicht, nicht delegierbar"
}

func (data RechteData) changed(family string, capability authz.Capability) bool {
	if data.Policy == nil {
		return false
	}
	for _, override := range data.Policy.Rules.Overrides {
		if override.RoleFamily == family && override.Capability == string(capability) {
			for _, roleFamily := range authz.RoleFamilies {
				if roleFamily.Key == family {
					for _, role := range roleFamily.Roles {
						if authz.RoleHasCapability(role, capability) != override.Allowed {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// Extra capabilities have no dedicated action in the historical matrix.
func additionalCapabilities() []authz.Capability {
	return []authz.Capability{authz.CapabilityManageUsers, authz.CapabilityManageParking, authz.CapabilityOwnerDocuments, authz.CapabilityVote, authz.CapabilityOversight}
}

type UserRightsData struct {
	Enabled, Editable bool
	CurrentTenant     string
	OrgKey            string
	Policy            *authz.Policy
	Profiles          []store.CapabilityProfile
	Scopes            map[string][]RightsScope
	Effective         map[string][]EffectiveRight
}

type RightsScope struct{ Slug, Label string }
type EffectiveRight struct {
	Label   string
	Allowed bool
}

func (data UserRightsData) grants(email string) []store.UserCapabilityGrant {
	var out []store.UserCapabilityGrant
	if data.Policy != nil {
		for _, grant := range data.Policy.Rules.Grants {
			if grant.Email == email {
				out = append(out, grant)
			}
		}
	}
	return out
}

func rightsEffectLabel(effect string) string {
	if effect == "deny" {
		return "Entzogen"
	}
	if effect == "grant" {
		return "Gewährt"
	}
	return "Aus Rolle"
}
func rightsScopeLabel(scope string) string {
	if scope == "" {
		return "Gesamte Organisation"
	}
	return scope
}

func rechteActionOptions(family, area string) []authz.MatrixGrant {
	options := authz.MatrixOptions(family, area)
	var out []authz.MatrixGrant
	for _, action := range []authz.MatrixAction{authz.MatrixActionView, authz.MatrixActionCreate, authz.MatrixActionChange, authz.MatrixActionApprove, authz.MatrixActionDelete} {
		option := authz.MatrixGrant{Action: action, Note: "unavailable"}
		for _, candidate := range options {
			if candidate.Action == action {
				option = candidate
				break
			}
		}
		out = append(out, option)
	}
	return out
}

func (data RechteData) cellChanged(family, area string) bool {
	for _, grant := range authz.MatrixOptions(family, area) {
		if data.changed(family, grant.Capability) {
			return true
		}
	}
	return false
}
