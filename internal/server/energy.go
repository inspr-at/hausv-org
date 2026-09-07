package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/mail"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/homeconnector"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/web"
)

const (
	energyConsumerDefaultStaleAfter = 10 * time.Minute
	energyConsumerMaxStaleMinutes   = 24 * 60
)

type energyCandidateView struct {
	EntityID    string
	DisplayName string
	SourceName  string
	Metric      string
	MetricLabel string
	Value       string
	Unit        string
	Checked     bool
}

type energyDiscoveryView struct {
	Recommended []energyCandidateView
	Additional  []energyCandidateView
}

type energyAssetOption struct {
	Kind    string
	Label   string
	Checked bool
}

type energyMetricView struct {
	AssetID     string
	Metric      string
	Kind        string
	Label       string
	Value       string
	Detail      string
	Tone        string
	Direction   string
	Numeric     float64
	Unit        string
	SourceState string
	LastUpdated time.Time
}

type energyConsumerMeasurementOption struct {
	EntityID          string `json:"entityId"`
	Name              string `json:"name"`
	Unit              string `json:"unit,omitempty"`
	Kind              string `json:"kind"`
	AssignedAssetID   string `json:"assignedAssetId,omitempty"`
	AssignedAssetName string `json:"assignedAssetName,omitempty"`
}

type energyConsumerMeasurementsResponse struct {
	Status   string                            `json:"status"`
	Message  string                            `json:"message"`
	Entities []energyConsumerMeasurementOption `json:"entities"`
}

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

type energyMappingSlotView struct {
	Key      string
	Label    string
	Purpose  string
	Status   string
	Detail   string
	Tone     string
	Required bool
	Derived  bool
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

type energyImportView struct {
	Filename string
	Date     string
	Format   string
}

type energyPeakView struct {
	Source string
	Value  string
}

type energyComparisonView struct {
	Tone    string
	Title   string
	Details string
}

type energyTariffView struct {
	ID          string
	Version     string
	Status      string
	SourceTitle string
	SourceURL   string
	Rule        string
	HasEstimate bool
	Disclaimer  string
	// Verrechnete Leistung und ihre Aufteilung an der Staffel. Getrennt
	// ausgewiesen, damit sichtbar wird, wo Kappen doppelt so viel bringt.
	PeakKW   string
	BilledKW string
	// PeakMeterPercent visualisiert die gemessene Spitze relativ zur
	// verrechneten Leistung. Der Wert ist serverseitig auf 0–100 begrenzt,
	// damit die ruhige Vergleichsgrafik nie eine andere Aussage als die Zahlen
	// daneben trifft.
	PeakMeterPercent int
	BelowKW          string
	AboveKW          string
	HasTier          bool
	MinimumReason    string
	AgreedKW         string
	HasAgreed        bool
	AgreedHint       string
	TierHint         string
	// MonthLabel und Basis benennen, woraus die Spitze stammt. Eine Kennzahl
	// ohne ihre Grundlage ist auf diesem Bildschirm wertlos: der Leistungstarif
	// bemisst je Kalendermonat, und ein halber Monat sieht aus wie ein ganzer.
	MonthLabel string
	// CoverageLabel is the compact visible statement. Basis keeps the full
	// month, source and quality explanation for progressive disclosure.
	CoverageLabel string
	Basis         string
	// PeakTime nennt den Zeitpunkt der teuersten Viertelstunde. Ohne ihn bleibt
	// die Spitze eine Zahl, mit ihm wird sie ein Ereignis, das man wiedererkennt.
	PeakTime    string
	HasPeakTime bool
	// AnnualPowerEUR ist ausschließlich der Leistungsanteil des Netztarifs.
	// Getrennt von einem Label geführt, weil genau diese Zahl unter einer
	// Überschrift über den Tarif 2027 als ganze Jahresrechnung missverstanden
	// wird — und ein Haushalt, dem eine Ersparnis versprochen wird, die nie
	// eintritt, ist dauerhaft verloren.
	AnnualPowerEUR string
	// Die Sätze der Modellrechnung kommen aus dem Regelprofil und nicht aus der
	// Vorlage: sonst behauptet die Oberfläche weiter 33,82 €, wenn im Profil
	// längst ein anderer Satz steht.
	BelowRateEUR string
	AboveRateEUR string
	ThresholdKW  string
	// MissingIsWaiting unterscheidet Warten von Handeln: liegt der Netzbezug
	// zugeordnet vor, kommt die Zahl von selbst.
	MissingIsWaiting bool
	// MissingReason erklärt im Leerzustand, warum keine Spitze dasteht. Eine
	// leere Kachel oder eine 0 wäre beides falsch.
	MissingReason string
}

type energyScenarioView struct {
	Title       string
	PeakBand    string
	EffectBand  string
	Uncertainty string
	Assumptions string
	// BilledBand übersetzt das Spitzenband in die verrechnete Leistung. Nur sie
	// steht auf der Rechnung: unterhalb der Mindestbemessung senkt eine
	// niedrigere Spitze nichts mehr, und genau dort entstünde sonst ein
	// Einsparversprechen, das nie eintritt.
	BilledBand    string
	HasBilledBand bool
	// BaselineNote nennt den Ausgangswert. Ein Band ohne seinen Ausgangspunkt
	// lädt dazu ein, die Differenz zu irgendeiner Zahl im Kopf zu bilden.
	BaselineNote string
	// FloorNote warnt, wenn die optimistische Seite des Bandes unter die
	// verrechenbare Untergrenze fällt. Die Spitze sinkt dort real weiter, der
	// verrechnete Betrag aber nicht — ohne den Hinweis liest sich das Szenario
	// als Ersparnis, die so nicht eintritt.
	FloorNote string
}

type energyCaretakerView struct {
	Email        string
	Name         string
	CanView      bool
	CanConfigure bool
	IsSelf       bool
	Editable     bool
}

type energyOption struct {
	Value string
	Label string
}

type energyHomeUnitOption struct {
	Value    string
	Label    string
	Selected bool
}

type energyMaintenanceView struct {
	ID             string
	AssetID        string
	AssetName      string
	Title          string
	IntervalMonths int
	NextDueValue   string
	DueLabel       string
	Tone           string
	LastCompleted  string
	ContactID      string
	ContactName    string
	DocumentID     string
	DocumentTitle  string
	IssueID        string
	IssueTitle     string
	EvidenceNote   string
	Active         bool
}

type energyTariffAssessmentView struct {
	ID            string
	Month         string
	Profile       string
	Peak          string
	Annual        string
	Quality       string
	Created       string
	SourceURL     string
	ProfileStatus string
}

type energyMeasureView struct {
	ID               string
	IssueID          string
	Title            string
	Status           string
	StatusLabel      string
	ContactID        string
	ContactName      string
	OfferNote        string
	AppointmentValue string
	Appointment      string
	WorkNote         string
	EvidenceNote     string
	BeforeFrom       string
	BeforeTo         string
	AfterFrom        string
	AfterTo          string
	BeforePeak       string
	AfterPeak        string
	BeforeQuality    string
	AfterQuality     string
	HasComparison    bool
	Completed        string
}

// energyFor hands back the energy store bound to the identity this request was
// authorized with.
//
// It exists so the tenant a query filters on comes from the authorization
// check, not from a slug re-resolved inside the storage layer on every call.
// The slug arguments below stay: the Storage API is shared with the in-memory
// implementation, whose keys ARE slugs, and the SQL store refuses outright if
// the two disagree.
func (a *app) energyFor(ac authCtx) energy.Storage {
	if a == nil || a.energyStore == nil {
		return nil
	}
	return a.energyStore.ForTenant(ac.tenantRef)
}

func (a *app) canViewEnergy(ac authCtx) bool {
	_, allowed := a.energyStoreForHome(ac, energy.DefaultHomeKey)
	return allowed
}

// energyStoreForHome is the authorization boundary for future home selectors
// and background handlers. Returning the already-scoped store makes it hard to
// authorize one home and accidentally query another one afterwards.
func (a *app) energyStoreForHome(ac authCtx, homeKey string) (energy.Storage, bool) {
	if !roleCanUseResidentAreas(ac.role) {
		return nil, false
	}
	// The tenant reference travels with the handle from here on. This is the
	// point where the caller's right to this house was just established, so it
	// is the point where the identity every query filters on should be fixed —
	// a slug that arrives later cannot re-point what this store answers for.
	store := a.energyFor(ac).ForHome(homeKey)
	if a.isEnergyHouseAdmin(ac) {
		return store, true
	}
	profile, exists, err := store.Profile(ac.tenant.Slug)
	if err != nil {
		return nil, false
	}
	if !exists || energyProfileUnclaimed(profile) {
		// The durable directory role may still be "resident" while the official
		// unit relation identifies this person as its owner. Keep that owner
		// relation authoritative for first-time and post-deletion onboarding.
		return store, a.ownerCanAccessEnergyProfile(ac, profile, exists)
	}
	if energy.NormalizeHomeKey(homeKey) != energy.DefaultHomeKey && profile.HomeType == energy.HomeApartment {
		linked, ok := a.effectiveEnergyUnit(ac.tenantRef, profile)
		if !ok {
			return nil, false
		}
		if !a.actorBelongsToEnergyUnit(ac, linked.ID, false) {
			return nil, false
		}
		return store, true
	}
	person := a.profileForTenant(ac.email, ac.tenant.Slug)
	if person.HasPermission(permissionEnergyView) ||
		person.HasPermission(permissionEnergyConfigure) ||
		person.HasPermission(permissionEnergyControl) ||
		person.HasPermission(permissionEnergyCaretaker) {
		return store, true
	}
	if profile.HomeType != energy.HomeApartment {
		return store, ac.role == roleOwner || ac.role == roleRenter || ac.role == roleResident || ac.role == roleBeirat
	}
	if linked, ok := a.effectiveEnergyUnit(ac.tenantRef, profile); ok {
		return store, a.actorBelongsToEnergyUnit(ac, linked.ID, false)
	}
	if len(a.energyResidentialUnits(ac.tenantRef)) == 0 && strings.TrimSpace(profile.UnitID) == "" {
		return store, ac.role == roleOwner || ac.role == roleRenter || ac.role == roleResident || ac.role == roleBeirat
	}
	return nil, false
}

func (a *app) canManageEnergy(ac authCtx) bool {
	if a.isEnergyHouseAdmin(ac) {
		return true
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	if err != nil {
		return false
	}
	person := a.profileForTenant(ac.email, ac.tenant.Slug)
	if exists && !energyProfileUnclaimed(profile) &&
		(person.HasPermission(permissionEnergyConfigure) || person.HasPermission(permissionEnergyCaretaker)) {
		return true
	}
	return a.ownerCanAccessEnergyProfile(ac, profile, exists)
}

func (a *app) canManageHomeIdentity(ac authCtx) bool {
	// A delegated technical caretaker may configure readings, but the shared
	// name and type of the home profile stay with owners and house management.
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	return err == nil && a.canManageHomeIdentityProfile(ac, profile, exists)
}

func (a *app) canControlEnergy(ac authCtx) bool {
	// The house-wide mode is a property decision, not a technical support
	// permission. Legacy energy-control grants intentionally do not widen it.
	if a.isEnergyHouseAdmin(ac) {
		return true
	}
	profile, exists, err := a.energyFor(ac).Profile(ac.tenant.Slug)
	return err == nil && a.ownerCanAccessEnergyProfile(ac, profile, exists)
}

func (a *app) isEnergyHouseAdmin(ac authCtx) bool {
	return ac.can(capabilityManageBuilding)
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
			http.Error(w, "Die Zuhause-Art ist an die offizielle Wohnung gebunden.", http.StatusForbidden)
			return
		}
		unitID, validUnit := a.resolveHomeIdentityUnitID(ac, profile, homeType, r.FormValue("unit_id"))
		if !validUnit {
			http.Error(w, "Bitte eine eigene offizielle Wohnung auswählen.", http.StatusBadRequest)
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
		http.Error(w, "Dieser Bereich ist Eigentümern der zugeordneten Wohnung und der Hausverwaltung vorbehalten.", http.StatusForbidden)
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
		http.Error(w, "Dieser Bereich ist Eigentümern der zugeordneten Wohnung und der Hausverwaltung vorbehalten.", http.StatusForbidden)
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
		http.Error(w, "Die Zuhause-Art ist an die offizielle Wohnung gebunden.", http.StatusForbidden)
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
		return "Ein einzelner Haushalt in einem Mehrparteienhaus. Der Überblick konzentriert sich auf die zugeordnete Wohnung und ihre eigenen Geräte."
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
	return "Offizielle Einheit", "Noch keine Wohnung angelegt"
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
	if updated, changed := a.ensureNamedEVMeasurementMappings(r.Context(), ac.tenant, assets, mappings); changed {
		mappings = updated
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
	if updated, changed := a.ensureNamedEVMeasurementMappings(r.Context(), ac.tenant, assets, mappings); changed {
		mappings = updated
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

func energyComparisonForView(intervals []energy.Interval, at time.Time) (energyComparisonView, bool) {
	comparison, ok := energy.CompareMonthlyPeaks(intervals, at, time.Local, "smart-meter", "home-assistant")
	if !ok {
		return energyComparisonView{}, false
	}
	if comparison.Status == "check" {
		return energyComparisonView{
			Tone:  "warning",
			Title: "Messquellen weichen sichtbar ab",
			Details: "Home Assistant liegt bei der Monatsspitze um " +
				formatEnergyValueUnit(formatEnergyNumber(comparison.DeltaPercent), "%") + " neben der Smart-Meter-Referenz. Zähler, Einheit und Vorzeichen prüfen.",
		}, true
	}
	return energyComparisonView{
		Tone:    "good",
		Title:   "Messquellen sind plausibel",
		Details: "Die Monatsspitzen liegen innerhalb von 10 % beieinander.",
	}, true
}

func energyTargetValue(value *float64) string {
	if value == nil {
		return ""
	}
	return formatEnergyNumber(*value)
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

func (a *app) energyContactOptions(contacts contactBookRepository) ([]energyOption, map[string]string) {
	options := []energyOption{}
	names := map[string]string{}
	if contacts == nil {
		return options, names
	}
	for _, item := range contacts.List(false) {
		label := managedContactDisplayName(item)
		if item.Company != "" && !strings.EqualFold(item.Company, label) {
			label += " · " + item.Company
		}
		if item.ServiceRegion != "" {
			label += " · " + item.ServiceRegion
		}
		names[item.ID] = label
		options = append(options, energyOption{Value: item.ID, Label: label})
	}
	sort.Slice(options, func(i, j int) bool { return strings.ToLower(options[i].Label) < strings.ToLower(options[j].Label) })
	return options, names
}

func (a *app) energyContact(contacts contactBookRepository, id string) (managedContact, bool) {
	if contacts == nil || strings.TrimSpace(id) == "" {
		return managedContact{}, false
	}
	for _, item := range contacts.List(false) {
		if item.ID == id {
			return item, true
		}
	}
	return managedContact{}, false
}

func (a *app) energyDocumentOptions(tenant store.TenantRef) ([]energyOption, map[string]string) {
	options := []energyOption{}
	names := map[string]string{}
	documents, ok := store.BindDocumentRepository(a.documentStore, tenant)
	if !ok {
		return options, names
	}
	for _, item := range documents.ListCurrent() {
		names[item.ID] = item.Title
		options = append(options, energyOption{Value: item.ID, Label: item.Title})
	}
	sort.Slice(options, func(i, j int) bool { return strings.ToLower(options[i].Label) < strings.ToLower(options[j].Label) })
	return options, names
}

func (a *app) energyIssueOptions(tenant store.TenantRef) ([]energyOption, map[string]string) {
	options := []energyOption{}
	names := map[string]string{}
	issues, ok := store.BindIssueRepository(a.issueStore, tenant)
	if !ok {
		return options, names
	}
	for _, item := range issues.List() {
		names[item.ID] = item.Title
		options = append(options, energyOption{Value: item.ID, Label: item.Title})
	}
	sort.Slice(options, func(i, j int) bool { return strings.ToLower(options[i].Label) < strings.ToLower(options[j].Label) })
	return options, names
}

func buildEnergyMaintenanceViews(plans []energy.MaintenancePlan, assets []energy.Asset, contacts, documents, issues map[string]string, now time.Time) []energyMaintenanceView {
	assetNames := map[string]string{}
	for _, asset := range assets {
		assetNames[asset.ID] = asset.Name
	}
	out := make([]energyMaintenanceView, 0, len(plans))
	for _, plan := range plans {
		days := int(plan.NextDueAt.Sub(now).Hours() / 24)
		dueLabel := "Fällig am " + plan.NextDueAt.In(time.Local).Format("02.01.2006")
		tone := ""
		if days < 0 {
			dueLabel = "Überfällig seit " + plan.NextDueAt.In(time.Local).Format("02.01.2006")
			tone = "danger"
		} else if days <= 30 {
			tone = "warning"
		}
		lastCompleted := ""
		if plan.LastCompletedAt != nil {
			lastCompleted = plan.LastCompletedAt.In(time.Local).Format("02.01.2006")
		}
		out = append(out, energyMaintenanceView{
			ID: plan.ID, AssetID: plan.AssetID, AssetName: firstNonEmpty(assetNames[plan.AssetID], "Anlage"),
			Title: plan.Title, IntervalMonths: plan.IntervalMonths, NextDueValue: plan.NextDueAt.In(time.Local).Format("2006-01-02"),
			DueLabel: dueLabel, Tone: tone, LastCompleted: lastCompleted, ContactID: plan.ContactID,
			ContactName: contacts[plan.ContactID], DocumentID: plan.DocumentID, DocumentTitle: documents[plan.DocumentID],
			IssueID: plan.IssueID, IssueTitle: issues[plan.IssueID], EvidenceNote: plan.EvidenceNote, Active: plan.Active,
		})
	}
	return out
}

func buildEnergyTariffAssessmentViews(items []energy.TariffAssessment) []energyTariffAssessmentView {
	out := make([]energyTariffAssessmentView, 0, len(items))
	for _, item := range items {
		month := item.AssessmentMonth
		if parsed, err := time.Parse("2006-01", item.AssessmentMonth); err == nil {
			month = parsed.Format("01/2006")
		}
		out = append(out, energyTariffAssessmentView{
			ID: item.ID, Month: month, Profile: item.ProfileID + " · " + item.ProfileVersion,
			Peak: formatEnergyValueUnit(formatEnergyNumber(item.PeakKW), "kW"), Annual: formatEnergyValueUnit(formatEnergyNumber(item.AnnualPowerEUR), "€") + " Modellwert/Jahr",
			Quality: energyQualityLabel(item.DataQuality), Created: item.CreatedAt.In(time.Local).Format("02.01.2006 15:04"),
			SourceURL: item.SourceURL, ProfileStatus: item.ProfileStatus,
		})
	}
	return out
}

func buildEnergyMeasureViews(items []energy.Measure, contacts map[string]string) []energyMeasureView {
	out := make([]energyMeasureView, 0, len(items))
	for _, item := range items {
		view := energyMeasureView{
			ID: item.ID, IssueID: item.IssueID, Title: item.Title, Status: item.Status,
			StatusLabel: energyMeasureStatusLabel(item.Status), ContactID: item.ContactID, ContactName: contacts[item.ContactID],
			OfferNote: item.OfferNote, WorkNote: item.WorkNote, EvidenceNote: item.EvidenceNote,
			BeforeQuality: energyQualityLabel(item.BeforeQuality), AfterQuality: energyQualityLabel(item.AfterQuality),
		}
		if item.AppointmentAt != nil {
			view.AppointmentValue = item.AppointmentAt.In(time.Local).Format("2006-01-02T15:04")
			view.Appointment = item.AppointmentAt.In(time.Local).Format("02.01.2006 15:04")
		}
		if item.CompletedAt != nil {
			view.Completed = item.CompletedAt.In(time.Local).Format("02.01.2006")
		}
		view.BeforeFrom = energyDateValue(item.BeforeFrom)
		view.BeforeTo = energyDateValue(item.BeforeTo)
		view.AfterFrom = energyDateValue(item.AfterFrom)
		view.AfterTo = energyDateValue(item.AfterTo)
		if item.BeforePeakKW != nil && item.AfterPeakKW != nil {
			view.BeforePeak = formatEnergyValueUnit(formatEnergyNumber(*item.BeforePeakKW), "kW")
			view.AfterPeak = formatEnergyValueUnit(formatEnergyNumber(*item.AfterPeakKW), "kW")
			view.HasComparison = true
		}
		out = append(out, view)
	}
	return out
}

func energyMeasureStatusLabel(status string) string {
	switch status {
	case energy.MeasureRequested:
		return "Anfrage vorbereitet"
	case energy.MeasureAssigned:
		return "Kontakt ausgewählt"
	case energy.MeasureScheduled:
		return "Termin vereinbart"
	case energy.MeasureCompleted:
		return "Abgeschlossen"
	case energy.MeasureCancelled:
		return "Nicht weiterverfolgt"
	default:
		return "Entwurf"
	}
}

func energyQualityLabel(quality string) string {
	switch quality {
	case energy.QualityMeasured:
		return "gemessen"
	case energy.QualityEstimated:
		return "teilweise geschätzt"
	case energy.QualityGap:
		return "mit Messlücken"
	case energy.QualityConflict:
		return "widersprüchlich"
	default:
		return "keine ausreichenden Daten"
	}
}

func energyDateValue(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.In(time.Local).Format("2006-01-02")
}

func parseEnergyLocalDate(raw string) (time.Time, error) {
	value, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(raw), time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return value.UTC(), nil
}

func parseEnergyLocalDateTime(raw string) (time.Time, error) {
	value, err := time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(raw), time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return value.UTC(), nil
}

func energyAssetBelongsToTenant(storage energy.Storage, tenantSlug, assetID string) bool {
	items, err := storage.ListAssets(tenantSlug)
	if err != nil {
		return false
	}
	for _, item := range items {
		if item.ID == assetID {
			return true
		}
	}
	return false
}

func findMaintenancePlan(storage energy.Storage, tenantSlug, id string) (energy.MaintenancePlan, bool) {
	items, err := storage.ListMaintenance(tenantSlug)
	if err != nil {
		return energy.MaintenancePlan{}, false
	}
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return energy.MaintenancePlan{}, false
}

func findMaintenancePlanByAsset(storage energy.Storage, tenantSlug, assetID string) (energy.MaintenancePlan, bool) {
	items, err := storage.ListMaintenance(tenantSlug)
	if err != nil {
		return energy.MaintenancePlan{}, false
	}
	for _, item := range items {
		if item.AssetID == assetID {
			return item, true
		}
	}
	return energy.MaintenancePlan{}, false
}

func (a *app) validEnergyReferences(tenant store.TenantRef, contacts contactBookRepository, contactID, documentID, issueID string) bool {
	if contactID != "" {
		if _, ok := a.energyContact(contacts, contactID); !ok {
			return false
		}
	}
	if documentID != "" {
		documents, ok := store.BindDocumentRepository(a.documentStore, tenant)
		if !ok {
			return false
		}
		if _, ok := documents.Get(documentID); !ok {
			return false
		}
	}
	if issueID != "" {
		issues, ok := store.BindIssueRepository(a.issueStore, tenant)
		if !ok {
			return false
		}
		if _, ok := issues.Get(issueID); !ok {
			return false
		}
	}
	return true
}

func (a *app) completeEnergyMeasureRanges(item *energy.Measure, r *http.Request) error {
	beforeFrom, err := parseEnergyLocalDate(r.FormValue("before_from"))
	if err != nil {
		return err
	}
	beforeTo, err := parseEnergyLocalDate(r.FormValue("before_to"))
	if err != nil {
		return err
	}
	afterFrom, err := parseEnergyLocalDate(r.FormValue("after_from"))
	if err != nil {
		return err
	}
	afterTo, err := parseEnergyLocalDate(r.FormValue("after_to"))
	if err != nil {
		return err
	}
	if beforeTo.Before(beforeFrom) || afterTo.Before(afterFrom) || !beforeTo.Before(afterFrom) ||
		beforeTo.Sub(beforeFrom) > 180*24*time.Hour || afterTo.Sub(afterFrom) > 180*24*time.Hour {
		return fmt.Errorf("invalid comparison ranges")
	}
	beforeIntervals, err := a.energyStore.ListIntervals(item.TenantSlug, beforeFrom, beforeTo.AddDate(0, 0, 1))
	if err != nil {
		return err
	}
	afterIntervals, err := a.energyStore.ListIntervals(item.TenantSlug, afterFrom, afterTo.AddDate(0, 0, 1))
	if err != nil {
		return err
	}
	item.BeforeFrom, item.BeforeTo = &beforeFrom, &beforeTo
	item.AfterFrom, item.AfterTo = &afterFrom, &afterTo
	item.BeforePeakKW, item.BeforeQuality = energy.PeakForRange(beforeIntervals)
	item.AfterPeakKW, item.AfterQuality = energy.PeakForRange(afterIntervals)
	if item.BeforePeakKW == nil || item.AfterPeakKW == nil ||
		strings.TrimSpace(item.WorkNote) == "" || strings.TrimSpace(item.EvidenceNote) == "" {
		return fmt.Errorf("comparison evidence incomplete")
	}
	return nil
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

func (a *app) updateEnergyMappings(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	if err := a.saveSelectedEnergyMappings(r.Context(), ac.tenant, r.Form["entities"]); err != nil {
		http.Error(w, "Messwerte konnten nicht aktualisiert werden.", http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/app/energie", http.StatusSeeOther)
}

func (a *app) updateEnergyAssets(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	if err := a.saveOnboardingAssets(ac.tenant.Slug, r.Form["assets"]); err != nil {
		http.Error(w, "Geräte konnten nicht aktualisiert werden.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/app/energie", http.StatusSeeOther)
}

func (a *app) importSmartMeter(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 17<<20)
	if err := r.ParseMultipartForm(17 << 20); err != nil {
		http.Redirect(w, r, "/app/energie?import=invalid", http.StatusSeeOther)
		return
	}
	file, header, err := r.FormFile("smart_meter_file")
	if err != nil {
		http.Redirect(w, r, "/app/energie?import=invalid", http.StatusSeeOther)
		return
	}
	defer file.Close()
	location, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		location = time.Local
	}
	record, intervals, err := energy.ParseSmartMeterCSV(io.LimitReader(file, 16<<20), ac.tenant.Slug, header.Filename, location)
	if err != nil {
		http.Redirect(w, r, "/app/energie?import=invalid", http.StatusSeeOther)
		return
	}
	inserted, err := a.energyFor(ac).PutImport(record, intervals)
	if err != nil {
		http.Error(w, "Smart-Meter-Daten konnten nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	status := "duplicate"
	if inserted {
		status = "added"
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.smart-meter.import",
		TargetType: "energy-import",
		TargetID:   record.ID,
		Summary:    "Smart-Meter-Referenz importiert",
		Details: map[string]string{
			"format":    record.Format,
			"digest":    shortImportDigest(record.SHA256),
			"intervals": strconv.Itoa(len(intervals)),
			"result":    status,
		},
	})
	http.Redirect(w, r, "/app/energie?import="+status, http.StatusSeeOther)
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

// updateEnergyAgreedPower speichert die mit dem Netzbetreiber vereinbarte
// Anschlussleistung. Sie ist keine Zielgröße, sondern eine Vertragstatsache:
// ab 2027 bemisst der Entwurf mindestens 20 % davon, auch in einem Monat ohne
// jede Spitze.
func (a *app) updateEnergyAgreedPower(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
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
	before := ""
	if profile.AgreedPowerKW != nil {
		before = formatEnergyNumber(*profile.AgreedPowerKW)
	}
	raw := strings.TrimSpace(r.FormValue("agreed_power_kw"))
	if raw == "" {
		// Leeren heißt "nicht erfasst" — die Mindestbemessung ruht dann wieder,
		// statt gegen einen alten Wert weiterzurechnen.
		profile.AgreedPowerKW = nil
	} else {
		value, parseErr := homeassistant.ParseFloat(raw)
		if parseErr != nil || !energy.ValidPowerKW(value, 1000) {
			http.Error(w, "Vereinbarte Anschlussleistung muss eine positive kW-Zahl sein.", http.StatusBadRequest)
			return
		}
		profile.AgreedPowerKW = &value
	}
	if err := a.energyFor(ac).SaveProfile(profile); err != nil {
		http.Error(w, "Anschlussleistung konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	after := ""
	if profile.AgreedPowerKW != nil {
		after = formatEnergyNumber(*profile.AgreedPowerKW)
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.agreed_power.change",
		TargetType: "home-profile",
		TargetID:   ac.tenant.Slug,
		Summary:    "Vereinbarte Anschlussleistung geändert",
		Details: map[string]string{
			"agreed_from_kw": before,
			"agreed_to_kw":   after,
		},
	})
	http.Redirect(w, r, "/app/energie?agreed=1#tarif", http.StatusSeeOther)
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

func (a *app) createEnergyMeasure(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canViewEnergy(ac) || !canCreateResidentIssue(ac.actor(), ac.resource()) {
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
	assets, _ := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	mappings, _ := a.energyFor(ac).ListMappings(ac.tenant.Slug)
	intervals, _ := a.energyFor(ac).ListIntervals(ac.tenant.Slug, time.Now().AddDate(0, -1, 0), time.Time{})
	recommendation := energy.NextRecommendation(profile, assets, mappings, intervals)
	if maintenance, listErr := a.energyFor(ac).ListMaintenance(ac.tenant.Slug); listErr == nil {
		if due, ok := energy.MaintenanceRecommendation(time.Now(), maintenance); ok {
			recommendation = due
		}
	}
	if strings.TrimSpace(r.FormValue("recommendation_id")) != recommendation.ID {
		http.Error(w, "Empfehlung ist nicht mehr aktuell.", http.StatusConflict)
		return
	}
	shared := []string{}
	for _, value := range r.Form["share"] {
		switch value {
		case "inventory", "measurements", "contact":
			shared = append(shared, value)
		}
	}
	pkg := energy.NewMeasurePackage(ac.tenant.Slug, recommendation)
	pkg.SharedFields = shared
	body := "Ziel: " + pkg.Goal + "\n\nAusgangslage: " + pkg.Baseline +
		"\n\nGewünschte Leistung: " + pkg.RequestedWork +
		"\n\nErwarteter Nachweis: " + pkg.ExpectedEvidence +
		"\n\nBewusst freigegebene Daten: " + energySharedFieldLabels(shared) +
		"\n\nOffene Vor-Ort-Fragen:\n- " + strings.Join(pkg.OpenSiteQuestions, "\n- ") +
		"\n\nKeine automatische Beauftragung, Preiszusage oder Vermittlungsprovision."
	now := time.Now().UTC()
	created, err := ac.repositories.issues.Create(residentIssue{
		TenantSlug:     ac.tenant.Slug,
		AuthorEmail:    normalizeEmail(ac.email),
		AuthorName:     a.profileForTenant(ac.email, ac.tenant.Slug).DisplayName(),
		Category:       "Sonstiges",
		Title:          "Energiemaßnahme: " + recommendation.Title,
		Body:           body,
		LocationType:   issueLocationCommon,
		LocationDetail: "Privates Hausprofil",
		Status:         issueStatusOpen,
		Priority:       issuePriorityNorm,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		http.Error(w, "Maßnahme konnte nicht angelegt werden.", http.StatusInternalServerError)
		return
	}
	profile.RecommendationID = recommendation.ID
	profile.RecommendationStatus = "measure-created"
	if err := a.energyFor(ac).SaveProfile(profile); err != nil {
		http.Error(w, "Maßnahme wurde angelegt, der Hausstatus konnte aber nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	measure := energy.Measure{
		TenantSlug:       ac.tenant.Slug,
		IssueID:          created.ID,
		RecommendationID: recommendation.ID,
		Title:            "Energiemaßnahme: " + recommendation.Title,
		Status:           energy.MeasureRequested,
		SharedFields:     shared,
	}
	if err := a.energyFor(ac).UpsertMeasure(measure); err != nil {
		http.Error(w, "Hausaufgabe wurde angelegt, der Maßnahmenkontext konnte aber nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.measure.create",
		TargetType: "issue",
		TargetID:   created.ID,
		Summary:    "Energieempfehlung als Maßnahme übernommen",
		Details: map[string]string{
			"recommendation_id": recommendation.ID,
			"shared_fields":     strings.Join(shared, ","),
			"marketplace_gate":  pkg.MarketplaceGate,
		},
	})
	http.Redirect(w, r, "/app/energie?measure=created#naechster-schritt", http.StatusSeeOther)
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

func (a *app) upsertEnergyMaintenance(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	assetID := strings.TrimSpace(r.FormValue("asset_id"))
	if !energyAssetBelongsToTenant(a.energyStore, ac.tenant.Slug, assetID) {
		http.Error(w, "Anlage gehört nicht zu diesem Haus.", http.StatusBadRequest)
		return
	}
	months, err := strconv.Atoi(strings.TrimSpace(r.FormValue("interval_months")))
	if err != nil {
		http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
		return
	}
	due, err := parseEnergyLocalDate(r.FormValue("next_due"))
	if err != nil {
		http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
		return
	}
	contactID := strings.TrimSpace(r.FormValue("contact_id"))
	documentID := strings.TrimSpace(r.FormValue("document_id"))
	issueID := strings.TrimSpace(r.FormValue("issue_id"))
	if !a.validEnergyReferences(ac.tenantRef, ac.repositories.contacts, contactID, documentID, issueID) {
		http.Error(w, "Verknüpfung gehört nicht zu diesem Haus.", http.StatusBadRequest)
		return
	}
	plan := energy.MaintenancePlan{
		ID:             strings.TrimSpace(r.FormValue("id")),
		TenantSlug:     ac.tenant.Slug,
		AssetID:        assetID,
		Title:          cleanEnergyText(r.FormValue("title"), 140),
		IntervalMonths: months,
		NextDueAt:      due,
		ContactID:      contactID,
		DocumentID:     documentID,
		IssueID:        issueID,
		EvidenceNote:   cleanEnergyText(r.FormValue("evidence_note"), 500),
		Active:         r.FormValue("active") != "false",
	}
	if plan.ID == "" {
		plan.ID = energy.NewID("maintenance")
	}
	if existing, ok := findMaintenancePlanByAsset(a.energyStore, ac.tenant.Slug, assetID); ok {
		plan.ID = existing.ID
		plan.CreatedAt = existing.CreatedAt
		plan.LastCompletedAt = existing.LastCompletedAt
	}
	if err := a.energyFor(ac).UpsertMaintenance(plan); err != nil {
		http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: "energy.maintenance.save", TargetType: "energy-maintenance", TargetID: plan.ID,
		Summary: "Wartungsplan gespeichert", Details: map[string]string{"asset_id": assetID, "interval_months": strconv.Itoa(months)},
	})
	http.Redirect(w, r, "/app/energie?maintenance=saved#wartung", http.StatusSeeOther)
}

func (a *app) completeEnergyMaintenance(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	plan, ok := findMaintenancePlan(a.energyStore, ac.tenant.Slug, strings.TrimSpace(r.FormValue("id")))
	if !ok {
		http.Error(w, "Wartungsplan nicht gefunden.", http.StatusNotFound)
		return
	}
	completed := time.Now().UTC()
	if raw := strings.TrimSpace(r.FormValue("completed_at")); raw != "" {
		parsed, err := parseEnergyLocalDate(raw)
		if err != nil {
			http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
			return
		}
		completed = parsed
	}
	plan.LastCompletedAt = &completed
	plan.NextDueAt = completed.AddDate(0, plan.IntervalMonths, 0)
	plan.EvidenceNote = cleanEnergyText(r.FormValue("evidence_note"), 500)
	if issueID := strings.TrimSpace(r.FormValue("issue_id")); issueID != "" {
		if _, found := ac.repositories.issues.Get(issueID); !found {
			http.Error(w, "Nachweis-Aufgabe gehört nicht zu diesem Haus.", http.StatusBadRequest)
			return
		}
		plan.IssueID = issueID
	}
	if plan.EvidenceNote == "" && plan.IssueID == "" {
		http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
		return
	}
	if err := a.energyFor(ac).UpsertMaintenance(plan); err != nil {
		http.Error(w, "Wartung konnte nicht abgeschlossen werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: "energy.maintenance.complete", TargetType: "energy-maintenance", TargetID: plan.ID,
		Summary: "Wartung abgeschlossen", Details: map[string]string{"next_due": plan.NextDueAt.Format("2006-01-02")},
	})
	http.Redirect(w, r, "/app/energie?maintenance=completed#wartung", http.StatusSeeOther)
}

func (a *app) saveEnergyTariffAssessment(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	intervals, err := a.energyFor(ac).ListIntervals(ac.tenant.Slug, monthStart.UTC(), monthStart.AddDate(0, 1, 0).UTC())
	if err != nil {
		http.Error(w, "Messwerte konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	peak := energy.PeakForMonth(intervals, now, time.Local)
	if peak <= 0 {
		http.Redirect(w, r, "/app/energie?assessment=no_data#tarif", http.StatusSeeOther)
		return
	}
	rules := energy.AustrianDraft2027()
	// Die vereinbarte Anschlussleistung entscheidet über die Mindestbemessung.
	// Fehlt das Profil, bleibt sie 0 und nur der 2-kW-Sockel greift.
	assessedProfile, _, _ := a.energyFor(ac).Profile(ac.tenant.Slug)
	estimate := rules.Estimate(peak, agreedPowerKW(assessedProfile))
	gaps, conflicts := intervalQualityCounts(intervals)
	quality := energy.QualityMeasured
	if gaps > 0 || conflicts > 0 {
		quality = energy.QualityGap
	}
	item := energy.TariffAssessment{
		ID:         energy.NewID("tariff"),
		TenantSlug: ac.tenant.Slug, AssessmentMonth: monthStart.Format("2006-01"),
		ProfileID: rules.ID, ProfileVersion: rules.Version, ProfileStatus: rules.Status, SourceURL: rules.SourceURL,
		PeakKW: peak, BilledKW: estimate.BilledKW, AnnualPowerEUR: estimate.AnnualPowerEUR, DataQuality: quality,
	}
	if err := a.energyFor(ac).SaveTariffAssessment(item); err != nil {
		http.Error(w, "Tarifbewertung konnte nicht festgehalten werden.", http.StatusInternalServerError)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: "energy.tariff.assessment", TargetType: "energy-tariff-assessment", TargetID: item.ID,
		Summary: "Tarifbewertung unveränderlich festgehalten", Details: map[string]string{"profile_id": rules.ID, "profile_version": rules.Version},
	})
	http.Redirect(w, r, "/app/energie?assessment=saved#tarif", http.StatusSeeOther)
}

func (a *app) updateEnergyMeasure(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	item, ok, err := a.energyFor(ac).GetMeasure(ac.tenant.Slug, strings.TrimSpace(r.FormValue("id")))
	if err != nil || !ok {
		http.Error(w, "Maßnahme nicht gefunden.", http.StatusNotFound)
		return
	}
	issue, found := ac.repositories.issues.Get(item.IssueID)
	if !found {
		http.Error(w, "Verknüpftes Anliegen nicht gefunden.", http.StatusConflict)
		return
	}
	status := strings.TrimSpace(r.FormValue("status"))
	switch status {
	case energy.MeasureRequested, energy.MeasureAssigned, energy.MeasureScheduled, energy.MeasureCompleted, energy.MeasureCancelled:
		item.Status = status
	default:
		http.Redirect(w, r, "/app/energie?measure_status=invalid#fachhilfe", http.StatusSeeOther)
		return
	}
	item.ContactID = strings.TrimSpace(r.FormValue("contact_id"))
	contact, contactOK := a.energyContact(ac.repositories.contacts, item.ContactID)
	if item.ContactID != "" && !contactOK {
		http.Error(w, "Fachkontakt gehört nicht zu diesem Haus.", http.StatusBadRequest)
		return
	}
	item.OfferNote = cleanEnergyText(r.FormValue("offer_note"), 1000)
	item.WorkNote = cleanEnergyText(r.FormValue("work_note"), 1000)
	item.EvidenceNote = cleanEnergyText(r.FormValue("evidence_note"), 1000)
	item.AppointmentAt = nil
	if raw := strings.TrimSpace(r.FormValue("appointment_at")); raw != "" {
		appointment, parseErr := parseEnergyLocalDateTime(raw)
		if parseErr != nil {
			http.Redirect(w, r, "/app/energie?measure_status=invalid#fachhilfe", http.StatusSeeOther)
			return
		}
		item.AppointmentAt = &appointment
	}
	if item.Status == energy.MeasureScheduled && item.AppointmentAt == nil {
		http.Redirect(w, r, "/app/energie?measure_status=appointment#fachhilfe", http.StatusSeeOther)
		return
	}
	if item.Status == energy.MeasureCompleted {
		if err := a.completeEnergyMeasureRanges(&item, r); err != nil {
			http.Redirect(w, r, "/app/energie?measure_status=ranges#fachhilfe", http.StatusSeeOther)
			return
		}
		completed := time.Now().UTC()
		item.CompletedAt = &completed
	}
	if item.ContactID != "" && item.Status == energy.MeasureRequested {
		item.Status = energy.MeasureAssigned
	}
	if err := a.energyFor(ac).UpsertMeasure(item); err != nil {
		http.Error(w, "Maßnahme konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	if contactOK && a.serviceAccessEnabled && normalizeEmail(contact.Email) != "" {
		updated, changed, updateErr := ac.repositories.issues.UpdateWorkflow(issue.ID, issueWorkflowUpdate{
			Status: issue.Status, Priority: issue.Priority, AssigneeEmail: normalizeEmail(contact.Email),
			ActorEmail: ac.email, ActorName: a.profileForTenant(ac.email, ac.tenant.Slug).DisplayName(), ChangedAt: time.Now(),
		})
		if updateErr != nil {
			http.Error(w, "Maßnahme ist gespeichert, aber der Dienstleisterzugriff konnte nicht aktualisiert werden.", http.StatusInternalServerError)
			return
		}
		if changed {
			a.handleIssueServiceAssignmentChange(r, ac.tenant, issue, updated, ac.email, ac.role)
		}
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: "energy.measure.update", TargetType: "energy-measure", TargetID: item.ID,
		Summary: "Energiemaßnahme aktualisiert", Details: map[string]string{"status": item.Status, "contact_id": item.ContactID},
	})
	http.Redirect(w, r, "/app/energie?measure_status=saved#fachhilfe", http.StatusSeeOther)
}

func energySharedFieldLabels(values []string) string {
	labels := map[string]string{
		"inventory":    "Anlageninventar",
		"measurements": "zusammengefasste Messwerte",
		"contact":      "Kontaktdaten",
	}
	out := []string{}
	for _, value := range values {
		if label := labels[value]; label != "" {
			out = append(out, label)
		}
	}
	if len(out) == 0 {
		return "keine"
	}
	return strings.Join(out, ", ")
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
		http.Error(w, "Person gehört nicht zu diesem Haus.", http.StatusBadRequest)
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
		Summary:    "Technischer Hauszugriff geändert",
		Details: map[string]string{
			"view":      strconv.FormatBool(grant["view"] || grant["configure"] || grant["control"]),
			"configure": strconv.FormatBool(grant["configure"]),
			"control":   strconv.FormatBool(grant["control"]),
		},
	})
	http.Redirect(w, r, "/app/energie?caretaker=1#betreuung", http.StatusSeeOther)
}

// energyCustomAssetSource marks consumers created outside the onboarding
// presets. It keeps their random identity stable when the preset selection is
// reconciled later.
const (
	energyCustomAssetSource   = "custom"
	energyFlowNodeAssetSource = "energy-flow-node"
	energyFlowHomeKind        = "flow-home"
	energyFlowGridKind        = "flow-grid"
	energyFlowParkingKind     = "flow-parking"
)

func isEnergyFlowSystemKind(kind string) bool {
	switch kind {
	case "pv", "battery", energyFlowHomeKind, energyFlowGridKind, energyFlowParkingKind:
		return true
	default:
		return false
	}
}

func energyFlowNodeID(tenantSlug, nodeType string) string {
	return energy.StableAssetID(tenantSlug, "flow-"+nodeType)
}

// addEnergyConsumer legt Verbraucher an oder bearbeitet eine bestehende
// Entität aus dem Cockpit-Dialog. Die Vorlagenidentität bleibt beim Bearbeiten
// erhalten; deleting is an explicit cockpit action for every consumer.
//
// Die Vorlagenliste bleibt unangetastet: sie deckt die häufigen Fälle ab, ist
// aber keine Obergrenze. Sauna, Durchlauferhitzer, Whirlpool oder Werkstatt
// treiben die Viertelstundenspitze genauso, und ohne eigene Entität wären sie
// im Lastmanagement unsichtbar.

func energySystemAssets(assets []energy.Asset) []energy.Asset {
	out := []energy.Asset{}
	for _, asset := range assets {
		switch asset.Kind {
		case "pv", "battery":
			out = append(out, asset)
		}
	}
	return out
}

func energyFlexibilityLabel(flexibility string) string {
	switch flexibility {
	case energy.FlexShift:
		return "zeitlich verschiebbar"
	case energy.FlexThrottle:
		return "kurz begrenzbar"
	case energy.FlexFixed:
		return "fest"
	default:
		return "Flexibilität offen"
	}
}

func buildEnergyConsumerKindOptions() []energyOption {
	kinds := []string{"other", "sauna", "instant-water-heater", "air-conditioning", "hot-water",
		"heat-pump", "ev", "wallbox"}
	out := make([]energyOption, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, energyOption{Value: kind, Label: energy.AssetKindLabel(kind)})
	}
	return out
}

func buildEnergyConsumerIconOptions() []energyOption {
	return []energyOption{
		{Value: "car-front", Label: "Auto"},
		{Value: "plug-zap", Label: "Ladestecker"},
		{Value: "heater", Label: "Warmwasser"},
		{Value: "fan", Label: "Wärmepumpe"},
		{Value: "washing-machine", Label: "Waschmaschine"},
		{Value: "flame", Label: "Sauna"},
		{Value: "drill", Label: "Werkstatt"},
		{Value: "waves-ladder", Label: "Pool"},
		{Value: "square-parking", Label: "Parkplatz"},
		{Value: "snowflake", Label: "Klimaanlage"},
		{Value: "shower-head", Label: "Dusche"},
		{Value: "plug", Label: "Sonstiges"},
	}
}

func defaultEnergyConsumerIcon(kind string) string {
	switch kind {
	case "ev":
		return "car-front"
	case "wallbox":
		return "plug-zap"
	case "hot-water":
		return "heater"
	case "instant-water-heater":
		return "shower-head"
	case "heat-pump":
		return "fan"
	case "sauna":
		return "flame"
	case "air-conditioning":
		return "snowflake"
	default:
		return "plug"
	}
}

func normalizeEnergyConsumerIcon(raw, kind string) string {
	wanted := strings.TrimSpace(strings.ToLower(raw))
	if web.IsLucideIcon(wanted) {
		return wanted
	}
	return defaultEnergyConsumerIcon(kind)
}

func energyConsumerIcon(asset energy.Asset) string {
	if asset.Metadata == nil {
		return defaultEnergyConsumerIcon(asset.Kind)
	}
	return normalizeEnergyConsumerIcon(asset.Metadata["icon"], asset.Kind)
}

func energyConsumerStaleOverrideMinutes(asset energy.Asset) int {
	minutes, err := strconv.Atoi(strings.TrimSpace(asset.Metadata["stale_after_minutes"]))
	if err != nil || minutes < 1 || minutes > energyConsumerMaxStaleMinutes {
		return 0
	}
	return minutes
}

func energyConsumerStaleAfter(asset energy.Asset) time.Duration {
	if minutes := energyConsumerStaleOverrideMinutes(asset); minutes > 0 {
		return time.Duration(minutes) * time.Minute
	}
	return energyConsumerDefaultStaleAfter
}

func setEnergyConsumerStaleOverride(metadata map[string]string, raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		delete(metadata, "stale_after_minutes")
		return true
	}
	minutes, err := strconv.Atoi(raw)
	if err != nil || minutes < 1 || minutes > energyConsumerMaxStaleMinutes {
		return false
	}
	metadata["stale_after_minutes"] = strconv.Itoa(minutes)
	return true
}

func energyVehicleSleepState(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "on", "true", "1", "sleep", "sleeping", "asleep", "schläft":
		return true, true
	case "off", "false", "0", "awake", "online":
		return false, true
	default:
		return false, false
	}
}

func energyVehicleSleepSignal(state homeassistant.EntityState) bool {
	return homeconnector.IsVehicleSleepReading(
		state.EntityID,
		state.State,
		haAttribute(state.Attributes, "friendly_name"),
	)
}

func consumerMeasurementKind(state homeassistant.EntityState) string {
	if energyVehicleSleepSignal(state) {
		return "sleep"
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(state.EntityID)), "sensor.") {
		return ""
	}
	deviceClass := strings.ToLower(strings.TrimSpace(haAttribute(state.Attributes, "device_class")))
	unit := strings.ToLower(strings.TrimSpace(haAttribute(state.Attributes, "unit_of_measurement")))
	switch {
	case deviceClass == "power" || unit == "w" || unit == "kw" || unit == "mw":
		return "power"
	case deviceClass == "energy" || unit == "wh" || unit == "kwh" || unit == "mwh":
		return "energy"
	case deviceClass == "battery" || unit == "%":
		return "percentage"
	default:
		return ""
	}
}

func consumerMeasurementMetric(kind string) string {
	switch kind {
	case "power":
		return energy.MetricConsumerPower
	case "percentage":
		return energy.MetricBatterySOC
	case "sleep":
		return energy.MetricConsumerSleep
	default:
		return energy.MetricConsumerEnergy
	}
}

func compactEnergyMatchKey(value string) string {
	return strings.Map(func(char rune) rune {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			return unicode.ToLower(char)
		}
		return -1
	}, value)
}

func namedEVMeasurementCandidate(name, kind string, states []homeassistant.EntityState, assigned map[string]bool) (homeassistant.EntityState, bool) {
	wanted := compactEnergyMatchKey(name)
	if len(wanted) < 4 {
		return homeassistant.EntityState{}, false
	}
	bestScore := 0
	bestCount := 0
	best := homeassistant.EntityState{}
	for _, state := range states {
		entityID := strings.ToLower(strings.TrimSpace(state.EntityID))
		if assigned[entityID] || consumerMeasurementKind(state) != kind {
			continue
		}
		haystack := compactEnergyMatchKey(entityID + " " + haAttribute(state.Attributes, "friendly_name"))
		if !strings.Contains(haystack, wanted) {
			continue
		}
		score := 100
		switch kind {
		case "power":
			switch {
			case strings.Contains(haystack, "ladeleistungzuhause"):
				score += 80
			case strings.Contains(haystack, "chargerpower"):
				score += 60
			case strings.Contains(haystack, "ladeleistung") || strings.Contains(haystack, "chargingpower"):
				score += 40
			}
		case "energy":
			switch {
			case strings.Contains(haystack, "ladeenergiezuhause"):
				score += 80
			case strings.Contains(haystack, "chargeenergyadded"):
				score += 60
			case strings.Contains(haystack, "ladeenergie") || strings.Contains(haystack, "chargingenergy"):
				score += 40
			}
		case "percentage":
			switch {
			case strings.Contains(haystack, "batterylevel") || strings.Contains(haystack, "ladestand"):
				score += 80
			case strings.Contains(haystack, "stateofcharge") || strings.Contains(haystack, "batterypercentage"):
				score += 60
			case strings.Contains(haystack, "soc"):
				score += 40
			}
		case "sleep":
			switch {
			case strings.Contains(haystack, "asleep") || strings.Contains(haystack, "schläft"):
				score += 80
			case strings.Contains(haystack, "sleeping") || strings.Contains(haystack, "schlaf"):
				score += 60
			case strings.Contains(haystack, "sleep"):
				score += 40
			}
		}
		if score > bestScore {
			best, bestScore, bestCount = state, score, 1
		} else if score == bestScore {
			bestCount++
		}
	}
	return best, bestScore > 100 && bestCount == 1
}

func (a *app) ensureNamedEVMeasurementMappings(ctx context.Context, tenant tenantConfig, assets []energy.Asset, mappings []energy.EntityMapping) ([]energy.EntityMapping, bool) {
	mapped := map[string]map[string]bool{}
	assigned := map[string]bool{}
	for _, mapping := range mappings {
		if !mapping.Confirmed {
			continue
		}
		assigned[strings.ToLower(mapping.EntityID)] = true
		if mapped[mapping.AssetID] == nil {
			mapped[mapping.AssetID] = map[string]bool{}
		}
		mapped[mapping.AssetID][mapping.Metric] = true
	}
	needsScan := false
	for _, asset := range assets {
		if asset.Kind == "ev" && (!mapped[asset.ID][energy.MetricConsumerPower] || !mapped[asset.ID][energy.MetricConsumerEnergy] ||
			!mapped[asset.ID][energy.MetricBatterySOC] || !mapped[asset.ID][energy.MetricConsumerSleep]) {
			needsScan = true
			break
		}
	}
	if !needsScan {
		return mappings, false
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	states, configured, err := a.energyStates(lookupCtx, tenant)
	cancel()
	if err != nil || !configured {
		return mappings, false
	}
	changed := false
	for _, asset := range assets {
		if asset.Kind != "ev" {
			continue
		}
		if mapped[asset.ID] == nil {
			mapped[asset.ID] = map[string]bool{}
		}
		for _, kind := range []string{"power", "energy", "percentage", "sleep"} {
			metric := consumerMeasurementMetric(kind)
			if mapped[asset.ID][metric] {
				continue
			}
			state, ok := namedEVMeasurementCandidate(asset.Name, kind, states, assigned)
			if !ok {
				continue
			}
			entityID := strings.ToLower(strings.TrimSpace(state.EntityID))
			mapping := energy.EntityMapping{
				TenantSlug: tenant.Slug, EntityID: entityID, AssetID: asset.ID, Metric: metric,
				DisplayName: firstNonEmpty(haAttribute(state.Attributes, "friendly_name"), entityID),
				Unit:        haAttribute(state.Attributes, "unit_of_measurement"), DeviceClass: haAttribute(state.Attributes, "device_class"), Confirmed: true,
			}
			seen := state.LastUpdated
			if seen.IsZero() {
				seen = state.LastChanged
			}
			if !seen.IsZero() {
				mapping.LastSeenAt = &seen
			}
			if err := a.energyStore.UpsertMapping(mapping); err != nil {
				continue
			}
			assigned[entityID] = true
			mapped[asset.ID][metric] = true
			changed = true
		}
	}
	if !changed {
		return mappings, false
	}
	updated, err := a.energyStore.ListMappings(tenant.Slug)
	if err != nil {
		return mappings, false
	}
	return updated, true
}

func (a *app) energyConsumerMeasurementOptions(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	assets, _ := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	mappings, _ := a.energyFor(ac).ListMappings(ac.tenant.Slug)
	assetNames := map[string]string{}
	for _, asset := range assets {
		assetNames[asset.ID] = asset.Name
	}
	byEntity := map[string]energy.EntityMapping{}
	for _, mapping := range mappings {
		byEntity[mapping.EntityID] = mapping
	}
	response := energyConsumerMeasurementsResponse{
		Status:   "not-configured",
		Message:  "Home Assistant ist für dieses Zuhause noch nicht verbunden.",
		Entities: []energyConsumerMeasurementOption{},
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	states, configured, err := a.energyStates(ctx, ac.tenant)
	cancel()
	if configured {
		if err == nil {
			response.Status = "ok"
			response.Message = "Home Assistant verbunden · alle Messwerte werden ausschließlich gelesen."
			for _, state := range states {
				kind := consumerMeasurementKind(state)
				if kind == "" {
					continue
				}
				entityID := strings.ToLower(strings.TrimSpace(state.EntityID))
				option := energyConsumerMeasurementOption{
					EntityID: entityID,
					Name:     firstNonEmpty(haAttribute(state.Attributes, "friendly_name"), entityID),
					Unit:     haAttribute(state.Attributes, "unit_of_measurement"),
					Kind:     kind,
				}
				if mapping, found := byEntity[entityID]; found && mapping.Confirmed {
					option.AssignedAssetID = mapping.AssetID
					option.AssignedAssetName = assetNames[mapping.AssetID]
					if mapping.AssetID == "" {
						option.AssignedAssetName = "Haus gesamt"
					}
				}
				response.Entities = append(response.Entities, option)
			}
		} else {
			response.Status = "offline"
			response.Message = "Home Assistant antwortet gerade nicht. Bestehende Zuordnungen bleiben erhalten."
		}
	}
	// Existing consumer mappings remain editable even while HA is offline or
	// an entity is currently unavailable and therefore absent from /states.
	seen := map[string]bool{}
	for _, option := range response.Entities {
		seen[option.EntityID] = true
	}
	for _, mapping := range mappings {
		kind := ""
		switch mapping.Metric {
		case energy.MetricConsumerPower, energy.MetricLoadPower, energy.MetricGridImportPower,
			energy.MetricGridExportPower, energy.MetricPVPower, energy.MetricBatteryPower,
			energy.MetricBatteryCharge, energy.MetricBatteryDischarge:
			kind = "power"
		case energy.MetricConsumerEnergy, energy.MetricGridImportEnergy:
			kind = "energy"
		case energy.MetricBatterySOC:
			kind = "percentage"
		case energy.MetricConsumerSleep:
			kind = "sleep"
		}
		if kind == "" || seen[mapping.EntityID] {
			continue
		}
		response.Entities = append(response.Entities, energyConsumerMeasurementOption{
			EntityID: mapping.EntityID, Name: firstNonEmpty(mapping.DisplayName, mapping.EntityID),
			Unit: mapping.Unit, Kind: kind, AssignedAssetID: mapping.AssetID,
			AssignedAssetName: assetNames[mapping.AssetID],
		})
	}
	sort.Slice(response.Entities, func(i, j int) bool {
		left := strings.ToLower(response.Entities[i].Name + " " + response.Entities[i].EntityID)
		right := strings.ToLower(response.Entities[j].Name + " " + response.Entities[j].EntityID)
		return left < right
	})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(response)
}

type energyConsumerMappingPlan struct {
	deleteIDs []string
	upserts   []energy.EntityMapping
}

func (a *app) planEnergyConsumerMappings(ctx context.Context, tenant tenantConfig, assetID, powerEntity, energyEntity, socEntity, sleepEntity string) (energyConsumerMappingPlan, error) {
	desired := map[string]string{
		"power":      strings.ToLower(strings.TrimSpace(powerEntity)),
		"energy":     strings.ToLower(strings.TrimSpace(energyEntity)),
		"percentage": strings.ToLower(strings.TrimSpace(socEntity)),
		"sleep":      strings.ToLower(strings.TrimSpace(sleepEntity)),
	}
	seenDesired := map[string]bool{}
	for _, entityID := range desired {
		if entityID != "" && seenDesired[entityID] {
			return energyConsumerMappingPlan{}, fmt.Errorf("one entity cannot fill multiple measurement slots")
		}
		seenDesired[entityID] = entityID != ""
	}
	mappings, err := a.energyStore.ListMappings(tenant.Slug)
	if err != nil {
		return energyConsumerMappingPlan{}, err
	}
	byEntity := map[string]energy.EntityMapping{}
	current := map[string]energy.EntityMapping{}
	for _, mapping := range mappings {
		byEntity[mapping.EntityID] = mapping
		if mapping.AssetID != assetID {
			continue
		}
		switch mapping.Metric {
		case energy.MetricConsumerPower:
			current["power"] = mapping
		case energy.MetricConsumerEnergy:
			current["energy"] = mapping
		case energy.MetricBatterySOC:
			current["percentage"] = mapping
		case energy.MetricConsumerSleep:
			current["sleep"] = mapping
		}
	}
	plan := energyConsumerMappingPlan{}
	needsLookup := false
	for kind, entityID := range desired {
		if existing, ok := current[kind]; ok && existing.EntityID != entityID {
			plan.deleteIDs = append(plan.deleteIDs, existing.ID)
		}
		if entityID == "" || current[kind].EntityID == entityID {
			continue
		}
		if !strings.HasPrefix(entityID, "sensor.") && (kind != "sleep" || !strings.HasPrefix(entityID, "binary_sensor.")) {
			return energyConsumerMappingPlan{}, fmt.Errorf("only sensor entities may be mapped")
		}
		if existing, found := byEntity[entityID]; found && existing.Confirmed && existing.AssetID != assetID {
			return energyConsumerMappingPlan{}, fmt.Errorf("entity already has another assignment")
		}
		needsLookup = true
	}
	statesByID := map[string]homeassistant.EntityState{}
	if needsLookup {
		lookupCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		states, configured, lookupErr := a.energyStates(lookupCtx, tenant)
		cancel()
		if lookupErr != nil || !configured {
			if lookupErr == nil {
				lookupErr = fmt.Errorf("home assistant is not configured")
			}
			return energyConsumerMappingPlan{}, lookupErr
		}
		for _, state := range states {
			statesByID[strings.ToLower(strings.TrimSpace(state.EntityID))] = state
		}
	}
	for kind, entityID := range desired {
		if entityID == "" || current[kind].EntityID == entityID {
			continue
		}
		state, found := statesByID[entityID]
		if !found || consumerMeasurementKind(state) != kind {
			return energyConsumerMappingPlan{}, fmt.Errorf("entity does not match measurement slot")
		}
		mapping := byEntity[entityID]
		mapping.TenantSlug = tenant.Slug
		mapping.EntityID = entityID
		mapping.AssetID = assetID
		mapping.Metric = consumerMeasurementMetric(kind)
		mapping.DisplayName = firstNonEmpty(haAttribute(state.Attributes, "friendly_name"), entityID)
		mapping.Unit = haAttribute(state.Attributes, "unit_of_measurement")
		mapping.DeviceClass = haAttribute(state.Attributes, "device_class")
		mapping.Confirmed = true
		seen := state.LastUpdated
		if seen.IsZero() {
			seen = state.LastChanged
		}
		if !seen.IsZero() {
			mapping.LastSeenAt = &seen
		}
		plan.upserts = append(plan.upserts, mapping)
	}
	return plan, nil
}

func (a *app) applyEnergyConsumerMappingPlan(tenantSlug string, plan energyConsumerMappingPlan) error {
	for _, id := range plan.deleteIDs {
		if _, err := a.energyStore.DeleteMapping(tenantSlug, id); err != nil {
			return err
		}
	}
	for _, mapping := range plan.upserts {
		if err := a.energyStore.UpsertMapping(mapping); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) planEnergyFlowNodeMappings(ctx context.Context, tenant tenantConfig, assetID string, slots []energyFlowMeasurementSlot, form url.Values) (energyConsumerMappingPlan, error) {
	desired := map[string]string{}
	wantedKind := map[string]string{}
	seenEntity := map[string]bool{}
	for _, slot := range slots {
		entityID := strings.ToLower(strings.TrimSpace(form.Get(slot.Name)))
		if entityID != "" && seenEntity[entityID] {
			return energyConsumerMappingPlan{}, fmt.Errorf("one entity cannot fill multiple measurement slots")
		}
		seenEntity[entityID] = entityID != ""
		desired[slot.Metric] = entityID
		wantedKind[slot.Metric] = slot.Kind
	}
	mappings, err := a.energyStore.ListMappings(tenant.Slug)
	if err != nil {
		return energyConsumerMappingPlan{}, err
	}
	byEntity := map[string]energy.EntityMapping{}
	current := map[string]energy.EntityMapping{}
	for _, mapping := range mappings {
		entityID := strings.ToLower(strings.TrimSpace(mapping.EntityID))
		byEntity[entityID] = mapping
		if _, wanted := desired[mapping.Metric]; !wanted {
			continue
		}
		legacyWholeHome := mapping.AssetID == "" && (mapping.Metric == energy.MetricLoadPower || mapping.Metric == energy.MetricGridImportPower || mapping.Metric == energy.MetricGridExportPower)
		if mapping.AssetID == assetID || legacyWholeHome {
			current[mapping.Metric] = mapping
		}
	}
	plan := energyConsumerMappingPlan{}
	needsLookup := false
	for metric, entityID := range desired {
		if existing, ok := current[metric]; ok && strings.ToLower(existing.EntityID) != entityID {
			plan.deleteIDs = append(plan.deleteIDs, existing.ID)
		}
		if entityID == "" || strings.EqualFold(current[metric].EntityID, entityID) {
			continue
		}
		if !strings.HasPrefix(entityID, "sensor.") {
			return energyConsumerMappingPlan{}, fmt.Errorf("only sensor entities may be mapped")
		}
		if existing, found := byEntity[entityID]; found && existing.Confirmed {
			belongsToNode := existing.AssetID == assetID
			for _, assigned := range current {
				belongsToNode = belongsToNode || strings.EqualFold(assigned.EntityID, entityID)
			}
			if !belongsToNode {
				return energyConsumerMappingPlan{}, fmt.Errorf("entity already has another assignment")
			}
		}
		needsLookup = true
	}
	statesByID := map[string]homeassistant.EntityState{}
	if needsLookup {
		lookupCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		states, configured, lookupErr := a.energyStates(lookupCtx, tenant)
		cancel()
		if lookupErr != nil || !configured {
			if lookupErr == nil {
				lookupErr = fmt.Errorf("home assistant is not configured")
			}
			return energyConsumerMappingPlan{}, lookupErr
		}
		for _, state := range states {
			statesByID[strings.ToLower(strings.TrimSpace(state.EntityID))] = state
		}
	}
	for metric, entityID := range desired {
		if entityID == "" || strings.EqualFold(current[metric].EntityID, entityID) {
			continue
		}
		state, found := statesByID[entityID]
		if !found || consumerMeasurementKind(state) != wantedKind[metric] {
			return energyConsumerMappingPlan{}, fmt.Errorf("entity does not match measurement slot")
		}
		mapping := byEntity[entityID]
		mapping.TenantSlug = tenant.Slug
		mapping.EntityID = entityID
		mapping.AssetID = assetID
		mapping.Metric = metric
		mapping.DisplayName = firstNonEmpty(haAttribute(state.Attributes, "friendly_name"), entityID)
		mapping.Unit = haAttribute(state.Attributes, "unit_of_measurement")
		mapping.DeviceClass = haAttribute(state.Attributes, "device_class")
		mapping.Confirmed = true
		seen := state.LastUpdated
		if seen.IsZero() {
			seen = state.LastChanged
		}
		if !seen.IsZero() {
			mapping.LastSeenAt = &seen
		}
		plan.upserts = append(plan.upserts, mapping)
	}
	return plan, nil
}

func (a *app) updateEnergyFlowNode(w http.ResponseWriter, r *http.Request, ac authCtx, nodeType string) {
	allowed := map[string]string{
		"home": energyFlowHomeKind, "grid": energyFlowGridKind, "pv": "pv",
		"storage": "battery", "parking": energyFlowParkingKind,
	}
	wantedKind, ok := allowed[nodeType]
	if !ok {
		http.Error(w, "Ungültiger Energieknoten.", http.StatusBadRequest)
		return
	}
	assets, err := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Energieknoten konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	asset := energyFlowNodeAsset(assets, ac.tenant.Slug, nodeType)
	postedID := strings.TrimSpace(r.FormValue("asset_id"))
	if postedID != asset.ID {
		http.Error(w, "Ungültiger Energieknoten.", http.StatusBadRequest)
		return
	}
	asset.Kind = wantedKind
	if nodeType == "home" || nodeType == "grid" || nodeType == "parking" || asset.Source == "" {
		asset.Source = energyFlowNodeAssetSource
	}
	asset.Confirmed = true
	asset.Name = cleanEnergyText(r.FormValue("name"), 80)
	if asset.Name == "" {
		http.Redirect(w, r, "/app/energie?verbraucher=name", http.StatusSeeOther)
		return
	}
	if asset.Metadata == nil {
		asset.Metadata = map[string]string{}
	}
	asset.Metadata["icon"] = normalizeEnergyConsumerIcon(r.FormValue("icon"), wantedKind)
	asset.Metadata["icon_configured"] = "true"
	asset.Metadata["color"] = normalizeEnergyFlowColor(r.FormValue("color"), wantedKind)
	if secondaryLabel := cleanEnergyText(r.FormValue("secondary_label"), 60); secondaryLabel != "" {
		asset.Metadata["secondary_label"] = secondaryLabel
	} else {
		delete(asset.Metadata, "secondary_label")
	}
	delete(asset.Metadata, "hidden")
	if nodeType == "parking" {
		if !setEnergyConsumerStaleOverride(asset.Metadata, r.FormValue("stale_after_minutes")) {
			http.Redirect(w, r, "/app/energie?verbraucher=schwelle", http.StatusSeeOther)
			return
		}
		asset.Flexibility = energyConsumerFlexibility(r.FormValue("flexibility"))
		asset.RatedPowerKW = nil
		if raw := strings.TrimSpace(r.FormValue("rated_power_kw")); raw != "" {
			value, parseErr := homeassistant.ParseFloat(raw)
			if parseErr != nil || !energy.ValidPowerKW(value, 1000) {
				http.Redirect(w, r, "/app/energie?verbraucher=leistung", http.StatusSeeOther)
				return
			}
			asset.RatedPowerKW = &value
		}
	}
	if nodeType == "storage" && strings.TrimSpace(r.FormValue("battery_power_entity")) != "" &&
		(strings.TrimSpace(r.FormValue("battery_charge_entity")) != "" || strings.TrimSpace(r.FormValue("battery_discharge_entity")) != "") {
		http.Redirect(w, r, "/app/energie?verbraucher=messwerte", http.StatusSeeOther)
		return
	}
	plan, err := a.planEnergyFlowNodeMappings(r.Context(), ac.tenant, asset.ID, energyFlowNodeSlots(nodeType, asset.ID, nil), r.Form)
	if err != nil {
		http.Redirect(w, r, "/app/energie?verbraucher=messwerte", http.StatusSeeOther)
		return
	}
	if err := a.energyFor(ac).UpsertAsset(asset); err != nil {
		http.Error(w, "Energieknoten konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	if err := a.applyEnergyConsumerMappingPlan(ac.tenant.Slug, plan); err != nil {
		http.Error(w, "Messwerte konnten nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	if nodeType == "parking" {
		_ = a.updateEnergyConsumerPriority(ac.tenant.Slug, asset.ID, r.FormValue("priority"))
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
		Action: "energy.flow-node.update", TargetType: "energy-asset", TargetID: asset.ID,
		Summary: "Energiefluss-Knoten bearbeitet", Details: map[string]string{"node_type": nodeType},
	})
	http.Redirect(w, r, "/app/energie?verbraucher=gespeichert", http.StatusSeeOther)
}

func (a *app) updateEnergyConsumerPriority(tenantSlug, assetID, rawPriority string) error {
	assets, err := a.energyStore.ListAssets(tenantSlug)
	if err != nil {
		return err
	}
	ordered := []string{}
	for _, asset := range sortEnergyConsumers(assets) {
		if isEnergyFlowSystemKind(asset.Kind) && asset.Kind != energyFlowParkingKind {
			continue
		}
		if asset.ID != assetID {
			ordered = append(ordered, asset.ID)
		}
	}
	priority, err := strconv.Atoi(strings.TrimSpace(rawPriority))
	if err != nil || priority < 1 {
		priority = len(ordered) + 1
	}
	if priority > len(ordered)+1 {
		priority = len(ordered) + 1
	}
	index := priority - 1
	ordered = append(ordered, "")
	copy(ordered[index+1:], ordered[index:])
	ordered[index] = assetID
	_, err = a.energyStore.UpdateAssetPriorities(tenantSlug, ordered)
	return err
}

func (a *app) ensureEnergyParkingAsset(tenant tenantConfig, assets []energy.Asset) ([]energy.Asset, error) {
	if !tenant.HA.ChargingConfigured() {
		return assets, nil
	}
	parkingID := energyFlowNodeID(tenant.Slug, "parking")
	for _, asset := range assets {
		if asset.ID == parkingID {
			return assets, nil
		}
	}
	parking := energyFlowNodeAsset(assets, tenant.Slug, "parking")
	if err := a.energyStore.UpsertAsset(parking); err != nil {
		return assets, err
	}
	return append(assets, parking), nil
}

func (a *app) addEnergyConsumer(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	if nodeType := strings.TrimSpace(r.FormValue("node_type")); nodeType != "" && nodeType != "consumer" {
		a.updateEnergyFlowNode(w, r, ac, nodeType)
		return
	}
	name := cleanEnergyText(r.FormValue("name"), 80)
	if name == "" {
		http.Redirect(w, r, "/app/energie?verbraucher=name", http.StatusSeeOther)
		return
	}
	kind := energyConsumerKind(r.FormValue("kind"))
	assets, err := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Verbraucher konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	assets, err = a.ensureEnergyParkingAsset(ac.tenant, assets)
	if err != nil {
		http.Error(w, "Verbraucher konnten nicht vorbereitet werden.", http.StatusInternalServerError)
		return
	}
	assetID := strings.TrimSpace(r.FormValue("asset_id"))
	asset := energy.Asset{}
	consumerCount := 0
	previousPriority := 1
	for _, candidate := range sortEnergyConsumers(assets) {
		if isEnergyFlowSystemKind(candidate.Kind) && candidate.Kind != energyFlowParkingKind {
			continue
		}
		if candidate.Kind == energyFlowParkingKind && candidate.Metadata["hidden"] == "true" {
			continue
		}
		consumerCount++
		if candidate.ID == assetID {
			previousPriority = consumerCount
		}
	}
	if assetID == "" {
		previousPriority = consumerCount + 1
	}
	if assetID != "" {
		found := false
		for _, candidate := range sortEnergyConsumers(assets) {
			if candidate.ID != assetID {
				continue
			}
			if isEnergyFlowSystemKind(candidate.Kind) {
				http.Error(w, "Diese Anlage ist kein Verbraucher.", http.StatusBadRequest)
				return
			}
			asset = candidate
			found = true
			break
		}
		if !found {
			http.Error(w, "Verbraucher nicht gefunden.", http.StatusBadRequest)
			return
		}
	} else {
		asset = energy.Asset{
			ID:         energy.NewID("asset"),
			TenantSlug: ac.tenant.Slug,
			Source:     energyCustomAssetSource,
			Confirmed:  true,
			Metadata:   map[string]string{},
		}
	}
	if asset.Metadata == nil {
		asset.Metadata = map[string]string{}
	}
	asset.Kind = kind
	asset.Name = name
	asset.Flexibility = energyConsumerFlexibility(r.FormValue("flexibility"))
	asset.Metadata["icon"] = normalizeEnergyConsumerIcon(r.FormValue("icon"), kind)
	asset.Metadata["color"] = normalizeEnergyFlowColor(r.FormValue("color"), kind)
	if secondaryLabel := cleanEnergyText(r.FormValue("secondary_label"), 60); secondaryLabel != "" {
		asset.Metadata["secondary_label"] = secondaryLabel
	} else {
		delete(asset.Metadata, "secondary_label")
	}
	if !setEnergyConsumerStaleOverride(asset.Metadata, r.FormValue("stale_after_minutes")) {
		http.Redirect(w, r, "/app/energie?verbraucher=schwelle", http.StatusSeeOther)
		return
	}
	asset.RatedPowerKW = nil
	if raw := strings.TrimSpace(r.FormValue("rated_power_kw")); raw != "" {
		value, err := homeassistant.ParseFloat(raw)
		if err != nil || !energy.ValidPowerKW(value, 1000) {
			http.Redirect(w, r, "/app/energie?verbraucher=leistung", http.StatusSeeOther)
			return
		}
		asset.RatedPowerKW = &value
	}
	mappingPlan := energyConsumerMappingPlan{}
	if r.FormValue("measurements_present") == "1" {
		var planErr error
		sleepEntity := r.FormValue("consumer_sleep_entity")
		if kind != "ev" {
			sleepEntity = ""
		}
		mappingPlan, planErr = a.planEnergyConsumerMappings(r.Context(), ac.tenant, asset.ID,
			r.FormValue("consumer_power_entity"), r.FormValue("consumer_energy_entity"),
			r.FormValue("consumer_soc_entity"), sleepEntity)
		if planErr != nil {
			http.Redirect(w, r, "/app/energie?verbraucher=messwerte", http.StatusSeeOther)
			return
		}
	}
	if err := a.energyFor(ac).UpsertAsset(asset); err != nil {
		http.Error(w, "Verbraucher konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	if err := a.applyEnergyConsumerMappingPlan(ac.tenant.Slug, mappingPlan); err != nil {
		http.Error(w, "Messwerte konnten nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	updatedAssets, err := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Priorität konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	orderedIDs := []string{}
	for _, candidate := range sortEnergyConsumers(updatedAssets) {
		if (isEnergyFlowSystemKind(candidate.Kind) && candidate.Kind != energyFlowParkingKind) || candidate.ID == asset.ID {
			continue
		}
		if candidate.Kind == energyFlowParkingKind && candidate.Metadata["hidden"] == "true" {
			continue
		}
		orderedIDs = append(orderedIDs, candidate.ID)
	}
	priority, parseErr := strconv.Atoi(strings.TrimSpace(r.FormValue("priority")))
	if parseErr != nil || priority < 1 {
		priority = previousPriority
	}
	if priority > len(orderedIDs)+1 {
		priority = len(orderedIDs) + 1
	}
	insertAt := priority - 1
	orderedIDs = append(orderedIDs, "")
	copy(orderedIDs[insertAt+1:], orderedIDs[insertAt:])
	orderedIDs[insertAt] = asset.ID
	if written, err := a.energyFor(ac).UpdateAssetPriorities(ac.tenant.Slug, orderedIDs); err != nil || !written {
		http.Error(w, "Priorität konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	action, summary, notice := "energy.consumer.add", "Verbraucher angelegt", "1"
	if assetID != "" {
		action, summary, notice = "energy.consumer.update", "Verbraucher bearbeitet", "gespeichert"
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     action,
		TargetType: "energy-asset",
		TargetID:   asset.ID,
		Summary:    summary,
		Details: map[string]string{
			"kind": asset.Kind, "flexibility": asset.Flexibility,
			"icon": asset.Metadata["icon"], "priority": strconv.Itoa(priority),
			"power_entity":        strings.TrimSpace(r.FormValue("consumer_power_entity")),
			"energy_entity":       strings.TrimSpace(r.FormValue("consumer_energy_entity")),
			"soc_entity":          strings.TrimSpace(r.FormValue("consumer_soc_entity")),
			"sleep_entity":        strings.TrimSpace(r.FormValue("consumer_sleep_entity")),
			"stale_after_minutes": asset.Metadata["stale_after_minutes"],
		},
	})
	http.Redirect(w, r, "/app/energie?verbraucher="+notice, http.StatusSeeOther)
}

// deleteEnergyConsumer removes any consumer chosen in the cockpit. PV and
// battery assets are still protected because they are managed as system
// assets. Home Assistant entities themselves are never changed; only their
// HAUSV assignment is removed.
func (a *app) deleteEnergyConsumer(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	assetID := strings.TrimSpace(r.FormValue("asset_id"))
	assets, err := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Verbraucher konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	mappings, _ := a.energyFor(ac).ListMappings(ac.tenant.Slug)
	parkingID := energyFlowNodeID(ac.tenant.Slug, "parking")
	if assetID == parkingID {
		asset := energyFlowNodeAsset(assets, ac.tenant.Slug, "parking")
		if asset.Metadata == nil {
			asset.Metadata = map[string]string{}
		}
		asset.Metadata["hidden"] = "true"
		asset.Source = energyFlowNodeAssetSource
		if err := a.energyFor(ac).UpsertAsset(asset); err != nil {
			http.Error(w, "Verbraucher konnte nicht entfernt werden.", http.StatusInternalServerError)
			return
		}
		for _, mapping := range mappings {
			if mapping.AssetID == assetID {
				_, _ = a.energyFor(ac).DeleteMapping(ac.tenant.Slug, mapping.ID)
			}
		}
		a.recordAudit(auditEvent{
			TenantSlug: ac.tenant.Slug, ActorEmail: ac.email, ActorRole: ac.role,
			Action: "energy.consumer.remove", TargetType: "energy-asset", TargetID: assetID,
			Summary: "Verbraucher entfernt", Details: map[string]string{"node_type": "parking"},
		})
		http.Redirect(w, r, "/app/energie?verbraucher=weg", http.StatusSeeOther)
		return
	}
	for _, asset := range assets {
		if asset.ID != assetID {
			continue
		}
		if isEnergyFlowSystemKind(asset.Kind) {
			http.Redirect(w, r, "/app/energie", http.StatusSeeOther)
			return
		}
		if _, err := a.energyFor(ac).DeleteAsset(ac.tenant.Slug, assetID); err != nil {
			http.Error(w, "Verbraucher konnte nicht entfernt werden.", http.StatusInternalServerError)
			return
		}
		for _, mapping := range mappings {
			if mapping.AssetID == assetID {
				_, _ = a.energyFor(ac).DeleteMapping(ac.tenant.Slug, mapping.ID)
			}
		}
		a.recordAudit(auditEvent{
			TenantSlug: ac.tenant.Slug,
			ActorEmail: ac.email,
			ActorRole:  ac.role,
			Action:     "energy.consumer.remove",
			TargetType: "energy-asset",
			TargetID:   assetID,
			Summary:    "Verbraucher entfernt",
		})
		http.Redirect(w, r, "/app/energie?verbraucher=weg", http.StatusSeeOther)
		return
	}
	// Unbekannte ID: nicht als Fehler behandeln, aber auch nichts löschen —
	// die Liste gehört einem anderen Haus oder ist bereits entfernt.
	http.Redirect(w, r, "/app/energie", http.StatusSeeOther)
}

func energyConsumerKind(raw string) string {
	kind := strings.ToLower(strings.TrimSpace(raw))
	switch kind {
	case "ev", "wallbox", "heat-pump", "hot-water",
		"sauna", "air-conditioning", "instant-water-heater":
		return kind
	default:
		return "other"
	}
}

func energyConsumerFlexibility(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case energy.FlexShift:
		return energy.FlexShift
	case energy.FlexThrottle:
		return energy.FlexThrottle
	case energy.FlexFixed:
		return energy.FlexFixed
	default:
		// Unbekannt heißt: zählt in den Verbrauch, aber nicht in die
		// Flexibilität. Lieber nichts versprechen als zu viel.
		return energy.FlexUnknown
	}
}

func (a *app) saveOnboardingAssets(tenantSlug string, selected []string) error {
	allowed := map[string]struct{}{
		"ev": {}, "wallbox": {}, "heat-pump": {}, "hot-water": {}, "pv": {},
		"battery": {}, "sauna": {}, "air-conditioning": {}, "instant-water-heater": {}, "other": {},
	}
	selectedSet := map[string]struct{}{}
	for _, raw := range selected {
		kind := strings.TrimSpace(raw)
		if _, ok := allowed[kind]; ok {
			selectedSet[kind] = struct{}{}
		}
	}
	for kind := range allowed {
		id := energy.StableAssetID(tenantSlug, kind)
		if _, keep := selectedSet[kind]; !keep {
			if _, err := a.energyStore.DeleteAsset(tenantSlug, id); err != nil {
				return err
			}
			// Remove the single-home legacy identity after upgrading. Tenant
			// scoping prevents this compatibility cleanup touching another home.
			if _, err := a.energyStore.DeleteAsset(tenantSlug, "asset-"+kind); err != nil {
				return err
			}
		}
	}
	current, err := a.energyStore.ListAssets(tenantSlug)
	if err != nil {
		return err
	}
	existing := map[string]energy.Asset{}
	for _, asset := range current {
		existing[asset.ID] = asset
	}
	for kind := range selectedSet {
		id := energy.StableAssetID(tenantSlug, kind)
		asset := energy.Asset{
			ID:          id,
			TenantSlug:  tenantSlug,
			Kind:        kind,
			Name:        energy.AssetKindLabel(kind),
			Flexibility: defaultAssetFlexibility(kind),
			Source:      "onboarding",
			Confirmed:   true,
		}
		// Dieser Schritt wählt nur die Arten aus. Was darüber hinaus erfasst ist
		// — Nennleistung, erklärte Flexibilität, Herkunft —, überlebt das
		// erneute Speichern: ein Seed mit 9 kW verlor sie sonst stillschweigend
		// und fiel auf die 1 kW der Vorbelegung zurück.
		if previous, ok := existing[id]; ok {
			asset.Name = previous.Name
			asset.RatedPowerKW = previous.RatedPowerKW
			asset.Source = previous.Source
			// Metadata trägt u.a. die Rail-Priorität (HAUSV-441) und darf ein
			// erneutes Speichern der Geräteauswahl nicht verlieren.
			asset.Metadata = previous.Metadata
			if previous.Flexibility != "" && previous.Flexibility != energy.FlexUnknown {
				asset.Flexibility = previous.Flexibility
			}
		}
		if err := a.energyStore.UpsertAsset(asset); err != nil {
			return err
		}
		// Pre-multi-home versions used asset-<kind>. The migration handles
		// durable SQL data; this also keeps in-memory/JSON-like test stores clean.
		if _, err := a.energyStore.DeleteAsset(tenantSlug, "asset-"+kind); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) saveSelectedEnergyMappings(ctx context.Context, tenant tenantConfig, selected []string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	states, configured, err := a.energyStates(timeoutCtx, tenant)
	if err != nil {
		return err
	}
	if !configured {
		return nil
	}
	selectedSet := map[string]struct{}{}
	for _, entityID := range selected {
		selectedSet[strings.ToLower(strings.TrimSpace(entityID))] = struct{}{}
	}
	existing, _ := a.energyStore.ListMappings(tenant.Slug)
	assets, _ := a.energyStore.ListAssets(tenant.Slug)
	existingByEntity := map[string]energy.EntityMapping{}
	for _, mapping := range existing {
		existingByEntity[strings.ToLower(mapping.EntityID)] = mapping
		if _, keep := selectedSet[mapping.EntityID]; !keep {
			if _, deleteErr := a.energyStore.DeleteMapping(tenant.Slug, mapping.ID); deleteErr != nil {
				return deleteErr
			}
		}
	}
	for _, state := range states {
		if _, keep := selectedSet[strings.ToLower(state.EntityID)]; !keep {
			continue
		}
		candidate, ok := classifyHAState(state)
		if !ok {
			continue
		}
		if confirmed, found := existingByEntity[strings.ToLower(candidate.EntityID)]; found && confirmed.Confirmed {
			// A repeated discovery must never replace a user's confirmed or
			// manually corrected meaning.
			continue
		}
		seen := candidate.LastUpdated
		if err := a.energyStore.UpsertMapping(energy.EntityMapping{
			TenantSlug:  tenant.Slug,
			EntityID:    candidate.EntityID,
			AssetID:     inferredEnergyAssetID(candidate.Metric, assets),
			Metric:      candidate.Metric,
			DisplayName: candidate.DisplayName,
			Unit:        candidate.Unit,
			DeviceClass: candidate.DeviceClass,
			Confirmed:   true,
			LastSeenAt:  &seen,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) saveManualEnergyMapping(tenantSlug, entityID, metric, displayName, unit, assetID string) error {
	entityID = strings.ToLower(strings.TrimSpace(entityID))
	metric = strings.TrimSpace(metric)
	if !strings.HasPrefix(entityID, "sensor.") {
		return fmt.Errorf("only sensor entities may be mapped")
	}
	switch metric {
	case energy.MetricGridImportPower, energy.MetricGridImportEnergy, energy.MetricGridExportPower,
		energy.MetricPVPower, energy.MetricBatteryPower, energy.MetricBatteryCharge,
		energy.MetricBatteryDischarge, energy.MetricBatterySOC, energy.MetricLoadPower:
	default:
		return fmt.Errorf("unsupported metric")
	}
	displayName = cleanEnergyText(displayName, 120)
	if displayName == "" {
		displayName = entityID
	}
	return a.energyStore.UpsertMapping(energy.EntityMapping{
		TenantSlug:  tenantSlug,
		EntityID:    entityID,
		AssetID:     strings.TrimSpace(assetID),
		Metric:      metric,
		DisplayName: displayName,
		Unit:        cleanEnergyText(unit, 24),
		Confirmed:   true,
	})
}

func inferredEnergyAssetID(metric string, assets []energy.Asset) string {
	kind := ""
	switch metric {
	case energy.MetricPVPower:
		kind = "pv"
	case energy.MetricBatteryPower, energy.MetricBatteryCharge, energy.MetricBatteryDischarge, energy.MetricBatterySOC:
		kind = "battery"
	default:
		return ""
	}
	match := ""
	for _, asset := range assets {
		if asset.Kind != kind {
			continue
		}
		if match != "" {
			return ""
		}
		match = asset.ID
	}
	return match
}

func discoverEnergyCandidates(ctx context.Context, cfg homeassistant.Config, mappings []energy.EntityMapping) (energyDiscoveryView, error) {
	states, err := cfg.States(ctx)
	if err != nil {
		return energyDiscoveryView{}, err
	}
	return discoverEnergyCandidatesFromStates(states, mappings)
}

func discoverEnergyCandidatesFromStates(states []homeassistant.EntityState, mappings []energy.EntityMapping) (energyDiscoveryView, error) {
	confirmed := map[string]bool{}
	confirmedMappings := map[string]energy.EntityMapping{}
	for _, mapping := range mappings {
		entityID := strings.ToLower(mapping.EntityID)
		confirmed[entityID] = mapping.Confirmed
		if mapping.Confirmed {
			confirmedMappings[entityID] = mapping
		}
	}
	candidates := []energy.EntityCandidate{}
	for _, state := range states {
		if candidate, ok := classifyHAState(state); ok {
			candidates = append(candidates, candidate)
		} else if mapping, found := confirmedMappings[strings.ToLower(state.EntityID)]; found {
			value, parseErr := homeassistant.ParseFloat(state.State)
			var valuePtr *float64
			if parseErr == nil {
				valuePtr = &value
			}
			candidates = append(candidates, energy.EntityCandidate{
				EntityID:    strings.ToLower(state.EntityID),
				DisplayName: firstNonEmpty(mapping.DisplayName, haAttribute(state.Attributes, "friendly_name"), state.EntityID),
				Unit:        firstNonEmpty(mapping.Unit, haAttribute(state.Attributes, "unit_of_measurement")),
				DeviceClass: firstNonEmpty(mapping.DeviceClass, haAttribute(state.Attributes, "device_class")),
				Metric:      mapping.Metric,
				Value:       valuePtr,
				LastUpdated: state.LastUpdated,
			})
		}
	}
	energy.SortCandidates(candidates)
	sort.SliceStable(candidates, func(i, j int) bool {
		leftConfirmed := confirmed[candidates[i].EntityID]
		rightConfirmed := confirmed[candidates[j].EntityID]
		if leftConfirmed != rightConfirmed {
			return leftConfirmed
		}
		left := energyCandidatePriority(candidates[i])
		right := energyCandidatePriority(candidates[j])
		if left != right {
			return left < right
		}
		return candidates[i].EntityID < candidates[j].EntityID
	})
	hasConfirmed := false
	for _, value := range confirmed {
		hasConfirmed = hasConfirmed || value
	}
	recommended := make([]energy.EntityCandidate, 0, 5)
	additional := make([]energy.EntityCandidate, 0, 8)
	recommendedEntities := map[string]bool{}
	recommendedMetrics := map[string]bool{}
	for _, candidate := range candidates {
		if len(recommended) >= 5 || !confirmed[candidate.EntityID] {
			continue
		}
		recommended = append(recommended, candidate)
		recommendedEntities[candidate.EntityID] = true
		recommendedMetrics[candidate.Metric] = true
	}
	for _, candidate := range candidates {
		if len(recommended) >= 5 {
			break
		}
		if recommendedEntities[candidate.EntityID] || recommendedMetrics[candidate.Metric] {
			continue
		}
		recommended = append(recommended, candidate)
		recommendedEntities[candidate.EntityID] = true
		recommendedMetrics[candidate.Metric] = true
	}
	for _, candidate := range candidates {
		if recommendedEntities[candidate.EntityID] || len(additional) >= 8 {
			continue
		}
		additional = append(additional, candidate)
	}
	toView := func(items []energy.EntityCandidate, isRecommended bool) []energyCandidateView {
		out := make([]energyCandidateView, 0, len(items))
		for _, candidate := range items {
			value := "–"
			if candidate.Value != nil {
				value = formatEnergyNumber(*candidate.Value)
			}
			metricLabel := energyMetricLabel(candidate.Metric)
			out = append(out, energyCandidateView{
				EntityID:    candidate.EntityID,
				DisplayName: metricLabel,
				SourceName:  firstNonEmpty(candidate.DisplayName, candidate.EntityID),
				Metric:      candidate.Metric,
				MetricLabel: metricLabel,
				Value:       value,
				Unit:        candidate.Unit,
				Checked:     confirmed[candidate.EntityID] || (isRecommended && !hasConfirmed),
			})
		}
		return out
	}
	return energyDiscoveryView{
		Recommended: toView(recommended, true),
		Additional:  toView(additional, false),
	}, nil
}

func energyCandidatePriority(candidate energy.EntityCandidate) int {
	name := strings.ToLower(candidate.DisplayName + " " + candidate.EntityID)
	score := 50
	preferred := []string{
		"grid_import_power",
		"netzbezug",
		"grid_import_energy",
		"grid_export_power",
		"netzeinspeisung",
		"pv_current_power",
		"current_power",
		"home_battery_soc",
		"battery_percentage",
		"home_consumption",
		"consumption_current",
		"battery_charge_power",
		"battery_discharge_power",
	}
	for index, marker := range preferred {
		if strings.Contains(name, marker) {
			score = index
			break
		}
	}
	if strings.Contains(name, "daily") || strings.Contains(name, "monthly") || strings.Contains(name, "yearly") {
		score += 30
	}
	return score
}

func classifyHAState(state homeassistant.EntityState) (energy.EntityCandidate, bool) {
	return energy.ClassifyCandidate(
		state.EntityID,
		haAttribute(state.Attributes, "friendly_name"),
		haAttribute(state.Attributes, "unit_of_measurement"),
		haAttribute(state.Attributes, "device_class"),
		haAttribute(state.Attributes, "state_class"),
		state.State,
		state.LastUpdated,
	)
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

// ── energy flow config (HAUSV-439) ──────────────────────────────────────────
// JSON payload for the client-side flow renderer (assets/energy-flow.js).
// Every display string is pre-formatted here; the kw numerics only drive
// ribbon widths and the proportional destination bands.

type energyFlowNodeConfig struct {
	ID             string                      `json:"id,omitempty"`
	NodeType       string                      `json:"nodeType,omitempty"`
	Icon           string                      `json:"icon,omitempty"`
	Label          string                      `json:"label,omitempty"`
	Value          string                      `json:"value,omitempty"`
	Unit           string                      `json:"unit,omitempty"`
	KW             float64                     `json:"kw"`
	Dir            string                      `json:"dir,omitempty"`
	Mode           string                      `json:"mode,omitempty"`
	Sub            string                      `json:"sub,omitempty"`
	Secondary      string                      `json:"secondary,omitempty"`
	SecondaryLabel string                      `json:"secondaryLabel,omitempty"`
	Metrics        []energyFlowMetricConfig    `json:"metrics,omitempty"`
	Color          string                      `json:"color,omitempty"`
	Flow           float64                     `json:"flow,omitempty"`
	Hover          string                      `json:"hover,omitempty"`
	Editable       bool                        `json:"editable,omitempty"`
	Measurements   []energyFlowMeasurementSlot `json:"measurements,omitempty"`
}

type energyFlowMetricConfig struct {
	Label string `json:"label,omitempty"`
	Value string `json:"value,omitempty"`
	Unit  string `json:"unit,omitempty"`
}

type energyFlowMeasurementSlot struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	Metric   string `json:"-"`
	EntityID string `json:"entity,omitempty"`
}

type energyFlowConsumerConfig struct {
	ID                string                      `json:"id,omitempty"`
	Icon              string                      `json:"icon,omitempty"`
	Title             string                      `json:"title"`
	Kind              string                      `json:"kind,omitempty"`
	RatedPower        string                      `json:"ratedPower,omitempty"`
	Flexibility       string                      `json:"flexibility,omitempty"`
	PowerEntity       string                      `json:"powerEntity,omitempty"`
	EnergyEntity      string                      `json:"energyEntity,omitempty"`
	Secondary         string                      `json:"secondary,omitempty"`
	SecondaryLabel    string                      `json:"secondaryLabel,omitempty"`
	Primary           *energyFlowMetricConfig     `json:"primary,omitempty"`
	Current           *energyFlowMetricConfig     `json:"current,omitempty"`
	Metrics           []energyFlowMetricConfig    `json:"metrics"`
	Color             string                      `json:"color,omitempty"`
	State             string                      `json:"state,omitempty"`
	DataStatus        string                      `json:"dataStatus,omitempty"`
	DataLabel         string                      `json:"dataLabel,omitempty"`
	Age               string                      `json:"age,omitempty"`
	StaleAfterMinutes int                         `json:"staleAfterMinutes,omitempty"`
	KW                float64                     `json:"kw"`
	Active            bool                        `json:"active,omitempty"`
	NodeType          string                      `json:"nodeType,omitempty"`
	Deletable         bool                        `json:"deletable,omitempty"`
	Measurements      []energyFlowMeasurementSlot `json:"measurements,omitempty"`
	Priority          int                         `json:"-"`
}

type energyFlowConfig struct {
	Home      energyFlowNodeConfig       `json:"home"`
	Producers []energyFlowNodeConfig     `json:"producers,omitempty"`
	Storage   *energyFlowNodeConfig      `json:"storage,omitempty"`
	Grid      *energyFlowNodeConfig      `json:"grid,omitempty"`
	Consumers []energyFlowConsumerConfig `json:"consumers,omitempty"`
	AddHint   bool                       `json:"addHint,omitempty"`
}

// formatEnergyFlowKW renders a power reading in kW without the unit suffix —
// the flow renderer appends the small raised unit marker itself. Always kW by
// decision (HAUSV-439); W vs kW may become a user setting later.
func formatEnergyFlowKW(watts float64) string {
	kw := watts / 1000
	digits := 2
	if math.Abs(kw) >= 10 {
		digits = 1
	}
	out := formatEnergyCompact(kw, digits)
	if out == "-0" {
		return "0"
	}
	return out
}

// energyConsumerPriority reads the stored rail position of an asset (kept in
// the metadata map so no schema change was needed, HAUSV-441). Entries without
// one sort behind every prioritised entry in their existing stable order.
func energyConsumerPriority(asset energy.Asset) int {
	value, err := strconv.Atoi(asset.Metadata["priority"])
	if err != nil || value <= 0 {
		return math.MaxInt32
	}
	return value
}

func defaultEnergyFlowColor(kind string) string {
	switch kind {
	case energyFlowHomeKind:
		return "#a24b42"
	case "pv":
		return "#4f7d49"
	case "battery":
		return "#7893a1"
	case energyFlowGridKind:
		return "#b8891f"
	default:
		return "#3e704c"
	}
}

func normalizeEnergyFlowColor(raw, kind string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if len(raw) == 7 && raw[0] == '#' {
		for _, char := range raw[1:] {
			if !strings.ContainsRune("0123456789abcdef", char) {
				return defaultEnergyFlowColor(kind)
			}
		}
		return raw
	}
	return defaultEnergyFlowColor(kind)
}

func defaultEnergySecondaryLabel(nodeType string) string {
	switch nodeType {
	case "home":
		return "Hausenergie heute"
	case "pv":
		return "PV-Ertrag heute"
	case "grid":
		return "Netzenergie heute"
	case "storage":
		return "Ladestand"
	case "parking":
		return "Ladeenergie"
	default:
		return "Energie"
	}
}

func energyAssetSecondaryLabel(asset energy.Asset, nodeType string) string {
	return firstNonEmpty(cleanEnergyText(asset.Metadata["secondary_label"], 60), defaultEnergySecondaryLabel(nodeType))
}

func energyAssetFlowColor(asset energy.Asset) string {
	return normalizeEnergyFlowColor(asset.Metadata["color"], asset.Kind)
}

func joinEnergySecondaryValues(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "–" {
			parts = append(parts, value)
		}
	}
	if len(parts) == 0 {
		return "–"
	}
	return strings.Join(parts, " · ")
}

func energyFlowMetric(label, formatted string) energyFlowMetricConfig {
	formatted = strings.TrimSpace(formatted)
	metric := energyFlowMetricConfig{Label: strings.TrimSpace(label), Value: formatted}
	if split := strings.LastIndex(formatted, "\u00a0"); split > 0 {
		metric.Value = formatted[:split]
		metric.Unit = strings.TrimSpace(formatted[split+len("\u00a0"):])
	}
	return metric
}

func energyFlowMetricPtr(label, formatted string) *energyFlowMetricConfig {
	metric := energyFlowMetric(label, formatted)
	if metric.Value == "" || metric.Value == "–" {
		return nil
	}
	return &metric
}

func energyConsumerAgeLabel(now, seen time.Time) string {
	if seen.IsZero() {
		return ""
	}
	age := now.Sub(seen)
	if age < 0 {
		age = 0
	}
	switch {
	case age < time.Minute:
		return fmt.Sprintf("vor %d Sek.", int(age/time.Second))
	case age < time.Hour:
		return fmt.Sprintf("vor %d Min.", int(age/time.Minute))
	case age < 24*time.Hour:
		return fmt.Sprintf("vor %d Std.", int(age/time.Hour))
	default:
		days := int(age / (24 * time.Hour))
		if days == 1 {
			return "vor 1 Tag"
		}
		return fmt.Sprintf("vor %d Tagen", days)
	}
}

func energyFlowConsumerSlots(asset energy.Asset, entities map[string]string) []energyFlowMeasurementSlot {
	slots := []energyFlowMeasurementSlot{
		{Name: "consumer_power_entity", Label: "Aktuelle Leistung", Kind: "power", EntityID: entities["power"]},
		{Name: "consumer_energy_entity", Label: "Energiezähler", Kind: "energy", EntityID: entities["energy"]},
		{Name: "consumer_soc_entity", Label: "Ladestand", Kind: "percentage", EntityID: entities["soc"]},
	}
	if asset.Kind == "ev" {
		slots = append(slots, energyFlowMeasurementSlot{
			Name: "consumer_sleep_entity", Label: "Schlafsignal", Kind: "sleep", EntityID: entities["sleep"],
		})
	}
	return slots
}

func energyFlowAssetMetric(metrics []energyMetricView, assetID, metric string) (energyMetricView, bool) {
	for _, reading := range metrics {
		if reading.AssetID == assetID && reading.Metric == metric {
			return reading, true
		}
	}
	return energyMetricView{}, false
}

func sortEnergyConsumers(assets []energy.Asset) []energy.Asset {
	out := append([]energy.Asset(nil), assets...)
	sort.SliceStable(out, func(i, j int) bool {
		return energyConsumerPriority(out[i]) < energyConsumerPriority(out[j])
	})
	return out
}

// reorderEnergyConsumers persists the rail order chosen by dragging or the
// keyboard (HAUSV-441). The client posts every visible consumer asset id in
// its new order; the configured parking charger follows the same contract.
func (a *app) reorderEnergyConsumers(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canManageEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	order := r.Form["order"]
	assets, err := a.energyFor(ac).ListAssets(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Energiedaten konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	parkingID := energyFlowNodeID(ac.tenant.Slug, "parking")
	for _, id := range order {
		if id != parkingID {
			continue
		}
		found := false
		for _, asset := range assets {
			found = found || asset.ID == parkingID
		}
		if !found && ac.tenant.HA.ChargingConfigured() {
			parking := energyFlowNodeAsset(assets, ac.tenant.Slug, "parking")
			if err := a.energyFor(ac).UpsertAsset(parking); err != nil {
				http.Error(w, "Reihenfolge konnte nicht gespeichert werden.", http.StatusInternalServerError)
				return
			}
			assets = append(assets, parking)
		}
	}
	byID := map[string]energy.Asset{}
	for _, asset := range assets {
		if isEnergyFlowSystemKind(asset.Kind) && asset.Kind != energyFlowParkingKind {
			continue
		}
		if asset.Kind == energyFlowParkingKind && asset.Metadata["hidden"] == "true" {
			continue
		}
		byID[asset.ID] = asset
	}
	if len(order) != len(byID) {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	seen := map[string]bool{}
	for _, id := range order {
		if _, ok := byID[id]; !ok || seen[id] {
			http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
			return
		}
		seen[id] = true
	}
	// Ein atomarer Schreibschritt: verschwindet ein Verbraucher zwischen
	// Validierung und Schreiben (paralleles Löschen), wird nichts geändert —
	// und ein Upsert könnte ihn auch nicht wieder auferstehen lassen.
	written, err := a.energyFor(ac).UpdateAssetPriorities(ac.tenant.Slug, order)
	if err != nil {
		http.Error(w, "Reihenfolge konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	if !written {
		http.Error(w, "Ungültige Eingabe", http.StatusConflict)
		return
	}
	a.recordAudit(auditEvent{
		TenantSlug: ac.tenant.Slug,
		ActorEmail: ac.email,
		ActorRole:  ac.role,
		Action:     "energy.consumer.reorder",
		TargetType: "energy-asset",
		TargetID:   strings.Join(order, ","),
		Summary:    "Verbraucher-Priorität geändert",
		Details:    map[string]string{"count": strconv.Itoa(len(order))},
	})
	w.WriteHeader(http.StatusNoContent)
}

func energyFlowMappingEntity(mappings []energy.EntityMapping, assetID, metric string) string {
	for _, mapping := range mappings {
		if !mapping.Confirmed || !energyFlowMappingMatchesMetric(mapping, metric) {
			continue
		}
		if mapping.AssetID == assetID || (mapping.AssetID == "" && (metric == energy.MetricLoadPower || metric == energy.MetricGridImportPower || metric == energy.MetricGridExportPower)) {
			return mapping.EntityID
		}
	}
	return ""
}

func energyFlowMappingMatchesMetric(mapping energy.EntityMapping, wanted string) bool {
	if mapping.Metric != energy.MetricBatteryPower {
		return mapping.Metric == wanted
	}
	legacyKind := energyMetricKind(mapping.Metric, mapping.DisplayName+" "+mapping.EntityID)
	switch wanted {
	case energy.MetricBatteryCharge:
		return legacyKind == "battery-charge"
	case energy.MetricBatteryDischarge:
		return legacyKind == "battery-discharge"
	case energy.MetricBatteryPower:
		return legacyKind == energy.MetricBatteryPower
	default:
		return false
	}
}

func energyFlowNodeAsset(assets []energy.Asset, tenantSlug, nodeType string) energy.Asset {
	wantedKind := ""
	defaultName, defaultIcon := "", "plug"
	switch nodeType {
	case "home":
		wantedKind, defaultName, defaultIcon = energyFlowHomeKind, "Hausverbrauch", "house"
	case "grid":
		wantedKind, defaultName, defaultIcon = energyFlowGridKind, "Netz", "utility-pole"
	case "pv":
		wantedKind, defaultName, defaultIcon = "pv", "PV-Leistung", "solar-panel"
	case "storage":
		wantedKind, defaultName, defaultIcon = "battery", "Speicher", "battery"
	case "parking":
		wantedKind, defaultName, defaultIcon = energyFlowParkingKind, "Parkplatz 20", "square-parking"
	}
	for _, asset := range assets {
		if asset.Kind != wantedKind {
			continue
		}
		metadata := map[string]string{}
		for key, value := range asset.Metadata {
			metadata[key] = value
		}
		asset.Metadata = metadata
		if strings.TrimSpace(asset.Name) == "" {
			asset.Name = defaultName
		}
		asset.Metadata["icon"] = normalizeEnergyConsumerIcon(asset.Metadata["icon"], wantedKind)
		if asset.Metadata["icon"] == "plug" && defaultIcon != "plug" && strings.TrimSpace(asset.Metadata["icon_configured"]) == "" {
			asset.Metadata["icon"] = defaultIcon
		}
		return asset
	}
	id := energyFlowNodeID(tenantSlug, nodeType)
	if nodeType == "pv" || nodeType == "storage" {
		id = energy.StableAssetID(tenantSlug, wantedKind)
	}
	return energy.Asset{
		ID: id, TenantSlug: tenantSlug, Kind: wantedKind, Name: defaultName,
		Source: energyFlowNodeAssetSource, Confirmed: true,
		Metadata: map[string]string{"icon": defaultIcon},
	}
}

func energyFlowNodeSlots(nodeType, assetID string, mappings []energy.EntityMapping) []energyFlowMeasurementSlot {
	definitions := []energyFlowMeasurementSlot{}
	switch nodeType {
	case "home":
		definitions = append(definitions,
			energyFlowMeasurementSlot{Name: "load_power_entity", Label: "Aktueller Hausverbrauch", Kind: "power", Metric: energy.MetricLoadPower},
			energyFlowMeasurementSlot{Name: "secondary_energy_entity", Label: "Hausenergie heute", Kind: "energy", Metric: energy.MetricConsumerEnergy})
	case "grid":
		definitions = append(definitions,
			energyFlowMeasurementSlot{Name: "grid_import_entity", Label: "Netzbezug", Kind: "power", Metric: energy.MetricGridImportPower},
			energyFlowMeasurementSlot{Name: "grid_export_entity", Label: "Netzeinspeisung", Kind: "power", Metric: energy.MetricGridExportPower},
			energyFlowMeasurementSlot{Name: "secondary_energy_entity", Label: "Netzenergie heute", Kind: "energy", Metric: energy.MetricConsumerEnergy})
	case "pv":
		definitions = append(definitions,
			energyFlowMeasurementSlot{Name: "pv_power_entity", Label: "Aktuelle PV-Leistung", Kind: "power", Metric: energy.MetricPVPower},
			energyFlowMeasurementSlot{Name: "secondary_energy_entity", Label: "PV-Ertrag heute", Kind: "energy", Metric: energy.MetricConsumerEnergy})
	case "storage":
		definitions = append(definitions,
			energyFlowMeasurementSlot{Name: "battery_power_entity", Label: "Nettoleistung mit Vorzeichen (alternativ)", Kind: "power", Metric: energy.MetricBatteryPower},
			energyFlowMeasurementSlot{Name: "battery_charge_entity", Label: "Ladeleistung", Kind: "power", Metric: energy.MetricBatteryCharge},
			energyFlowMeasurementSlot{Name: "battery_discharge_entity", Label: "Entladeleistung", Kind: "power", Metric: energy.MetricBatteryDischarge},
			energyFlowMeasurementSlot{Name: "battery_soc_entity", Label: "Ladestand", Kind: "percentage", Metric: energy.MetricBatterySOC},
			energyFlowMeasurementSlot{Name: "secondary_energy_entity", Label: "Speicherenergie heute (optional)", Kind: "energy", Metric: energy.MetricConsumerEnergy})
	case "parking":
		definitions = append(definitions,
			energyFlowMeasurementSlot{Name: "consumer_power_entity", Label: "Aktuelle Ladeleistung", Kind: "power", Metric: energy.MetricConsumerPower},
			energyFlowMeasurementSlot{Name: "consumer_energy_entity", Label: "Ladeenergiezähler", Kind: "energy", Metric: energy.MetricConsumerEnergy},
			energyFlowMeasurementSlot{Name: "consumer_soc_entity", Label: "Ladestand", Kind: "percentage", Metric: energy.MetricBatterySOC})
	}
	for index := range definitions {
		definitions[index].EntityID = energyFlowMappingEntity(mappings, assetID, definitions[index].Metric)
	}
	return definitions
}

func buildEnergyFlowConfig(tenantSlug string, live energyLiveView, assets []energy.Asset, mappings []energy.EntityMapping, metrics []energyMetricView, charging parkingLiveView, canManage bool) energyFlowConfig {
	return buildEnergyFlowConfigAt(tenantSlug, live, assets, mappings, metrics, charging, canManage, time.Now())
}

func buildEnergyFlowConfigAt(tenantSlug string, live energyLiveView, assets []energy.Asset, mappings []energy.EntityMapping, metrics []energyMetricView, charging parkingLiveView, canManage bool, now time.Time) energyFlowConfig {
	cfg := energyFlowConfig{AddHint: canManage}
	homeAsset := energyFlowNodeAsset(assets, tenantSlug, "home")
	cfg.Home = energyFlowNodeConfig{
		ID: homeAsset.ID, NodeType: "home", Icon: energyConsumerIcon(homeAsset),
		Value: "–", Label: homeAsset.Name, Secondary: "–",
		SecondaryLabel: energyAssetSecondaryLabel(homeAsset, "home"), Color: energyAssetFlowColor(homeAsset), Editable: canManage,
		Measurements: energyFlowNodeSlots("home", homeAsset.ID, mappings),
	}
	if reading, ok := energyFlowAssetMetric(metrics, homeAsset.ID, energy.MetricConsumerEnergy); ok {
		cfg.Home.Secondary = reading.Value
		cfg.Home.Metrics = append(cfg.Home.Metrics, energyFlowMetric(cfg.Home.SecondaryLabel, reading.Value))
	}
	if live.HasMain {
		watts := math.Abs(energyPowerWatts(&live.Main))
		cfg.Home.Value, cfg.Home.Unit, cfg.Home.KW = formatEnergyFlowKW(watts), "kW", watts/1000
	}
	pvAsset := energyFlowNodeAsset(assets, tenantSlug, "pv")
	for index := range live.Flows {
		item := live.Flows[index]
		if item.Metric != energy.MetricPVPower {
			continue
		}
		// Negative inverter standby draw is not production: clamp to zero so
		// no producer ribbon appears at night.
		watts := math.Max(0, energyPowerWatts(&item))
		cfg.Producers = append(cfg.Producers, energyFlowNodeConfig{
			ID: pvAsset.ID, NodeType: "pv", Icon: energyConsumerIcon(pvAsset), Label: pvAsset.Name, Editable: canManage,
			Value: formatEnergyFlowKW(watts), Unit: "kW", KW: watts / 1000, Secondary: "–",
			SecondaryLabel: energyAssetSecondaryLabel(pvAsset, "pv"), Color: energyAssetFlowColor(pvAsset),
			Measurements: energyFlowNodeSlots("pv", pvAsset.ID, mappings),
		})
		if reading, ok := energyFlowAssetMetric(metrics, pvAsset.ID, energy.MetricConsumerEnergy); ok {
			cfg.Producers[len(cfg.Producers)-1].Secondary = reading.Value
			cfg.Producers[len(cfg.Producers)-1].Metrics = append(
				cfg.Producers[len(cfg.Producers)-1].Metrics,
				energyFlowMetric(cfg.Producers[len(cfg.Producers)-1].SecondaryLabel, reading.Value),
			)
		}
	}
	storageAsset := energyFlowNodeAsset(assets, tenantSlug, "storage")
	storageSOC, hasStorageSOC := energyFlowAssetMetric(metrics, storageAsset.ID, energy.MetricBatterySOC)
	if !hasStorageSOC && live.HasBatterySOC {
		storageSOC, hasStorageSOC = live.BatterySOC, true
	}
	if live.HasBattery || hasStorageSOC {
		watts := math.Abs(energyPowerWatts(&live.Battery))
		mode := "wartet"
		if live.HasBattery && watts >= 1 {
			switch live.Battery.Direction {
			case "charging":
				mode = "lädt"
			case "discharging":
				mode = "entlädt"
			}
		}
		flow := watts / 1000
		if mode == "wartet" {
			// Below the 1 W threshold no direction is claimed — draw no ribbon.
			flow = 0
		}
		cfg.Storage = &energyFlowNodeConfig{
			ID: storageAsset.ID, NodeType: "storage", Icon: energyConsumerIcon(storageAsset), Label: storageAsset.Name,
			Editable: canManage, Measurements: energyFlowNodeSlots("storage", storageAsset.ID, mappings),
			Value: formatEnergyFlowKW(watts), Unit: "kW", Secondary: "–",
			SecondaryLabel: energyAssetSecondaryLabel(storageAsset, "storage"), Color: energyAssetFlowColor(storageAsset),
			Mode: mode, Flow: flow, Sub: storageAsset.Name + " · " + mode,
		}
		if hasStorageSOC {
			cfg.Storage.Secondary = storageSOC.Value
			socLabel := cleanEnergyText(storageAsset.Metadata["secondary_label"], 60)
			if socLabel == "" {
				socLabel = "Ladestand"
			}
			cfg.Storage.Metrics = append(cfg.Storage.Metrics, energyFlowMetric(socLabel, storageSOC.Value))
		}
		if reading, ok := energyFlowAssetMetric(metrics, storageAsset.ID, energy.MetricConsumerEnergy); ok {
			cfg.Storage.Secondary = joinEnergySecondaryValues(cfg.Storage.Secondary, reading.Value)
			cfg.Storage.Metrics = append(cfg.Storage.Metrics, energyFlowMetric("Speicherenergie", reading.Value))
		}
	}
	if live.HasGrid {
		gridAsset := energyFlowNodeAsset(assets, tenantSlug, "grid")
		dir := "import"
		if live.Grid.Metric == energy.MetricGridExportPower {
			dir = "export"
		}
		watts := math.Abs(energyPowerWatts(&live.Grid))
		cfg.Grid = &energyFlowNodeConfig{
			ID: gridAsset.ID, NodeType: "grid", Icon: energyConsumerIcon(gridAsset), Label: gridAsset.Name,
			Editable: canManage, Measurements: energyFlowNodeSlots("grid", gridAsset.ID, mappings),
			Value: formatEnergyFlowKW(watts), Unit: "kW", KW: watts / 1000, Dir: dir, Secondary: "–",
			SecondaryLabel: energyAssetSecondaryLabel(gridAsset, "grid"), Color: energyAssetFlowColor(gridAsset),
		}
		if reading, ok := energyFlowAssetMetric(metrics, gridAsset.ID, energy.MetricConsumerEnergy); ok {
			cfg.Grid.Secondary = reading.Value
			cfg.Grid.Metrics = append(cfg.Grid.Metrics, energyFlowMetric(cfg.Grid.SecondaryLabel, reading.Value))
		}
	}
	consumerPower := map[string]energyMetricView{}
	consumerEnergy := map[string]energyMetricView{}
	consumerSOC := map[string]energyMetricView{}
	consumerSleep := map[string]energyMetricView{}
	for _, metric := range metrics {
		switch metric.Metric {
		case energy.MetricConsumerPower:
			if _, exists := consumerPower[metric.AssetID]; metric.AssetID != "" && !exists {
				consumerPower[metric.AssetID] = metric
			}
		case energy.MetricConsumerEnergy:
			if _, exists := consumerEnergy[metric.AssetID]; metric.AssetID != "" && !exists {
				consumerEnergy[metric.AssetID] = metric
			}
		case energy.MetricBatterySOC:
			if _, exists := consumerSOC[metric.AssetID]; metric.AssetID != "" && !exists {
				consumerSOC[metric.AssetID] = metric
			}
		case energy.MetricConsumerSleep:
			if _, exists := consumerSleep[metric.AssetID]; metric.AssetID != "" && !exists {
				consumerSleep[metric.AssetID] = metric
			}
		}
	}
	measurementEntities := map[string]map[string]string{}
	for _, mapping := range mappings {
		if !mapping.Confirmed || mapping.AssetID == "" {
			continue
		}
		if measurementEntities[mapping.AssetID] == nil {
			measurementEntities[mapping.AssetID] = map[string]string{}
		}
		switch mapping.Metric {
		case energy.MetricConsumerPower:
			measurementEntities[mapping.AssetID]["power"] = mapping.EntityID
		case energy.MetricConsumerEnergy:
			measurementEntities[mapping.AssetID]["energy"] = mapping.EntityID
		case energy.MetricBatterySOC:
			measurementEntities[mapping.AssetID]["soc"] = mapping.EntityID
		case energy.MetricConsumerSleep:
			measurementEntities[mapping.AssetID]["sleep"] = mapping.EntityID
		}
	}
	for _, asset := range sortEnergyConsumers(assets) {
		if isEnergyFlowSystemKind(asset.Kind) {
			continue
		}
		title := strings.TrimSpace(asset.Name)
		if title == "" {
			title = energy.AssetKindLabel(asset.Kind)
		}
		ratedPower := ""
		if asset.RatedPowerKW != nil {
			ratedPower = formatEnergyCompact(*asset.RatedPowerKW, 1)
		}
		state := "Bereit · " + energyFlexibilityLabel(asset.Flexibility)
		secondary := ""
		var current *energyFlowMetricConfig
		var primaryMetric *energyMetricView
		kw := 0.0
		active := false
		dataStatus := ""
		dataLabel := ""
		age := ""
		primary, hasPrimary := consumerPower[asset.ID]
		if !hasPrimary {
			primary, hasPrimary = consumerEnergy[asset.ID]
		}
		if !hasPrimary {
			primary, hasPrimary = consumerSOC[asset.ID]
		}
		if hasPrimary {
			age = energyConsumerAgeLabel(now, primary.LastUpdated)
			switch primary.SourceState {
			case "unavailable":
				state = "Nicht verfügbar"
				dataStatus = "unavailable"
			case "unknown":
				state = "Unbekannt"
				dataStatus = "unknown"
			default:
				if primary.Metric == energy.MetricConsumerPower {
					watts := math.Abs(energyPowerWatts(&primary))
					state = formatEnergyValueUnit(formatEnergyFlowKW(watts), "kW")
					current = energyFlowMetricPtr("Jetzt", state)
					if !primary.LastUpdated.IsZero() && now.Sub(primary.LastUpdated) > energyConsumerStaleAfter(asset) {
						dataStatus = "stale"
						dataLabel = "Veraltet"
					} else {
						kw = watts / 1000
						active = watts >= 50
						if active {
							primaryMetric = &primary
						}
					}
				} else {
					state = primary.Value
					if !primary.LastUpdated.IsZero() && now.Sub(primary.LastUpdated) > energyConsumerStaleAfter(asset) {
						dataStatus = "stale"
						dataLabel = "Veraltet"
					}
				}
			}
		}
		if energyReading, hasEnergy := consumerEnergy[asset.ID]; hasEnergy && energyReading.SourceState == "" {
			secondary = energyReading.Value
		}
		if reading, ok := consumerSOC[asset.ID]; ok && reading.SourceState == "" {
			secondary = joinEnergySecondaryValues(reading.Value, secondary)
		}
		if sleep, ok := consumerSleep[asset.ID]; asset.Kind == "ev" && ok && sleep.SourceState == "sleep" {
			seen := primary.LastUpdated
			if seen.IsZero() {
				seen = sleep.LastUpdated
			}
			state = "schläft"
			if !seen.IsZero() {
				state += " · Stand " + seen.In(time.Local).Format("15:04")
			}
			dataStatus, dataLabel, age = "sleep", "", ""
			kw, active = 0, false
			primaryMetric = nil
		}
		secondaryLabel := energyAssetSecondaryLabel(asset, "consumer")
		if _, hasSOC := consumerSOC[asset.ID]; hasSOC && strings.TrimSpace(asset.Metadata["secondary_label"]) == "" {
			secondaryLabel = "Ladestand"
		}
		quietMetrics := make([]energyFlowMetricConfig, 0, 2)
		if primaryMetric == nil && dataStatus != "unavailable" && dataStatus != "unknown" && dataStatus != "sleep" {
			if reading, ok := consumerSOC[asset.ID]; ok && reading.SourceState == "" {
				primaryMetric = &reading
			} else if reading, ok := consumerEnergy[asset.ID]; ok && reading.SourceState == "" {
				primaryMetric = &reading
			} else if hasPrimary && primary.SourceState == "" {
				primaryMetric = &primary
			}
		}
		if reading, ok := consumerSOC[asset.ID]; ok && reading.SourceState == "" &&
			(primaryMetric == nil || primaryMetric.Metric != energy.MetricBatterySOC) {
			label := "Ladestand"
			if custom := cleanEnergyText(asset.Metadata["secondary_label"], 60); custom != "" {
				label = custom
			}
			quietMetrics = append(quietMetrics, energyFlowMetric(label, reading.Value))
		}
		if reading, ok := consumerEnergy[asset.ID]; ok && reading.SourceState == "" &&
			(primaryMetric == nil || primaryMetric.Metric != energy.MetricConsumerEnergy) {
			label := "Energie"
			if _, hasSOC := consumerSOC[asset.ID]; !hasSOC {
				label = secondaryLabel
			}
			if asset.Kind == energyFlowParkingKind {
				label = "Ladeenergie"
			}
			quietMetrics = append(quietMetrics, energyFlowMetric(label, reading.Value))
		}
		var primaryDisplay *energyFlowMetricConfig
		if primaryMetric != nil {
			label := secondaryLabel
			switch primaryMetric.Metric {
			case energy.MetricConsumerPower:
				if current != nil {
					copy := *current
					primaryDisplay = &copy
				}
			case energy.MetricBatterySOC:
				label = secondaryLabel
			}
			if primaryDisplay == nil {
				primaryDisplay = energyFlowMetricPtr(label, primaryMetric.Value)
			}
		}
		cfg.Consumers = append(cfg.Consumers, energyFlowConsumerConfig{
			ID: asset.ID, Icon: energyConsumerIcon(asset), Title: title,
			Kind: asset.Kind, RatedPower: ratedPower, Flexibility: asset.Flexibility,
			PowerEntity: measurementEntities[asset.ID]["power"], EnergyEntity: measurementEntities[asset.ID]["energy"],
			Secondary: secondary, SecondaryLabel: secondaryLabel, Color: energyAssetFlowColor(asset),
			Primary: primaryDisplay, Current: current, Metrics: quietMetrics,
			State: state, DataStatus: dataStatus, DataLabel: dataLabel, Age: age,
			StaleAfterMinutes: energyConsumerStaleOverrideMinutes(asset),
			KW:                kw, Active: active, NodeType: "consumer", Deletable: true,
			Priority:     energyConsumerPriority(asset),
			Measurements: energyFlowConsumerSlots(asset, measurementEntities[asset.ID]),
		})
	}
	if charging.Available {
		parkingAsset := energyFlowNodeAsset(assets, tenantSlug, "parking")
		if parkingAsset.Metadata["hidden"] != "true" {
			state := charging.ModeLabel
			secondary := ""
			var current *energyFlowMetricConfig
			var primaryDisplay *energyFlowMetricConfig
			quietMetrics := []energyFlowMetricConfig{}
			kw := 0.0
			active := false
			dataStatus, dataLabel, age := "", "", ""
			reading, hasReading := consumerPower[parkingAsset.ID]
			if !hasReading && (!charging.PowerLastUpdated.IsZero() || charging.PowerSourceState != "") {
				reading = energyMetricView{
					AssetID: parkingAsset.ID, Metric: energy.MetricConsumerPower,
					Numeric: charging.PowerKW, Unit: "kW",
					SourceState: charging.PowerSourceState, LastUpdated: charging.PowerLastUpdated,
				}
				hasReading = true
			}
			if hasReading {
				age = energyConsumerAgeLabel(now, reading.LastUpdated)
				switch reading.SourceState {
				case "unavailable":
					state, dataStatus = "Nicht verfügbar", "unavailable"
				case "unknown":
					state, dataStatus = "Unbekannt", "unknown"
				default:
					watts := math.Abs(energyPowerWatts(&reading))
					state = formatEnergyValueUnit(formatEnergyFlowKW(watts), "kW")
					current = energyFlowMetricPtr("Jetzt", state)
					if !reading.LastUpdated.IsZero() && now.Sub(reading.LastUpdated) > energyConsumerStaleAfter(parkingAsset) {
						dataStatus, dataLabel = "stale", "Veraltet"
					} else {
						kw = watts / 1000
						active = kw > 0
						if active {
							primaryDisplay = energyFlowMetricPtr("Jetzt", state)
						}
					}
				}
				if energyReading, hasEnergy := consumerEnergy[parkingAsset.ID]; hasEnergy && energyReading.SourceState == "" {
					secondary = energyReading.Value
					if primaryDisplay == nil {
						primaryDisplay = energyFlowMetricPtr("Ladeenergie", energyReading.Value)
					} else {
						quietMetrics = append(quietMetrics, energyFlowMetric("Ladeenergie", energyReading.Value))
					}
				}
			} else if (charging.Mode == "surplus" || charging.Mode == "manual") && charging.PowerKW > 0.05 {
				kw = charging.PowerKW
				active = true
				state = charging.ModeLabel + " mit " + formatEnergyValueUnit(formatEnergyFlowKW(kw*1000), "kW")
				primaryDisplay = energyFlowMetricPtr("Jetzt", formatEnergyValueUnit(formatEnergyFlowKW(kw*1000), "kW"))
			}
			if primaryDisplay == nil && current != nil && dataStatus != "unavailable" && dataStatus != "unknown" {
				copy := *current
				primaryDisplay = &copy
			}
			cfg.Consumers = append(cfg.Consumers, energyFlowConsumerConfig{
				ID: parkingAsset.ID, Icon: energyConsumerIcon(parkingAsset), Title: parkingAsset.Name,
				Kind: "other", Flexibility: parkingAsset.Flexibility, NodeType: "parking", Deletable: true,
				Secondary: secondary, SecondaryLabel: energyAssetSecondaryLabel(parkingAsset, "parking"), Color: energyAssetFlowColor(parkingAsset),
				Primary: primaryDisplay, Current: current, Metrics: quietMetrics,
				State: state, DataStatus: dataStatus, DataLabel: dataLabel, Age: age,
				StaleAfterMinutes: energyConsumerStaleOverrideMinutes(parkingAsset),
				KW:                kw, Active: active, Priority: energyConsumerPriority(parkingAsset),
				Measurements: []energyFlowMeasurementSlot{
					{Name: "consumer_power_entity", Label: "Aktuelle Ladeleistung", Kind: "power", EntityID: firstNonEmpty(energyFlowMappingEntity(mappings, parkingAsset.ID, energy.MetricConsumerPower), charging.PowerEntity)},
					{Name: "consumer_energy_entity", Label: "Ladeenergiezähler", Kind: "energy", EntityID: firstNonEmpty(energyFlowMappingEntity(mappings, parkingAsset.ID, energy.MetricConsumerEnergy), charging.EnergyEntity)},
					{Name: "consumer_soc_entity", Label: "Ladestand", Kind: "percentage", EntityID: energyFlowMappingEntity(mappings, parkingAsset.ID, energy.MetricBatterySOC)},
				},
			})
		}
	}
	sort.SliceStable(cfg.Consumers, func(i, j int) bool { return cfg.Consumers[i].Priority < cfg.Consumers[j].Priority })
	applyEnergyFlowHovers(&cfg, live)
	sanitizeEnergyFlowConfig(&cfg)
	return cfg
}

// applyEnergyFlowHovers erklärt jede Kachel per Hover und macht die Bilanz
// mathematisch schlüssig: Hausverbrauch = Quellen − Speicherladung −
// Einspeisung ± einer ausgewiesenen Differenz (Messabweichung oder nicht
// einzeln erfasste Quellen). Nichts wird stillschweigend passend gemacht.
func applyEnergyFlowHovers(cfg *energyFlowConfig, live energyLiveView) {
	if !live.HasMain {
		return
	}
	loadW := math.Abs(energyPowerWatts(&live.Main))
	if math.IsNaN(loadW) || math.IsInf(loadW, 0) {
		// Ein transienter "NaN"/"Inf"-Sensorwert darf keine Bilanz behaupten.
		return
	}
	finite := func(watts float64) float64 {
		if math.IsNaN(watts) || math.IsInf(watts, 0) {
			return 0
		}
		return watts
	}
	pvW := 0.0
	for index := range live.Flows {
		item := live.Flows[index]
		if item.Metric == energy.MetricPVPower {
			pvW += math.Max(0, finite(energyPowerWatts(&item)))
		}
	}
	importW, exportW := 0.0, 0.0
	for index := range live.Flows {
		item := live.Flows[index]
		switch item.Metric {
		case energy.MetricGridImportPower:
			importW = finite(math.Abs(energyPowerWatts(&item)))
		case energy.MetricGridExportPower:
			exportW = finite(math.Abs(energyPowerWatts(&item)))
		}
	}
	dischargeW, chargeW := 0.0, 0.0
	if live.HasBattery {
		watts := finite(math.Abs(energyPowerWatts(&live.Battery)))
		if watts >= 1 {
			switch live.Battery.Direction {
			case "discharging":
				dischargeW = watts
			case "charging":
				chargeW = watts
			}
		}
	}
	k := func(watts float64) string { return formatEnergyValueUnit(formatEnergyFlowKW(watts), "kW") }
	terms := []string{}
	if pvW > 0 {
		terms = append(terms, "PV "+k(pvW))
	}
	if dischargeW > 0 {
		terms = append(terms, "Speicher "+k(dischargeW))
	}
	if importW > 0 {
		terms = append(terms, "Netzbezug "+k(importW))
	}
	uses := []string{}
	if chargeW > 0 {
		uses = append(uses, "Speicherladung "+k(chargeW))
	}
	if exportW > 0 {
		uses = append(uses, "Einspeisung "+k(exportW))
	}
	available := pvW + dischargeW + importW - chargeW - exportW
	residual := loadW - available
	equation := "Hausverbrauch " + k(loadW)
	if len(terms) > 0 {
		equation += " · gedeckt aus " + strings.Join(terms, " + ")
		if len(uses) > 0 {
			equation += " − " + strings.Join(uses, " − ")
		}
	} else if len(uses) > 0 {
		equation += " · abzüglich " + strings.Join(uses, " − ")
	}
	if math.Abs(residual) >= 15 {
		if residual > 0 {
			equation += " · Differenz " + k(residual) + " nicht einzeln erfasst (Messung oder weitere Quellen)"
		} else {
			equation += " · Differenz " + k(-residual) + " nicht in der Hausverbrauch-Messung enthalten (separate Verbraucher oder Messabweichung)"
		}
	} else {
		equation += " · Bilanz geht auf"
	}
	cfg.Home.Hover = equation
	for index := range cfg.Producers {
		cfg.Producers[index].Hover = "PV erzeugt gerade " + k(pvW) + " für Haus, Speicher und Netz."
	}
	if cfg.Storage != nil {
		switch {
		case dischargeW > 0:
			cfg.Storage.Hover = "Der Speicher entlädt " + k(dischargeW) + " und deckt damit einen Teil des Hausverbrauchs."
		case chargeW > 0:
			cfg.Storage.Hover = "Der Speicher lädt gerade mit " + k(chargeW) + "."
		default:
			cfg.Storage.Hover = "Der Speicher wartet – kein nennenswerter Lade- oder Entladefluss."
		}
	}
	if cfg.Grid != nil {
		if cfg.Grid.Dir == "import" {
			cfg.Grid.Hover = "Aus dem Netz kommen gerade " + k(importW) + "."
		} else {
			cfg.Grid.Hover = "Überschuss von " + k(exportW) + " geht ins Netz."
		}
	}
}

func combineBatteryMetrics(metrics []energyMetricView, selected map[int]bool) (energyMetricView, bool) {
	var charge, discharge, generic *energyMetricView
	chargeIndex, dischargeIndex, genericIndex := -1, -1, -1
	for index := range metrics {
		item := metrics[index]
		if item.Metric != energy.MetricBatteryPower && item.Metric != energy.MetricBatteryCharge && item.Metric != energy.MetricBatteryDischarge {
			continue
		}
		switch item.Kind {
		case "battery-charge":
			if charge == nil {
				copy := item
				charge = &copy
				chargeIndex = index
			}
		case "battery-discharge":
			if discharge == nil {
				copy := item
				discharge = &copy
				dischargeIndex = index
			}
		default:
			if generic == nil {
				copy := item
				generic = &copy
				genericIndex = index
			}
		}
	}
	if charge != nil || discharge != nil {
		chargeW := energyPowerWatts(charge)
		dischargeW := energyPowerWatts(discharge)
		netW := dischargeW - chargeW
		item := energyMetricView{Metric: energy.MetricBatteryPower, Kind: "battery-power", Label: "Speicher"}
		// The flow config reads Numeric/Unit (HAUSV-439); without them the
		// split charge/discharge pair rendered a permanently idle battery.
		item.Numeric = math.Abs(netW)
		item.Unit = "W"
		if netW > 0 {
			item.Value = formatEnergyReading(netW, "W")
			item.Detail = "liefert Energie"
			item.Direction = "discharging"
		} else if netW < 0 {
			item.Value = formatEnergyReading(-netW, "W")
			item.Detail = "lädt"
			item.Direction = "charging"
		} else {
			item.Value = formatEnergyValueUnit("0", "W")
			item.Detail = "in Ruhe"
			item.Direction = "idle"
		}
		if chargeIndex >= 0 {
			selected[chargeIndex] = true
		}
		if dischargeIndex >= 0 {
			selected[dischargeIndex] = true
		}
		return item, true
	}
	if generic != nil {
		netW := energyPowerWatts(generic)
		generic.Label = "Speicher"
		generic.Numeric = math.Abs(netW)
		generic.Unit = "W"
		switch {
		case netW < 0:
			generic.Value = formatEnergyReading(-netW, "W")
			generic.Detail = "lädt"
			generic.Direction = "charging"
		case netW > 0:
			generic.Value = formatEnergyReading(netW, "W")
			generic.Detail = "liefert Energie"
			generic.Direction = "discharging"
		default:
			generic.Value = formatEnergyValueUnit("0", "W")
			generic.Detail = "in Ruhe"
			generic.Direction = "idle"
		}
		selected[genericIndex] = true
		return *generic, true
	}
	return energyMetricView{}, false
}

func energyPowerWatts(item *energyMetricView) float64 {
	if item == nil {
		return 0
	}
	switch strings.ToLower(strings.TrimSpace(item.Unit)) {
	case "kw":
		return item.Numeric * 1000
	case "mw":
		return item.Numeric * 1000000
	default:
		return item.Numeric
	}
}

func energyFlowDetail(metric string) string {
	switch metric {
	case energy.MetricPVPower:
		return "erzeugt"
	case energy.MetricGridImportPower:
		return "Bezug"
	case energy.MetricGridExportPower:
		return "Einspeisung"
	default:
		return ""
	}
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

func buildEnergyAssetOptions(assets []energy.Asset) []energyAssetOption {
	selected := map[string]bool{}
	for _, asset := range assets {
		selected[asset.Kind] = true
	}
	kinds := []string{"pv", "battery", "ev", "wallbox", "heat-pump", "hot-water", "sauna", "air-conditioning", "instant-water-heater", "other"}
	out := make([]energyAssetOption, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, energyAssetOption{Kind: kind, Label: energy.AssetKindLabel(kind), Checked: selected[kind]})
	}
	return out
}

func buildEnergyMappingAssetOptions(assets []energy.Asset) []energyOption {
	out := make([]energyOption, 0, len(assets))
	for _, asset := range assets {
		out = append(out, energyOption{Value: asset.ID, Label: asset.Name})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Label) < strings.ToLower(out[j].Label)
	})
	return out
}

func buildEnergyMappingSlots(assets []energy.Asset, mappings []energy.EntityMapping) []energyMappingSlotView {
	hasAsset := map[string]bool{}
	for _, asset := range assets {
		if asset.Confirmed {
			hasAsset[asset.Kind] = true
		}
	}
	hasMetric := map[string]bool{}
	for _, mapping := range mappings {
		if !mapping.Confirmed {
			continue
		}
		hasMetric[energyMetricForMapping(mapping)] = true
	}
	hasPowerForPeak := hasMetric[energy.MetricGridImportPower] || hasMetric[energy.MetricLoadPower]
	type slot struct {
		key      string
		label    string
		purpose  string
		metrics  []string
		required bool
		derived  bool
	}
	slots := []slot{
		{
			key: "load", label: "Hausverbrauch", purpose: "Zeigt, wie viel das Zuhause insgesamt gerade benötigt.",
			metrics: []string{energy.MetricLoadPower}, required: true,
		},
		{
			key: "grid", label: "Netzleistung", purpose: "Trennt aktuellen Bezug und Einspeisung am Hausanschluss.",
			metrics: []string{energy.MetricGridImportPower, energy.MetricGridExportPower}, required: true,
		},
		{
			key: "pv", label: "PV-Erzeugung", purpose: "Macht Eigenversorgung und Überschüsse sichtbar.",
			metrics: []string{energy.MetricPVPower}, required: hasAsset["pv"] || hasMetric[energy.MetricPVPower],
		},
		{
			key: "battery-power", label: "Speicherleistung", purpose: "Zeigt Nettoleistung oder getrennte Lade- und Entladeleistung.",
			metrics: []string{energy.MetricBatteryPower, energy.MetricBatteryCharge, energy.MetricBatteryDischarge}, required: hasAsset["battery"] || hasMetric[energy.MetricBatteryPower] || hasMetric[energy.MetricBatteryCharge] || hasMetric[energy.MetricBatteryDischarge],
		},
		{
			key: "battery-soc", label: "Speicherfüllstand", purpose: "Zeigt, wie viel Energie relativ zur Kapazität verfügbar ist.",
			metrics: []string{energy.MetricBatterySOC}, required: hasAsset["battery"] || hasMetric[energy.MetricBatterySOC],
		},
		{
			key: "peaks", label: "Viertelstunden-Spitzen", purpose: "Wird aus dem Leistungsverlauf berechnet; kein eigener Sensor nötig.",
			required: true, derived: true,
		},
	}
	out := make([]energyMappingSlotView, 0, len(slots))
	for _, item := range slots {
		found := false
		count := 0
		for _, metric := range item.metrics {
			if hasMetric[metric] {
				found = true
				count++
			}
		}
		view := energyMappingSlotView{
			Key: item.key, Label: item.label, Purpose: item.purpose, Required: item.required, Derived: item.derived,
		}
		switch {
		case item.derived && hasPowerForPeak:
			view.Status, view.Detail, view.Tone = "Wird berechnet", "Aus den zugeordneten Leistungswerten", "good"
		case item.derived:
			view.Status, view.Detail, view.Tone = "Noch nicht möglich", "Hausverbrauch oder Netzbezug zuordnen", "warning"
		case found:
			view.Status, view.Tone = "Zugeordnet", "good"
			if item.key == "grid" && count == 1 {
				view.Detail = "Bezug oder Einspeisung vorhanden; die zweite Richtung ist optional."
			} else if item.key == "battery-power" {
				view.Detail = "Ein vorzeichenbehafteter Sensor oder getrennte Lade-/Entladewerte."
			} else {
				view.Detail = "Bereit für Übersicht und Verlauf."
			}
		case item.required:
			view.Status, view.Detail, view.Tone = "Noch zuordnen", "Einen passenden Home-Assistant-Sensor auswählen.", "warning"
		default:
			view.Status, view.Detail, view.Tone = "Derzeit nicht benötigt", "Kann später ergänzt werden.", "quiet"
		}
		out = append(out, view)
	}
	return out
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

func energyPeakViews(intervals []energy.Interval, at time.Time) []energyPeakView {
	bySource := map[string][]energy.Interval{}
	for _, interval := range intervals {
		if interval.Quality == energy.QualityMeasured || interval.Quality == energy.QualityEstimated {
			bySource[interval.Source] = append(bySource[interval.Source], interval)
		}
	}
	sources := make([]string, 0, len(bySource))
	for source := range bySource {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	out := make([]energyPeakView, 0, len(sources))
	for _, source := range sources {
		label := source
		switch source {
		case "smart-meter":
			label = "Smart Meter"
		case "home-assistant":
			label = "Home Assistant"
		}
		out = append(out, energyPeakView{
			Source: label,
			Value:  formatEnergyValueUnit(formatEnergyNumber(energy.PeakForMonth(bySource[source], at, time.Local)), "kW"),
		})
	}
	return out
}

func intervalQualityCounts(intervals []energy.Interval) (int, int) {
	gaps, conflicts := 0, 0
	for _, interval := range intervals {
		switch interval.Quality {
		case energy.QualityGap:
			gaps++
		case energy.QualityConflict:
			conflicts++
		}
	}
	return gaps, conflicts
}

func latestEnergySeen(mappings []energy.EntityMapping, intervals []energy.Interval, live time.Time) time.Time {
	latest := live
	for _, mapping := range mappings {
		if mapping.LastSeenAt != nil && mapping.LastSeenAt.After(latest) {
			latest = *mapping.LastSeenAt
		}
	}
	for _, interval := range intervals {
		end := interval.StartsAt.Add(interval.Duration)
		if end.After(latest) {
			latest = end
		}
	}
	return latest
}

// recordsItself sagt, ob ein bestaetigter Netzbezug vorliegt. Nur dann fuellt
// sich die Karte von selbst — der Leerzustand ist dann eine Wartezeit und
// keine Aufforderung, eine Datei zu suchen.
func buildEnergyTariffView(profile energy.HomeProfile, intervals []energy.Interval, recordsItself bool) energyTariffView {
	now := time.Now()
	rules := energy.AustrianDraft2027()
	view := energyTariffView{
		ID:          rules.ID,
		Version:     rules.Version,
		Status:      "Entwurf · nicht verbindlich",
		SourceTitle: rules.SourceTitle,
		SourceURL:   rules.SourceURL,
		Rule:        "Höchste abgeschlossene Viertelstunde des Monats; Entwurfsannahme mindestens 2 kW und 20 % der vereinbarten Leistung.",
		Disclaimer:  "Keine Tarif- oder Einspargarantie. Neue Verordnungsversionen können ausgetauscht werden, ohne Messdaten zu verändern.",
		MonthLabel:  energyMonthLabel(now),

		BelowRateEUR: formatEnergyValueUnit(formatEnergyCompact(rules.AnnualBelowEURPerKW, 2), "€"),
		AboveRateEUR: formatEnergyValueUnit(formatEnergyCompact(rules.AnnualAboveEURPerKW, 2), "€"),
		ThresholdKW:  formatEnergyValueUnit(formatEnergyCompact(rules.TierThresholdKW, 0), "kW"),
	}
	peak := energy.PeakForMonth(intervals, now, time.Local)
	if peak <= 0 {
		if recordsItself {
			view.MissingReason = "Die erste vollständige Viertelstunde wird gerade aufgezeichnet und erscheint hier, sobald sie zu Ende ist."
			view.MissingIsWaiting = true
		} else {
			view.MissingReason = "Sobald der Netzbezug einem Messwert zugeordnet ist, zeichnet HAUSV die Viertelstunden selbst auf."
		}
		return view
	}
	estimate := rules.Estimate(peak, agreedPowerKW(profile))
	view.AnnualPowerEUR = formatEnergyValueUnit(formatEnergyNumber(estimate.AnnualPowerEUR), "€")
	view.HasEstimate = true
	coverage := buildEnergyTariffCoverageView(intervals, now)
	view.CoverageLabel = coverage.Label
	view.Basis = coverage.Basis
	if at, ok := peakQuarterOfMonth(intervals, now); ok {
		view.PeakTime = at.In(time.Local).Format("02.01. um 15:04")
		view.HasPeakTime = true
	}
	view.PeakKW = formatEnergyValueUnit(formatEnergyCompact(peak, 1), "kW")
	view.BilledKW = formatEnergyValueUnit(formatEnergyCompact(estimate.BilledKW, 1), "kW")
	if estimate.BilledKW > 0 {
		view.PeakMeterPercent = int(math.Round(math.Max(0, math.Min(1, peak/estimate.BilledKW)) * 100))
	}
	view.MinimumReason = estimate.MinimumReason
	if estimate.AboveKW > 0 {
		view.HasTier = true
		view.BelowKW = formatEnergyValueUnit(formatEnergyCompact(estimate.BelowKW, 1), "kW")
		view.AboveKW = formatEnergyValueUnit(formatEnergyCompact(estimate.AboveKW, 1), "kW")
		view.TierHint = "Der Anteil über " + formatEnergyValueUnit(formatEnergyCompact(rules.TierThresholdKW, 0), "kW") + " wird im Entwurf mit dem höheren Satz bemessen. Dort wirkt Kappen etwa doppelt so stark."
	}
	if agreed := agreedPowerKW(profile); agreed > 0 {
		view.HasAgreed = true
		view.AgreedKW = formatEnergyValueUnit(formatEnergyCompact(agreed, 1), "kW")
	} else {
		view.AgreedHint = "Ohne vereinbarte Anschlussleistung rechnet die Schätzung nur mit dem 2-kW-Sockel. Der Wert steht auf Ihrer Netzrechnung."
	}
	return view
}

// energyMonthLabel schreibt den Kalendermonat aus. Der Leistungstarif bemisst
// je Kalendermonat; „08/2026“ oder gar nichts wäre auf diesem Bildschirm die
// schlechtere Antwort.
func energyMonthLabel(at time.Time) string {
	names := []string{"Jänner", "Februar", "März", "April", "Mai", "Juni",
		"Juli", "August", "September", "Oktober", "November", "Dezember"}
	local := at.In(time.Local)
	return names[int(local.Month())-1] + " " + strconv.Itoa(local.Year())
}

type energyTariffCoverageView struct {
	Label     string
	Basis     string
	Measured  int
	Estimated int
}

// buildEnergyTariffCoverageView benennt, worauf die Monatsspitze beruht: wie
// viele abgeschlossene Viertelstunden aus welcher Quelle und wie viel des
// Monats damit abgedeckt ist. Label bleibt kurz genug fuer die Karte; Basis
// traegt Monat, Quelle und die gemessen/geschaetzt-Unterscheidung fuer die
// progressive Offenlegung.
func buildEnergyTariffCoverageView(intervals []energy.Interval, at time.Time) energyTariffCoverageView {
	local := at.In(time.Local)
	monthStart := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, time.Local)
	monthIntervals := make([]energy.Interval, 0, len(intervals))
	sources := map[string]struct{}{}
	for _, interval := range intervals {
		start := interval.StartsAt.In(time.Local)
		if start.Before(monthStart) || !start.Before(monthStart.AddDate(0, 1, 0)) {
			continue
		}
		monthIntervals = append(monthIntervals, interval)
		if interval.Quality == energy.QualityMeasured || interval.Quality == energy.QualityEstimated {
			sources[interval.Source] = struct{}{}
		}
	}
	counts := energy.CountUsableQuarters(monthIntervals)
	if counts.Total == 0 {
		return energyTariffCoverageView{}
	}
	elapsed := int(local.Sub(monthStart) / (15 * time.Minute))
	if elapsed < counts.Total {
		elapsed = counts.Total
	}
	label := strconv.Itoa(counts.Total) + " von " + strconv.Itoa(elapsed) + " Viertelstunden abgedeckt"
	basis := strconv.Itoa(counts.Total) + " von " + strconv.Itoa(elapsed) +
		" bisherigen Viertelstunden im " + energyMonthLabel(local)
	quality := make([]string, 0, 2)
	if counts.Measured > 0 {
		quality = append(quality, strconv.Itoa(counts.Measured)+" direkt gemessen")
	}
	if counts.Estimated > 0 {
		quality = append(quality, strconv.Itoa(counts.Estimated)+" aus Momentanwerten geschätzt")
	}
	if len(quality) > 0 {
		basis += " · " + strings.Join(quality, ", ")
	}
	if names := energySourceLabels(sources); names != "" {
		basis += " · Quelle: " + names
	}
	return energyTariffCoverageView{
		Label:     label,
		Basis:     basis,
		Measured:  counts.Measured,
		Estimated: counts.Estimated,
	}
}

func energyTariffBasis(intervals []energy.Interval, at time.Time) string {
	return buildEnergyTariffCoverageView(intervals, at).Basis
}

// energySourceLabels übersetzt die internen Quellenschlüssel in Klartext und
// hält die Reihenfolge stabil, damit zwei Aufrufe nicht unterschiedlich lauten.
func energySourceLabels(sources map[string]struct{}) string {
	keys := make([]string, 0, len(sources))
	for key := range sources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	labels := make([]string, 0, len(keys))
	for _, key := range keys {
		switch key {
		case "smart-meter":
			labels = append(labels, "Smart-Meter-Export")
		case "home-assistant":
			labels = append(labels, "Home Assistant")
		default:
			labels = append(labels, key)
		}
	}
	return strings.Join(labels, " und ")
}

// peakQuarterOfMonth liefert den Beginn der teuersten Viertelstunde des
// laufenden Monats. PeakForMonth gibt nur den Wert zurück; für die Oberfläche
// zählt aber, wann er entstanden ist — daran erkennt ein Haushalt die eigene
// Gewohnheit wieder.
func peakQuarterOfMonth(intervals []energy.Interval, at time.Time) (time.Time, bool) {
	local := at.In(time.Local)
	monthStart := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, 0)
	best := time.Time{}
	bestKW := 0.0
	found := false
	for _, interval := range intervals {
		if interval.Quality != energy.QualityMeasured && interval.Quality != energy.QualityEstimated {
			continue
		}
		start := interval.StartsAt.In(time.Local)
		if start.Before(monthStart) || !start.Before(monthEnd) {
			continue
		}
		if !found || interval.AverageKW > bestKW {
			best, bestKW, found = start, interval.AverageKW, true
		}
	}
	return best, found
}

// agreedPowerKW liefert 0, solange die vereinbarte Anschlussleistung nicht
// erfasst ist. Estimate lässt die 20-%-Mindestbemessung dann bewusst aus,
// statt sie gegen einen geratenen Wert zu rechnen.
func agreedPowerKW(profile energy.HomeProfile) float64 {
	if profile.AgreedPowerKW == nil {
		return 0
	}
	return *profile.AgreedPowerKW
}

// assetFlexContribution liefert Leistung und Flexibilität eines Verbrauchers.
//
// Reihenfolge ist Absicht: gesetzte Werte am Asset schlagen die Vorbelegung
// aus der Kategorie. Damit bleibt die Kategorie eine Vorlage und wird nicht
// zur Verhaltensregel — das ist der Kern von HAUSV-422.
func assetFlexContribution(asset energy.Asset) (float64, string) {
	// Erzeugung verschiebt die Bezugsspitze nicht. Die Nennleistung einer
	// PV-Anlage bleibt erfasst — sie beschreibt die Anlagengröße —, darf aber
	// nicht als abschaltbare Last verrechnet werden.
	if asset.Kind == "pv" {
		return 0, energy.FlexUnknown
	}
	kw := defaultAssetPowerKW(asset.Kind)
	if asset.RatedPowerKW != nil {
		kw = *asset.RatedPowerKW
	}
	flexibility := asset.Flexibility
	if flexibility == "" || flexibility == energy.FlexUnknown {
		if asset.Source == energyCustomAssetSource {
			// Frei angelegt und ohne Angabe: nichts versprechen. Der
			// Verbraucher zählt im Verbrauch, aber nicht in der Peak-Wirkung.
			return 0, energy.FlexUnknown
		}
		flexibility = defaultAssetFlexibility(asset.Kind)
	}
	return kw, flexibility
}

// defaultAssetPowerKW ist die Vorbelegung einer Vorlage ohne eigene Angabe.
// Bewusst zurückhaltend: es sind Modellannahmen, keine Messwerte. PV liefert
// 0, weil Erzeugung die Bezugsspitze nicht verschiebt.
func defaultAssetPowerKW(kind string) float64 {
	switch kind {
	case "ev", "wallbox", "battery":
		return 3
	case "hot-water", "heat-pump":
		return 1
	default:
		return 0
	}
}

func assetFlexAssumption(asset energy.Asset, suffix string) string {
	name := strings.TrimSpace(asset.Name)
	if name == "" {
		name = energy.AssetKindLabel(asset.Kind)
	}
	return name + " " + suffix
}

func isChargingPreset(asset energy.Asset) bool {
	return asset.Source != energyCustomAssetSource && (asset.Kind == "ev" || asset.Kind == "wallbox")
}

// chargingPresetIndex wählt aus den Vorlagen für die Ladelast die
// aussagekräftigste aus: eine erfasste Nennleistung schlägt die Vorbelegung,
// sonst die höhere Leistung. Ohne diese Wahl entschiede die Sortierung — die
// Liste kommt nach Art sortiert, also gewänne immer das E-Auto mit seinen
// vorbelegten 3 kW, selbst neben einer erfassten 22-kW-Wallbox.
func chargingPresetIndex(assets []energy.Asset) int {
	best := -1
	for index, asset := range assets {
		if !isChargingPreset(asset) {
			continue
		}
		if best < 0 {
			best = index
			continue
		}
		current, previous := assets[index], assets[best]
		if (current.RatedPowerKW != nil) != (previous.RatedPowerKW != nil) {
			if current.RatedPowerKW != nil {
				best = index
			}
			continue
		}
		currentKW, _ := assetFlexContribution(current)
		previousKW, _ := assetFlexContribution(previous)
		if currentKW > previousKW {
			best = index
		}
	}
	return best
}

func buildEnergyScenarioViews(profile energy.HomeProfile, assets []energy.Asset, intervals []energy.Interval) []energyScenarioView {
	baseline := energy.PeakForMonth(intervals, time.Now(), time.Local)
	if baseline <= 0 {
		return nil
	}
	shiftable, throttle, battery := 0.0, 0.0, 0.0
	assumptions := []string{}
	// Je Asset gerechnet, nicht je Kategorie. Die Kategorie liefert nur
	// Vorbelegungen; eine gesetzte Nennleistung und eine gesetzte Flexibilität
	// gewinnen immer. Vorher verschmolz `has[kind]` zwei gleichartige
	// Verbraucher zu einem — zwei Wallboxen ergaben dieselben 3 kW wie eine.
	chargingPreset := chargingPresetIndex(assets)
	for index, asset := range assets {
		kw, flexibility := assetFlexContribution(asset)
		if kw <= 0 {
			continue
		}
		// E-Auto und Wallbox beschreiben als Vorlagen dieselbe Ladelast; beide
		// anzurechnen würde die Flexibilität erfinden. Frei angelegte
		// Verbraucher zählen dagegen einzeln — sie wurden bewusst so benannt.
		if isChargingPreset(asset) && index != chargingPreset {
			continue
		}
		switch {
		case asset.Kind == "battery" && flexibility != energy.FlexFixed:
			// Ein Speicher ist keine drosselbare Last, sondern eine eigene
			// Rolle mit eigenem Gewicht in der Simulation. Wer ihn ausdrücklich
			// als fest erklärt, bekommt dieses Gewicht nicht.
			battery += kw
			assumptions = append(assumptions, assetFlexAssumption(asset, "Speicherleistung"))
		case flexibility == energy.FlexShift:
			shiftable += kw
			assumptions = append(assumptions, assetFlexAssumption(asset, "zeitlich verschiebbar"))
		case flexibility == energy.FlexThrottle:
			throttle += kw
			assumptions = append(assumptions, assetFlexAssumption(asset, "kurz begrenzbar"))
		}
	}
	if shiftable+throttle+battery == 0 {
		return nil
	}
	counts := energy.CountUsableQuarters(intervals)
	quality := energy.QualityUnavailable
	if counts.Measured > 0 {
		quality = energy.QualityMeasured
	}
	// A mixed basis stays conservative: one estimated quarter means the
	// scenario must not present the complete baseline as directly measured.
	if counts.Estimated > 0 {
		quality = energy.QualityEstimated
	}
	gaps, conflicts := intervalQualityCounts(intervals)
	if gaps > 0 || conflicts > 0 {
		quality = energy.QualityGap
	}
	result := energy.SimulateScenario(energy.ScenarioInput{
		Name: "Bestehende Flexibilität nutzen", BaselinePeakKW: baseline,
		ShiftableKW: shiftable, ThrottleKW: throttle, BatteryKW: battery, DataQuality: quality,
	})
	view := energyScenarioView{
		Title: result.Name,
		BaselineNote: "Ausgangswert: die gemessene Monatsspitze von " +
			formatEnergyValueUnit(formatEnergyCompact(baseline, 1), "kW") + " im " + energyMonthLabel(time.Now()) + ".",
		PeakBand:    formatEnergyValueUnit(formatEnergyNumber(result.ExpectedPeakLowKW)+"–"+formatEnergyNumber(result.ExpectedPeakHighKW), "kW"),
		EffectBand:  formatEnergyValueUnit(formatEnergyNumber(result.PeakEffectLowKW)+"–"+formatEnergyNumber(result.PeakEffectHighKW), "kW") + " mögliche Peak-Wirkung",
		Uncertainty: result.Uncertainty,
		Assumptions: strings.Join(assumptions, " · "),
	}
	// Estimate(0, agreed) liefert genau die Untergrenze: den 2-kW-Sockel oder
	// die 20 % der vereinbarten Leistung, je nachdem was höher ist.
	rules := energy.AustrianDraft2027()
	agreed := agreedPowerKW(profile)
	billedLow := rules.Estimate(result.ExpectedPeakLowKW, agreed).BilledKW
	billedHigh := rules.Estimate(result.ExpectedPeakHighKW, agreed).BilledKW
	// Nur zeigen, wenn die Mindestbemessung das Band tatsächlich anhebt. Sonst
	// stünde zweimal dieselbe Zahl da und die Aussage ginge im Rauschen unter.
	if billedLow > result.ExpectedPeakLowKW+0.05 || billedHigh > result.ExpectedPeakHighKW+0.05 {
		view.BilledBand = formatEnergyValueUnit(formatEnergyCompact(billedLow, 1)+"–"+formatEnergyCompact(billedHigh, 1), "kW")
		if billedLow == billedHigh {
			view.BilledBand = formatEnergyValueUnit(formatEnergyCompact(billedLow, 1), "kW")
		}
		view.HasBilledBand = true
	}
	if floor := rules.Estimate(0, agreed).BilledKW; result.ExpectedPeakLowKW < floor {
		view.FloorNote = "Unter " + formatEnergyValueUnit(formatEnergyCompact(floor, 1), "kW") + " sinkt der verrechnete Betrag nicht weiter: so weit reicht die Mindestbemessung. Weiteres Kappen senkt die Spitze, nicht die Rechnung."
	}
	return []energyScenarioView{view}
}

// defaultAssetFlexibility hält die Vorbelegung an einer Stelle: im
// energy-Paket, das sie auch beim Seeding speichert.
func defaultAssetFlexibility(kind string) string {
	return energy.DefaultAssetFlexibility(kind)
}

func energyMetricLabel(metric string) string {
	switch metric {
	case energy.MetricGridImportPower:
		return "Netzbezug jetzt"
	case energy.MetricGridImportEnergy:
		return "Netzbezug gesamt"
	case energy.MetricGridExportPower:
		return "Einspeisung"
	case energy.MetricPVPower:
		return "PV-Leistung"
	case energy.MetricBatteryPower:
		return "Batterieleistung"
	case energy.MetricBatteryCharge:
		return "Batterie-Ladeleistung"
	case energy.MetricBatteryDischarge:
		return "Batterie-Entladeleistung"
	case energy.MetricBatterySOC:
		return "Batteriestand"
	case energy.MetricLoadPower:
		return "Hausverbrauch"
	case energy.MetricConsumerPower:
		return "Verbraucherleistung"
	case energy.MetricConsumerEnergy:
		return "Verbrauchszähler"
	default:
		return "Messwert"
	}
}

func energyMetricTone(metric string, value float64, unit string, target *float64) string {
	if metric == energy.MetricGridImportPower && target != nil && *target > 0 {
		valueKW := value
		switch strings.ToLower(strings.TrimSpace(unit)) {
		case "w":
			valueKW = value / 1000
		case "mw":
			valueKW = value * 1000
		}
		if valueKW >= *target {
			return "danger"
		}
		if valueKW >= *target*0.8 {
			return "warning"
		}
	}
	if metric == energy.MetricPVPower || metric == energy.MetricBatterySOC {
		return "good"
	}
	return ""
}

func formatEnergyNumber(value float64) string {
	precision := 1
	if value >= 100 {
		precision = 0
	}
	return strings.ReplaceAll(strconv.FormatFloat(value, 'f', precision, 64), ".", ",")
}

func formatEnergyValueUnit(value, unit string) string {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return value
	}
	return value + "\u00a0" + unit
}

func formatEnergyReading(value float64, unit string) string {
	normalized := strings.ToLower(strings.TrimSpace(unit))
	switch normalized {
	case "w":
		if value >= 1000 || value <= -1000 {
			return formatEnergyValueUnit(formatEnergyDisplayCompact(value/1000, 2), "kW")
		}
		return formatEnergyValueUnit(formatEnergyDisplayCompact(value, 0), "W")
	case "kw":
		return formatEnergyValueUnit(formatEnergyDisplayCompact(value, 2), "kW")
	case "%":
		return formatEnergyValueUnit(formatEnergyDisplayCompact(value, 1), "%")
	case "kwh":
		return formatEnergyValueUnit(formatEnergyDisplayCompact(value, 1), "kWh")
	case "mwh":
		return formatEnergyValueUnit(formatEnergyDisplayCompact(value, 2), "MWh")
	default:
		return formatEnergyNumber(value) + energyUnitSuffix(unit)
	}
}

func formatEnergyDisplayCompact(value float64, precision int) string {
	raw := formatDecimal(value, precision)
	if precision > 0 {
		raw = strings.TrimRight(strings.TrimRight(raw, "0"), ",")
	}
	return raw
}

func formatEnergyCompact(value float64, precision int) string {
	raw := strconv.FormatFloat(value, 'f', precision, 64)
	if precision > 0 {
		raw = strings.TrimRight(strings.TrimRight(raw, "0"), ".")
	}
	return strings.ReplaceAll(raw, ".", ",")
}

func energyUnitSuffix(unit string) string {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return ""
	}
	return "\u00a0" + unit
}

func haAttribute(attributes map[string]any, key string) string {
	if attributes == nil {
		return ""
	}
	value, ok := attributes[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func cleanEnergyText(raw string, limit int) string {
	value := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	runes := []rune(value)
	if len(runes) > limit {
		value = string(runes[:limit])
	}
	return value
}

// sanitizeEnergyFlowConfig keeps the flow config encodable. A sensor that
// reports "unavailable" or a division that produced ±Inf must not turn into a
// json.Marshal error — encoding/json refuses NaN and Inf — because the page
// then ships "null" and the client renderer silently leaves the no-JS fallback
// in place (HAUSV-671). Display strings are already formatted; only the kw
// numerics that drive ribbon widths are replaced with 0.
func sanitizeEnergyFlowConfig(cfg *energyFlowConfig) {
	finite := func(kw float64) float64 {
		if math.IsNaN(kw) || math.IsInf(kw, 0) {
			return 0
		}
		return kw
	}
	cfg.Home.KW = finite(cfg.Home.KW)
	for i := range cfg.Producers {
		cfg.Producers[i].KW = finite(cfg.Producers[i].KW)
	}
	if cfg.Storage != nil {
		cfg.Storage.KW = finite(cfg.Storage.KW)
	}
	if cfg.Grid != nil {
		cfg.Grid.KW = finite(cfg.Grid.KW)
	}
	for i := range cfg.Consumers {
		cfg.Consumers[i].KW = finite(cfg.Consumers[i].KW)
	}
}
