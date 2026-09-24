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

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeassistant"
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

type energyImportView struct {
	Filename string
	Date     string
	Format   string
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
