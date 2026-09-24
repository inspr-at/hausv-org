package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/homeconnector"
	"github.com/inspr-at/hausv-org/internal/web"
)

const (
	energyConsumerDefaultStaleAfter = 10 * time.Minute
	energyConsumerMaxStaleMinutes   = 24 * 60
)

type energyAssetOption struct {
	Kind    string
	Label   string
	Checked bool
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
