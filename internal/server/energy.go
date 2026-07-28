package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	Metric      string
	MetricLabel string
	Value       string
	Unit        string
	Checked     bool
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
	CanControl   bool
	IsSelf       bool
	Editable     bool
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
	profile := a.profileForTenant(ac.email, ac.tenant.Slug)
	return hasCapability(ac.role, capabilityControlEnergy) || profile.HasPermission(permissionEnergyControl)
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
	step := profile.OnboardingStep
	if raw := strings.TrimSpace(r.URL.Query().Get("step")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed >= 1 && parsed <= 5 {
			step = parsed
		}
	}
	candidates := []energyCandidateView{}
	connectorMessage := "Home Assistant ist noch nicht verbunden. Das ist okay – Sie können später weitermachen."
	connectorOK := false
	if step == 4 && ac.tenant.HA.Configured() {
		ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
		defer cancel()
		if discovered, discoverErr := discoverEnergyCandidates(ctx, ac.tenant.HA, mappings); discoverErr == nil {
			candidates = discovered
			connectorOK = true
			connectorMessage = fmt.Sprintf("%d passende Messwerte gefunden. Sie entscheiden, welche übernommen werden.", len(candidates))
		} else {
			connectorMessage = "Home Assistant antwortet gerade nicht. Ihre bisherigen Angaben bleiben erhalten."
		}
	}
	a.render(w, "homeOnboarding", a.withBase(ac, map[string]any{
		"Title":              "Mein Zuhause einrichten",
		"ActivePage":         "energy",
		"Profile":            profile,
		"Step":               step,
		"Progress":           step * 20,
		"AssetOptions":       buildEnergyAssetOptions(assets),
		"Candidates":         candidates,
		"HasCandidates":      len(candidates) > 0,
		"ConnectorOK":        connectorOK,
		"ConnectorMessage":   connectorMessage,
		"CanManageEnergy":    a.canManageEnergy(ac),
		"CanControlEnergy":   a.canControlEnergy(ac),
		"IsObserveMode":      profile.OperatingMode == energy.ModeObserve,
		"OnboardingComplete": profile.OnboardingComplete,
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
			if err := a.saveManualEnergyMapping(ac.tenant.Slug, r.FormValue("manual_entity_id"), r.FormValue("manual_metric"), r.FormValue("manual_name"), r.FormValue("manual_unit")); err != nil {
				http.Error(w, "Die manuelle Zuordnung ist ungültig.", http.StatusBadRequest)
				return
			}
		}
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
		stored, hasStored := userProfile{}, false
		if a.inviteStore != nil {
			stored, hasStored = a.inviteStore.Get(member.Email)
		}
		_, isEnv := a.profiles[normalizeEmail(member.Email)]
		editable := hasStored && (!isEnv || stored.Adopted)
		out = append(out, energyCaretakerView{
			Email:        member.Email,
			Name:         member.DisplayName(),
			CanView:      member.HasPermission(permissionEnergyView) || member.HasPermission(permissionEnergyConfigure) || member.HasPermission(permissionEnergyControl) || member.HasPermission(permissionEnergyCaretaker),
			CanConfigure: member.HasPermission(permissionEnergyConfigure) || member.HasPermission(permissionEnergyCaretaker),
			CanControl:   member.HasPermission(permissionEnergyControl),
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
		if grant["view"] || grant["configure"] || grant["control"] {
			out = append(out, permissionEnergyView)
		}
		if grant["configure"] {
			out = append(out, permissionEnergyConfigure)
		}
		if grant["control"] {
			out = append(out, permissionEnergyControl)
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
		id := "asset-" + kind
		if _, keep := selectedSet[kind]; !keep {
			if _, err := a.energyStore.DeleteAsset(tenantSlug, id); err != nil {
				return err
			}
		}
	}
	for kind := range selectedSet {
		if err := a.energyStore.UpsertAsset(energy.Asset{
			ID:          "asset-" + kind,
			TenantSlug:  tenantSlug,
			Kind:        kind,
			Name:        energy.AssetKindLabel(kind),
			Flexibility: defaultAssetFlexibility(kind),
			Source:      "onboarding",
			Confirmed:   true,
		}); err != nil {
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

func (a *app) saveManualEnergyMapping(tenantSlug, entityID, metric, displayName, unit string) error {
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
		Metric:      metric,
		DisplayName: displayName,
		Unit:        cleanEnergyText(unit, 24),
		Confirmed:   true,
	})
}

func discoverEnergyCandidates(ctx context.Context, cfg homeassistant.Config, mappings []energy.EntityMapping) ([]energyCandidateView, error) {
	states, err := cfg.States(ctx)
	if err != nil {
		return nil, err
	}
	confirmed := map[string]bool{}
	for _, mapping := range mappings {
		confirmed[mapping.EntityID] = mapping.Confirmed
	}
	candidates := []energy.EntityCandidate{}
	for _, state := range states {
		if candidate, ok := classifyHAState(state); ok {
			candidates = append(candidates, candidate)
		}
	}
	energy.SortCandidates(candidates)
	out := make([]energyCandidateView, 0, len(candidates))
	for _, candidate := range candidates {
		value := "–"
		if candidate.Value != nil {
			value = formatEnergyNumber(*candidate.Value)
		}
		out = append(out, energyCandidateView{
			EntityID:    candidate.EntityID,
			DisplayName: firstNonEmpty(candidate.DisplayName, candidate.EntityID),
			Metric:      candidate.Metric,
			MetricLabel: energyMetricLabel(candidate.Metric),
			Value:       value,
			Unit:        candidate.Unit,
			Checked:     confirmed[candidate.EntityID],
		})
	}
	return out, nil
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
