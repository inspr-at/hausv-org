package server

import (
	"net/http"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

// renderHomeIdentitySettingsTempl is the templ counterpart of the legacy
// "homeIdentitySettings" template. It stays behind a.portalTemplEnabled so the
// Go-string template keeps rendering the route while the switch is off.
func (a *app) renderHomeIdentitySettingsTempl(w http.ResponseWriter, r *http.Request, ac authCtx, profile energy.HomeProfile, data map[string]any) {
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.HomeSettingsPage(web.HomeSettingsPageData{
		Portal:              a.settingsPortalContext(ac, data["Title"].(string), "settings"),
		HomeIdentity:        a.homeIdentityFromProfile(profile),
		HouseholdName:       profile.HouseholdName,
		HomeType:            profile.HomeType,
		HomeTypeLabel:       data["HomeTypeLabel"].(string),
		HomeTypeDescription: data["HomeTypeDescription"].(string),
		HomeTypeLocked:      data["HomeTypeLocked"].(bool),
		HomeUnitLabel:       data["HomeUnitLabel"].(string),
		HomeUnitID:          data["HomeUnitID"].(string),
		HasHomeUnit:         data["HasHomeUnit"].(bool),
		OfficialUnitTitle:   data["OfficialUnitTitle"].(string),
		OfficialUnitSummary: data["OfficialUnitSummary"].(string),
		UnitOptions:         homeIdentitySelectOptions(data["UnitOptions"].([]energyHomeUnitOption)),
		HasUnitOptions:      data["HasUnitOptions"].(bool),
		From:                data["From"].(string),
		BackURL:             data["BackURL"].(string),
		BackLabel:           data["BackLabel"].(string),
		Saved:               data["Saved"].(bool),
		Invalid:             data["Invalid"].(bool),
	}))
}

func homeIdentitySelectOptions(options []energyHomeUnitOption) []view.SelectOption {
	converted := make([]view.SelectOption, 0, len(options))
	for _, option := range options {
		converted = append(converted, view.SelectOption{Value: option.Value, Label: option.Label, Selected: option.Selected})
	}
	return converted
}
