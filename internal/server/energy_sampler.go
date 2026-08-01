package server

// Der Viertelstunden-Sampler schließt die Lücke zwischen einer laufenden
// Home-Assistant-Verbindung und dem Leistungstarif.
//
// Bis hierher war der CSV-Import (PutImport) der einzige Schreiber von
// Interval-Zeilen. Ein Haus, das seit Monaten live gelesen wird, aber nie
// exportiert hat, las deshalb dauerhaft „Für diesen Monat liegt noch keine
// abgeschlossene Viertelstunde vor" — und hätte das für immer gelesen. Ohne
// aufgezeichnete Viertelstunden bleibt das gesamte Cockpit (Spitze,
// verrechnete Leistung, Staffel, Szenarien) leer.
//
// Drei Grenzen sind hier bewusst gezogen:
//
//   - Messen ist nicht Steuern. Der Sampler liest ausschließlich
//     GET /api/states/<entity>. Er ruft keinen Dienst auf und schreibt nichts
//     nach Home Assistant; er läuft deshalb auch im Beobachtungsmodus.
//   - Nur mit bestätigter Zuordnung. Ohne bestätigten Netzbezug misst HAUSV
//     nichts. Die Erkennung bleibt unverändert.
//   - Nur abgeschlossene Viertelstunden. Eine laufende Viertelstunde wird nie
//     geschrieben.

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/markus-barta/hausv-org/internal/energy"
	"github.com/markus-barta/hausv-org/internal/homeassistant"
)

const (
	// energySampleReadTimeout begrenzt einen einzelnen Zustandsabruf.
	energySampleReadTimeout = 8 * time.Second

	// energySampleStaleAfter deckt sich mit der Frist in energy.AssessQuality:
	// ein Messwert, der länger nicht mehr aktualisiert wurde, gilt als
	// eingefroren und zählt nicht als Beleg für die Viertelstunde.
	energySampleStaleAfter = 10 * time.Minute

	// energySampleMaxBuffer begrenzt den Messwertpuffer je Haus. Er wird an
	// jeder Viertelstundengrenze geleert; die Grenze schützt nur davor, dass
	// eine stehengebliebene Uhr unbegrenzt Speicher belegt.
	energySampleMaxBuffer = 4096
)

// energySamplerState ist der laufende Puffer eines Hauses. Er hält die
// Messwerte der aktuellen Viertelstunde und genau einen Messwert davor: dieser
// „Vorhalt" beschreibt die Leistung, die zu Beginn der Viertelstunde galt.
type energySamplerState struct {
	entityID      string
	quarterStart  time.Time
	samples       []energy.PowerSample
	staleReadings int
}

// startEnergyIntervalSampler startet den Hintergrundlauf; die zurückgegebene
// Funktion stoppt ihn. Das Intervall ist injizierbar, damit Tests nie warten
// müssen — sie rufen sampleEnergyTenant direkt mit einer gesetzten Zeit auf.
func (a *app) startEnergyIntervalSampler() func() {
	if a.energyStore == nil {
		logInfo("energy interval sampler disabled", "reason", "no_energy_store")
		return func() {}
	}
	if a.energySampleInterval <= 0 {
		logInfo("energy interval sampler disabled", "reason", "interval_disabled")
		return func() {}
	}
	configured := 0
	for _, tenant := range a.tenants {
		if tenant.HA.Configured() {
			configured++
		}
	}
	if configured == 0 {
		logInfo("energy interval sampler disabled", "reason", "no_configured_home_assistant_tenants")
		return func() {}
	}
	logInfo("energy interval sampler enabled", "tenants", configured, "interval", a.energySampleInterval)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(a.energySampleInterval)
		defer ticker.Stop()
		a.sampleEnergyTenants(ctx, time.Now())
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				a.sampleEnergyTenants(ctx, now)
			}
		}
	}()
	return cancel
}

func (a *app) sampleEnergyTenants(ctx context.Context, now time.Time) {
	for _, tenant := range a.tenants {
		if !tenant.HA.Configured() {
			continue
		}
		a.sampleEnergyTenant(ctx, tenant, now)
	}
}

// sampleEnergyTenant liest genau einen Netzbezugswert dieses Hauses und
// schreibt die vorige Viertelstunde fest, sobald sie vorbei ist. Zurückgegeben
// werden die dabei geschriebenen Viertelstunden.
func (a *app) sampleEnergyTenant(ctx context.Context, tenant tenantConfig, now time.Time) []energy.Interval {
	mapping, ok := a.confirmedGridImportMapping(tenant.Slug)
	if !ok {
		// Ohne bestätigten Netzbezug wird nicht gemessen, und ein früher
		// aufgebauter Puffer beschreibt nichts mehr, was noch zugeordnet ist.
		a.resetEnergySampler(tenant.Slug)
		return nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	value, usable, stale := a.readEnergyGridImport(ctx, tenant, mapping, now)

	a.energySamplerMu.Lock()
	defer a.energySamplerMu.Unlock()
	state := a.energySamplerStateLocked(tenant.Slug)
	if state.entityID != mapping.EntityID {
		// Andere Messstelle: der alte Puffer beschreibt eine andere Größe.
		*state = energySamplerState{entityID: mapping.EntityID}
	}

	quarter := energy.QuarterStart(now, time.Local).UTC()
	if state.quarterStart.IsZero() {
		state.quarterStart = quarter
	}
	recorded := []energy.Interval{}
	if quarter.After(state.quarterStart) {
		if interval, complete := energy.QuarterFromPowerSamples(energy.QuarterInput{
			TenantSlug:    tenant.Slug,
			Source:        energy.SourceHomeAssistant,
			Start:         state.quarterStart,
			Now:           now,
			Location:      time.Local,
			Samples:       state.samples,
			StaleReadings: state.staleReadings,
			Policy:        a.energySamplingPolicy(),
		}); complete {
			if err := a.energyStore.PutInterval(interval); err != nil {
				logError("home assistant quarter hour not recorded", err, "tenant", tenant.Slug)
			} else {
				recorded = append(recorded, interval)
				logInfo("home assistant quarter hour recorded",
					"tenant", tenant.Slug,
					"starts_at", interval.StartsAt.Format(time.RFC3339),
					"average_kw", interval.AverageKW,
					"quality", interval.Quality,
					"samples", len(state.samples),
				)
			}
		}
		// Übersprungene Viertelstunden werden bewusst nicht nachgetragen: für sie
		// wurde nichts beobachtet, und eine Zeile wäre die Behauptung, doch
		// hingesehen zu haben.
		state.samples = energyCarryForward(state.samples, quarter)
		state.staleReadings = 0
		state.quarterStart = quarter
	}

	switch {
	case usable:
		state.samples = append(state.samples, energy.PowerSample{At: now.UTC(), KW: value})
		if len(state.samples) > energySampleMaxBuffer {
			state.samples = state.samples[len(state.samples)-energySampleMaxBuffer:]
		}
	case stale:
		state.staleReadings++
	}
	return recorded
}

// confirmedGridImportMapping liefert die bestätigte Netzbezugs-Leistung dieses
// Hauses.
//
// Gemessen wird MetricGridImportPower und nicht MetricGridImportEnergy: der
// Leistungstarif bemisst die mittlere Bezugsleistung einer Viertelstunde, und
// genau diese Größe wird hier abgetastet. Ein kumulativer kWh-Zähler wäre
// theoretisch genauer, ist in Home Assistant aber typischerweise auf 0,1 oder
// 1 kWh gerundet — über eine Viertelstunde entspräche das mehreren hundert
// Watt Unsicherheit auf einem Wert, der verrechnet wird.
func (a *app) confirmedGridImportMapping(tenantSlug string) (energy.EntityMapping, bool) {
	if a.energyStore == nil {
		return energy.EntityMapping{}, false
	}
	mappings, err := a.energyStore.ListMappings(tenantSlug)
	if err != nil {
		logError("energy mappings unavailable for quarter hour sampling", err, "tenant", tenantSlug)
		return energy.EntityMapping{}, false
	}
	var best energy.EntityMapping
	found := false
	for _, mapping := range mappings {
		if !mapping.Confirmed || mapping.Metric != energy.MetricGridImportPower {
			continue
		}
		if strings.TrimSpace(mapping.EntityID) == "" {
			continue
		}
		// Mehrere bestätigte Netzbezugssensoren sind ungewöhnlich, aber möglich.
		// Die Auswahl muss über Neustarts hinweg dieselbe bleiben, sonst wechselt
		// die Messgrundlage unbemerkt.
		if !found || mapping.EntityID < best.EntityID {
			best = mapping
			found = true
		}
	}
	return best, found
}

// readEnergyGridImport liest den Zustand einer Entität. Nur lesend: State ruft
// GET /api/states/<entity> auf. Rückgabe: Wert in kW, ob er brauchbar ist, und
// ob die Antwort zwar kam, der Messwert aber eingefroren war.
func (a *app) readEnergyGridImport(ctx context.Context, tenant tenantConfig, mapping energy.EntityMapping, now time.Time) (float64, bool, bool) {
	readCtx, cancel := context.WithTimeout(ctx, energySampleReadTimeout)
	defer cancel()
	state, err := tenant.HA.State(readCtx, mapping.EntityID)
	if err != nil {
		return 0, false, false
	}
	if energyReadingStale(state, now) {
		return 0, false, true
	}
	value, ok := energyGridImportKW(mapping, state)
	if !ok {
		return 0, false, false
	}
	return value, true, false
}

// energyReadingStale erkennt einen eingefrorenen Messwert. „unavailable" ist
// etwas anderes und fällt weiter unten durch die Zahlenprüfung: eine Lücke
// bleibt eine Lücke.
func energyReadingStale(state homeassistant.EntityState, now time.Time) bool {
	seen := state.LastUpdated
	if seen.IsZero() {
		seen = state.LastChanged
	}
	if seen.IsZero() {
		return false
	}
	return now.Sub(seen) > energySampleStaleAfter
}

// energyGridImportKW rechnet den Zustand in kW um. Eine unbekannte Einheit wird
// verworfen statt geraten — aus „2,4" ohne Einheit lässt sich keine
// Leistungsangabe machen, die später verrechnet wird.
func energyGridImportKW(mapping energy.EntityMapping, state homeassistant.EntityState) (float64, bool) {
	value, err := homeassistant.ParseFloat(state.State)
	if err != nil {
		return 0, false
	}
	unit := strings.TrimSpace(mapping.Unit)
	if unit == "" {
		unit = haAttribute(state.Attributes, "unit_of_measurement")
	}
	factor, ok := energyPowerUnitFactorKW(unit)
	if !ok {
		return 0, false
	}
	kw := value * factor
	if math.IsNaN(kw) || math.IsInf(kw, 0) {
		return 0, false
	}
	if kw < 0 {
		// Ein vorzeichenbehafteter Netzsensor meldet Einspeisung negativ. Für den
		// Bezug ist das null und kein negativer Bezug.
		kw = 0
	}
	return kw, true
}

func energyPowerUnitFactorKW(unit string) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "w":
		return 0.001, true
	case "kw":
		return 1, true
	case "mw":
		return 1000, true
	default:
		return 0, false
	}
}

// energyCarryForward behält genau einen Messwert vor der neuen Viertelstunde.
// Ohne ihn wäre der Beginn jeder Viertelstunde unbelegt und die erste
// Viertelstunde nach jedem Wechsel eine Messlücke.
func energyCarryForward(samples []energy.PowerSample, start time.Time) []energy.PowerSample {
	out := make([]energy.PowerSample, 0, len(samples)+1)
	carry := energy.PowerSample{}
	hasCarry := false
	for _, sample := range samples {
		if sample.At.Before(start) {
			carry = sample
			hasCarry = true
			continue
		}
		out = append(out, sample)
	}
	if !hasCarry {
		return out
	}
	return append([]energy.PowerSample{carry}, out...)
}

func (a *app) energySamplingPolicy() energy.SamplingPolicy {
	return energy.DefaultSamplingPolicy(a.energySampleInterval)
}

func (a *app) energySamplerStateLocked(tenantSlug string) *energySamplerState {
	if a.energySamplers == nil {
		a.energySamplers = map[string]*energySamplerState{}
	}
	state, ok := a.energySamplers[tenantSlug]
	if !ok {
		state = &energySamplerState{}
		a.energySamplers[tenantSlug] = state
	}
	return state
}

func (a *app) resetEnergySampler(tenantSlug string) {
	a.energySamplerMu.Lock()
	defer a.energySamplerMu.Unlock()
	delete(a.energySamplers, tenantSlug)
}

// StartEnergyIntervalSampler startet den Viertelstunden-Sampler; die
// zurückgegebene Funktion stoppt ihn.
func (a *app) StartEnergyIntervalSampler() func() { return a.startEnergyIntervalSampler() }
