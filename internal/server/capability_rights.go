package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/authz"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

func rightsOrganisation(tenant tenantConfig) string {
	return normalizeSlug(tenant.Organisation)
}

func (a *app) capabilityPolicy(ctx context.Context, tenant tenantConfig, preview bool) *authz.Policy {
	if a == nil || a.capabilityRepo == nil || rightsOrganisation(tenant) == "" {
		return nil
	}
	rules, err := a.capabilityRepo(rightsOrganisation(tenant)).Get(ctx)
	if err != nil {
		logError("capability rules load failed", err)
		return &authz.Policy{Unavailable: true}
	}
	if preview {
		rules.Grants = nil
	}
	return &authz.Policy{Rules: rules}
}

func (a *app) actorFor(person, tenantSlug, role string) authorizationActor {
	actor := actorFor(person, tenantSlug, role)
	if a == nil {
		return actor
	}
	if tenant, ok := a.tenantBySlug(tenantSlug); ok {
		actor.Organisation = rightsOrganisation(tenant)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		actor.Policy = a.capabilityPolicy(ctx, tenant, false)
	}
	return actor
}

func (a *app) canEditRights(ac authCtx) bool {
	if ac.preview != nil || a.capabilityRepo == nil {
		return false
	}
	return ac.can(capabilityPlatformAdmin) && rightsOrganisation(ac.tenant) != "" && a.isOrganisationAdmin(&ac)
}

func (a *app) rightsData(ac authCtx) web.RechteData {
	data := web.RechteData{Editable: a.canEditRights(ac), Policy: ac.policy, OrgKey: rightsOrganisation(ac.tenant), UserCounts: map[string]int{}}
	seen := map[string]bool{}
	for slug, tenant := range a.tenants {
		if rightsOrganisation(tenant) != data.OrgKey {
			continue
		}
		// Count people once per family, even when they belong to several houses.
		for _, profile := range a.directoryProfiles() {
			if !profile.HasTenant(slug) {
				continue
			}
			family := authz.RoleFamily(a.roleFor(profile.Email, slug))
			key := family + ":" + normalizeEmail(profile.Email)
			if !seen[key] {
				data.UserCounts[family]++
				seen[key] = true
			}
		}
	}
	return data
}

func (a *app) updateRechte(w http.ResponseWriter, r *http.Request, ac authCtx) {
	ac = a.organisationHouseContext(r.Context(), &ac)
	if !a.canEditRights(ac) {
		http.Error(w, "Nur die Organisationsadministration kann Rechte ändern.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültiges Formular.", http.StatusBadRequest)
		return
	}
	a.rightsMu.Lock()
	defer a.rightsMu.Unlock()
	org := rightsOrganisation(ac.tenant)
	repo := a.capabilityRepo(org)
	rules, err := repo.Get(r.Context())
	if err != nil {
		http.Error(w, "Rechte konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	changed := false
	switch r.Form.Get("operation") {
	case "reset":
		if r.Form.Get("confirm") != "yes" {
			http.Error(w, "Bitte das Zurücksetzen bestätigen.", http.StatusBadRequest)
			return
		}
		if err := repo.ResetOverrides(r.Context()); err != nil {
			http.Error(w, "Rechte konnten nicht zurückgesetzt werden.", http.StatusInternalServerError)
			return
		}
		for _, override := range rules.Overrides {
			role := ""
			for _, family := range authz.RoleFamilies {
				if family.Key == override.RoleFamily {
					role = family.RepresentativeRole
				}
			}
			a.auditRights(ac, store.AuditActionCapabilityReset, override.RoleFamily, "", override.Capability, "", strconv.FormatBool(override.Allowed), strconv.FormatBool(authz.RoleHasCapability(role, authz.Capability(override.Capability))))
		}
	case "set":
		family, area, capability := r.Form.Get("family"), r.Form.Get("area"), authz.Capability(r.Form.Get("capability"))
		role := ""
		for _, candidate := range authz.RoleFamilies {
			if candidate.Key == family {
				role = candidate.RepresentativeRole
			}
		}
		validArea := area == "additional"
		for _, option := range authz.MatrixOptions(family, area) {
			if option.Capability == capability {
				validArea = true
			}
		}
		if role == "" || !validArea || !store.DelegableCapability(string(capability)) || authz.LockedFor(family, capability) || (r.Form.Get("allowed") != "" && r.Form.Get("allowed") != "1") {
			http.Error(w, "Dieses Recht kann nicht geändert werden.", http.StatusBadRequest)
			return
		}
		allowed := r.Form.Get("allowed") == "1"
		for _, candidate := range authz.RoleFamilies {
			if candidate.Key == family {
				for _, memberRole := range candidate.Roles {
					if authz.RoleHasCapability(memberRole, capability) != allowed {
						changed = true
					}
				}
			}
		}
		policy := authz.Policy{Rules: rules}
		before := policy.RoleAllowed(org, role, capability)
		if err := repo.SetOverride(r.Context(), store.CapabilityOverride{RoleFamily: family, Capability: string(capability), Allowed: allowed, UpdatedBy: ac.email}); err != nil {
			http.Error(w, "Rechte konnten nicht gespeichert werden.", http.StatusInternalServerError)
			return
		}
		a.auditRights(ac, store.AuditActionCapabilityOverride, family, area, string(capability), "", strconv.FormatBool(before), strconv.FormatBool(allowed))
	default:
		http.Error(w, "Unbekannte Änderung.", http.StatusBadRequest)
		return
	}
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"saved":true,"changed":%t}`, changed)
		return
	}
	http.Redirect(w, r, "/app/verwaltung/rechte?rights=saved", http.StatusSeeOther)
}

func (a *app) auditRights(ac authCtx, action, family, area, capability, email, before, after string) {
	if area == "" {
		for _, cell := range authz.RoleMatrix {
			for _, grant := range cell.Grants {
				if string(grant.Capability) == capability {
					if area != "" {
						area += ", "
					}
					area += cell.AreaKey
					break
				}
			}
		}
	}
	summary := "Berechtigung geändert: " + authz.CapabilityLabel(authz.Capability(capability))
	if action == store.AuditActionCapabilityReset {
		summary = "Rollenrecht auf Standard zurückgesetzt: " + authz.CapabilityLabel(authz.Capability(capability))
	}
	if action == store.AuditActionCapabilityProfile {
		summary = "Berechtigungsprofil gespeichert: " + capability
	}
	a.recordAudit(auditEvent{TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role, Action: action, TargetType: "capability", TargetID: capability, Summary: summary, Details: map[string]string{"organisation": rightsOrganisation(ac.tenant), "family": family, "area": area, "capability": capability, "email": email, "before": before, "after": after}})
}

func (a *app) updateUserRights(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canEditRights(ac) {
		http.Error(w, "Nur die Organisationsadministration kann eigene Rechte ändern.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültiges Formular.", http.StatusBadRequest)
		return
	}
	email := normalizeEmail(r.Form.Get("email"))
	profile, found := a.directoryProfile(email)
	if !found || !profile.HasTenant(ac.tenant.Slug) {
		http.Error(w, "Person gehört nicht zu dieser Liegenschaft.", http.StatusNotFound)
		return
	}
	scope := r.Form.Get("tenant_slug")
	if scope != "" {
		tenant, ok := a.tenantBySlug(scope)
		if !ok || rightsOrganisation(tenant) != rightsOrganisation(ac.tenant) || !profile.HasTenant(scope) {
			http.Error(w, "Ungültige Liegenschaft.", http.StatusBadRequest)
			return
		}
	}
	a.rightsMu.Lock()
	defer a.rightsMu.Unlock()
	repo := a.capabilityRepo(rightsOrganisation(ac.tenant))
	rules, err := repo.Get(r.Context())
	if err != nil {
		http.Error(w, "Rechte konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	before := map[string]string{}
	for _, grant := range rules.Grants {
		if grant.Email == email && grant.TenantSlug == scope {
			before[grant.Capability] = grant.Effect
		}
	}
	var after []store.UserCapabilityGrant
	operation := r.Form.Get("operation")
	switch operation {
	case "grant":
		capability, effect := r.Form.Get("capability"), r.Form.Get("effect")
		if !store.DelegableCapability(capability) || (effect != "" && effect != "grant" && effect != "deny") {
			http.Error(w, "Dieses Recht kann nicht geändert werden.", http.StatusBadRequest)
			return
		}
		if err := repo.SetGrant(r.Context(), store.UserCapabilityGrant{Email: email, Capability: capability, Effect: effect, TenantSlug: scope, UpdatedBy: ac.email}); err != nil {
			http.Error(w, "Recht konnte nicht gespeichert werden.", http.StatusInternalServerError)
			return
		}
		a.auditUserRight(ac, email, scope, capability, before[capability], effect)
	case "apply-profile":
		name := r.Form.Get("profile")
		found := name == "Standard"
		for _, profile := range rules.Profiles {
			if profile.Name == name {
				after = profile.Capabilities
				found = true
			}
		}
		if !found {
			http.Error(w, "Profil nicht gefunden.", http.StatusBadRequest)
			return
		}
		if err := repo.ReplaceUserGrants(r.Context(), email, scope, after, ac.email); err != nil {
			http.Error(w, "Profil konnte nicht angewendet werden.", http.StatusInternalServerError)
			return
		}
		next := map[string]string{}
		for _, grant := range after {
			next[grant.Capability] = grant.Effect
		}
		for _, capability := range authz.AllCapabilities {
			key := string(capability)
			if before[key] != next[key] {
				a.auditUserRight(ac, email, scope, key, before[key], next[key])
			}
		}
	case "save-profile":
		name := strings.TrimSpace(r.Form.Get("profile_name"))
		for _, capability := range authz.AllCapabilities {
			if effect := before[string(capability)]; effect != "" {
				after = append(after, store.UserCapabilityGrant{Capability: string(capability), Effect: effect})
			}
		}
		// Names identify templates; prevent an unnoticed overwrite of a shared one.
		for _, profile := range rules.Profiles {
			if strings.EqualFold(profile.Name, name) {
				http.Error(w, "Dieser Profilname ist bereits vergeben.", http.StatusConflict)
				return
			}
		}
		if err := repo.SaveProfile(r.Context(), store.CapabilityProfile{Name: name, Capabilities: after}); err != nil {
			http.Error(w, "Profil konnte nicht gespeichert werden. Bitte Namen prüfen.", http.StatusBadRequest)
			return
		}
		a.auditRights(ac, store.AuditActionCapabilityProfile, "", "", name, email, "", fmt.Sprint(len(after))+" Rechte")
	default:
		http.Error(w, "Unbekannte Änderung.", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/app/settings/users?rights=saved#access-"+email, http.StatusSeeOther)
}

func (a *app) auditUserRight(ac authCtx, email, scope, capability, before, after string) {
	if before == "" {
		before = "role"
	}
	if after == "" {
		after = "role"
	}
	if scope == "" {
		scope = "Gesamte Organisation"
	}
	a.recordAudit(auditEvent{TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role, Action: store.AuditActionUserCapability, TargetType: "user", TargetID: email, Summary: "Eigenes Benutzerrecht geändert: " + authz.CapabilityLabel(authz.Capability(capability)), Details: map[string]string{"organisation": rightsOrganisation(ac.tenant), "email": email, "capability": capability, "tenant_slug": scope, "before": before, "after": after}})
}

func (a *app) directoryProfiles() []userProfile {
	emails := map[string]bool{}
	for email := range a.profiles {
		emails[email] = true
	}
	for email := range a.admins {
		emails[email] = true
	}
	if a.inviteStore != nil {
		for _, profile := range a.inviteStore.List() {
			emails[profile.Email] = true
		}
	}
	out := make([]userProfile, 0, len(emails))
	for email := range emails {
		if profile, ok := a.directoryProfile(email); ok {
			out = append(out, profile)
		}
	}
	return out
}

func effectiveRights(actor authorizationActor) []web.EffectiveRight {
	var out []web.EffectiveRight
	for _, capability := range authz.AllCapabilities {
		out = append(out, web.EffectiveRight{Label: authz.CapabilityLabel(capability), Allowed: authz.Can(actor, capability, resourceFor(actor.Tenant))})
	}
	return out
}

func (a *app) userRightsData(ac authCtx, users []userRow) web.UserRightsData {
	data := web.UserRightsData{Enabled: ac.policy != nil, Editable: a.canEditRights(ac), CurrentTenant: ac.tenant.Slug, OrgKey: rightsOrganisation(ac.tenant), Policy: ac.policy, Scopes: map[string][]web.RightsScope{}, Effective: map[string][]web.EffectiveRight{}}
	if ac.policy != nil {
		data.Profiles = ac.policy.Rules.Profiles
	}
	for _, user := range users {
		actor := ac.actor()
		actor.Person = user.Email
		actor.Role = a.roleFor(user.Email, ac.tenant.Slug)
		data.Effective[user.Email] = effectiveRights(actor)
		profile, ok := a.directoryProfile(user.Email)
		if !ok {
			continue
		}
		for _, slug := range profile.Tenants {
			if tenant, ok := a.tenantBySlug(slug); ok && rightsOrganisation(tenant) == data.OrgKey {
				data.Scopes[user.Email] = append(data.Scopes[user.Email], web.RightsScope{Slug: slug, Label: tenant.Name})
			}
		}
	}
	return data
}

func (ac authCtx) parkingManagementAllowed() bool {
	if allowed, configured := ac.policy.Configured(ac.actor(), capabilityManageParking); configured {
		return allowed
	}
	return ac.can(capabilityManageUsers) || ac.can(capabilityManageParking)
}
