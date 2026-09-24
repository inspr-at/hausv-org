package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

type energyLiveView struct {
	Main             energyMetricView
	HasMain          bool
	Flows            []energyMetricView
	Grid             energyMetricView
	HasGrid          bool
	Battery          energyMetricView
	HasBattery       bool
	BatterySOC       energyMetricView
	HasBatterySOC    bool
	BatteryFill      string
	Additional       []energyMetricView
	HasAdditional    bool
	AdditionalCount  int
	AdditionalTopics string
}

type energyLiveRefreshResponse struct {
	Flow      energyFlowConfig `json:"flow"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type energyChartSeriesView struct {
	Key            string
	Label          string
	Path           string
	MobilePath     string
	AreaPath       string
	MobileAreaPath string
	Latest         string
}

type energyChartTickView struct {
	Position       string
	MobilePosition string
	Label          string
}

type energyChartSampleValueView struct {
	Key            string
	Label          string
	Value          string
	Position       string
	MobilePosition string
}

type energyChartSampleView struct {
	Index          int
	Time           string
	Position       string
	MobilePosition string
	HitPosition    string
	HitWidth       string
	MobileHit      string
	MobileHitWidth string
	Values         []energyChartSampleValueView
}

type energyChartView struct {
	HasData                 bool
	IsToday                 bool
	Title                   string
	DialogTitle             string
	Series                  []energyChartSeriesView
	XTicks                  []energyChartTickView
	YTicks                  []energyChartTickView
	Samples                 []energyChartSampleView
	Summary                 string
	Detail                  string
	Status                  string
	Range                   string
	PointSet                int
	ThresholdLabel          string
	ThresholdPosition       string
	ThresholdMobilePosition string
	ThresholdValue          string
	HasThreshold            bool
}

type energyChartData struct {
	Key     string
	Label   string
	Values  []float64
	Present []bool
}

type energyObservationProgressView struct {
	Completed int
	Target    int
	Percent   int
	Label     string
	Title     string
	Measured  int
	Estimated int
}

type energyChartCacheEntry struct {
	View      energyChartView
	ExpiresAt time.Time
}

type energyCoverageView struct {
	Label  string
	Status string
	Detail string
	Tone   string
}

type energyRoadmapStep struct {
	Number  int
	Title   string
	Detail  string
	State   string
	Action  string
	URL     string
	Current bool
}

type energyCaretakerView struct {
	Email        string
	Name         string
	CanView      bool
	CanConfigure bool
	IsSelf       bool
	Editable     bool
}

type energyHomeUnitOption struct {
	Value    string
	Label    string
	Selected bool
}

func (a *app) canManageHomeIdentity(ac authCtx) bool {
	// A delegated technical caretaker may configure readings, but the shared
	// name and type of the home profile stay with owners and house management.
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	return err == nil && a.canManageHomeIdentityProfile(ac, profile, exists)
}

func (a *app) energyResidentialUnits(tenant store.TenantRef) []unit {
	if a.unitStore == nil {
		return nil
	}
	units, ok := store.BindUnitRepository(a.unitStore, tenant)
	if !ok {
		return nil
	}
	all := units.List()
	out := make([]unit, 0, len(all))
	for _, item := range all {
		if normalizeUnitType(item.UnitType) == unitTypeResidential {
			out = append(out, item)
		}
	}
	return out
}

func (a *app) effectiveEnergyUnit(tenant store.TenantRef, profile energy.HomeProfile) (unit, bool) {
	if profile.HomeType != energy.HomeApartment {
		return unit{}, false
	}
	units := a.energyResidentialUnits(tenant)
	unitID := normalizeUnitID(profile.UnitID)
	if unitID == "" {
		return unit{}, false
	}
	for _, item := range units {
		if normalizeUnitID(item.ID) == unitID {
			return item, true
		}
	}
	return unit{}, false
}

func (a *app) actorBelongsToEnergyUnit(ac authCtx, unitID string, ownerOnly bool) bool {
	if a.unitStore == nil {
		return false
	}
	units := ac.repositories.units
	if units == nil {
		var ok bool
		units, ok = store.BindUnitRepository(a.unitStore, ac.tenantRef)
		if !ok {
			return false
		}
	}
	unitID = normalizeUnitID(unitID)
	for _, membership := range units.UnitsForEmail(ac.email) {
		if normalizeUnitID(membership.Unit.ID) != unitID {
			continue
		}
		return !ownerOnly || normalizeRole(membership.Relation) == roleOwner
	}
	return false
}

func (a *app) ownerCanAccessEnergyProfile(ac authCtx, profile energy.HomeProfile, exists bool) bool {
	units := a.energyResidentialUnits(ac.tenantRef)
	if !exists || energyProfileUnclaimed(profile) {
		if len(units) == 0 {
			return normalizeRole(ac.role) == roleOwner
		}
		// A tenant-wide singleton must not be claimed by whichever owner happens
		// to start first. Owners may self-onboard only when one official home is
		// unambiguous; multi-unit recovery belongs to house management.
		return len(units) == 1 && a.actorBelongsToEnergyUnit(ac, units[0].ID, true)
	}
	if profile.HomeType != energy.HomeApartment {
		if len(units) == 0 {
			return normalizeRole(ac.role) == roleOwner
		}
		if len(units) == 1 {
			return a.actorBelongsToEnergyUnit(ac, units[0].ID, true)
		}
		return false
	}
	if linked, ok := a.effectiveEnergyUnit(ac.tenantRef, profile); ok {
		return a.actorBelongsToEnergyUnit(ac, linked.ID, true)
	}
	return normalizeRole(ac.role) == roleOwner &&
		len(units) == 0 &&
		strings.TrimSpace(profile.UnitID) == ""
}

func energyProfileUnclaimed(profile energy.HomeProfile) bool {
	return !profile.OnboardingComplete &&
		profile.OnboardingStep <= 1 &&
		strings.TrimSpace(profile.UnitID) == "" &&
		strings.TrimSpace(profile.HouseholdName) == ""
}

func (a *app) canManageHomeIdentityProfile(ac authCtx, profile energy.HomeProfile, exists bool) bool {
	if a.isEnergyHouseAdmin(ac) {
		return true
	}
	return a.ownerCanAccessEnergyProfile(ac, profile, exists)
}

func (a *app) homeIdentityUnitOptions(ac authCtx, profile energy.HomeProfile) []energyHomeUnitOption {
	selectedID := normalizeUnitID(profile.UnitID)
	if selectedID == "" {
		if linked, ok := a.effectiveEnergyUnit(ac.tenantRef, profile); ok {
			selectedID = normalizeUnitID(linked.ID)
		}
	}
	options := make([]energyHomeUnitOption, 0)
	for _, item := range a.energyResidentialUnits(ac.tenantRef) {
		if !a.isEnergyHouseAdmin(ac) && !a.actorBelongsToEnergyUnit(ac, item.ID, true) {
			continue
		}
		options = append(options, energyHomeUnitOption{
			Value:    item.ID,
			Label:    item.Label,
			Selected: normalizeUnitID(item.ID) == selectedID,
		})
	}
	if selectedID == "" && len(options) == 1 {
		options[0].Selected = true
	}
	return options
}

func (a *app) resolveHomeIdentityUnitID(ac authCtx, profile energy.HomeProfile, homeType string, raw string) (string, bool) {
	if homeType != energy.HomeApartment {
		return "", true
	}
	if linked, ok := a.effectiveEnergyUnit(ac.tenantRef, profile); ok {
		requested := normalizeUnitID(raw)
		if requested == "" || requested == normalizeUnitID(linked.ID) {
			return linked.ID, true
		}
		return "", false
	}
	options := a.homeIdentityUnitOptions(ac, profile)
	if len(options) == 0 {
		return "", strings.TrimSpace(profile.UnitID) == "" || a.isEnergyHouseAdmin(ac)
	}
	requested := normalizeUnitID(raw)
	if requested == "" {
		for _, option := range options {
			if option.Selected {
				requested = normalizeUnitID(option.Value)
				break
			}
		}
	}
	for _, option := range options {
		if normalizeUnitID(option.Value) == requested {
			return option.Value, true
		}
	}
	return "", false
}

func (a *app) homeIdentityTypeLocked(tenant store.TenantRef, profile energy.HomeProfile) bool {
	_, linked := a.effectiveEnergyUnit(tenant, profile)
	return profile.OnboardingComplete && linked
}

func (a *app) homeOnboarding(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canViewEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Hausprofil konnte nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	if !exists || energyProfileUnclaimed(profile) {
		if !exists {
			profile = energy.DefaultProfile(ac.tenant.Slug, time.Now())
		}
		profile.HouseholdName = houseDisplayName(ac.tenant)
		// A brand-new or deliberately reset profile has no history that could be
		// exposed. Preselect the sole official apartment; the first POST persists
		// this exact relation while preserving contract metadata on a reset marker.
		if units := a.energyResidentialUnits(ac.tenantRef); len(units) == 1 {
			profile.UnitID = units[0].ID
		}
	}
	unitOptions := a.homeIdentityUnitOptions(ac, profile)
	homeUnitID := ""
	homeUnitLabel, hasHomeUnit := a.energyHomeUnitLabel(ac.tenantRef, profile)
	if linked, ok := a.effectiveEnergyUnit(ac.tenantRef, profile); ok {
		homeUnitID = linked.ID
	}
	assets, _ := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	mappings, _ := a.energyFor(ac).ListMappings(ac.tenant.Slug)
	mappingSlots := buildEnergyMappingSlots(assets, mappings)
	intervals, _ := a.energyFor(ac).ListIntervals(ac.tenant.Slug, time.Now().AddDate(0, -1, 0), time.Time{})
	finishRecommendation := energy.NextRecommendation(profile, assets, mappings, intervals)
	step := profile.OnboardingStep
	if raw := strings.TrimSpace(r.URL.Query().Get("step")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed >= 1 && parsed <= 5 {
			step = parsed
		}
	}
	if step == 2 && !a.canManageHomeIdentityProfile(ac, profile, exists) {
		http.Error(w, "Name und Zuordnung dürfen nur zuständige Eigentümer oder die Hausverwaltung ändern.", http.StatusForbidden)
		return
	}
	discovery := energyDiscoveryView{}
	connectorMessage := "Home Assistant ist noch nicht verbunden. Das ist okay – Sie können später weitermachen."
	if step == 4 {
		ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
		defer cancel()
		states, configured, sourceErr := a.energyStates(ctx, ac.tenant)
		if discovered, discoverErr := discoverEnergyCandidatesFromStates(states, mappings); configured && sourceErr == nil && discoverErr == nil {
			discovery = discovered
			if len(discovery.Recommended) > 0 {
				connectorMessage = fmt.Sprintf("%d sichere Vorschläge gefunden. Sie bleiben vollständig lesend.", len(discovery.Recommended))
			} else {
				connectorMessage = "Verbindung hergestellt, aber noch kein eindeutig passender Hausenergie-Messwert gefunden."
			}
		} else if configured {
			connectorMessage = "Home Assistant antwortet gerade nicht. Ihre bisherigen Angaben bleiben erhalten."
		}
	}
	a.renderOnboardingTempl(w, r, web.OnboardingPageData{
		Portal:               a.onboardingPortalContext(ac),
		Step:                 step,
		Progress:             step * 20,
		ProfileReset:         r.URL.Query().Get("reset") == "1",
		HouseholdName:        profile.HouseholdName,
		HomeType:             profile.HomeType,
		HomeTypeLabel:        energyHomeTypeLabel(profile.HomeType),
		HomeTypeDescription:  energyHomeTypeDescription(profile.HomeType),
		HomeTypeLocked:       a.homeIdentityTypeLocked(ac.tenantRef, profile),
		HasHomeUnit:          hasHomeUnit,
		HomeUnitID:           homeUnitID,
		HomeUnitLabel:        homeUnitLabel,
		HasUnitOptions:       len(unitOptions) > 0,
		UnitOptions:          onboardingUnitOptions(unitOptions),
		AssetOptions:         onboardingAssetOptions(buildEnergyAssetOptions(assets)),
		MappingSlots:         onboardingMappingSlots(mappingSlots),
		MappingAssetOptions:  onboardingOptions(buildEnergyMappingAssetOptions(assets)),
		ConnectorMessage:     connectorMessage,
		Candidates:           onboardingCandidates(discovery.Recommended),
		HasCandidates:        len(discovery.Recommended) > 0,
		RecommendedCount:     len(discovery.Recommended),
		AdditionalCandidates: onboardingCandidates(discovery.Additional),
		HasAdditional:        len(discovery.Additional) > 0,
		FinishRecommendation: onboardingRecommendation(finishRecommendation),
		CanControlEnergy:     a.canControlEnergy(ac),
	})
}

func (a *app) updateHomeOnboarding(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Hausprofil konnte nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	if !exists || energyProfileUnclaimed(profile) {
		if !exists {
			profile = energy.DefaultProfile(ac.tenant.Slug, time.Now())
		}
		profile.HouseholdName = houseDisplayName(ac.tenant)
		// Persist the sole official apartment with a fresh or reset profile so
		// step two remains reachable. A retained free-period start stays intact.
		// The unfinished profile may still deliberately choose another type.
		if units := a.energyResidentialUnits(ac.tenantRef); len(units) == 1 {
			profile.UnitID = units[0].ID
		}
	}
	if !a.canManageHomeIdentityProfile(ac, profile, exists) {
		http.Error(w, "Die Einrichtung ist zuständigen Eigentümern und der Hausverwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	action := strings.TrimSpace(r.FormValue("action"))
	nextStep := profile.OnboardingStep
	switch action {
	case "understand":
		nextStep = 2
	case "profile":
		name := cleanEnergyText(r.FormValue("household_name"), 100)
		homeType := strings.TrimSpace(r.FormValue("home_type"))
		if name == "" || !validEnergyHomeType(homeType) {
			http.Error(w, "Name und Art des Zuhauses sind erforderlich.", http.StatusBadRequest)
			return
		}
		if a.homeIdentityTypeLocked(ac.tenantRef, profile) && homeType != profile.HomeType {
			http.Error(w, "Die Zuhause-Art ist an die offizielle Einheit gebunden.", http.StatusForbidden)
			return
		}
		unitID, validUnit := a.resolveHomeIdentityUnitID(ac, profile, homeType, r.FormValue("unit_id"))
		if !validUnit {
			http.Error(w, "Bitte eine eigene offizielle Einheit auswählen.", http.StatusBadRequest)
			return
		}
		profile.HouseholdName = name
		profile.HomeType = homeType
		profile.UnitID = unitID
		nextStep = 3
	case "assets":
		if err := a.saveOnboardingAssets(ac.tenant.Slug, r.Form["assets"]); err != nil {
			http.Error(w, "Geräte konnten nicht gespeichert werden.", http.StatusInternalServerError)
			return
		}
		nextStep = 4
	case "mappings":
		if err := a.saveSelectedEnergyMappings(r.Context(), ac.tenant, r.Form["entities"]); err != nil {
			http.Error(w, "Messwerte konnten nicht gespeichert werden.", http.StatusBadGateway)
			return
		}
		if strings.TrimSpace(r.FormValue("manual_entity_id")) != "" {
			if err := a.saveManualEnergyMapping(ac.tenant.Slug, r.FormValue("manual_entity_id"), r.FormValue("manual_metric"), r.FormValue("manual_name"), r.FormValue("manual_unit"), r.FormValue("manual_asset_id")); err != nil {
				http.Error(w, "Die manuelle Zuordnung ist ungültig.", http.StatusBadRequest)
				return
			}
		}
		nextStep = 5
	case "skip-mappings":
		nextStep = 5
	case "finish":
		profile.OnboardingComplete = true
		if profile.FreeStartedAt == nil {
			started := time.Now().UTC()
			until := started.AddDate(1, 0, 0)
			profile.FreeStartedAt = &started
			profile.FreeUntilAt = &until
		}
		nextStep = 5
	case "back":
		if nextStep > 1 {
			nextStep--
		}
	default:
		http.Error(w, "Ungültiger Schritt", http.StatusBadRequest)
		return
	}
	profile.OnboardingStep = nextStep
	if err := a.energyFor(ac).SaveProfile(profile); err != nil {
		http.Error(w, "Hausprofil konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.onboarding.update",
		TargetType: "home-profile",
		TargetID:   ac.tenant.Slug,
		Summary:    "Hausprofil eingerichtet",
		Details: map[string]string{
			"step": strconv.Itoa(nextStep),
		},
	})
	if profile.OnboardingComplete {
		http.Redirect(w, r, "/app/energie?welcome=1", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/zuhause/onboarding?step="+strconv.Itoa(nextStep), http.StatusSeeOther)
}

func (a *app) homeIdentitySettings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Hausprofil konnte nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	if !a.canManageHomeIdentityProfile(ac, profile, exists) {
		http.Error(w, "Dieser Bereich ist Eigentümern der zugeordneten Einheit und der Hausverwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !exists || !profile.OnboardingComplete {
		http.Redirect(w, r, "/app/zuhause/onboarding?step=2", http.StatusSeeOther)
		return
	}
	from := normalizeHomeIdentityReturn(r.URL.Query().Get("from"), ac.can(capabilityManageBuilding))
	backURL, backLabel := homeIdentityBackLink(from)
	unitOptions := a.homeIdentityUnitOptions(ac, profile)
	unitTitle, unitSummary := a.homeIdentityUnitContext(ac, profile)
	unitLabel, hasUnit := a.energyHomeUnitLabel(ac.tenantRef, profile)
	unitID := ""
	if linked, ok := a.effectiveEnergyUnit(ac.tenantRef, profile); ok {
		unitID = linked.ID
	}
	pageData := map[string]any{
		"Title":               profile.HouseholdName + " · Mein Zuhause",
		"ActivePage":          "settings",
		"Profile":             profile,
		"HomeTypeLabel":       energyHomeTypeLabel(profile.HomeType),
		"From":                from,
		"BackURL":             backURL,
		"BackLabel":           backLabel,
		"OfficialUnitTitle":   unitTitle,
		"OfficialUnitSummary": unitSummary,
		"HomeUnitLabel":       unitLabel,
		"HomeUnitID":          unitID,
		"HasHomeUnit":         hasUnit,
		"UnitOptions":         unitOptions,
		"HasUnitOptions":      len(unitOptions) > 0,
		"HomeTypeLocked":      a.homeIdentityTypeLocked(ac.tenantRef, profile),
		"HomeTypeDescription": energyHomeTypeDescription(profile.HomeType),
		"CanManageBuilding":   ac.can(capabilityManageBuilding),
		"Saved":               r.URL.Query().Get("saved") == "1",
		"Invalid":             r.URL.Query().Get("invalid") == "1",
	}
	a.renderHomeIdentitySettingsTempl(w, r, ac, profile, pageData)
}

func (a *app) updateHomeIdentity(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Hausprofil konnte nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	if !a.canManageHomeIdentityProfile(ac, profile, exists) {
		http.Error(w, "Dieser Bereich ist Eigentümern der zugeordneten Einheit und der Hausverwaltung vorbehalten.", http.StatusForbidden)
		return
	}
	if !exists || !profile.OnboardingComplete {
		http.Redirect(w, r, "/app/zuhause/onboarding?step=2", http.StatusSeeOther)
		return
	}
	from := normalizeHomeIdentityReturn(r.FormValue("from"), ac.can(capabilityManageBuilding))
	name := cleanEnergyText(r.FormValue("household_name"), 100)
	homeType := strings.TrimSpace(r.FormValue("home_type"))
	if name == "" || !validEnergyHomeType(homeType) {
		http.Redirect(w, r, "/app/settings/home?invalid=1&from="+from, http.StatusSeeOther)
		return
	}
	if a.homeIdentityTypeLocked(ac.tenantRef, profile) && homeType != profile.HomeType {
		http.Error(w, "Die Zuhause-Art ist an die offizielle Einheit gebunden.", http.StatusForbidden)
		return
	}
	unitID, validUnit := a.resolveHomeIdentityUnitID(ac, profile, homeType, r.FormValue("unit_id"))
	if !validUnit {
		http.Redirect(w, r, "/app/settings/home?invalid=1&from="+from, http.StatusSeeOther)
		return
	}
	changed := make([]string, 0, 3)
	if profile.HouseholdName != name {
		changed = append(changed, "Anzeigename")
	}
	if profile.HomeType != homeType {
		changed = append(changed, "Zuhause-Art")
	}
	if normalizeUnitID(profile.UnitID) != normalizeUnitID(unitID) {
		changed = append(changed, "Einheiten-Zuordnung")
	}
	profile.HouseholdName = name
	profile.HomeType = homeType
	profile.UnitID = unitID
	if err := a.energyFor(ac).SaveProfile(profile); err != nil {
		http.Error(w, "Hausprofil konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	if len(changed) > 0 {
		a.recordAudit(auditEvent{
			TenantSlug: ac.tenant.Slug,
			ActorEmail: ac.email,
			ActorRole:  ac.role,
			Action:     store.AuditActionEnergyIdentity,
			TargetType: "home-profile",
			TargetID:   ac.tenant.Slug,
			Summary:    "Darstellung von Mein Zuhause geändert",
			Details: map[string]string{
				"changed_fields": strings.Join(changed, ", "),
			},
		})
	}
	switch from {
	case "energy":
		http.Redirect(w, r, "/app/energie?profile=1", http.StatusSeeOther)
	case "building":
		http.Redirect(w, r, "/app/settings/building?section=units&home=saved", http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/app/settings/home?saved=1", http.StatusSeeOther)
	}
}

func validEnergyHomeType(value string) bool {
	switch strings.TrimSpace(value) {
	case energy.HomeApartment, energy.HomeHouse, energy.HomeCommunity:
		return true
	default:
		return false
	}
}

func energyHomeTypeLabel(value string) string {
	switch strings.TrimSpace(value) {
	case energy.HomeHouse:
		return "Einfamilienhaus"
	case energy.HomeCommunity:
		return "Hausgemeinschaft"
	default:
		return "Wohnung"
	}
}

func energyHomeTypeDescription(value string) string {
	switch strings.TrimSpace(value) {
	case energy.HomeHouse:
		return "Ein Haushalt mit eigenem Gebäude. Haus-, Heiz- und Energietechnik können gemeinsam betrachtet werden."
	case energy.HomeCommunity:
		return "Mehrere Parteien und gemeinsam genutzte Anlagen. Der Überblick richtet sich an Eigentümergemeinschaft oder Hausverwaltung."
	default:
		return "Ein einzelner Haushalt in einem Mehrparteienhaus. Der Überblick konzentriert sich auf die zugeordnete Einheit und ihre eigenen Geräte."
	}
}

func normalizeHomeIdentityReturn(value string, canManageBuilding bool) string {
	switch strings.TrimSpace(value) {
	case "energy":
		return "energy"
	case "building":
		if canManageBuilding {
			return "building"
		}
	}
	return "settings"
}

func homeIdentityBackLink(from string) (string, string) {
	switch from {
	case "energy":
		return "/app/energie", "Zurück zu Mein Zuhause"
	case "building":
		return "/app/settings/building?section=units", "Zurück zu Gebäude & Einheiten"
	default:
		return "/app/settings", "Zurück zu Einstellungen"
	}
}

func (a *app) energyHomeUnitLabel(tenant store.TenantRef, profile energy.HomeProfile) (string, bool) {
	linked, ok := a.effectiveEnergyUnit(tenant, profile)
	if !ok {
		return "", false
	}
	label := strings.TrimSpace(linked.Label)
	if label == "" {
		label = linked.ID
	}
	return label, label != ""
}

func (a *app) homeIdentityUnitContext(ac authCtx, profile energy.HomeProfile) (string, string) {
	if profile.HomeType != energy.HomeApartment {
		return "Geltungsbereich", "Gesamte Liegenschaft"
	}
	if label, ok := a.energyHomeUnitLabel(ac.tenantRef, profile); ok {
		return "Offizielle Einheit", label
	}
	if len(a.homeIdentityUnitOptions(ac, profile)) > 0 {
		return "Offizielle Einheit", "Noch nicht zugeordnet"
	}
	return "Offizielle Einheit", "Noch keine Einheit angelegt"
}

func buildEnergyObservationProgressView(intervals []energy.Interval) energyObservationProgressView {
	const target = 96
	counts := energy.CountUsableQuarters(intervals)
	completed := counts.Total
	if completed > target {
		completed = target
	}
	remaining := target - completed
	remainingMinutes := remaining * 15
	title := "Noch 1 Tag beobachten"
	if remaining == 1 {
		title = "Noch 1 Viertelstunde beobachten"
	} else if remaining > 1 && remainingMinutes < 60 {
		title = fmt.Sprintf("Noch %d Viertelstunden beobachten", remaining)
	} else if remainingMinutes > 0 && remainingMinutes < 24*60 {
		hours := remainingMinutes / 60
		minutes := remainingMinutes % 60
		if minutes == 0 {
			title = fmt.Sprintf("Noch %d Std. beobachten", hours)
		} else {
			title = fmt.Sprintf("Noch %d Std. %d Min. beobachten", hours, minutes)
		}
	}
	return energyObservationProgressView{
		Completed: completed,
		Target:    target,
		Percent:   completed * 100 / target,
		Label:     fmt.Sprintf("%d von %d Viertelstunden", completed, target),
		Title:     title,
		Measured:  counts.Measured,
		Estimated: counts.Estimated,
	}
}

func (a *app) energyCockpit(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canViewEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Energiedaten konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	if !exists || !profile.OnboardingComplete {
		http.Redirect(w, r, "/app/zuhause/onboarding", http.StatusSeeOther)
		return
	}
	assets, _ := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	mappings, _ := a.energyFor(ac).ListMappings(ac.tenant.Slug)
	// Named EVs from the pilot inventory predate per-consumer mappings. When
	// Home Assistant exposes an unambiguous matching home-charging sensor, bind
	// it once and then use the normal persisted measurement path.
	if ac.supportView == nil {
		if updated, changed := a.ensureNamedEVMeasurementMappings(r.Context(), ac.tenant, assets, mappings); changed {
			mappings = updated
		}
	}
	metrics, _, liveLastSeen := a.currentEnergyMetrics(r.Context(), ac.tenant, mappings, profile)
	live := buildEnergyLiveView(metrics)
	chart := a.energy24HourChart(r.Context(), ac.tenant, mappings, profile, time.Now(), r.URL.Query().Get("zeitraum"))
	coverage, coverageSummary := buildEnergyCoverageViews(assets, mappings)
	canManageEnergyData := a.canManageHomeIdentity(ac)
	imports := []energy.ImportRecord{}
	if canManageEnergyData {
		imports, _ = a.energyFor(ac).ListImports(ac.tenant.Slug)
	}
	importViews := make([]energyImportView, 0, len(imports))
	for _, item := range imports {
		importViews = append(importViews, energyImportView{
			Filename: item.Filename,
			Date:     item.ImportedAt.In(time.Local).Format("02.01.2006 15:04"),
			Format:   item.Format,
		})
	}
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
	monthIntervals, _ := a.energyFor(ac).ListIntervals(ac.tenant.Slug, monthStart.UTC(), time.Time{})
	peakViews := energyPeakViews(monthIntervals, time.Now())
	comparisonView, hasComparison := energyComparisonForView(monthIntervals, time.Now())
	recommendation := energy.NextRecommendation(profile, assets, mappings, monthIntervals)
	observationProgress := buildEnergyObservationProgressView(monthIntervals)
	gaps, conflicts := intervalQualityCounts(monthIntervals)
	lastSeen := latestEnergySeen(mappings, monthIntervals, liveLastSeen)
	quality := energy.AssessQuality(time.Now(), lastSeen, gaps, conflicts, len(monthIntervals))
	_, recordsItself := a.confirmedGridImportMapping(ac.tenant.Slug)
	tariffView := buildEnergyTariffView(profile, monthIntervals, recordsItself)
	scenarioViews := buildEnergyScenarioViews(profile, assets, monthIntervals)
	caretakers := a.energyCaretakerViews(ac)
	maintenance, _ := a.energyFor(ac).ListMaintenance(ac.tenant.Slug)
	if maintenanceRecommendation, ok := energy.MaintenanceRecommendation(time.Now(), maintenance); ok {
		recommendation = maintenanceRecommendation
	}
	contactOptions, contactNames := a.energyContactOptions(ac.repositories.contacts)
	documentOptions, documentNames := a.energyDocumentOptions(ac.tenantRef)
	issueOptions, issueNames := a.energyIssueOptions(ac.tenantRef)
	maintenanceViews := buildEnergyMaintenanceViews(maintenance, assets, contactNames, documentNames, issueNames, time.Now())
	assessments, _ := a.energyFor(ac).ListTariffAssessments(ac.tenant.Slug)
	assessmentViews := buildEnergyTariffAssessmentViews(assessments)
	measures, _ := a.energyFor(ac).ListMeasures(ac.tenant.Slug)
	measureViews := buildEnergyMeasureViews(measures, contactNames)
	freeUntil := ""
	if profile.FreeUntilAt != nil {
		freeUntil = profile.FreeUntilAt.In(time.Local).Format("02.01.2006")
	}
	homeUnitLabel, hasHomeUnit := a.energyHomeUnitLabel(ac.tenantRef, profile)
	chargingCtx, cancelCharging := context.WithTimeout(r.Context(), 5*time.Second)
	charging := a.chargingLiveView(chargingCtx, ac.tenant, false, false)
	cancelCharging()
	flowConfig := buildEnergyFlowConfig(ac.tenant.Slug, live, assets, mappings, metrics, charging, canManageEnergyData)
	flowConfigJSON, err := json.Marshal(flowConfig)
	if err != nil {
		// Never silent: without the config the client keeps the no-JS fallback.
		logError("energy flow config not encodable", err, "tenant", ac.tenant.Slug)
		flowConfigJSON = []byte("null")
	}
	lucideIconNamesJSON, err := json.Marshal(web.LucideIconNames())
	if err != nil {
		lucideIconNamesJSON = []byte("[]")
	}
	systemAssets := energySystemAssets(assets)
	a.renderEnergyTempl(w, r, web.EnergyPageData{
		Portal:                  a.energyPortalContext(ac, profile.HouseholdName),
		FlowConfigJSON:          energyWebJSON(flowConfigJSON, "null"),
		LucideIconNamesJSON:     energyWebJSON(lucideIconNamesJSON, "[]"),
		HouseholdName:           profile.HouseholdName,
		HomeIdentity:            a.homeIdentityFromProfile(ac.tenantRef, profile),
		HomeTypeLabel:           energyHomeTypeLabel(profile.HomeType),
		HomeUnitLabel:           homeUnitLabel,
		HasHomeUnit:             hasHomeUnit,
		Welcome:                 r.URL.Query().Get("welcome") == "1",
		ModeChanged:             r.URL.Query().Get("mode") == "1",
		ProfileChanged:          r.URL.Query().Get("profile") == "1",
		ConsumerNotice:          r.URL.Query().Get("verbraucher"),
		MeasureCreated:          r.URL.Query().Get("measure") == "created",
		RecommendationDeferred:  profile.RecommendationID == recommendation.ID && profile.RecommendationStatus == "deferred",
		RecommendationDismissed: profile.RecommendationID == recommendation.ID && profile.RecommendationStatus == "dismissed",
		Recommendation: web.EnergyRecommendationView{
			ID: recommendation.ID, Title: recommendation.Title, Reason: recommendation.Reason,
			Benefit: recommendation.Benefit, Effort: recommendation.Effort, ImpactRange: recommendation.ImpactRange,
		},
		RecommendationURL: recommendationURL(recommendation.ID),
		ObservationProgress: web.EnergyObservationProgressView{
			Completed: observationProgress.Completed, Target: observationProgress.Target,
			Percent: observationProgress.Percent, Label: observationProgress.Label, Title: observationProgress.Title,
		},
		IsActiveMode:           profile.OperatingMode == energy.ModeActive,
		IsShadowMode:           profile.AutomationStage == energy.StageShadow,
		CanManageEnergy:        a.canManageEnergy(ac),
		CanControlEnergy:       a.canControlEnergy(ac),
		CanManageHomeIdentity:  canManageEnergyData,
		CanManageEnergyData:    canManageEnergyData,
		CanGrantEnergyAccess:   a.canControlEnergy(ac),
		CanInviteEnergyAccess:  a.canControlEnergy(ac),
		MetricCount:            len(metrics),
		HasMetrics:             len(metrics) > 0,
		Live:                   energyWebLive(live),
		Chart:                  energyWebChart(chart),
		Tariff:                 energyWebTariff(tariffView),
		TargetChanged:          r.URL.Query().Get("target") == "1",
		AgreedPowerChanged:     r.URL.Query().Get("agreed") == "1",
		TargetPeakValue:        energyTargetValue(profile.TargetPeakKW),
		AgreedPowerValue:       energyTargetValue(profile.AgreedPowerKW),
		TariffAssessmentStatus: r.URL.Query().Get("assessment"),
		TariffAssessments:      energyWebAssessments(assessmentViews),
		HasTariffAssessments:   len(assessmentViews) > 0,
		Quality: web.EnergyQualityView{
			Status: quality.Status, Label: quality.Label, Effect: quality.Effect, NextAction: quality.NextAction,
		},
		Coverage:          energyWebCoverage(coverage),
		CoverageSummary:   coverageSummary,
		Roadmap:           energyWebRoadmap(energyRoadmap(profile, len(assets), len(mappings))),
		SystemAssets:      energyWebSystemAssets(systemAssets),
		HasSystemAssets:   len(systemAssets) > 0,
		FreeUntil:         freeUntil,
		Assets:            energyWebAssets(assets),
		HasAssets:         len(assets) > 0,
		Maintenance:       energyWebMaintenance(maintenanceViews),
		HasMaintenance:    len(maintenanceViews) > 0,
		MaintenanceStatus: r.URL.Query().Get("maintenance"),
		ContactOptions:    energyWebOptions(contactOptions),
		DocumentOptions:   energyWebOptions(documentOptions),
		IssueOptions:      energyWebOptions(issueOptions),
		ImportStatus:      r.URL.Query().Get("import"),
		Imports:           energyWebImports(importViews),
		HasImports:        len(importViews) > 0,
		Peaks:             energyWebPeaks(peakViews),
		HasPeaks:          len(peakViews) > 0,
		Comparison: web.EnergyComparisonView{
			Tone: comparisonView.Tone, Title: comparisonView.Title, Details: comparisonView.Details,
		},
		HasComparison:         hasComparison,
		Caretakers:            energyWebCaretakers(caretakers),
		HasCaretakers:         len(caretakers) > 0,
		CaretakerChanged:      r.URL.Query().Get("caretaker") == "1",
		CaretakerInviteStatus: r.URL.Query().Get("caretaker_invite"),
		Measures:              energyWebMeasures(measureViews),
		HasMeasures:           len(measureViews) > 0,
		MeasureStatus:         r.URL.Query().Get("measure_status"),
		Scenarios:             energyWebScenarios(scenarioViews),
		HasScenarios:          len(scenarioViews) > 0,
		ConsumerKindOptions:   energyWebOptions(buildEnergyConsumerKindOptions()),
		ConsumerIconOptions:   energyWebOptions(buildEnergyConsumerIconOptions()),
	})
}

// energyLiveRefresh returns only the data required to redraw the live flow.
// Keeping this separate from the full cockpit avoids page jumps, scroll loss
// and expensive chart/report rebuilding on the ten-second refresh cycle.
func (a *app) energyLiveRefresh(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canViewEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Energiedaten konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	if !exists || !profile.OnboardingComplete {
		http.Error(w, "Energie-Cockpit ist noch nicht eingerichtet.", http.StatusConflict)
		return
	}
	assets, _ := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	mappings, _ := a.energyFor(ac).ListMappings(ac.tenant.Slug)
	if ac.supportView == nil {
		if updated, changed := a.ensureNamedEVMeasurementMappings(r.Context(), ac.tenant, assets, mappings); changed {
			mappings = updated
		}
	}
	metrics, _, _ := a.currentEnergyMetrics(r.Context(), ac.tenant, mappings, profile)
	if len(metrics) == 0 {
		http.Error(w, "Home Assistant liefert gerade keine Live-Werte.", http.StatusServiceUnavailable)
		return
	}
	live := buildEnergyLiveView(metrics)
	chargingCtx, cancelCharging := context.WithTimeout(r.Context(), 5*time.Second)
	charging := a.chargingLiveView(chargingCtx, ac.tenant, false, false)
	cancelCharging()

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(energyLiveRefreshResponse{
		Flow:      buildEnergyFlowConfig(ac.tenant.Slug, live, assets, mappings, metrics, charging, a.canManageHomeIdentity(ac)),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		return
	}
}

func recommendationURL(id string) string {
	if strings.HasPrefix(id, "maintenance-") {
		return "/app/energie#wartung"
	}
	switch id {
	case "inventory":
		return "/app/zuhause/onboarding?step=3"
	case "measure":
		return "/app/zuhause/onboarding?step=4"
	case "target":
		return "/app/energie#tarif"
	case "simulate":
		return "/app/energie#szenarien"
	default:
		return ""
	}
}

func (a *app) updateEnergyMode(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canControlEnergy(ac) {
		http.Error(w, "Nur Eigentümer oder Hausadministration dürfen den Modus ändern.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil || !exists {
		http.Error(w, "Hausprofil fehlt.", http.StatusBadRequest)
		return
	}
	requested := strings.TrimSpace(r.FormValue("mode"))
	before := profile.OperatingMode
	switch requested {
	case energy.ModeObserve:
		profile.OperatingMode = energy.ModeObserve
		profile.AutomationStage = energy.StageObserve
	case energy.ModeActive:
		if r.FormValue("confirm") != "yes" || strings.TrimSpace(r.FormValue("confirmation_text")) != "TESTLAUF" {
			http.Error(w, "Der Testlauf muss bewusst bestätigt werden.", http.StatusBadRequest)
			return
		}
		profile.OperatingMode = energy.ModeActive
		// A house-wide grant opens the gate, but starts in Shadow Mode. A
		// separately reviewed device policy must promote it to real actions.
		profile.AutomationStage = energy.StageShadow
	default:
		http.Error(w, "Unbekannter Modus", http.StatusBadRequest)
		return
	}
	if err := a.energyFor(ac).SaveProfile(profile); err != nil {
		http.Error(w, "Modus konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.mode.change",
		TargetType: "home-profile",
		TargetID:   ac.tenant.Slug,
		Summary:    "Energiemodus geändert",
		Details: map[string]string{
			"mode_from": before,
			"mode_to":   profile.OperatingMode,
		},
	})
	http.Redirect(w, r, "/app/energie?mode=1", http.StatusSeeOther)
}

func (a *app) updateEnergyTarget(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	value, err := homeassistant.ParseFloat(r.FormValue("target_peak_kw"))
	if err != nil || !energy.ValidPowerKW(value, 1000) {
		http.Error(w, "Peak-Ziel muss eine positive kW-Zahl sein.", http.StatusBadRequest)
		return
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil || !exists {
		http.Error(w, "Hausprofil fehlt.", http.StatusBadRequest)
		return
	}
	before := ""
	if profile.TargetPeakKW != nil {
		before = formatEnergyNumber(*profile.TargetPeakKW)
	}
	profile.TargetPeakKW = &value
	if err := a.energyFor(ac).SaveProfile(profile); err != nil {
		http.Error(w, "Peak-Ziel konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.target.change",
		TargetType: "home-profile",
		TargetID:   ac.tenant.Slug,
		Summary:    "Peak-Ziel geändert",
		Details: map[string]string{
			"target_from_kw": before,
			"target_to_kw":   formatEnergyNumber(value),
		},
	})
	http.Redirect(w, r, "/app/energie?target=1#tarif", http.StatusSeeOther)
}

func (a *app) updateEnergyRecommendation(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canViewEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil || !exists {
		http.Error(w, "Hausprofil fehlt.", http.StatusBadRequest)
		return
	}
	id := cleanEnergyText(r.FormValue("recommendation_id"), 80)
	status := strings.TrimSpace(r.FormValue("status"))
	if id == "" || (status != "deferred" && status != "dismissed") {
		http.Error(w, "Ungültige Auswahl", http.StatusBadRequest)
		return
	}
	profile.RecommendationID = id
	profile.RecommendationStatus = status
	if err := a.energyFor(ac).SaveProfile(profile); err != nil {
		http.Error(w, "Auswahl konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.recommendation.update",
		TargetType: "energy-recommendation",
		TargetID:   id,
		Summary:    "Energieempfehlung eingeordnet",
		Details:    map[string]string{"status": status},
	})
	http.Redirect(w, r, "/app/energie#naechster-schritt", http.StatusSeeOther)
}

func (a *app) inviteEnergyCaretaker(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canControlEnergy(ac) {
		http.Error(w, "Nur Eigentümer oder Hausadministration dürfen technische Betreuung einladen.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	email := normalizeEmail(r.FormValue("email"))
	if _, err := mail.ParseAddress(email); err != nil {
		http.Redirect(w, r, "/app/energie?caretaker_invite=invalid#betreuung", http.StatusSeeOther)
		return
	}
	scopes := map[string]bool{}
	for _, scope := range r.Form["scope"] {
		scopes[scope] = true
	}
	permissions := []string{permissionEnergyView}
	if scopes["configure"] {
		permissions = append(permissions, permissionEnergyConfigure)
	}
	profile := userProfile{
		Email:       email,
		FirstName:   cleanEnergyText(r.FormValue("first_name"), 80),
		LastName:    cleanEnergyText(r.FormValue("last_name"), 80),
		Role:        roleResident,
		Status:      "Eingeladen",
		Tenants:     []string{ac.tenant.Slug},
		Permissions: permissions,
		AuthMethods: defaultAuthMethods(),
	}
	if existing, ok := a.directoryProfile(email); ok {
		if existing.HasTenant(ac.tenant.Slug) {
			http.Redirect(w, r, "/app/energie?caretaker_invite=exists#betreuung", http.StatusSeeOther)
			return
		}
		if _, stored := a.inviteStore.Get(email); !stored {
			http.Redirect(w, r, "/app/energie?caretaker_invite=error#betreuung", http.StatusSeeOther)
			return
		}
		if _, found, membershipErr := a.inviteStore.SetTenantMembership(email, ac.tenant.Slug, roleResident, permissions); membershipErr != nil || !found {
			http.Redirect(w, r, "/app/energie?caretaker_invite=error#betreuung", http.StatusSeeOther)
			return
		}
	} else {
		added, addErr := a.inviteStore.Add(profile)
		if addErr != nil || !added {
			http.Redirect(w, r, "/app/energie?caretaker_invite=error#betreuung", http.StatusSeeOther)
			return
		}
	}
	mailStatus := "verschickt"
	if err := a.mailer.SendInvite(email, a.publicBaseURL(r, ac.tenant)+"/", ac.tenant.Address); err != nil {
		mailStatus = "nicht zugestellt"
		logWarn("energy caretaker invite delivery failed",
			"tenant", ac.tenant.Slug,
			"error_type", fmt.Sprintf("%T", err),
		)
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.caretaker.invite",
		TargetType: "user",
		TargetID:   email,
		Summary:    "Technische Vertrauensperson eingeladen",
		Details: map[string]string{
			"permissions": strings.Join(permissionLabelList(permissions), ", "),
			"mail_status": mailStatus,
		},
	})
	status := "invited"
	if mailStatus != "verschickt" {
		status = "saved_no_mail"
	}
	http.Redirect(w, r, "/app/energie?caretaker_invite="+status+"#betreuung", http.StatusSeeOther)
}

func (a *app) energyCaretakerViews(ac authCtx) []energyCaretakerView {
	candidates := map[string]userProfile{}
	for email, profile := range a.profiles {
		candidates[normalizeEmail(email)] = profile
	}
	if a.inviteStore != nil {
		for _, profile := range a.inviteStore.List() {
			candidates[normalizeEmail(profile.Email)] = profile
		}
	}
	out := []energyCaretakerView{}
	for email, candidate := range candidates {
		if effective, ok := a.directoryProfile(email); ok {
			candidate = effective
		}
		if !candidate.HasTenant(ac.tenant.Slug) || normalizeEmail(candidate.Email) == normalizeEmail(ac.email) {
			continue
		}
		member := candidate.ForTenant(ac.tenant.Slug)
		if isServiceProviderRole(member.Role) {
			continue
		}
		canView := member.HasPermission(permissionEnergyView) || member.HasPermission(permissionEnergyConfigure) ||
			member.HasPermission(permissionEnergyControl) || member.HasPermission(permissionEnergyCaretaker)
		if !canView {
			continue
		}
		stored, hasStored := userProfile{}, false
		if a.inviteStore != nil {
			stored, hasStored = a.inviteStore.Get(member.Email)
		}
		_, isEnv := a.profiles[normalizeEmail(member.Email)]
		editable := hasStored && (!isEnv || stored.Adopted)
		out = append(out, energyCaretakerView{
			Email:        member.Email,
			Name:         member.DisplayName(),
			CanView:      canView,
			CanConfigure: member.HasPermission(permissionEnergyConfigure) || member.HasPermission(permissionEnergyCaretaker),
			Editable:     editable,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func (a *app) updateEnergyCaretaker(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canControlEnergy(ac) {
		http.Error(w, "Nur Eigentümer oder Hausadministration dürfen technische Zugriffe vergeben.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	email := normalizeEmail(r.FormValue("email"))
	existing, ok := a.inviteStore.Get(email)
	if !ok || !existing.HasTenant(ac.tenant.Slug) || isServiceProviderRole(existing.ForTenant(ac.tenant.Slug).Role) {
		http.Error(w, "Person gehört nicht zu dieser Liegenschaft.", http.StatusBadRequest)
		return
	}
	grant := map[string]bool{}
	for _, scope := range r.Form["scope"] {
		grant[scope] = true
	}
	_, found, err := a.inviteStore.MutateTenantPermissions(email, ac.tenant.Slug, func(current []string) []string {
		out := []string{}
		for _, permission := range current {
			switch permission {
			case permissionEnergyView, permissionEnergyConfigure, permissionEnergyControl, permissionEnergyCaretaker:
				continue
			default:
				out = append(out, permission)
			}
		}
		if grant["view"] || grant["configure"] {
			out = append(out, permissionEnergyView)
		}
		if grant["configure"] {
			out = append(out, permissionEnergyConfigure)
		}
		return out
	})
	if err != nil || !found {
		http.Error(w, "Zugriff konnte nicht geändert werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.caretaker.scope",
		TargetType: "user",
		TargetID:   email,
		Summary:    "Technischer Zugriff auf die Liegenschaft geändert",
		Details: map[string]string{
			"view":      strconv.FormatBool(grant["view"] || grant["configure"] || grant["control"]),
			"configure": strconv.FormatBool(grant["configure"]),
			"control":   strconv.FormatBool(grant["control"]),
		},
	})
	http.Redirect(w, r, "/app/energie?caretaker=1#betreuung", http.StatusSeeOther)
}

func (a *app) currentEnergyMetrics(ctx context.Context, tenant tenantConfig, mappings []energy.EntityMapping, profile energy.HomeProfile) ([]energyMetricView, string, time.Time) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	states := []homeassistant.EntityState{}
	if !tenant.HA.Configured() {
		var configured bool
		var sourceErr error
		states, configured, sourceErr = a.energyStates(timeoutCtx, tenant)
		if !configured {
			return nil, "Noch keine Messquelle verbunden", time.Time{}
		}
		if sourceErr != nil {
			return nil, "Messquelle ist gerade nicht erreichbar", time.Time{}
		}
	}
	type reading struct {
		mapping energy.EntityMapping
		state   homeassistant.EntityState
	}
	readings := make([]reading, 0, len(mappings))
	for _, mapping := range mappings {
		if !mapping.Confirmed {
			continue
		}
		var state homeassistant.EntityState
		var err error
		if tenant.HA.Configured() {
			state, err = tenant.HA.State(timeoutCtx, mapping.EntityID)
		} else {
			state, err = energyStateByID(states, mapping.EntityID)
		}
		if err != nil {
			continue
		}
		readings = append(readings, reading{mapping: mapping, state: state})
	}
	sort.Slice(readings, func(i, j int) bool {
		if readings[i].mapping.Metric == readings[j].mapping.Metric {
			return readings[i].mapping.EntityID < readings[j].mapping.EntityID
		}
		return readings[i].mapping.Metric < readings[j].mapping.Metric
	})
	metrics := make([]energyMetricView, 0, len(readings))
	var latest time.Time
	for _, reading := range readings {
		unit := reading.mapping.Unit
		if unit == "" {
			unit = haAttribute(reading.state.Attributes, "unit_of_measurement")
		}
		detail := reading.mapping.DisplayName
		if detail == "" {
			detail = reading.mapping.EntityID
		}
		metric := energyMetricForDisplay(reading.mapping, reading.state)
		seen := reading.state.LastUpdated
		if seen.IsZero() {
			seen = reading.state.LastChanged
		}
		if seen.After(latest) {
			latest = seen
		}
		item := energyMetricView{
			AssetID:     reading.mapping.AssetID,
			Metric:      metric,
			Kind:        energyMetricKind(metric, detail),
			Label:       energyMetricLabel(metric),
			Detail:      detail,
			Unit:        unit,
			LastUpdated: seen,
		}
		sourceState := strings.ToLower(strings.TrimSpace(reading.state.State))
		if metric == energy.MetricConsumerSleep {
			if sleeping, known := energyVehicleSleepState(sourceState); known {
				if sleeping {
					item.SourceState = "sleep"
				} else {
					item.SourceState = "awake"
				}
			} else if sourceState == "unknown" || sourceState == "unavailable" {
				item.SourceState = sourceState
			} else {
				continue
			}
			metrics = append(metrics, item)
			continue
		}
		if sourceState == "unknown" || sourceState == "unavailable" {
			item.SourceState = sourceState
			metrics = append(metrics, item)
			continue
		}
		value, err := homeassistant.ParseFloat(reading.state.State)
		if err != nil {
			continue
		}
		item.Value = formatEnergyReading(value, unit)
		item.Tone = energyMetricTone(metric, value, unit, profile.TargetPeakKW)
		item.Numeric = value
		metrics = append(metrics, item)
	}
	if len(metrics) == 0 {
		return nil, "Messquelle verbunden, aber noch keine bestätigten Live-Werte", time.Time{}
	}
	return metrics, "Live aus Home Assistant · nur gelesen", latest
}

func energyMetricForDisplay(mapping energy.EntityMapping, state homeassistant.EntityState) string {
	if mapping.Metric != energy.MetricBatteryPower {
		return mapping.Metric
	}
	// Older auto-discovery classified a few battery-prefixed consumption
	// sensors as battery power. Correct that narrow presentation mistake
	// without mutating the user's confirmed mapping.
	candidate, ok := classifyHAState(state)
	if ok && candidate.Metric == energy.MetricLoadPower {
		return candidate.Metric
	}
	return energyMetricForMapping(mapping)
}

func energyMetricForMapping(mapping energy.EntityMapping) string {
	if mapping.Metric != energy.MetricBatteryPower {
		return mapping.Metric
	}
	name := strings.ToLower(mapping.DisplayName + " " + mapping.EntityID)
	if strings.Contains(name, "home current consumption") ||
		strings.Contains(name, "home_consumption") ||
		strings.Contains(name, "consumption_current") ||
		strings.Contains(name, "hausverbrauch") {
		return energy.MetricLoadPower
	}
	return mapping.Metric
}

func energyMetricKind(metric, detail string) string {
	if metric == energy.MetricBatteryCharge {
		return "battery-charge"
	}
	if metric == energy.MetricBatteryDischarge {
		return "battery-discharge"
	}
	if metric != energy.MetricBatteryPower {
		return metric
	}
	name := strings.ToLower(detail)
	switch {
	case strings.Contains(name, "discharge") || strings.Contains(name, "entlad"):
		return "battery-discharge"
	case strings.Contains(name, "charge") || strings.Contains(name, "lade"):
		return "battery-charge"
	default:
		return metric
	}
}

func buildEnergyLiveView(metrics []energyMetricView) energyLiveView {
	usable := make([]energyMetricView, 0, len(metrics))
	for _, metric := range metrics {
		if metric.SourceState == "" && metric.Metric != energy.MetricConsumerSleep {
			usable = append(usable, metric)
		}
	}
	metrics = usable
	var view energyLiveView
	selected := map[int]bool{}

	first := func(metric string) (energyMetricView, int, bool) {
		for index, item := range metrics {
			if !selected[index] && item.Metric == metric {
				return item, index, true
			}
		}
		return energyMetricView{}, 0, false
	}

	if item, index, ok := first(energy.MetricLoadPower); ok {
		item.Label = "Hausverbrauch"
		item.Detail = "Momentan aus allen Quellen"
		view.Main = item
		view.HasMain = true
		selected[index] = true
	}

	for _, metric := range []string{
		energy.MetricPVPower,
		energy.MetricGridImportPower,
		energy.MetricGridExportPower,
	} {
		if item, index, ok := first(metric); ok {
			item.Detail = energyFlowDetail(metric)
			view.Flows = append(view.Flows, item)
			selected[index] = true
		}
	}
	for index := range view.Flows {
		item := view.Flows[index]
		if item.Metric != energy.MetricGridImportPower && item.Metric != energy.MetricGridExportPower {
			continue
		}
		if !view.HasGrid || math.Abs(energyPowerWatts(&item)) > math.Abs(energyPowerWatts(&view.Grid)) {
			view.Grid = item
			view.HasGrid = true
		}
	}

	view.Battery, view.HasBattery = combineBatteryMetrics(metrics, selected)
	if item, index, ok := first(energy.MetricBatterySOC); ok {
		fill := math.Max(0, math.Min(100, item.Numeric))
		item.Label = "Speicher"
		item.Detail = "Ladestand"
		item.Numeric = fill
		item.Value = formatEnergyReading(fill, "%")
		view.BatterySOC = item
		view.HasBatterySOC = true
		view.BatteryFill = formatEnergySVGNumber(fill)
		selected[index] = true
	}

	topics := map[string]bool{}
	for index, item := range metrics {
		if selected[index] {
			continue
		}
		item.Detail = ""
		view.Additional = append(view.Additional, item)
		switch item.Metric {
		case energy.MetricGridImportEnergy:
			topics["Verbrauch"] = true
		case energy.MetricBatteryPower, energy.MetricBatteryCharge, energy.MetricBatteryDischarge:
			topics["Speicher"] = true
		default:
			topics[item.Label] = true
		}
	}
	view.HasAdditional = len(view.Additional) > 0
	view.AdditionalCount = len(view.Additional)
	labels := make([]string, 0, len(topics))
	for label := range topics {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	view.AdditionalTopics = strings.Join(labels, ", ")
	return view
}

func (a *app) energy24HourChart(ctx context.Context, tenant tenantConfig, mappings []energy.EntityMapping, profile energy.HomeProfile, now time.Time, requestedRange string) energyChartView {
	today := requestedRange == "heute"
	view := energyChartView{
		IsToday:     today,
		Title:       "Letzte 24 Stunden",
		DialogTitle: "Letzte 24 Stunden im Detail",
		Status:      "Für den 24-Stunden-Verlauf fehlen noch ausreichend aufgezeichnete Leistungswerte.",
		Range:       "Letzte 24 Stunden",
	}
	if today {
		view.Title = "Heute · 00 bis 24 Uhr"
		view.DialogTitle = "Der heutige Tag im Detail"
		view.Range = "Heute · vollständige Tagesachse von 00 bis 24 Uhr"
	}
	if !tenant.HA.Configured() {
		return view
	}

	selected := map[string]energy.EntityMapping{}
	entitySet := map[string]bool{}
	for _, mapping := range mappings {
		if !mapping.Confirmed {
			continue
		}
		metric := energyMetricForMapping(mapping)
		key := metric
		if metric == energy.MetricBatteryPower || metric == energy.MetricBatteryCharge || metric == energy.MetricBatteryDischarge {
			key = energyMetricKind(metric, mapping.DisplayName+" "+mapping.EntityID)
		}
		switch key {
		case energy.MetricLoadPower, energy.MetricPVPower, energy.MetricGridImportPower,
			energy.MetricGridExportPower, "battery-charge", "battery-discharge", energy.MetricBatteryPower:
		default:
			continue
		}
		if _, exists := selected[key]; exists {
			continue
		}
		selected[key] = mapping
		entitySet[mapping.EntityID] = true
	}
	entityIDs := make([]string, 0, len(entitySet))
	for entityID := range entitySet {
		entityIDs = append(entityIDs, entityID)
	}
	sort.Strings(entityIDs)
	if len(entityIDs) == 0 {
		return view
	}
	// Die Staffelschwelle dient hier nur als sinnvolle Vorbelegung der
	// Planungsgrenze. Sie bleibt eine Produktannahme und wird als solche
	// beschriftet — sie bildet weder die Tarifstaffel noch den § 18-Referenzwert ab.
	threshold := energy.AustrianDraft2027().TierThresholdKW
	view.ThresholdLabel = "Planungsgrenze"
	if profile.TargetPeakKW != nil && *profile.TargetPeakKW > 0 {
		threshold = *profile.TargetPeakKW
		view.ThresholdLabel = "Persönliches Ziel"
	}
	view.ThresholdValue = formatEnergyValueUnit(formatEnergyCompact(threshold, 1), "kW")
	location := time.Local
	if vienna, err := time.LoadLocation("Europe/Vienna"); err == nil {
		location = vienna
	}
	end := now.UTC()
	start := end.Add(-24 * time.Hour)
	historyEnd := end
	rangeKey := "rolling"
	if today {
		localNow := now.In(location)
		localStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
		localEnd := localStart.AddDate(0, 0, 1)
		start = localStart.UTC()
		end = localEnd.UTC()
		historyEnd = now.UTC()
		if historyEnd.After(end) {
			historyEnd = end
		}
		rangeKey = "today-" + localStart.Format("20060102")
	}
	cacheKey := tenant.Slug + "|" + strings.Join(entityIDs, ",") + "|" + formatEnergySVGNumber(threshold) + "|" + rangeKey
	if cached, ok := a.cachedEnergyChart(cacheKey, now); ok {
		return cached
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	history, err := tenant.HA.History(timeoutCtx, start, historyEnd, entityIDs)
	if err != nil {
		a.cacheEnergyChart(cacheKey, view, now.Add(30*time.Second))
		return view
	}

	points := int(end.Sub(start)/(15*time.Minute)) + 1
	valuesFor := func(key string) ([]float64, []bool) {
		mapping, ok := selected[key]
		if !ok {
			return make([]float64, points), make([]bool, points)
		}
		return energyHistoryBuckets(history[mapping.EntityID], mapping.Unit, start, end, historyEnd, points)
	}

	loadValues, loadPresent := valuesFor(energy.MetricLoadPower)
	pvValues, pvPresent := valuesFor(energy.MetricPVPower)
	importValues, importPresent := valuesFor(energy.MetricGridImportPower)
	exportValues, exportPresent := valuesFor(energy.MetricGridExportPower)
	chargeValues, chargePresent := valuesFor("battery-charge")
	dischargeValues, dischargePresent := valuesFor("battery-discharge")
	genericBattery, genericBatteryPresent := valuesFor(energy.MetricBatteryPower)

	gridValues, gridPresent := combineEnergyHistory(importValues, importPresent, exportValues, exportPresent, -1)
	batteryValues, batteryPresent := combineEnergyHistory(dischargeValues, dischargePresent, chargeValues, chargePresent, -1)
	if countPresent(batteryPresent) == 0 {
		batteryValues, batteryPresent = genericBattery, genericBatteryPresent
	}

	data := []energyChartData{
		{Key: "load", Label: "Hausverbrauch", Values: loadValues, Present: loadPresent},
		{Key: "pv", Label: "PV-Erzeugung", Values: pvValues, Present: pvPresent},
		{Key: "grid", Label: "Netz", Values: gridValues, Present: gridPresent},
		{Key: "battery", Label: "Speicher", Values: batteryValues, Present: batteryPresent},
	}
	minimumPoints := 2
	if today {
		minimumPoints = 1
	}
	maxAbsolute := threshold
	for _, series := range data {
		if countPresent(series.Present) < minimumPoints {
			continue
		}
		for index, value := range series.Values {
			if !series.Present[index] {
				continue
			}
			maxAbsolute = math.Max(maxAbsolute, math.Abs(value))
		}
	}
	if maxAbsolute == 0 {
		a.cacheEnergyChart(cacheKey, view, now.Add(30*time.Second))
		return view
	}
	limit := energyChartRoundedLimit(maxAbsolute)
	minValue, maxValue := -limit, limit
	zeroY := energyChartValuePosition(0, minValue, maxValue)

	for _, series := range data {
		if countPresent(series.Present) < minimumPoints {
			continue
		}
		areaPath, mobileAreaPath := "", ""
		if series.Key == "load" {
			areaPath = energyChartAreaPath(series.Values, series.Present, minValue, maxValue, 52, 788, zeroY)
			mobileAreaPath = energyChartAreaPath(series.Values, series.Present, minValue, maxValue, 44, 388, zeroY)
		}
		view.Series = append(view.Series, energyChartSeriesView{
			Key:            series.Key,
			Label:          series.Label,
			Path:           energyChartPath(series.Values, series.Present, minValue, maxValue, 52, 788),
			MobilePath:     energyChartPath(series.Values, series.Present, minValue, maxValue, 44, 388),
			AreaPath:       areaPath,
			MobileAreaPath: mobileAreaPath,
			Latest:         latestEnergyChartValue(series.Values, series.Present),
		})
		view.PointSet += countPresent(series.Present)
	}
	if len(view.Series) == 0 {
		a.cacheEnergyChart(cacheKey, view, now.Add(30*time.Second))
		return view
	}

	if today {
		view.XTicks = energyChartDayTicks(start, end, location)
	} else {
		view.XTicks = energyChartTimeTicks(start, end, location)
	}
	view.YTicks = energyChartValueTicks(limit)
	view.Samples = energyChartSamples(start, end, location, data, minValue, maxValue, minimumPoints)
	if threshold > 0 && threshold <= maxValue {
		position := energyChartValuePosition(threshold, minValue, maxValue)
		view.ThresholdPosition = formatEnergySVGNumber(position)
		view.ThresholdMobilePosition = view.ThresholdPosition
		view.HasThreshold = true
	}
	if today {
		view.Range = "Heute, 00:00 bis 24:00 · Messwerte bis " + historyEnd.In(location).Format("15:04") + " Uhr"
	} else {
		view.Range = start.In(location).Format("02.01. · 15:04") + " bis " + end.In(location).Format("02.01. · 15:04")
	}
	view.Summary, view.Detail = energyChartSummary(
		start, end, loadValues, loadPresent, pvValues, pvPresent, gridValues, gridPresent, batteryValues, batteryPresent, location,
	)
	if today && countPresent(loadPresent) == 0 {
		view.Summary = "Der heutige Tag ist als Verlauf sichtbar."
	}
	view.Status = "Viertelstunden-Ansicht · nur gelesen"
	view.HasData = true
	a.cacheEnergyChart(cacheKey, view, now.Add(5*time.Minute))
	return view
}

func (a *app) cachedEnergyChart(key string, now time.Time) (energyChartView, bool) {
	a.energyChartMu.Lock()
	defer a.energyChartMu.Unlock()
	if a.energyChartCache == nil {
		return energyChartView{}, false
	}
	a.pruneEnergyChartCacheLocked(now)
	item, ok := a.energyChartCache[key]
	if !ok {
		return energyChartView{}, false
	}
	return item.View, true
}

func (a *app) cacheEnergyChart(key string, view energyChartView, expiresAt time.Time) {
	a.energyChartMu.Lock()
	defer a.energyChartMu.Unlock()
	if a.energyChartCache == nil {
		a.energyChartCache = map[string]energyChartCacheEntry{}
	}
	a.pruneEnergyChartCacheLocked(time.Now())
	a.energyChartCache[key] = energyChartCacheEntry{View: view, ExpiresAt: expiresAt}
}

func (a *app) clearEnergyChartCache(tenantSlug string) {
	a.energyChartMu.Lock()
	defer a.energyChartMu.Unlock()
	prefix := strings.TrimSpace(tenantSlug) + "|"
	for key := range a.energyChartCache {
		if strings.HasPrefix(key, prefix) {
			delete(a.energyChartCache, key)
		}
	}
}

func (a *app) pruneEnergyChartCacheLocked(now time.Time) {
	for key, item := range a.energyChartCache {
		if !item.ExpiresAt.After(now) {
			delete(a.energyChartCache, key)
		}
	}
}

func energyHistoryBuckets(items []homeassistant.HistoryState, unit string, start, end, availableUntil time.Time, points int) ([]float64, []bool) {
	values := make([]float64, points)
	present := make([]bool, points)
	if points < 2 || len(items) == 0 || !end.After(start) {
		return values, present
	}
	type sample struct {
		at    time.Time
		value float64
	}
	samples := make([]sample, 0, len(items))
	for _, item := range items {
		value, err := homeassistant.ParseFloat(item.State)
		if err != nil {
			continue
		}
		at := item.LastChanged
		if at.IsZero() {
			at = item.LastUpdated
		}
		if at.IsZero() {
			continue
		}
		samples = append(samples, sample{at: at.UTC(), value: energyPowerKW(value, unit)})
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].at.Before(samples[j].at) })
	if len(samples) == 0 {
		return values, present
	}
	step := end.Sub(start) / time.Duration(points-1)
	current := 0.0
	hasCurrent := false
	index := 0
	for point := 0; point < points; point++ {
		target := start.Add(time.Duration(point) * step)
		if target.After(availableUntil) {
			continue
		}
		for index < len(samples) && !samples[index].at.After(target) {
			current = samples[index].value
			hasCurrent = true
			index++
		}
		if hasCurrent {
			values[point] = current
			present[point] = true
		}
	}
	return values, present
}

func energyPowerKW(value float64, unit string) float64 {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "w":
		return value / 1000
	case "mw":
		return value * 1000
	default:
		return value
	}
}

func combineEnergyHistory(primary []float64, primaryPresent []bool, secondary []float64, secondaryPresent []bool, secondaryFactor float64) ([]float64, []bool) {
	count := len(primary)
	if len(secondary) > count {
		count = len(secondary)
	}
	values := make([]float64, count)
	present := make([]bool, count)
	for index := 0; index < count; index++ {
		if index < len(primary) && index < len(primaryPresent) && primaryPresent[index] {
			values[index] += primary[index]
			present[index] = true
		}
		if index < len(secondary) && index < len(secondaryPresent) && secondaryPresent[index] {
			values[index] += secondary[index] * secondaryFactor
			present[index] = true
		}
	}
	return values, present
}

func countPresent(items []bool) int {
	count := 0
	for _, present := range items {
		if present {
			count++
		}
	}
	return count
}

func energyChartPath(values []float64, present []bool, minValue, maxValue, left, right float64) string {
	if len(values) < 2 || maxValue <= minValue {
		return ""
	}
	var path strings.Builder
	drawing := false
	presentCount := 0
	singleX, singleY := 0.0, 0.0
	for index, value := range values {
		if index >= len(present) || !present[index] {
			drawing = false
			continue
		}
		x := left + (right-left)*float64(index)/float64(len(values)-1)
		y := energyChartValuePosition(value, minValue, maxValue)
		presentCount++
		singleX, singleY = x, y
		command := "L"
		if !drawing {
			command = "M"
			drawing = true
		}
		fmt.Fprintf(&path, "%s%.1f %.1f", command, x, y)
	}
	if presentCount == 1 {
		return fmt.Sprintf("M%.1f %.1fL%.1f %.1f", singleX, singleY, singleX+0.1, singleY)
	}
	return path.String()
}

func energyChartAreaPath(values []float64, present []bool, minValue, maxValue, left, right, baseline float64) string {
	if len(values) < 2 || maxValue <= minValue {
		return ""
	}
	var path strings.Builder
	drawing := false
	lastX := 0.0
	closeSegment := func() {
		if !drawing {
			return
		}
		fmt.Fprintf(&path, "L%.1f %.1fZ", lastX, baseline)
		drawing = false
	}
	for index, value := range values {
		if index >= len(present) || !present[index] {
			closeSegment()
			continue
		}
		x := left + (right-left)*float64(index)/float64(len(values)-1)
		y := energyChartValuePosition(value, minValue, maxValue)
		if !drawing {
			fmt.Fprintf(&path, "M%.1f %.1fL%.1f %.1f", x, baseline, x, y)
			drawing = true
		} else {
			fmt.Fprintf(&path, "L%.1f %.1f", x, y)
		}
		lastX = x
	}
	closeSegment()
	return path.String()
}

func energyChartRoundedLimit(maxAbsolute float64) float64 {
	wholeKW := math.Ceil(math.Max(0, maxAbsolute))
	return math.Max(5, math.Ceil((wholeKW*1.25)/5)*5)
}

func energyChartValuePosition(value, minValue, maxValue float64) float64 {
	const (
		top    = 16.0
		bottom = 204.0
	)
	if maxValue <= minValue {
		return bottom
	}
	return bottom - (value-minValue)/(maxValue-minValue)*(bottom-top)
}

func latestEnergyChartValue(values []float64, present []bool) string {
	for index := len(values) - 1; index >= 0; index-- {
		if index < len(present) && present[index] {
			return formatEnergyValueUnit(formatEnergyCompact(values[index], 2), "kW")
		}
	}
	return ""
}

func energyChartTimeTicks(start, end time.Time, location *time.Location) []energyChartTickView {
	out := make([]energyChartTickView, 0, 5)
	for index := 0; index < 5; index++ {
		ratio := float64(index) / 4
		at := start.Add(time.Duration(ratio * float64(end.Sub(start)))).In(location)
		out = append(out, energyChartTickView{
			Position:       formatEnergySVGNumber(52 + ratio*(788-52)),
			MobilePosition: formatEnergySVGNumber(44 + ratio*(388-44)),
			Label:          at.Format("15:04"),
		})
	}
	return out
}

func energyChartDayTicks(start, end time.Time, location *time.Location) []energyChartTickView {
	if !end.After(start) {
		return nil
	}
	localStart := start.In(location)
	out := make([]energyChartTickView, 0, 5)
	for index, hour := range []int{0, 6, 12, 18, 24} {
		at := time.Date(localStart.Year(), localStart.Month(), localStart.Day(), hour, 0, 0, 0, location)
		ratio := float64(at.Sub(start)) / float64(end.Sub(start))
		label := at.Format("15:04")
		if index == 4 {
			label = "24:00"
		}
		out = append(out, energyChartTickView{
			Position:       formatEnergySVGNumber(52 + ratio*(788-52)),
			MobilePosition: formatEnergySVGNumber(44 + ratio*(388-44)),
			Label:          label,
		})
	}
	return out
}

func energyChartValueTicks(limit float64) []energyChartTickView {
	if limit <= 0 {
		return nil
	}
	step := 5.0
	if limit > 15 {
		step = math.Ceil((limit/3)/5) * 5
	}
	values := []float64{limit}
	for value := limit - step; value > 0; value -= step {
		values = append(values, value)
	}
	values = append(values, 0)
	for value := step; value < limit; value += step {
		values = append(values, -value)
	}
	values = append(values, -limit)
	out := make([]energyChartTickView, 0, len(values))
	for _, value := range values {
		y := energyChartValuePosition(value, -limit, limit)
		out = append(out, energyChartTickView{
			Position:       formatEnergySVGNumber(y),
			MobilePosition: formatEnergySVGNumber(y),
			Label:          formatEnergyValueUnit(formatEnergyCompact(value, 0), "kW"),
		})
	}
	return out
}

func energyChartSamples(start, end time.Time, location *time.Location, data []energyChartData, minValue, maxValue float64, minimumPoints int) []energyChartSampleView {
	pointCount := 0
	for _, series := range data {
		if len(series.Values) > pointCount {
			pointCount = len(series.Values)
		}
	}
	if pointCount < 2 || maxValue <= minValue {
		return nil
	}
	step := end.Sub(start) / time.Duration(pointCount-1)
	desktopStep := (788.0 - 52.0) / float64(pointCount-1)
	mobileStep := (388.0 - 44.0) / float64(pointCount-1)
	out := make([]energyChartSampleView, 0, pointCount)
	for index := 0; index < pointCount; index++ {
		desktopX := 52.0 + float64(index)*desktopStep
		mobileX := 44.0 + float64(index)*mobileStep
		desktopLeft := desktopX - desktopStep/2
		mobileLeft := mobileX - mobileStep/2
		desktopWidth := desktopStep
		mobileWidth := mobileStep
		if index == 0 {
			desktopLeft = 52
			mobileLeft = 44
			desktopWidth = desktopStep / 2
			mobileWidth = mobileStep / 2
		} else if index == pointCount-1 {
			desktopWidth = desktopStep / 2
			mobileWidth = mobileStep / 2
		}
		sample := energyChartSampleView{
			Index:          index,
			Time:           start.Add(time.Duration(index) * step).In(location).Format("15:04"),
			Position:       formatEnergySVGNumber(desktopX),
			MobilePosition: formatEnergySVGNumber(mobileX),
			HitPosition:    formatEnergySVGNumber(desktopLeft),
			HitWidth:       formatEnergySVGNumber(desktopWidth),
			MobileHit:      formatEnergySVGNumber(mobileLeft),
			MobileHitWidth: formatEnergySVGNumber(mobileWidth),
		}
		for _, series := range data {
			if countPresent(series.Present) < minimumPoints || index >= len(series.Values) || index >= len(series.Present) || !series.Present[index] {
				continue
			}
			value := series.Values[index]
			label := series.Label
			displayValue := value
			switch series.Key {
			case "grid":
				if value < 0 {
					label = "Einspeisung"
					displayValue = math.Abs(value)
				} else {
					label = "Netzbezug"
				}
			case "battery":
				if value < 0 {
					label = "Speicher lädt"
					displayValue = math.Abs(value)
				} else {
					label = "Speicher entlädt"
				}
			}
			sample.Values = append(sample.Values, energyChartSampleValueView{
				Key:            series.Key,
				Label:          label,
				Value:          formatEnergyValueUnit(formatEnergyCompact(displayValue, 2), "kW"),
				Position:       formatEnergySVGNumber(energyChartValuePosition(value, minValue, maxValue)),
				MobilePosition: formatEnergySVGNumber(energyChartValuePosition(value, minValue, maxValue)),
			})
		}
		if len(sample.Values) > 0 {
			out = append(out, sample)
		}
	}
	return out
}

func formatEnergySVGNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64)
}

func energyChartSummary(start, end time.Time, load []float64, loadPresent []bool, pv []float64, pvPresent []bool, grid []float64, gridPresent []bool, battery []float64, batteryPresent []bool, location *time.Location) (string, string) {
	peakIndex := -1
	peak := 0.0
	for index, value := range load {
		if index < len(loadPresent) && loadPresent[index] && (peakIndex < 0 || value > peak) {
			peakIndex = index
			peak = value
		}
	}
	if peakIndex < 0 {
		return "Die letzten 24 Stunden sind als Verlauf sichtbar.", "Noch fehlt ein durchgängiger Hausverbrauch für eine belastbare Einordnung."
	}
	step := 15 * time.Minute
	if len(load) > 1 && end.After(start) {
		step = end.Sub(start) / time.Duration(len(load)-1)
	}
	at := start.Add(time.Duration(peakIndex) * step).In(location)
	summary := "Die höchste Last lag um " + at.Format("15:04") + " Uhr bei " + formatEnergyValueUnit(formatEnergyCompact(peak, 2), "kW") + "."
	valueAt := func(values []float64, present []bool) float64 {
		if peakIndex < len(values) && peakIndex < len(present) && present[peakIndex] {
			return values[peakIndex]
		}
		return 0
	}
	pvAt := math.Max(0, valueAt(pv, pvPresent))
	gridAt := math.Max(0, valueAt(grid, gridPresent))
	batteryAt := math.Max(0, valueAt(battery, batteryPresent))
	switch {
	case pvAt+batteryAt >= peak*0.6:
		return summary, "PV und Speicher deckten zu diesem Zeitpunkt den größten Teil."
	case pvAt >= peak*0.5:
		return summary, "Die PV deckte zu diesem Zeitpunkt einen großen Teil."
	case gridAt >= peak*0.5:
		return summary, "Der größte Anteil kam zu diesem Zeitpunkt aus dem Netz."
	default:
		return summary, "Die Herkunft verteilt sich auf mehrere gemessene Quellen."
	}
}

func buildEnergyCoverageViews(assets []energy.Asset, mappings []energy.EntityMapping) ([]energyCoverageView, string) {
	confirmedMetrics := map[string]bool{}
	confirmedAssets := map[string]bool{}
	for _, mapping := range mappings {
		if !mapping.Confirmed {
			continue
		}
		confirmedMetrics[mapping.Metric] = true
		if mapping.AssetID != "" {
			confirmedAssets[mapping.AssetID] = true
		}
	}
	houseMeasured := confirmedMetrics[energy.MetricGridImportPower] || confirmedMetrics[energy.MetricGridImportEnergy] ||
		confirmedMetrics[energy.MetricLoadPower]
	houseStatus := "Noch offen"
	houseDetail := "Noch kein hausweiter Referenzwert bestätigt."
	houseTone := "open"
	if houseMeasured {
		houseStatus = "Gemessen"
		houseDetail = "Hausweiter Bezug oder Verbrauch ist bestätigt."
		houseTone = "good"
	}
	out := []energyCoverageView{{
		Label:  "Hausanschluss",
		Status: houseStatus,
		Detail: houseDetail,
		Tone:   houseTone,
	}}
	measured := 0
	if houseMeasured {
		measured++
	}
	for _, asset := range assets {
		if asset.Kind == energyFlowHomeKind || asset.Kind == energyFlowGridKind || asset.Kind == energyFlowParkingKind {
			continue
		}
		hasMeasurement := confirmedAssets[asset.ID]
		if hasMeasurement {
			measured++
		}
		status := "Nur erfasst"
		detail := "Noch kein separater Messwert bestätigt."
		tone := "open"
		if hasMeasurement {
			status = "Gemessen"
			detail = "Mindestens ein Messwert ist dieser Anlage zugeordnet."
			tone = "good"
		}
		out = append(out, energyCoverageView{
			Label:  asset.Name,
			Status: status,
			Detail: detail,
			Tone:   tone,
		})
	}
	return out, fmt.Sprintf("%d von %d Bereichen gemessen", measured, len(out))
}

func energyRoadmap(profile energy.HomeProfile, assetCount, mappingCount int) []energyRoadmapStep {
	steps := []energyRoadmapStep{
		{Number: 1, Title: "Verstehen", Detail: "Verbraucher und Möglichkeiten erfassen", State: "Erledigt"},
		{Number: 2, Title: "Messen", Detail: "Bestehende Messwerte sicher lesen", State: "Offen", Action: "Messwerte prüfen", URL: "/app/zuhause/onboarding?step=4"},
		{Number: 3, Title: "Planen", Detail: "15-Minuten-Spitze und Ziel festlegen", State: "Danach"},
		{Number: 4, Title: "Optimieren", Detail: "Erst simulieren, später bewusst freigeben", State: "Später"},
	}
	if assetCount == 0 {
		steps[0].State = "Jetzt"
		steps[0].Current = true
		steps[0].Action = "Verbraucher erfassen"
		steps[0].URL = "/app/zuhause/onboarding?step=3"
		return steps
	}
	if mappingCount == 0 {
		steps[1].State = "Jetzt"
		steps[1].Current = true
		return steps
	}
	steps[1].State = "Erledigt"
	steps[2].State = "Jetzt"
	steps[2].Current = true
	if profile.TargetPeakKW == nil {
		steps[2].Action = "Ziel vorbereiten"
		steps[2].URL = "/app/energie#fahrplan"
	}
	return steps
}
