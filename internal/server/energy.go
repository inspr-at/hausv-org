package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
)

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

type energyOption struct {
	Value string
	Label string
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
	if allowed, configured := ac.policy.Configured(ac.actor(), capabilityManageEnergy); configured {
		return store, allowed
	}
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
	if allowed, configured := ac.policy.Configured(ac.actor(), capabilityManageEnergy); configured {
		return allowed && roleCanUseResidentAreas(ac.role)
	}
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

func (a *app) canControlEnergy(ac authCtx) bool {
	if allowed, configured := ac.policy.Configured(ac.actor(), capabilityControlEnergy); configured {
		return allowed && roleCanUseResidentAreas(ac.role)
	}
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

func energyTargetValue(value *float64) string {
	if value == nil {
		return ""
	}
	return formatEnergyNumber(*value)
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
