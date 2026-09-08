package server

import (
	"bytes"
	"io"
	"net/http"

	"github.com/inspr-at/hausv-org/internal/energy"

	"github.com/inspr-at/hausv-org/internal/web"
)

// onboardingPortalContext keeps the setup wizard inside the same
// permission-gated portal shell as every other converted route. The active
// section stays "energy" because the wizard is the entry point of Mein Zuhause.
func (a *app) onboardingPortalContext(ac authCtx) web.PortalPageData {
	return a.portalBaseData(ac, "energy", "Mein Zuhause einrichten")
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
