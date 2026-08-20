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

	"github.com/inspr-at/hausv-org/internal/textutil"
)

const (
	DefaultHomeKey = "default"

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
	MetricBatteryCharge    = "battery-charge-power"
	MetricBatteryDischarge = "battery-discharge-power"
	MetricBatterySOC       = "battery-soc"
	MetricLoadPower        = "load-power"
	MetricConsumerPower    = "consumer-power"
	MetricConsumerEnergy   = "consumer-energy"
	MetricConsumerSleep    = "consumer-sleep"
	MetricUnknown          = "unknown"
)

type HomeProfile struct {
	TenantSlug         string
	HomeKey            string
	UnitID             string
	HomeType           string
	HouseholdName      string
	OperatingMode      string
	AutomationStage    string
	OnboardingStep     int
	OnboardingComplete bool
	TargetPeakKW       *float64
	// AgreedPowerKW ist die mit dem Netzbetreiber vereinbarte Anschlussleistung.
	// nil heißt "nicht erfasst": die Mindestbemessung bleibt dann stumm, statt
	// stillschweigend mit 0 zu rechnen.
	AgreedPowerKW        *float64
	RecommendationID     string
	RecommendationStatus string
	FreeStartedAt        *time.Time
	FreeUntilAt          *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Asset struct {
	ID           string
	TenantSlug   string
	HomeKey      string
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
	HomeKey     string
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
	HomeKey    string
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
	return DefaultProfileForHome(tenantSlug, DefaultHomeKey, now)
}

func DefaultProfileForHome(tenantSlug, homeKey string, now time.Time) HomeProfile {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	return HomeProfile{
		TenantSlug:      normalizeSlug(tenantSlug),
		HomeKey:         NormalizeHomeKey(homeKey),
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
	profile.HomeKey = NormalizeHomeKey(profile.HomeKey)
	profile.UnitID = textutil.UnitID(profile.UnitID)
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
		if !ValidPowerKW(value, 10000) {
			profile.TargetPeakKW = nil
		} else {
			profile.TargetPeakKW = &value
		}
	}
	if profile.AgreedPowerKW != nil {
		value := math.Round(*profile.AgreedPowerKW*100) / 100
		// Ein Hausanschluss auf Netzebene 7 liegt realistisch zwischen wenigen
		// und einigen hundert kW. Unplausibles verwerfen statt zu speichern:
		// der Wert steuert die Mindestbemessung und damit eine Geldgröße.
		if !ValidPowerKW(value, 1000) {
			profile.AgreedPowerKW = nil
		} else {
			profile.AgreedPowerKW = &value
		}
	}
	if profile.FreeStartedAt != nil {
		started := profile.FreeStartedAt.UTC()
		profile.FreeStartedAt = &started
		if profile.FreeUntilAt == nil {
			// Profiles that predate the explicit end marker keep the original
			// three-year pilot promise. New onboarding always supplies both
			// timestamps and therefore receives the current twelve-month term.
			until := started.AddDate(3, 0, 0)
			profile.FreeUntilAt = &until
		}
	}
	if profile.FreeUntilAt != nil {
		until := profile.FreeUntilAt.UTC()
		profile.FreeUntilAt = &until
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
	asset.HomeKey = NormalizeHomeKey(asset.HomeKey)
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
		if !ValidPowerKW(value, 10000) {
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
	mapping.HomeKey = NormalizeHomeKey(mapping.HomeKey)
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
	return StableAssetIDForHome(tenantSlug, DefaultHomeKey, kind)
}

func StableAssetIDForHome(tenantSlug, homeKey, kind string) string {
	base := "asset-" + normalizeSlug(tenantSlug)
	if normalized := NormalizeHomeKey(homeKey); normalized != DefaultHomeKey {
		base += "-" + normalized
	}
	return base + "-" + normalizeToken(kind, "other")
}

// NormalizeHomeKey returns the stable, tenant-local key used to isolate one
// Zuhause. The default keeps every pre-0.78 single-home installation working
// without new configuration.
//
// It deliberately keeps the strict token normalizer rather than following the
// tenant slug onto textutil.Slug. A home key is tenant-LOCAL: it never resolves
// to an identity and nothing outside this package agrees on what it means, so
// there is no second normalizer to reconcile with. Widening it would silently
// change the stored key of every existing home whose key contains anything
// outside [a-z0-9-] and make those rows unreachable, which is a data migration,
// not a cleanup. TestHomeKeyNormalizationIsNotTheTenantNormalizer pins this.
func NormalizeHomeKey(raw string) string {
	if normalized := normalizeToken(raw, ""); normalized != "" {
		return normalized
	}
	return DefaultHomeKey
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
	case isPower && isHouseLoad && !isForecast:
		// Some stationary battery integrations prefix every sensor with the
		// battery product name. Consumption is still the house load and must
		// win over the broader battery-name heuristic below.
		metric = MetricLoadPower
	case isPower && !isPortableOrVehicleBattery && (isStationaryBattery || isBatteryFlow):
		metric = MetricBatteryPower
	case normalizedUnit == "%" && !isPortableOrVehicleBattery &&
		(deviceClass == "battery" || isStationaryBattery) &&
		containsAny(name, "battery", "batter", "speicher", "state of charge", "state_of_charge", " soc", "_soc"):
		metric = MetricBatterySOC
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

// ValidPowerKW prüft eine Leistungsangabe in kW.
//
// NaN und ±Inf müssen ausdrücklich abgefangen werden: jeder Vergleich mit NaN
// ist falsch, ein bloßes "value <= 0 || value > max" ließe sie also durch.
// "NaN" ist eine gültige Eingabe für strconv.ParseFloat, und ein NaN im Modell
// macht aus jeder Kennzahl "NaN kW".
func ValidPowerKW(value, max float64) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}
	return value > 0 && value <= max
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
	case MetricGridImportPower, MetricGridImportEnergy, MetricGridExportPower, MetricPVPower, MetricBatteryPower, MetricBatteryCharge, MetricBatteryDischarge, MetricBatterySOC, MetricLoadPower, MetricConsumerPower, MetricConsumerEnergy, MetricConsumerSleep:
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

// normalizeSlug canonicalises a TENANT slug, and it is textutil.Slug because
// that is what every other layer already means by "slug": internal/config
// accepts textutil.Slug output as the legal form, and internal/tenantid resolves
// an identity under it.
//
// It used to be normalizeToken, which strips everything outside [a-z0-9]. That
// made two normalizers for one concept, and the moment energy writes started
// carrying a tenant_id the divergence stopped being cosmetic: for a configured
// house like "haus.a" or "haus-grün" the boot minted one identity and the first
// energy write minted a SECOND, stamping the wrong id onto the row. A wrong
// tenant_id is invisible to the boot-time completeness check, which counts NULLs.
//
// It also collided: "haus-a" and "haus.a" both collapsed to "haus-a", so two
// houses shared one home_profiles row and the second could not store energy data
// under its own identity at all.
//
// TestTenantSlugNormalizersAgree fails if the two ever drift apart again.
func normalizeSlug(raw string) string {
	return textutil.Slug(raw)
}
