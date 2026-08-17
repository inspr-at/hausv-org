package server

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

// onboardingPortalContext keeps the setup wizard inside the same
// permission-gated portal shell as every other converted route. The active
// section stays "energy" because the wizard is the entry point of Mein Zuhause.
func (a *app) onboardingPortalContext(ac authCtx) web.PortalPageData {
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	modules := a.portalModulesFor(ac.tenant.Slug)
	contexts := a.portalContextsFor(ac.email, ac.tenant.Slug, ac.role)
	portalContexts := make([]web.PortalContext, 0, len(contexts))
	for _, context := range contexts {
		portalContexts = append(portalContexts, web.PortalContext{
			TenantSlug: context.TenantSlug,
			HouseName:  context.HouseName,
			Address:    context.Address,
			Role:       context.Role,
			Current:    context.Current,
		})
	}
	unreadAnnouncements := 0
	if ac.repositories.announcements != nil && ac.repositories.announcementReads != nil && strings.TrimSpace(ac.email) != "" {
		now := time.Now()
		unreadAnnouncements = unreadAnnouncementCount(ac.repositories.announcements.Visible(now), ac.repositories.announcementReads.LastSeen(ac.email), now)
	}
	openIssues := 0
	if a.issueStore != nil {
		openIssues = issueOpenCount(a.visibleIssuesForActor(ac.tenantRef, ac.email, ac.role))
	}
	return web.PortalPageData{
		Title:               "Mein Zuhause einrichten · " + houseDisplayName(ac.tenant) + " · " + ac.role,
		TenantSlug:          ac.tenant.Slug,
		HouseName:           houseDisplayName(ac.tenant),
		Address:             ac.tenant.Address,
		MapURL:              tenantMapURL(ac.tenant.Address),
		DisplayName:         profile.DisplayName(),
		Initials:            profile.Initials(),
		Role:                ac.role,
		DisplayVersion:      version.DisplayVersion(version.Version),
		ActivePage:          "energy",
		Modules:             web.PortalModules{Energy: modules.Energy, Announcements: modules.Announcements, Events: modules.Events, Contacts: modules.Contacts, Documents: modules.Documents, Issues: modules.Issues, Votes: modules.Votes, Parking: modules.Parking, Handovers: modules.Handovers, Users: modules.Users, Audit: modules.Audit, Help: modules.Help},
		CanUseResidentAreas: roleCanUseResidentAreas(ac.role),
		CanViewEnergy:       modules.Energy && a.canViewEnergy(ac),
		CanManageIssues:     ac.can(capabilityManageIssues),
		CanSeeParking:       modules.Parking && (ac.can(capabilityPlatformAdmin) || profile.HasPermission(permissionParking)),
		CanManageHandovers:  modules.Handovers && canManageHandovers(ac.actor(), ac.resource()),
		CanManageUsers:      modules.Users && ac.can(capabilityManageUsers),
		CanViewAudit:        modules.Audit && canViewAudit(ac.actor(), ac.resource()),
		Issues:              make([]view.IssueView, openIssues),
		UnreadAnnouncements: unreadAnnouncements,
		Contexts:            portalContexts,
		ReleaseNotes:        version.Notes(),
	}
}

func onboardingUnitOptions(options []energyHomeUnitOption) []web.OnboardingUnitOption {
	out := make([]web.OnboardingUnitOption, 0, len(options))
	for _, option := range options {
		out = append(out, web.OnboardingUnitOption{Value: option.Value, Label: option.Label, Selected: option.Selected})
	}
	return out
}

func onboardingAssetOptions(options []energyAssetOption) []web.OnboardingAssetOption {
	out := make([]web.OnboardingAssetOption, 0, len(options))
	for _, option := range options {
		out = append(out, web.OnboardingAssetOption{Kind: option.Kind, Label: option.Label, Checked: option.Checked})
	}
	return out
}

func onboardingOptions(options []energyOption) []web.OnboardingOption {
	out := make([]web.OnboardingOption, 0, len(options))
	for _, option := range options {
		out = append(out, web.OnboardingOption{Value: option.Value, Label: option.Label})
	}
	return out
}

func onboardingMappingSlots(slots []energyMappingSlotView) []web.OnboardingMappingSlot {
	out := make([]web.OnboardingMappingSlot, 0, len(slots))
	for _, slot := range slots {
		out = append(out, web.OnboardingMappingSlot{
			Key:     slot.Key,
			Label:   slot.Label,
			Purpose: slot.Purpose,
			Status:  slot.Status,
			Detail:  slot.Detail,
			Tone:    slot.Tone,
		})
	}
	return out
}

func onboardingCandidates(candidates []energyCandidateView) []web.OnboardingCandidate {
	out := make([]web.OnboardingCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, web.OnboardingCandidate{
			EntityID:    candidate.EntityID,
			DisplayName: candidate.DisplayName,
			SourceName:  candidate.SourceName,
			MetricLabel: candidate.MetricLabel,
			Checked:     candidate.Checked,
		})
	}
	return out
}

func onboardingRecommendation(recommendation energy.Recommendation) web.OnboardingRecommendation {
	return web.OnboardingRecommendation{Title: recommendation.Title, Reason: recommendation.Reason}
}

func (a *app) renderOnboardingTempl(w http.ResponseWriter, r *http.Request, data web.OnboardingPageData) {
	var rendered bytes.Buffer
	if err := web.OnboardingPage(data).Render(r.Context(), &rendered); err != nil {
		logError("templ onboarding render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), data.Portal.TenantSlug))
}
