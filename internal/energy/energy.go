// Package energy contains the vendor-neutral HAUSV Zuhause energy model.
//
// Home Assistant and device manufacturers are adapters around this package.
// In particular, entity ids, access tokens and service calls are deliberately
// absent from the core profile and asset model.
package energy

import (
	"crypto/rand"
	"encoding/hex"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ModeObserve = "observe"
	ModeActive  = "active"

	StageObserve   = "observe"
	StageRecommend = "recommend"
	StageShadow    = "shadow"
	StageActive    = "active"

	HomeApartment = "apartment"
	HomeHouse     = "house"
	HomeCommunity = "community"

	FlexUnknown  = "unknown"
	FlexFixed    = "fixed"
	FlexShift    = "shift"
	FlexThrottle = "throttle"

	MetricGridImportPower  = "grid-import-power"
	MetricGridImportEnergy = "grid-import-energy"
	MetricGridExportPower  = "grid-export-power"
	MetricPVPower          = "pv-power"
	MetricBatteryPower     = "battery-power"
	MetricBatterySOC       = "battery-soc"
	MetricLoadPower        = "load-power"
	MetricUnknown          = "unknown"
)

type HomeProfile struct {
	TenantSlug           string
	HomeType             string
	HouseholdName        string
	OperatingMode        string
	AutomationStage      string
	OnboardingStep       int
	OnboardingComplete   bool
	TargetPeakKW         *float64
	RecommendationID     string
	RecommendationStatus string
	FreeStartedAt        *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Asset struct {
	ID           string
	TenantSlug   string
	Kind         string
	Name         string
	RatedPowerKW *float64
	Flexibility  string
	Source       string
	Confirmed    bool
	Metadata     map[string]string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type EntityMapping struct {
	ID          string
	TenantSlug  string
	EntityID    string
	AssetID     string
	Metric      string
	DisplayName string
	Unit        string
	DeviceClass string
	Confirmed   bool
	LastSeenAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Interval struct {
	TenantSlug string
	StartsAt   time.Time
	Duration   time.Duration
	ImportKWh  float64
	AverageKW  float64
	Quality    string
	Source     string
	CreatedAt  time.Time
}

type EntityCandidate struct {
	EntityID    string
	DisplayName string
	Unit        string
	DeviceClass string
	StateClass  string
	Metric      string
	Value       *float64
	LastUpdated time.Time
}

type PeakSummary struct {
	CurrentQuarterStart time.Time
	CurrentAverageKW    float64
	ProjectedAverageKW  float64
	MonthlyPeakKW       float64
	TargetPeakKW        *float64
	HeadroomKW          *float64
	Quality             string
}

func DefaultProfile(tenantSlug string, now time.Time) HomeProfile {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	return HomeProfile{
		TenantSlug:      normalizeSlug(tenantSlug),
		HomeType:        HomeApartment,
		OperatingMode:   ModeObserve,
		AutomationStage: StageObserve,
		OnboardingStep:  1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func NormalizeProfile(profile HomeProfile, now time.Time) HomeProfile {
	if now.IsZero() {
		now = time.Now()
	}
	profile.TenantSlug = normalizeSlug(profile.TenantSlug)
	profile.HouseholdName = strings.TrimSpace(profile.HouseholdName)
	profile.RecommendationID = normalizeToken(profile.RecommendationID, "")
	switch strings.ToLower(strings.TrimSpace(profile.RecommendationStatus)) {
	case "deferred", "dismissed", "measure-created":
		profile.RecommendationStatus = strings.ToLower(strings.TrimSpace(profile.RecommendationStatus))
	default:
		profile.RecommendationStatus = ""
	}
	switch strings.ToLower(strings.TrimSpace(profile.HomeType)) {
	case HomeHouse:
		profile.HomeType = HomeHouse
	case HomeCommunity:
		profile.HomeType = HomeCommunity
	default:
		profile.HomeType = HomeApartment
	}
	if profile.OperatingMode != ModeActive {
		profile.OperatingMode = ModeObserve
	}
	switch profile.AutomationStage {
	case StageRecommend, StageShadow:
		// Recommendation and shadow are safe while the house remains observe-only.
	case StageActive:
		if profile.OperatingMode != ModeActive {
			profile.AutomationStage = StageShadow
		}
	default:
		profile.AutomationStage = StageObserve
	}
	if profile.OnboardingStep < 1 {
		profile.OnboardingStep = 1
	}
	if profile.TargetPeakKW != nil {
		value := math.Round(*profile.TargetPeakKW*100) / 100
		if value <= 0 || value > 10000 {
			profile.TargetPeakKW = nil
		} else {
			profile.TargetPeakKW = &value
		}
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now.UTC()
	}
	profile.CreatedAt = profile.CreatedAt.UTC()
	profile.UpdatedAt = now.UTC()
	return profile
}

func NormalizeAsset(asset Asset, now time.Time) Asset {
	if now.IsZero() {
		now = time.Now()
	}
	asset.ID = strings.TrimSpace(asset.ID)
	if asset.ID == "" {
		asset.ID = NewID("asset")
	}
	asset.TenantSlug = normalizeSlug(asset.TenantSlug)
	asset.Kind = normalizeToken(asset.Kind, "other")
	asset.Name = strings.TrimSpace(asset.Name)
	if asset.Name == "" {
		asset.Name = AssetKindLabel(asset.Kind)
	}
	switch strings.ToLower(strings.TrimSpace(asset.Flexibility)) {
	case FlexFixed, FlexShift, FlexThrottle:
		asset.Flexibility = strings.ToLower(strings.TrimSpace(asset.Flexibility))
	default:
		asset.Flexibility = FlexUnknown
	}
	asset.Source = normalizeToken(asset.Source, "manual")
	if asset.RatedPowerKW != nil {
		value := math.Round(*asset.RatedPowerKW*100) / 100
		if value <= 0 || value > 10000 {
			asset.RatedPowerKW = nil
		} else {
			asset.RatedPowerKW = &value
		}
	}
	if asset.Metadata == nil {
		asset.Metadata = map[string]string{}
	}
	if asset.CreatedAt.IsZero() {
		asset.CreatedAt = now.UTC()
	}
	asset.CreatedAt = asset.CreatedAt.UTC()
	asset.UpdatedAt = now.UTC()
	return asset
}

func NormalizeMapping(mapping EntityMapping, now time.Time) EntityMapping {
	if now.IsZero() {
		now = time.Now()
	}
	mapping.ID = strings.TrimSpace(mapping.ID)
	if mapping.ID == "" {
		mapping.ID = NewID("mapping")
	}
	mapping.TenantSlug = normalizeSlug(mapping.TenantSlug)
	mapping.EntityID = strings.ToLower(strings.TrimSpace(mapping.EntityID))
	mapping.AssetID = strings.TrimSpace(mapping.AssetID)
	mapping.Metric = normalizeMetric(mapping.Metric)
	mapping.DisplayName = strings.TrimSpace(mapping.DisplayName)
	mapping.Unit = strings.TrimSpace(mapping.Unit)
	mapping.DeviceClass = strings.ToLower(strings.TrimSpace(mapping.DeviceClass))
	if mapping.CreatedAt.IsZero() {
		mapping.CreatedAt = now.UTC()
	}
	mapping.CreatedAt = mapping.CreatedAt.UTC()
	mapping.UpdatedAt = now.UTC()
	return mapping
}

func AssetKindLabel(kind string) string {
	switch normalizeToken(kind, "other") {
	case "ev":
		return "E-Auto"
	case "wallbox":
		return "Wallbox"
	case "heat-pump":
		return "Wärmepumpe"
	case "hot-water":
		return "Warmwasser"
	case "pv":
		return "PV-Anlage"
	case "battery":
		return "Batteriespeicher"
	case "sauna":
		return "Sauna"
	case "air-conditioning":
		return "Klimaanlage"
	case "instant-water-heater":
		return "Durchlauferhitzer"
	default:
		return "Weiterer Verbraucher"
	}
}

func NewID(prefix string) string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return normalizeToken(prefix, "item") + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return normalizeToken(prefix, "item") + "-" + hex.EncodeToString(raw[:])
}

// StableAssetID keeps bootstrapped and onboarding-managed asset identities
// deterministic without colliding when several homes use the same asset kind.
func StableAssetID(tenantSlug, kind string) string {
	return "asset-" + normalizeSlug(tenantSlug) + "-" + normalizeToken(kind, "other")
}

// ClassifyCandidate translates Home Assistant metadata into a conservative
// measurement suggestion. It never grants write capability.
func ClassifyCandidate(entityID, displayName, unit, deviceClass, stateClass, rawValue string, updated time.Time) (EntityCandidate, bool) {
	entityID = strings.ToLower(strings.TrimSpace(entityID))
	unit = strings.TrimSpace(unit)
	deviceClass = strings.ToLower(strings.TrimSpace(deviceClass))
	stateClass = strings.ToLower(strings.TrimSpace(stateClass))
	name := strings.ToLower(strings.TrimSpace(displayName + " " + entityID))
	if !strings.HasPrefix(entityID, "sensor.") {
		return EntityCandidate{}, false
	}
	normalizedUnit := strings.ToLower(strings.ReplaceAll(unit, " ", ""))
	isPowerUnit := normalizedUnit == "w" || normalizedUnit == "kw" || normalizedUnit == "mw"
	isEnergyUnit := normalizedUnit == "wh" || normalizedUnit == "kwh" || normalizedUnit == "mwh"
	isPower := deviceClass == "power" && isPowerUnit
	isEnergy := deviceClass == "energy" && isEnergyUnit
	isStationaryBattery := containsAny(
		name,
		"sonnenbatterie",
		"home battery",
		"home_battery",
		"house battery",
		"house_battery",
		"hausspeicher",
		"haus speicher",
		"energiespeicher",
		"energy storage",
		"energy_storage",
		"battery storage",
		"battery_storage",
		"speicherstand",
		"battery soc",
		"battery_soc",
		"battery state of charge",
		"battery_state_of_charge",
	)
	isPortableOrVehicleBattery := containsAny(
		name,
		"iphone",
		"ipad",
		"phone",
		"mobile",
		"watch",
		"macbook",
		"laptop",
		"tablet",
		"vehicle",
		"tesla",
		"model 3",
		"model y",
		"auto batterie",
		"car battery",
		"robot",
		"robi",
		"nuki",
		"lock",
		"remote",
		"zigbee",
		"button",
	)
	isForecast := containsAny(name, "forecast", "prediction", "predicted", "estimate", "prognose")
	isGrid := containsAny(name, "grid", "netz")
	isExport := containsAny(name, "export", "feed in", "feed_in", "einspeis")
	isImport := containsAny(name, "import", "bezug", "netzbezug", "grid_import")
	isPV := containsAny(name, "solar", "photovolta", " pv", "pv_", "pv.")
	isBatteryFlow := containsAny(name, "battery", "batter", "speicher") &&
		containsAny(name, "charge", "discharge", "laden", "entladen", "inout", "in_out")
	isHouseLoad := containsAny(
		name,
		"home consumption",
		"home_consumption",
		"house consumption",
		"house_consumption",
		"hausverbrauch",
		"gesamtverbrauch",
		"total consumption",
		"total_consumption",
		"load power",
		"load_power",
		"consumption current",
		"consumption_current",
	)
	metric := MetricUnknown
	switch {
	case isEnergy && isGrid && isExport:
		// Exported energy has no supported metric yet. Treating it as imported
		// energy would invert the meaning of the reading.
		return EntityCandidate{}, false
	case isEnergy && isGrid && isImport:
		metric = MetricGridImportEnergy
	case isPower && isGrid && isExport:
		metric = MetricGridExportPower
	case isPower && isGrid && isImport:
		metric = MetricGridImportPower
	case isPower && isPV && !isForecast:
		metric = MetricPVPower
	case isPower && !isPortableOrVehicleBattery && (isStationaryBattery || isBatteryFlow):
		metric = MetricBatteryPower
	case normalizedUnit == "%" && !isPortableOrVehicleBattery &&
		(deviceClass == "battery" || isStationaryBattery) &&
		containsAny(name, "battery", "batter", "speicher", "state of charge", "state_of_charge", " soc", "_soc"):
		metric = MetricBatterySOC
	case isPower && isHouseLoad && !isForecast:
		metric = MetricLoadPower
	default:
		return EntityCandidate{}, false
	}
	var valuePtr *float64
	value, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(rawValue), ",", "."), 64)
	if err == nil && !math.IsNaN(value) && !math.IsInf(value, 0) {
		valuePtr = &value
	}
	return EntityCandidate{
		EntityID:    entityID,
		DisplayName: strings.TrimSpace(displayName),
		Unit:        unit,
		DeviceClass: deviceClass,
		StateClass:  stateClass,
		Metric:      metric,
		Value:       valuePtr,
		LastUpdated: updated.UTC(),
	}, true
}

func SortCandidates(items []EntityCandidate) {
	metricRank := map[string]int{
		MetricGridImportPower:  0,
		MetricGridImportEnergy: 1,
		MetricPVPower:          2,
		MetricBatteryPower:     3,
		MetricBatterySOC:       4,
		MetricGridExportPower:  5,
		MetricLoadPower:        6,
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, lok := metricRank[items[i].Metric]
		right, rok := metricRank[items[j].Metric]
		if !lok {
			left = 99
		}
		if !rok {
			right = 99
		}
		if left == right {
			return items[i].DisplayName < items[j].DisplayName
		}
		return left < right
	})
}

func QuarterStart(at time.Time, location *time.Location) time.Time {
	if location == nil {
		location = time.Local
	}
	local := at.In(location)
	return local.Add(
		-time.Duration(local.Minute()%15)*time.Minute -
			time.Duration(local.Second())*time.Second -
			time.Duration(local.Nanosecond()),
	)
}

func AveragePowerKW(energyKWh float64, duration time.Duration) float64 {
	hours := duration.Hours()
	if hours <= 0 {
		return 0
	}
	return energyKWh / hours
}

func PeakForMonth(intervals []Interval, at time.Time, location *time.Location) float64 {
	if location == nil {
		location = time.Local
	}
	want := at.In(location)
	peak := 0.0
	for _, interval := range intervals {
		local := interval.StartsAt.In(location)
		if local.Year() == want.Year() && local.Month() == want.Month() && interval.AverageKW > peak {
			peak = interval.AverageKW
		}
	}
	return peak
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func normalizeMetric(raw string) string {
	switch strings.TrimSpace(raw) {
	case MetricGridImportPower, MetricGridImportEnergy, MetricGridExportPower, MetricPVPower, MetricBatteryPower, MetricBatterySOC, MetricLoadPower:
		return strings.TrimSpace(raw)
	default:
		return MetricUnknown
	}
}

func normalizeToken(raw, fallback string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	dash := false
	for _, r := range raw {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if b.Len() > 0 && !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return fallback
	}
	return out
}

func normalizeSlug(raw string) string {
	return normalizeToken(raw, "")
}
