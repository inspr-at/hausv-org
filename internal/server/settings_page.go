package server

import (
	"bytes"
	"html/template"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/view"
	"github.com/inspr-at/hausv-org/internal/web"
)

func (a *app) settingsPortalContext(ac authCtx, title, activePage string) web.PortalPageData {
	return a.portalBaseData(ac, activePage, title)
}

func (a *app) renderSettingsComponent(w http.ResponseWriter, r *http.Request, tenantSlug string, component templ.Component) {
	var rendered bytes.Buffer
	if err := component.Render(r.Context(), &rendered); err != nil {
		logError("templ settings render failed", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, prefixTenantHTMLPaths(rendered.String(), tenantSlug))
}

func (a *app) renderSettingsHubTempl(w http.ResponseWriter, r *http.Request, ac authCtx, data map[string]any) {
	modules := a.portalModulesFor(ac.tenant.Slug)
	canViewEnergy := modules.Energy && a.canViewEnergy(ac)
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.SettingsHubPage(web.SettingsHubPageData{
		Portal:                      a.settingsPortalContext(ac, "Einstellungen", "settings"),
		Email:                       ac.email,
		SettingsDisplayName:         data["SettingsDisplayName"].(string),
		SettingsNotificationSummary: data["SettingsNotificationSummary"].(string),
		SettingsHomeURL:             data["SettingsHomeURL"].(string),
		CalendarFeedURL:             data["CalendarFeedURL"].(string),
		HasCalendarFeedURL:          data["HasCalendarFeedURL"].(bool),
		SettingsCanManageEnergyData: data["SettingsCanManageEnergyData"].(bool),
		CanManageHomeIdentity:       modules.Energy && a.canManageHomeIdentity(ac),
		CanManageBuilding:           ac.can(capabilityManageBuilding),
		CanManageDocuments:          ac.can(capabilityManageDocuments),
		CanManageHandovers:          canManageHandovers(ac.actor(), ac.resource()),
		CanViewAudit:                canViewAudit(ac.actor(), ac.resource()),
		IsAdmin:                     ac.can(capabilityPlatformAdmin),
		HomeIdentity:                a.homeIdentityForActor(ac, canViewEnergy),
		ProfilePictureURL:           data["ProfilePictureURL"].(string),
		HasProfilePicture:           data["ProfilePictureURL"].(string) != "",
		ProfilePictureMsg:           data["ProfilePictureMsg"].(string),
		ProfilePictureOK:            data["ProfilePictureOK"].(bool),
	}))
}

func (a *app) renderProfileSettingsTempl(w http.ResponseWriter, r *http.Request, ac authCtx, data map[string]any) {
	profile := data["Profile"].(userProfile)
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.ProfileSettingsPage(web.ProfileSettingsPageData{
		EffectiveRights: effectiveRights(ac.actor()),
		RightsTenant:    ac.tenant.Name,
		Portal:          a.settingsPortalContext(ac, "Profil", "settings"),
		Email:           ac.email,
		Profile: web.SettingsProfile{
			Title:          profile.Title,
			FirstName:      profile.FirstName,
			LastName:       profile.LastName,
			Phone:          profile.Phone,
			DirectoryOptIn: profile.DirectoryOptIn,
		},
		ProfileMsg:     data["ProfileMsg"].(string),
		ProfileOK:      data["ProfileOK"].(bool),
		PermissionList: data["PermissionList"].([]string),
		AuthList:       data["AuthList"].([]string),
		Units:          data["Units"].([]profileUnitView),
	}))
}

func (a *app) renderNotificationSettingsTempl(w http.ResponseWriter, r *http.Request, ac authCtx, data map[string]any) {
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.NotificationSettingsPage(web.NotificationSettingsPageData{
		Portal:                    a.settingsPortalContext(ac, "Benachrichtigungen", "settings"),
		Email:                     ac.email,
		NotifyMsg:                 data["NotifyMsg"].(string),
		NotifyOK:                  data["NotifyOK"].(bool),
		EmailNotificationsEnabled: data["EmailNotificationsEnabled"].(bool),
		NotificationEvents:        data["NotificationEvents"].([]notificationEventOption),
		NotificationEnabledCount:  data["NotificationEnabledCount"].(int),
		NotificationEventCount:    data["NotificationEventCount"].(int),
	}))
}

func (a *app) renderBuildingSettingsTempl(w http.ResponseWriter, r *http.Request, ac authCtx, data map[string]any) {
	tenant := ac.tenant
	_, _, _, mapConfigured := tenantMapCoordinates(tenant)
	homeProfile := data["HomeProfile"].(energy.HomeProfile)
	lucideJSON := data["LucideIconNamesJSON"].(template.JS)
	a.renderSettingsComponent(w, r, tenant.Slug, web.BuildingSettingsPage(web.BuildingSettingsPageData{
		Portal:       a.settingsPortalContext(ac, "Gebäude & Einheiten", "settings"),
		AssetVersion: version.AssetVersion(),
		Tenant: web.BuildingTenant{
			Name: tenant.Name, Address: tenant.Address, BrandIcon: tenant.BrandIcon, BrandAbbreviation: tenant.BrandAbbreviation,
			ContactName: tenant.ContactName, ContactAddress: tenant.ContactAddress, ContactEmail: tenant.ContactEmail, ContactPhone: tenant.ContactPhone,
			EmergencyName: tenant.EmergencyName, EmergencyPhone: tenant.EmergencyPhone,
			CaretakerName: tenant.CaretakerName, CaretakerEmail: tenant.CaretakerEmail, CaretakerPhone: tenant.CaretakerPhone,
			HeroImageURL: tenant.HeroImageURL, MapLatitude: tenant.MapLatitude, MapLongitude: tenant.MapLongitude,
		},
		BuildingMsg:           data["BuildingMsg"].(string),
		BuildingOK:            data["BuildingOK"].(bool),
		BuildingSection:       data["BuildingSection"].(string),
		BrandIconOptions:      data["BrandIconOptions"].([]view.SelectOption),
		BrandIconLabel:        data["BrandIconLabel"].(string),
		BrandIconIsLucide:     data["BrandIconIsLucide"].(bool),
		LucideIconNamesJSON:   string(lucideJSON),
		HeroMsg:               data["HeroMsg"].(string),
		HeroOK:                data["HeroOK"].(bool),
		HasCustomHero:         data["HasCustomHero"].(bool),
		Units:                 data["Units"].([]buildingUnitView),
		NewUnitTypeOptions:    data["NewUnitTypeOptions"].([]view.SelectOption),
		NewUnitPaymentOptions: data["NewUnitPaymentOptions"].([]view.SelectOption),
		UnitTotal:             data["UnitTotal"].(int),
		BillableUnits:         data["BillableUnits"].(string),
		BillableLabel:         data["BillableLabel"].(string),
		FairUseFreeUnits:      data["FairUseFreeUnits"].(int),
		FairUseExceeded:       data["FairUseExceeded"].(bool),
		UnitsEmpty:            data["UnitsEmpty"].(emptyStateView),
		UnitMsg:               data["UnitMsg"].(string),
		UnitOK:                data["UnitOK"].(bool),
		PaymentMsg:            data["PaymentMsg"].(string),
		PaymentOK:             data["PaymentOK"].(bool),
		HasHomeProfile:        data["HasHomeProfile"].(bool),
		HomeProfileName:       homeProfile.HouseholdName,
		HomeProfileType:       string(homeProfile.HomeType),
		HasHomeProfileUnit:    data["HasHomeProfileUnit"].(bool),
		HomeProfileSaved:      data["HomeProfileSaved"].(bool),
		MapConfigured:         mapConfigured,
	}))
}

func (a *app) renderUserSettingsTempl(w http.ResponseWriter, r *http.Request, ac authCtx, data map[string]any) {
	a.renderSettingsComponent(w, r, ac.tenant.Slug, web.UserSettingsPage(web.UserSettingsPageData{
		Rights:                       a.userRightsData(ac, data["Users"].([]userRow)),
		RightsSaved:                  r.URL.Query().Get("rights") == "saved",
		Portal:                       a.settingsPortalContext(ac, "Benutzer & Rechte", "users"),
		AssetVersion:                 version.AssetVersion(),
		Users:                        data["Users"].([]userRow),
		UserCount:                    data["UserCount"].(int),
		ActiveUserCount:              data["ActiveUserCount"].(int),
		InvitedUserCount:             data["InvitedUserCount"].(int),
		DisabledUserCount:            data["DisabledUserCount"].(int),
		UsersEmpty:                   data["UsersEmpty"].(emptyStateView),
		InviteMsg:                    data["InviteMsg"].(string),
		InviteOK:                     data["InviteOK"].(bool),
		IsAdmin:                      ac.can(capabilityPlatformAdmin),
		ServiceProviderAccessEnabled: a.serviceAccessEnabled,
	}))
}
