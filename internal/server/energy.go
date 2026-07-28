package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/markus-barta/hausv-org/internal/energy"
	"github.com/markus-barta/hausv-org/internal/homeassistant"
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
	Label  string
	Value  string
	Detail string
	Tone   string
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
	Estimate    string
	HasEstimate bool
	Disclaimer  string
}

type energyScenarioView struct {
	Title       string
	PeakBand    string
	EffectBand  string
	Uncertainty string
	Assumptions string
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

func (a *app) canViewEnergy(ac authCtx) bool {
	if !canUseResidentAreas(ac.role) {
		return false
	}
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	return hasCapability(ac.role, capabilityManageEnergy) ||
		profile.HasPermission(permissionEnergyView) ||
		profile.HasPermission(permissionEnergyConfigure) ||
		profile.HasPermission(permissionEnergyControl) ||
		profile.HasPermission(permissionEnergyCaretaker) ||
		ac.role == roleRenter || ac.role == roleResident || ac.role == roleBeirat
}

func (a *app) canManageEnergy(ac authCtx) bool {
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	return hasCapability(ac.role, capabilityManageEnergy) ||
		profile.HasPermission(permissionEnergyConfigure) ||
		profile.HasPermission(permissionEnergyCaretaker)
}

func (a *app) canControlEnergy(ac authCtx) bool {
	// The house-wide mode is a property decision, not a technical support
	// permission. Legacy energy-control grants intentionally do not widen it.
	return hasCapability(ac.role, capabilityControlEnergy)
}

func (a *app) homeOnboarding(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canViewEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	profile, exists, err := a.energyStore.Profile(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Hausprofil konnte nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	if !exists {
		profile = energy.DefaultProfile(ac.tenant.Slug, time.Now())
		profile.HouseholdName = houseDisplayName(ac.tenant)
	}
	assets, _ := a.energyStore.ListAssets(ac.tenant.Slug)
	mappings, _ := a.energyStore.ListMappings(ac.tenant.Slug)
	intervals, _ := a.energyStore.ListIntervals(ac.tenant.Slug, time.Now().AddDate(0, -1, 0), time.Time{})
	finishRecommendation := energy.NextRecommendation(profile, assets, mappings, intervals)
	step := profile.OnboardingStep
	if raw := strings.TrimSpace(r.URL.Query().Get("step")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed >= 1 && parsed <= 5 {
			step = parsed
		}
	}
	discovery := energyDiscoveryView{}
	connectorMessage := "Home Assistant ist noch nicht verbunden. Das ist okay – Sie können später weitermachen."
	connectorOK := false
	if step == 4 && ac.tenant.HA.Configured() {
		ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
		defer cancel()
		if discovered, discoverErr := discoverEnergyCandidates(ctx, ac.tenant.HA, mappings); discoverErr == nil {
			discovery = discovered
			connectorOK = true
			if len(discovery.Recommended) > 0 {
				connectorMessage = fmt.Sprintf("%d sichere Vorschläge gefunden. Sie bleiben vollständig lesend.", len(discovery.Recommended))
			} else {
				connectorMessage = "Verbindung hergestellt, aber noch kein eindeutig passender Hausenergie-Messwert gefunden."
			}
		} else {
			connectorMessage = "Home Assistant antwortet gerade nicht. Ihre bisherigen Angaben bleiben erhalten."
		}
	}
	a.render(w, "homeOnboarding", a.withBase(ac, map[string]any{
		"Title":                "Mein Zuhause einrichten",
		"ActivePage":           "energy",
		"Profile":              profile,
		"Step":                 step,
		"Progress":             step * 20,
		"AssetOptions":         buildEnergyAssetOptions(assets),
		"MappingAssetOptions":  buildEnergyMappingAssetOptions(assets),
		"Candidates":           discovery.Recommended,
		"HasCandidates":        len(discovery.Recommended) > 0,
		"RecommendedCount":     len(discovery.Recommended),
		"AdditionalCandidates": discovery.Additional,
		"HasAdditional":        len(discovery.Additional) > 0,
		"ConnectorOK":          connectorOK,
		"ConnectorMessage":     connectorMessage,
		"FinishRecommendation": finishRecommendation,
		"CanManageEnergy":      a.canManageEnergy(ac),
		"CanControlEnergy":     a.canControlEnergy(ac),
		"IsObserveMode":        profile.OperatingMode == energy.ModeObserve,
		"OnboardingComplete":   profile.OnboardingComplete,
	}))
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
	profile, exists, err := a.energyStore.Profile(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Hausprofil konnte nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	if !exists {
		profile = energy.DefaultProfile(ac.tenant.Slug, time.Now())
		profile.HouseholdName = houseDisplayName(ac.tenant)
	}
	action := strings.TrimSpace(r.FormValue("action"))
	nextStep := profile.OnboardingStep
	switch action {
	case "understand":
		nextStep = 2
	case "profile":
		profile.HouseholdName = cleanEnergyText(r.FormValue("household_name"), 100)
		profile.HomeType = strings.TrimSpace(r.FormValue("home_type"))
		nextStep = 3
	case "assets":
		if err := a.saveOnboardingAssets(ac.tenant.Slug, r.Form["assets"]); err != nil {
			http.Error(w, "Geräte konnten nicht gespeichert werden.", http.StatusInternalServerError)
			return
		}
		nextStep = 4
	case "mappings":
		if ac.tenant.HA.Configured() {
			if err := a.saveSelectedEnergyMappings(r.Context(), ac.tenant, r.Form["entities"]); err != nil {
				http.Error(w, "Messwerte konnten nicht gespeichert werden.", http.StatusBadGateway)
				return
			}
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
			profile.FreeStartedAt = &started
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
	if err := a.energyStore.SaveProfile(profile); err != nil {
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

func (a *app) energyCockpit(w http.ResponseWriter, r *http.Request, ac authCtx) {
	if !a.canViewEnergy(ac) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	profile, exists, err := a.energyStore.Profile(ac.tenant.Slug)
	if err != nil {
		http.Error(w, "Energiedaten konnten nicht geladen werden.", http.StatusInternalServerError)
		return
	}
	if !exists || !profile.OnboardingComplete {
		http.Redirect(w, r, "/app/zuhause/onboarding", http.StatusSeeOther)
		return
	}
	assets, _ := a.energyStore.ListAssets(ac.tenant.Slug)
	mappings, _ := a.energyStore.ListMappings(ac.tenant.Slug)
	metrics, sourceStatus := a.currentEnergyMetrics(r.Context(), ac.tenant, mappings, profile)
	coverage, coverageSummary := buildEnergyCoverageViews(assets, mappings)
	imports, _ := a.energyStore.ListImports(ac.tenant.Slug)
	importViews := make([]energyImportView, 0, len(imports))
	for _, item := range imports {
		importViews = append(importViews, energyImportView{
			Filename: item.Filename,
			Date:     item.ImportedAt.In(time.Local).Format("02.01.2006 15:04"),
			Format:   item.Format,
		})
	}
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
	monthIntervals, _ := a.energyStore.ListIntervals(ac.tenant.Slug, monthStart.UTC(), time.Time{})
	peakViews := energyPeakViews(monthIntervals, time.Now())
	comparisonView, hasComparison := energyComparisonForView(monthIntervals, time.Now())
	recommendation := energy.NextRecommendation(profile, assets, mappings, monthIntervals)
	gaps, conflicts := intervalQualityCounts(monthIntervals)
	lastSeen := latestEnergySeen(mappings, monthIntervals)
	quality := energy.AssessQuality(time.Now(), lastSeen, gaps, conflicts)
	tariffView := buildEnergyTariffView(profile, monthIntervals)
	scenarioViews := buildEnergyScenarioViews(assets, monthIntervals)
	caretakers := a.energyCaretakerViews(ac)
	maintenance, _ := a.energyStore.ListMaintenance(ac.tenant.Slug)
	if maintenanceRecommendation, ok := energy.MaintenanceRecommendation(time.Now(), maintenance); ok {
		recommendation = maintenanceRecommendation
	}
	contactOptions, contactNames := a.energyContactOptions(ac.tenant.Slug)
	documentOptions, documentNames := a.energyDocumentOptions(ac.tenant.Slug)
	issueOptions, issueNames := a.energyIssueOptions(ac.tenant.Slug)
	maintenanceViews := buildEnergyMaintenanceViews(maintenance, assets, contactNames, documentNames, issueNames, time.Now())
	assessments, _ := a.energyStore.ListTariffAssessments(ac.tenant.Slug)
	assessmentViews := buildEnergyTariffAssessmentViews(assessments)
	measures, _ := a.energyStore.ListMeasures(ac.tenant.Slug)
	measureViews := buildEnergyMeasureViews(measures, contactNames)
	freeUntil := ""
	if profile.FreeStartedAt != nil {
		freeUntil = profile.FreeStartedAt.AddDate(3, 0, 0).In(time.Local).Format("02.01.2006")
	}
	a.render(w, "energyCockpit", a.withBase(ac, map[string]any{
		"Title":                   "Mein Zuhause",
		"ActivePage":              "energy",
		"Profile":                 profile,
		"Assets":                  assets,
		"HasAssets":               len(assets) > 0,
		"Mappings":                mappings,
		"HasMappings":             len(mappings) > 0,
		"Metrics":                 metrics,
		"HasMetrics":              len(metrics) > 0,
		"Coverage":                coverage,
		"CoverageSummary":         coverageSummary,
		"SourceStatus":            sourceStatus,
		"Roadmap":                 energyRoadmap(profile, len(assets), len(mappings)),
		"CanManageEnergy":         a.canManageEnergy(ac),
		"CanControlEnergy":        a.canControlEnergy(ac),
		"IsObserveMode":           profile.OperatingMode == energy.ModeObserve,
		"IsActiveMode":            profile.OperatingMode == energy.ModeActive,
		"IsShadowMode":            profile.AutomationStage == energy.StageShadow,
		"FreeUntil":               freeUntil,
		"Welcome":                 r.URL.Query().Get("welcome") == "1",
		"ModeChanged":             r.URL.Query().Get("mode") == "1",
		"Imports":                 importViews,
		"HasImports":              len(importViews) > 0,
		"Peaks":                   peakViews,
		"HasPeaks":                len(peakViews) > 0,
		"Comparison":              comparisonView,
		"HasComparison":           hasComparison,
		"ImportStatus":            r.URL.Query().Get("import"),
		"Recommendation":          recommendation,
		"RecommendationURL":       recommendationURL(recommendation.ID),
		"RecommendationDeferred":  profile.RecommendationID == recommendation.ID && profile.RecommendationStatus == "deferred",
		"RecommendationDismissed": profile.RecommendationID == recommendation.ID && profile.RecommendationStatus == "dismissed",
		"MeasureCreated":          r.URL.Query().Get("measure") == "created",
		"Quality":                 quality,
		"Tariff":                  tariffView,
		"Scenarios":               scenarioViews,
		"HasScenarios":            len(scenarioViews) > 0,
		"TargetChanged":           r.URL.Query().Get("target") == "1",
		"TargetPeakValue":         energyTargetValue(profile.TargetPeakKW),
		"Caretakers":              caretakers,
		"HasCaretakers":           len(caretakers) > 0,
		"CaretakerChanged":        r.URL.Query().Get("caretaker") == "1",
		"CanGrantEnergyAccess":    hasCapability(ac.role, capabilityControlEnergy),
		"CanInviteEnergyAccess":   hasCapability(ac.role, capabilityControlEnergy),
		"CaretakerInviteStatus":   r.URL.Query().Get("caretaker_invite"),
		"Maintenance":             maintenanceViews,
		"HasMaintenance":          len(maintenanceViews) > 0,
		"MaintenanceStatus":       r.URL.Query().Get("maintenance"),
		"ContactOptions":          contactOptions,
		"DocumentOptions":         documentOptions,
		"IssueOptions":            issueOptions,
		"TariffAssessments":       assessmentViews,
		"HasTariffAssessments":    len(assessmentViews) > 0,
		"TariffAssessmentStatus":  r.URL.Query().Get("assessment"),
		"Measures":                measureViews,
		"HasMeasures":             len(measureViews) > 0,
		"MeasureStatus":           r.URL.Query().Get("measure_status"),
		"ServiceAccessEnabled":    a.serviceAccessEnabled,
	}))
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
				formatEnergyNumber(comparison.DeltaPercent) + " % neben der Smart-Meter-Referenz. Zähler, Einheit und Vorzeichen prüfen.",
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

func (a *app) energyContactOptions(tenantSlug string) ([]energyOption, map[string]string) {
	options := []energyOption{}
	names := map[string]string{}
	if a.contactStore == nil {
		return options, names
	}
	for _, item := range a.contactStore.ListTenant(tenantSlug, false) {
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

func (a *app) energyContact(tenantSlug, id string) (managedContact, bool) {
	if a.contactStore == nil || strings.TrimSpace(id) == "" {
		return managedContact{}, false
	}
	for _, item := range a.contactStore.ListTenant(tenantSlug, false) {
		if item.ID == id {
			return item, true
		}
	}
	return managedContact{}, false
}

func (a *app) energyDocumentOptions(tenantSlug string) ([]energyOption, map[string]string) {
	options := []energyOption{}
	names := map[string]string{}
	if a.documentStore == nil {
		return options, names
	}
	for _, item := range a.documentStore.ListCurrentTenant(tenantSlug) {
		names[item.ID] = item.Title
		options = append(options, energyOption{Value: item.ID, Label: item.Title})
	}
	sort.Slice(options, func(i, j int) bool { return strings.ToLower(options[i].Label) < strings.ToLower(options[j].Label) })
	return options, names
}

func (a *app) energyIssueOptions(tenantSlug string) ([]energyOption, map[string]string) {
	options := []energyOption{}
	names := map[string]string{}
	if a.issueStore == nil {
		return options, names
	}
	for _, item := range a.issueStore.ListTenant(tenantSlug) {
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
			Peak: formatEnergyNumber(item.PeakKW) + " kW", Annual: formatEnergyNumber(item.AnnualPowerEUR) + " € Modellwert/Jahr",
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
			view.BeforePeak = formatEnergyNumber(*item.BeforePeakKW) + " kW"
			view.AfterPeak = formatEnergyNumber(*item.AfterPeakKW) + " kW"
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

func (a *app) validEnergyReferences(tenantSlug, contactID, documentID, issueID string) bool {
	if contactID != "" {
		if _, ok := a.energyContact(tenantSlug, contactID); !ok {
			return false
		}
	}
	if documentID != "" {
		if a.documentStore == nil {
			return false
		}
		if _, ok := a.documentStore.Get(tenantSlug, documentID); !ok {
			return false
		}
	}
	if issueID != "" {
		if a.issueStore == nil {
			return false
		}
		if _, ok := a.issueStore.Get(tenantSlug, issueID); !ok {
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
	profile, exists, err := a.energyStore.Profile(ac.tenant.Slug)
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
		if r.FormValue("confirm") != "yes" || strings.TrimSpace(r.FormValue("confirmation_text")) != "AKTIVIEREN" {
			http.Error(w, "Aktivierung muss bewusst bestätigt werden.", http.StatusBadRequest)
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
	if err := a.energyStore.SaveProfile(profile); err != nil {
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
	inserted, err := a.energyStore.PutImport(record, intervals)
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
			"filename":  record.Filename,
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
	if err != nil || value <= 0 || value > 1000 {
		http.Error(w, "Peak-Ziel muss eine positive kW-Zahl sein.", http.StatusBadRequest)
		return
	}
	profile, exists, err := a.energyStore.Profile(ac.tenant.Slug)
	if err != nil || !exists {
		http.Error(w, "Hausprofil fehlt.", http.StatusBadRequest)
		return
	}
	before := ""
	if profile.TargetPeakKW != nil {
		before = formatEnergyNumber(*profile.TargetPeakKW)
	}
	profile.TargetPeakKW = &value
	if err := a.energyStore.SaveProfile(profile); err != nil {
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
	profile, exists, err := a.energyStore.Profile(ac.tenant.Slug)
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
	if err := a.energyStore.SaveProfile(profile); err != nil {
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
	if !a.canViewEnergy(ac) || !canCreateResidentIssue(ac.role) {
		http.Error(w, "Kein Zugriff", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültige Eingabe", http.StatusBadRequest)
		return
	}
	profile, exists, err := a.energyStore.Profile(ac.tenant.Slug)
	if err != nil || !exists {
		http.Error(w, "Hausprofil fehlt.", http.StatusBadRequest)
		return
	}
	assets, _ := a.energyStore.ListAssets(ac.tenant.Slug)
	mappings, _ := a.energyStore.ListMappings(ac.tenant.Slug)
	intervals, _ := a.energyStore.ListIntervals(ac.tenant.Slug, time.Now().AddDate(0, -1, 0), time.Time{})
	recommendation := energy.NextRecommendation(profile, assets, mappings, intervals)
	if maintenance, listErr := a.energyStore.ListMaintenance(ac.tenant.Slug); listErr == nil {
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
	created, err := a.issueStore.Create(residentIssue{
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
	if err := a.energyStore.SaveProfile(profile); err != nil {
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
	if err := a.energyStore.UpsertMeasure(measure); err != nil {
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
	if !hasCapability(ac.role, capabilityControlEnergy) {
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
		logError("energy caretaker invite delivery failed", err, "recipient", redactedEmail(email), "tenant", ac.tenant.Slug)
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
	if !a.validEnergyReferences(ac.tenant.Slug, contactID, documentID, issueID) {
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
	if err := a.energyStore.UpsertMaintenance(plan); err != nil {
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
		if _, found := a.issueStore.Get(ac.tenant.Slug, issueID); !found {
			http.Error(w, "Nachweis-Aufgabe gehört nicht zu diesem Haus.", http.StatusBadRequest)
			return
		}
		plan.IssueID = issueID
	}
	if plan.EvidenceNote == "" && plan.IssueID == "" {
		http.Redirect(w, r, "/app/energie?maintenance=invalid#wartung", http.StatusSeeOther)
		return
	}
	if err := a.energyStore.UpsertMaintenance(plan); err != nil {
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
	intervals, err := a.energyStore.ListIntervals(ac.tenant.Slug, monthStart.UTC(), monthStart.AddDate(0, 1, 0).UTC())
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
	estimate := rules.Estimate(peak, 0)
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
	if err := a.energyStore.SaveTariffAssessment(item); err != nil {
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
	item, ok, err := a.energyStore.GetMeasure(ac.tenant.Slug, strings.TrimSpace(r.FormValue("id")))
	if err != nil || !ok {
		http.Error(w, "Maßnahme nicht gefunden.", http.StatusNotFound)
		return
	}
	issue, found := a.issueStore.Get(ac.tenant.Slug, item.IssueID)
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
	contact, contactOK := a.energyContact(ac.tenant.Slug, item.ContactID)
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
	if err := a.energyStore.UpsertMeasure(item); err != nil {
		http.Error(w, "Maßnahme konnte nicht gespeichert werden.", http.StatusInternalServerError)
		return
	}
	if contactOK && a.serviceAccessEnabled && normalizeEmail(contact.Email) != "" {
		updated, changed, updateErr := a.issueStore.UpdateWorkflow(ac.tenant.Slug, issue.ID, issueWorkflowUpdate{
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
	if !hasCapability(ac.role, capabilityControlEnergy) {
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
	for kind := range selectedSet {
		if err := a.energyStore.UpsertAsset(energy.Asset{
			ID:          energy.StableAssetID(tenantSlug, kind),
			TenantSlug:  tenantSlug,
			Kind:        kind,
			Name:        energy.AssetKindLabel(kind),
			Flexibility: defaultAssetFlexibility(kind),
			Source:      "onboarding",
			Confirmed:   true,
		}); err != nil {
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
	if !tenant.HA.Configured() {
		return nil
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	states, err := tenant.HA.States(timeoutCtx)
	if err != nil {
		return err
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
		energy.MetricPVPower, energy.MetricBatteryPower, energy.MetricBatterySOC, energy.MetricLoadPower:
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
	case energy.MetricBatteryPower, energy.MetricBatterySOC:
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

func (a *app) currentEnergyMetrics(ctx context.Context, tenant tenantConfig, mappings []energy.EntityMapping, profile energy.HomeProfile) ([]energyMetricView, string) {
	if !tenant.HA.Configured() {
		return nil, "Noch keine Messquelle verbunden"
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	type reading struct {
		mapping energy.EntityMapping
		state   homeassistant.EntityState
	}
	readings := make([]reading, 0, len(mappings))
	for _, mapping := range mappings {
		if !mapping.Confirmed {
			continue
		}
		state, err := tenant.HA.State(timeoutCtx, mapping.EntityID)
		if err != nil {
			continue
		}
		readings = append(readings, reading{mapping: mapping, state: state})
	}
	sort.Slice(readings, func(i, j int) bool { return readings[i].mapping.Metric < readings[j].mapping.Metric })
	metrics := make([]energyMetricView, 0, len(readings))
	for _, reading := range readings {
		value, err := homeassistant.ParseFloat(reading.state.State)
		if err != nil {
			continue
		}
		unit := reading.mapping.Unit
		if unit == "" {
			unit = haAttribute(reading.state.Attributes, "unit_of_measurement")
		}
		detail := reading.mapping.DisplayName
		if detail == "" {
			detail = reading.mapping.EntityID
		}
		metrics = append(metrics, energyMetricView{
			Label:  energyMetricLabel(reading.mapping.Metric),
			Value:  formatEnergyNumber(value) + energyUnitSuffix(unit),
			Detail: detail,
			Tone:   energyMetricTone(reading.mapping.Metric, value, profile.TargetPeakKW),
		})
	}
	if len(metrics) == 0 {
		return nil, "Messquelle verbunden, aber noch keine bestätigten Live-Werte"
	}
	return metrics, "Live aus Home Assistant · nur gelesen"
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
			Value:  formatEnergyNumber(energy.PeakForMonth(bySource[source], at, time.Local)) + " kW",
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

func latestEnergySeen(mappings []energy.EntityMapping, intervals []energy.Interval) time.Time {
	var latest time.Time
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

func buildEnergyTariffView(profile energy.HomeProfile, intervals []energy.Interval) energyTariffView {
	rules := energy.AustrianDraft2027()
	view := energyTariffView{
		ID:          rules.ID,
		Version:     rules.Version,
		Status:      "Entwurf · nicht verbindlich",
		SourceTitle: rules.SourceTitle,
		SourceURL:   rules.SourceURL,
		Rule:        "Höchste abgeschlossene Viertelstunde des Monats; Entwurfsannahme mindestens 2 kW und 20 % der vereinbarten Leistung.",
		Disclaimer:  "Keine Tarif- oder Einspargarantie. Neue Verordnungsversionen können ausgetauscht werden, ohne Messdaten zu verändern.",
	}
	peak := energy.PeakForMonth(intervals, time.Now(), time.Local)
	if peak <= 0 {
		return view
	}
	estimate := rules.Estimate(peak, 0)
	view.Estimate = formatEnergyNumber(estimate.AnnualPowerEUR) + " € pro Jahr als reine Modellgröße"
	view.HasEstimate = true
	return view
}

func buildEnergyScenarioViews(assets []energy.Asset, intervals []energy.Interval) []energyScenarioView {
	baseline := energy.PeakForMonth(intervals, time.Now(), time.Local)
	if baseline <= 0 {
		return nil
	}
	has := map[string]bool{}
	for _, asset := range assets {
		has[asset.Kind] = true
	}
	shiftable, throttle, battery := 0.0, 0.0, 0.0
	assumptions := []string{}
	if has["ev"] || has["wallbox"] {
		shiftable += 3
		assumptions = append(assumptions, "E-Auto zeitlich verschiebbar")
	}
	if has["hot-water"] {
		shiftable += 1
		assumptions = append(assumptions, "Warmwasser mit Komfortdeadline")
	}
	if has["heat-pump"] {
		throttle += 1
		assumptions = append(assumptions, "Wärmepumpe kurz begrenzbar")
	}
	if has["battery"] {
		battery += 3
		assumptions = append(assumptions, "Speicherleistung vorerst mit 3 kW modelliert")
	}
	if shiftable+throttle+battery == 0 {
		return nil
	}
	quality := energy.QualityMeasured
	gaps, conflicts := intervalQualityCounts(intervals)
	if gaps > 0 || conflicts > 0 {
		quality = energy.QualityGap
	}
	result := energy.SimulateScenario(energy.ScenarioInput{
		Name: "Bestehende Flexibilität nutzen", BaselinePeakKW: baseline,
		ShiftableKW: shiftable, ThrottleKW: throttle, BatteryKW: battery, DataQuality: quality,
	})
	return []energyScenarioView{{
		Title:       result.Name,
		PeakBand:    formatEnergyNumber(result.ExpectedPeakLowKW) + "–" + formatEnergyNumber(result.ExpectedPeakHighKW) + " kW",
		EffectBand:  formatEnergyNumber(result.PeakEffectLowKW) + "–" + formatEnergyNumber(result.PeakEffectHighKW) + " kW mögliche Peak-Wirkung",
		Uncertainty: result.Uncertainty,
		Assumptions: strings.Join(assumptions, " · "),
	}}
}

func defaultAssetFlexibility(kind string) string {
	switch kind {
	case "ev", "wallbox", "hot-water", "sauna":
		return energy.FlexShift
	case "heat-pump", "battery", "air-conditioning":
		return energy.FlexThrottle
	default:
		return energy.FlexUnknown
	}
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
	case energy.MetricBatterySOC:
		return "Batteriestand"
	case energy.MetricLoadPower:
		return "Verbrauch"
	default:
		return "Messwert"
	}
}

func energyMetricTone(metric string, value float64, target *float64) string {
	if metric == energy.MetricGridImportPower && target != nil && *target > 0 {
		if value >= *target {
			return "danger"
		}
		if value >= *target*0.8 {
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

func energyUnitSuffix(unit string) string {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return ""
	}
	return " " + unit
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
