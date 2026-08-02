package energy

import (
	"math"
	"sort"
	"strings"
	"time"
)

type TariffRuleProfile struct {
	ID             string
	Version        string
	Status         string
	SourceTitle    string
	SourceURL      string
	ValidFrom      time.Time
	ValidUntil     *time.Time
	QuarterMinutes int
	MonthlyMaximum bool
	MinimumKW      float64
	MinimumShare   float64
	// TierThresholdKW ist die laufende Tarifstaffel nach § 6 Abs. 2 des
	// Entwurfs: bis hierher gilt der günstigere Satz, darüber der höhere.
	//
	// Bewusst NICHT "ReferenceKW": der 10-kW-Referenzwert nach § 18 ist eine
	// einmalige Übergangsbestimmung für Bestandsanschlüsse und hat mit der
	// laufenden Staffel nichts zu tun. Beide tragen denselben Zahlenwert, und
	// sie zu verwechseln ist laut Quelle der häufigste Fehler im Thema.
	TierThresholdKW     float64
	AnnualBelowEURPerKW float64
	AnnualAboveEURPerKW float64
	Assumptions         []string
}

func AustrianDraft2027() TariffRuleProfile {
	return TariffRuleProfile{
		ID:                  "at-ne7-draft-2027-v1",
		Version:             "2026-07-28",
		Status:              "draft",
		SourceTitle:         "E-Control, Begutachtungsentwurf SNE-G-V (V SNE 01_26); Einordnung enercab",
		SourceURL:           "https://www.enercab.eu/leistungstarif-2027.html",
		ValidFrom:           time.Date(2027, time.January, 1, 0, 0, 0, 0, time.Local),
		QuarterMinutes:      15,
		MonthlyMaximum:      true,
		MinimumKW:           2,
		MinimumShare:        0.20,
		TierThresholdKW:     10,
		AnnualBelowEURPerKW: 33.82,
		AnnualAboveEURPerKW: 67.64,
		Assumptions: []string{
			"Höchste mittlere Bezugsleistung eines Kalendermonats.",
			"Mindestbemessung: Entwurfsannahme 20 % der vereinbarten Leistung, mindestens 2 kW.",
			"Preiswerte und die 10-kW-Staffel nach § 6 Abs. 2 sind unverbindliche Modellannahmen, keine geltenden Tarife.",
			"Die Staffel ist nicht der 10-kW-Referenzwert nach § 18; dieser betrifft nur Bestandsanschlüsse einmalig.",
			"SNAP und WiNAP wirken ausschließlich auf den Arbeitspreis und verändern die Monatsspitze nicht.",
			"Energiegemeinschaften senken den Arbeitspreis, nicht die verrechnete Leistung.",
		},
	}
}

type TariffEstimate struct {
	ProfileID      string
	BilledKW       float64
	AnnualPowerEUR float64
	// BelowKW und AboveKW teilen die verrechnete Leistung an der Staffel auf.
	// Getrennt ausgewiesen, weil der Grenznutzen des Kappens oberhalb der
	// Schwelle doppelt so hoch ist — das ist die eigentliche Handlungsaussage.
	BelowKW float64
	AboveKW float64
	// MinimumReason benennt, warum die verrechnete Leistung über der gemessenen
	// Spitze liegt. Leer, solange die Spitze selbst maßgeblich ist. Ohne das
	// wirkt die Mindestbemessung wie ein Rechenfehler.
	MinimumReason   string
	AssumptionLabel string
	Guaranteed      bool
}

// Estimate rechnet die Monatsspitze in die verrechnete Leistung um. agreedKW
// ist die vereinbarte Anschlussleistung; 0 heißt "nicht erfasst", dann bleibt
// die 20-%-Mindestbemessung außen vor und nur der 2-kW-Sockel greift.
func (profile TariffRuleProfile) Estimate(monthlyPeakKW, agreedKW float64) TariffEstimate {
	billed := monthlyPeakKW
	reason := ""
	if billed < profile.MinimumKW {
		billed = profile.MinimumKW
		reason = "Sockel der Mindestbemessung"
	}
	if agreedKW > 0 {
		share := agreedKW * profile.MinimumShare
		if share > billed {
			billed = share
			reason = "Mindestbemessung aus der vereinbarten Leistung"
		}
	}
	below := math.Min(billed, profile.TierThresholdKW)
	above := math.Max(0, billed-profile.TierThresholdKW)
	return TariffEstimate{
		ProfileID:       profile.ID,
		BilledKW:        round2(billed),
		AnnualPowerEUR:  round2(below*profile.AnnualBelowEURPerKW + above*profile.AnnualAboveEURPerKW),
		BelowKW:         round2(below),
		AboveKW:         round2(above),
		MinimumReason:   reason,
		AssumptionLabel: "Unverbindliche Modellrechnung auf Basis eines Begutachtungsentwurfs",
		Guaranteed:      false,
	}
}

type DeviceCapability struct {
	ID              string
	AssetID         string
	Kind            string
	ReadOnly        bool
	MinValue        *float64
	MaxValue        *float64
	Unit            string
	Adapter         string
	ManualOverride  bool
	MinimumRun      time.Duration
	ComfortDeadline *time.Time
}

const (
	CapabilityMeasure  = "measure"
	CapabilityForecast = "forecast"
	CapabilityLimit    = "limit"
	CapabilitySwitch   = "switch"

	DecisionNoop      = "noop"
	DecisionRecommend = "recommend"
	DecisionShadow    = "shadow"
	DecisionApply     = "apply"
)

type ControlInput struct {
	Mode              string
	Stage             string
	Now               time.Time
	LastMeasurementAt time.Time
	CurrentKW         float64
	ProjectedKW       float64
	TargetKW          float64
	Capabilities      []DeviceCapability
}

type ControlDecision struct {
	Action       string
	CapabilityID string
	Requested    *float64
	Reason       string
	Safe         bool
}

func DecideControl(input ControlInput) ControlDecision {
	if input.Now.IsZero() {
		input.Now = time.Now()
	}
	if input.LastMeasurementAt.IsZero() || input.Now.Sub(input.LastMeasurementAt) > 10*time.Minute {
		return ControlDecision{Action: DecisionNoop, Reason: "Messwert ist nicht aktuell; lokal sicher nicht eingreifen.", Safe: true}
	}
	if input.TargetKW <= 0 || input.ProjectedKW <= input.TargetKW {
		return ControlDecision{Action: DecisionNoop, Reason: "Prognose bleibt innerhalb des gesetzten Ziels.", Safe: true}
	}
	candidates := make([]DeviceCapability, 0, len(input.Capabilities))
	for _, capability := range input.Capabilities {
		if capability.ReadOnly || capability.ManualOverride || (capability.Kind != CapabilityLimit && capability.Kind != CapabilitySwitch) {
			continue
		}
		candidates = append(candidates, capability)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	if len(candidates) == 0 {
		return ControlDecision{Action: DecisionRecommend, Reason: "Spitze erwartet, aber kein ausdrücklich freigegebener Geräteadapter verfügbar.", Safe: true}
	}
	action := DecisionRecommend
	if input.Stage == StageShadow {
		action = DecisionShadow
	}
	if input.Mode == ModeActive && input.Stage == StageActive {
		action = DecisionApply
	}
	requested := input.TargetKW
	return ControlDecision{
		Action: action, CapabilityID: candidates[0].ID, Requested: &requested,
		Reason: "Prognose überschreitet das Hausziel; flexible Last zuerst begrenzen.", Safe: true,
	}
}

type ScenarioInput struct {
	Name           string
	BaselinePeakKW float64
	Assets         []Asset
	ShiftableKW    float64
	ThrottleKW     float64
	BatteryKW      float64
	DataQuality    string
}

type ScenarioResult struct {
	Name               string
	BaselinePeakKW     float64
	ExpectedPeakLowKW  float64
	ExpectedPeakHighKW float64
	PeakEffectLowKW    float64
	PeakEffectHighKW   float64
	ComfortAssumption  string
	EnergyAssumption   string
	Uncertainty        string
	Guaranteed         bool
}

func SimulateScenario(input ScenarioInput) ScenarioResult {
	flexible := math.Max(0, input.ShiftableKW)*0.45 + math.Max(0, input.ThrottleKW)*0.35 + math.Max(0, input.BatteryKW)*0.70
	highEffect := math.Min(math.Max(0, input.BaselinePeakKW), flexible)
	lowFactor := 0.35
	uncertainty := "mittel"
	if input.DataQuality == QualityMeasured {
		lowFactor = 0.60
		uncertainty = "niedrig bis mittel"
	} else if input.DataQuality == QualityGap || input.DataQuality == QualityUnavailable {
		lowFactor = 0.15
		uncertainty = "hoch – zuerst messen"
	}
	lowEffect := highEffect * lowFactor
	lowPeak := math.Max(0, input.BaselinePeakKW-highEffect)
	highPeak := math.Max(lowPeak, input.BaselinePeakKW-lowEffect)
	return ScenarioResult{
		Name:               strings.TrimSpace(input.Name),
		BaselinePeakKW:     round2(input.BaselinePeakKW),
		ExpectedPeakLowKW:  round2(lowPeak),
		ExpectedPeakHighKW: round2(highPeak),
		PeakEffectLowKW:    round2(lowEffect),
		PeakEffectHighKW:   round2(highEffect),
		ComfortAssumption:  "Verschiebbare Lasten halten ihre vom Haushalt gesetzten Deadlines und Komfortgrenzen ein.",
		EnergyAssumption:   "Gesamtenergie wird überwiegend zeitlich verschoben, nicht als Einsparung behauptet.",
		Uncertainty:        uncertainty,
		Guaranteed:         false,
	}
}

type Recommendation struct {
	ID           string
	Title        string
	Reason       string
	Benefit      string
	Prerequisite string
	Effort       string
	ImpactRange  string
	State        string
}

func NextRecommendation(profile HomeProfile, assets []Asset, mappings []EntityMapping, intervals []Interval) Recommendation {
	if len(assets) == 0 {
		return Recommendation{ID: "inventory", Title: "Große Verbraucher erfassen", Reason: "Ohne Inventar bleibt unklar, was sich verschieben lässt.", Benefit: "Ausgangslage verstehen", Prerequisite: "Keine", Effort: "5 Minuten", ImpactRange: "Noch keine Peak-Schätzung", State: "now"}
	}
	if len(mappings) == 0 && len(intervals) == 0 {
		return Recommendation{ID: "measure", Title: "Netzbezug messen", Reason: "Ein einzelner Netzbezugswert reicht für den ersten echten Überblick.", Benefit: "15-Minuten-Spitze sichtbar machen", Prerequisite: "Smart Meter oder vorhandener Sensor", Effort: "10–30 Minuten", ImpactRange: "Messung statt Vermutung", State: "now"}
	}
	if CountUsableQuarters(intervals).Total < 96 {
		return Recommendation{ID: "observe", Title: "Einen vollständigen Tag beobachten", Reason: "Für eine Empfehlung fehlen noch ausreichend abgeschlossene Viertelstunden.", Benefit: "Normale Schwankungen von echten Spitzen trennen", Prerequisite: "Aktuelle Messwerte", Effort: "Automatisch", ImpactRange: "Noch keine belastbare Wirkung", State: "now"}
	}
	if profile.TargetPeakKW == nil {
		return Recommendation{ID: "target", Title: "Persönliches Peak-Ziel festlegen", Reason: "Die Messbasis ist da; nun braucht der Fahrplan eine verständliche Zielmarke.", Benefit: "Szenarien vergleichbar machen", Prerequisite: "Mindestens ein Messtag", Effort: "5 Minuten", ImpactRange: "Modellbandbreite, keine Garantie", State: "now"}
	}
	return Recommendation{ID: "simulate", Title: "Zeitverschiebung zuerst simulieren", Reason: "Vor Hardware oder Steuerung sollte sichtbar sein, was bestehende Verbraucher beitragen.", Benefit: "Kaufentscheidungen vermeiden oder begründen", Prerequisite: "Peak-Ziel und Verbraucher", Effort: "10 Minuten", ImpactRange: "Bandbreite aus vorhandenen Daten", State: "now"}
}

type MeasurePackage struct {
	ID                string
	TenantSlug        string
	RecommendationID  string
	Goal              string
	Baseline          string
	RequestedWork     string
	ExpectedEvidence  string
	SharedFields      []string
	OpenSiteQuestions []string
	Status            string
	MarketplaceGate   string
}

func NewMeasurePackage(tenantSlug string, recommendation Recommendation) MeasurePackage {
	return MeasurePackage{
		ID:                NewID("measure"),
		TenantSlug:        normalizeSlug(tenantSlug),
		RecommendationID:  recommendation.ID,
		Goal:              recommendation.Benefit,
		Baseline:          recommendation.Reason,
		RequestedWork:     recommendation.Title,
		ExpectedEvidence:  "Ausführung, Datum, relevante Unterlagen und nachvollziehbarer Vorher-/Nachher-Messzeitraum",
		SharedFields:      []string{},
		OpenSiteQuestions: []string{"Einbausituation vor Ort prüfen", "Erforderlichen Fachnachweis festlegen"},
		Status:            "draft",
		MarketplaceGate:   "closed-until-pilot-evidence",
	}
}

type PP20Adapter struct {
	MeterEnergyEntity string
	PowerEntity       string
	PlugSwitchEntity  string
	BatterySOCEntity  string
	GridFeedInEntity  string
}

func (adapter PP20Adapter) Capabilities() []DeviceCapability {
	out := []DeviceCapability{
		{ID: "pp20-energy", AssetID: "pp20", Kind: CapabilityMeasure, ReadOnly: true, Unit: "kWh", Adapter: "pp20"},
		{ID: "pp20-power", AssetID: "pp20", Kind: CapabilityMeasure, ReadOnly: true, Unit: "W", Adapter: "pp20"},
	}
	if strings.TrimSpace(adapter.PlugSwitchEntity) != "" {
		out = append(out, DeviceCapability{ID: "pp20-switch", AssetID: "pp20", Kind: CapabilitySwitch, Adapter: "pp20", ManualOverride: true})
	}
	return out
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
