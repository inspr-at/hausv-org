package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeassistant"
)

// HAUSV-428: Ohne diesen Sampler entsteht auf einem Haus mit reiner
// Home-Assistant-Anbindung nie eine abgeschlossene Viertelstunde — und damit
// nie eine Monatsspitze, nie eine verrechnete Leistung, nie ein Szenario.
//
// Die Prüfungen fahren die Uhr selbst; nichts wartet.

const (
	sampleIntervalHAUSV428 = 30 * time.Second
	samplesPerQuarterHAUSV = int(15 * time.Minute / sampleIntervalHAUSV428)
	gridEntityHAUSV428     = "sensor.grid_import_power"
)

// fakeHomeAssistantHAUSV428 ist eine ausschließlich lesbare Fixture. Sie
// protokolliert jeden Zugriff, damit bewiesen werden kann, dass der Sampler
// nichts nach Home Assistant schreibt.
type fakeHomeAssistantHAUSV428 struct {
	mu          sync.Mutex
	kw          float64
	lastUpdated time.Time
	offline     bool
	unavailable bool
	requests    []string
	writes      int
	server      *httptest.Server
}

func newFakeHomeAssistantHAUSV428(t *testing.T) *fakeHomeAssistantHAUSV428 {
	t.Helper()
	fake := &fakeHomeAssistantHAUSV428{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		fake.requests = append(fake.requests, r.Method+" "+r.URL.Path)
		// Alles, was kein lesender Zustandsabruf ist, gilt als Schreibzugriff —
		// Dienstaufrufe (/api/services/...) eingeschlossen.
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/api/states") {
			fake.writes++
		}
		offline, unavailable, kw, updated := fake.offline, fake.unavailable, fake.kw, fake.lastUpdated
		fake.mu.Unlock()

		if offline {
			http.Error(w, "offline", http.StatusServiceUnavailable)
			return
		}
		state := "unavailable"
		if !unavailable {
			state = fmt.Sprintf("%g", kw)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entity_id":    gridEntityHAUSV428,
			"state":        state,
			"attributes":   map[string]any{"unit_of_measurement": "kW", "device_class": "power"},
			"last_updated": updated.UTC().Format(time.RFC3339Nano),
			"last_changed": updated.UTC().Format(time.RFC3339Nano),
		})
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

func (f *fakeHomeAssistantHAUSV428) reading(kw float64, at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kw, f.lastUpdated, f.offline, f.unavailable = kw, at, false, false
}

func (f *fakeHomeAssistantHAUSV428) goOffline() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.offline = true
}

func (f *fakeHomeAssistantHAUSV428) goUnavailable(at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unavailable, f.offline, f.lastUpdated = true, false, at
}

// freeze lässt die Verbindung antworten, den Messwert aber einfrieren.
func (f *fakeHomeAssistantHAUSV428) freeze(kw float64, frozenAt time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kw, f.lastUpdated, f.offline, f.unavailable = kw, frozenAt, false, false
}

func (f *fakeHomeAssistantHAUSV428) auditReadOnly(t *testing.T) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writes != 0 {
		t.Fatalf("Messen ist nicht Steuern: %d schreibende Zugriffe auf Home Assistant (%v)", f.writes, f.requests)
	}
	if len(f.requests) == 0 {
		t.Fatal("es wurde überhaupt nicht gelesen")
	}
	for _, request := range f.requests {
		if request != "GET /api/states/"+gridEntityHAUSV428 {
			t.Fatalf("unerwarteter Zugriff %q; erlaubt ist nur GET /api/states/<entity>", request)
		}
	}
}

func (f *fakeHomeAssistantHAUSV428) config() homeassistant.Config {
	return homeassistant.NewConfig(f.server.URL, "fixture", "", "", "")
}

// quarterAnchorHAUSV428 verankert die Prüfung im laufenden Kalendermonat.
// PeakForMonth filtert nach Kalendermonat — ein fest verdrahtetes Datum wäre am
// Monatswechsel tot, und genau das ist hier schon zweimal passiert.
func quarterAnchorHAUSV428() time.Time {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	anchor := energy.QuarterStart(now, time.Local).Add(-15 * time.Minute)
	if anchor.Before(monthStart) {
		return monthStart
	}
	return anchor
}

func samplerAppHAUSV428(t *testing.T) *app {
	t.Helper()
	a := newTestPortalApp(t, userProfile{
		Email:       "owner@example.com",
		Role:        roleOwner,
		Tenants:     []string{"demo"},
		AuthMethods: defaultAuthMethods(),
	})
	a.energySampleInterval = sampleIntervalHAUSV428
	a.energySamplers = map[string]*energySamplerState{}
	return a
}

func confirmGridImportHAUSV428(t *testing.T, a *app, tenantSlug, entityID string) {
	t.Helper()
	if err := a.energyStore.UpsertMapping(energy.EntityMapping{
		ID:          energy.NewID("mapping"),
		TenantSlug:  tenantSlug,
		EntityID:    entityID,
		Metric:      energy.MetricGridImportPower,
		DisplayName: "Netzbezug",
		Unit:        "kW",
		Confirmed:   true,
	}); err != nil {
		t.Fatalf("Zuordnung speichern: %v", err)
	}
}

func samplerTenantHAUSV428(a *app, slug string, fake *fakeHomeAssistantHAUSV428) tenantConfig {
	tenant := a.tenants[slug]
	tenant.Slug = slug
	tenant.HA = fake.config()
	return tenant
}

// runQuarterHAUSV428 fährt eine vollständige Viertelstunde ab und überquert die
// Grenze, sodass sie festgeschrieben wird.
func runQuarterHAUSV428(t *testing.T, a *app, tenant tenantConfig, fake *fakeHomeAssistantHAUSV428, start time.Time, kw float64) []energy.Interval {
	t.Helper()
	for i := 0; i < samplesPerQuarterHAUSV; i++ {
		at := start.Add(time.Duration(i) * sampleIntervalHAUSV428)
		fake.reading(kw, at)
		a.sampleEnergyTenant(t.Context(), tenant, at)
	}
	boundary := start.Add(15 * time.Minute)
	fake.reading(kw, boundary)
	return a.sampleEnergyTenant(t.Context(), tenant, boundary)
}

func TestSamplerRecordsCompletedQuarterHourFromHomeAssistantHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	tenant := samplerTenantHAUSV428(a, "demo", fake)
	start := quarterAnchorHAUSV428()

	recorded := runQuarterHAUSV428(t, a, tenant, fake, start, 4.81)
	if len(recorded) != 1 {
		t.Fatalf("festgeschriebene Viertelstunden = %d, erwartet 1", len(recorded))
	}
	intervals, err := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	if err != nil || len(intervals) != 1 {
		t.Fatalf("Intervalle = %+v err=%v", intervals, err)
	}
	got := intervals[0]
	if got.Source != "home-assistant" {
		t.Fatalf("Quelle = %q, erwartet home-assistant", got.Source)
	}
	if math.Abs(got.AverageKW-4.81) > 1e-9 || math.Abs(got.ImportKWh-4.81/4) > 1e-9 {
		t.Fatalf("Mittelwert = %v kW / %v kWh", got.AverageKW, got.ImportKWh)
	}
	if got.Quality != energy.QualityEstimated {
		t.Fatalf("Qualität = %q, erwartet %q", got.Quality, energy.QualityEstimated)
	}
	if !got.StartsAt.Equal(start.UTC()) || got.Duration != 15*time.Minute {
		t.Fatalf("Ausrichtung = %s / %v, erwartet %s / 15m", got.StartsAt, got.Duration, start.UTC())
	}
	// Der eigentliche Zweck: die Monatsspitze existiert jetzt.
	if peak := energy.PeakForMonth(intervals, start, time.Local); math.Abs(peak-4.81) > 1e-9 {
		t.Fatalf("Monatsspitze = %v kW, erwartet 4,81 kW", peak)
	}
	fake.auditReadOnly(t)
}

func TestSamplerNeverWritesARunningQuarterHourHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	tenant := samplerTenantHAUSV428(a, "demo", fake)
	start := quarterAnchorHAUSV428()

	// Vollständige Abtastung, aber die Viertelstunde läuft noch.
	for i := 0; i < samplesPerQuarterHAUSV; i++ {
		at := start.Add(time.Duration(i) * sampleIntervalHAUSV428)
		fake.reading(7.2, at)
		if recorded := a.sampleEnergyTenant(t.Context(), tenant, at); len(recorded) != 0 {
			t.Fatalf("bei %s wurde eine laufende Viertelstunde geschrieben: %+v", at, recorded)
		}
	}
	intervals, _ := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	if len(intervals) != 0 {
		t.Fatalf("die laufende Viertelstunde darf nicht im Speicher stehen: %+v", intervals)
	}
	// Das Cockpit unterscheidet bewusst: erst mit dem Überschreiten der Grenze
	// entsteht eine abgeschlossene Viertelstunde.
	fake.reading(7.2, start.Add(15*time.Minute))
	if recorded := a.sampleEnergyTenant(t.Context(), tenant, start.Add(15*time.Minute)); len(recorded) != 1 {
		t.Fatalf("nach der Grenze wurde nichts festgeschrieben: %+v", recorded)
	}
}

func TestSamplerRefusesValueForPartialQuarterHourHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	tenant := samplerTenantHAUSV428(a, "demo", fake)
	start := quarterAnchorHAUSV428()

	// Der Sampler steigt erst in der Mitte ein — etwa nach einem Neustart.
	for i := samplesPerQuarterHAUSV / 2; i < samplesPerQuarterHAUSV; i++ {
		at := start.Add(time.Duration(i) * sampleIntervalHAUSV428)
		fake.reading(11.4, at)
		a.sampleEnergyTenant(t.Context(), tenant, at)
	}
	boundary := start.Add(15 * time.Minute)
	fake.reading(11.4, boundary)
	a.sampleEnergyTenant(t.Context(), tenant, boundary)

	intervals, _ := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	if len(intervals) != 1 {
		t.Fatalf("Intervalle = %+v", intervals)
	}
	if intervals[0].Quality != energy.QualityGap {
		t.Fatalf("Qualität = %q, erwartet %q", intervals[0].Quality, energy.QualityGap)
	}
	if intervals[0].AverageKW != 0 {
		t.Fatalf("aus einer halben Viertelstunde darf kein Wert entstehen: %+v", intervals[0])
	}
	if peak := energy.PeakForMonth(intervals, start, time.Local); peak != 0 {
		t.Fatalf("Monatsspitze = %v kW, erwartet 0", peak)
	}
}

func TestSamplerDegradesQualityOnConnectionHoleHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	tenant := samplerTenantHAUSV428(a, "demo", fake)
	start := quarterAnchorHAUSV428()

	for i := 0; i < samplesPerQuarterHAUSV; i++ {
		at := start.Add(time.Duration(i) * sampleIntervalHAUSV428)
		switch {
		case i < 5:
			fake.reading(12.0, at)
		case i < 25:
			// Home Assistant ist zehn Minuten lang nicht erreichbar.
			fake.goOffline()
		default:
			fake.reading(1.0, at)
		}
		a.sampleEnergyTenant(t.Context(), tenant, at)
	}
	boundary := start.Add(15 * time.Minute)
	fake.reading(1.0, boundary)
	a.sampleEnergyTenant(t.Context(), tenant, boundary)

	intervals, _ := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	if len(intervals) != 1 || intervals[0].Quality != energy.QualityGap {
		t.Fatalf("Qualität = %+v, erwartet eine Messlücke", intervals)
	}
	// Über das Loch zu mitteln hätte 12 kW über zehn Minuten fortgeschrieben und
	// damit eine Monatsspitze erfunden, die niemand gemessen hat.
	if intervals[0].AverageKW != 0 {
		t.Fatalf("über die Lücke wurde gemittelt: %+v", intervals[0])
	}
	gaps, conflicts := intervalQualityCounts(intervals)
	quality := energy.AssessQuality(boundary, boundary, gaps, conflicts, len(intervals))
	if quality.Status != energy.QualityGap {
		t.Fatalf("das Cockpit muss die Lücke benennen: %+v", quality)
	}
}

func TestSamplerNamesFrozenReadingsStaleHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	tenant := samplerTenantHAUSV428(a, "demo", fake)
	start := quarterAnchorHAUSV428()

	// Die Verbindung antwortet, der Messwert steht seit einer Stunde still.
	frozen := start.Add(-time.Hour)
	for i := 0; i < samplesPerQuarterHAUSV; i++ {
		at := start.Add(time.Duration(i) * sampleIntervalHAUSV428)
		fake.freeze(9.9, frozen)
		a.sampleEnergyTenant(t.Context(), tenant, at)
	}
	boundary := start.Add(15 * time.Minute)
	fake.freeze(9.9, frozen)
	a.sampleEnergyTenant(t.Context(), tenant, boundary)

	intervals, _ := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	if len(intervals) != 1 || intervals[0].Quality != energy.QualityStale {
		t.Fatalf("Qualität = %+v, erwartet %q", intervals, energy.QualityStale)
	}
	if intervals[0].AverageKW != 0 {
		t.Fatalf("ein eingefrorener Wert darf keine Leistung behaupten: %+v", intervals[0])
	}
}

func TestSamplerTreatsUnavailableAsHoleHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	tenant := samplerTenantHAUSV428(a, "demo", fake)
	start := quarterAnchorHAUSV428()

	// Einzelne „unavailable"-Aussetzer kommen auf echten Anlagen vor. Einer darf
	// eine sonst vollständige Viertelstunde nicht verwerfen.
	for i := 0; i < samplesPerQuarterHAUSV; i++ {
		at := start.Add(time.Duration(i) * sampleIntervalHAUSV428)
		if i == 17 {
			fake.goUnavailable(at)
		} else {
			fake.reading(5.5, at)
		}
		a.sampleEnergyTenant(t.Context(), tenant, at)
	}
	boundary := start.Add(15 * time.Minute)
	fake.reading(5.5, boundary)
	a.sampleEnergyTenant(t.Context(), tenant, boundary)

	intervals, _ := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	if len(intervals) != 1 || intervals[0].Quality != energy.QualityEstimated {
		t.Fatalf("einzelner Aussetzer = %+v", intervals)
	}
	if math.Abs(intervals[0].AverageKW-5.5) > 1e-9 {
		t.Fatalf("Mittelwert = %v kW", intervals[0].AverageKW)
	}
}

func TestSamplerIsIdempotentAcrossRestartHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	tenant := samplerTenantHAUSV428(a, "demo", fake)
	start := quarterAnchorHAUSV428()

	runQuarterHAUSV428(t, a, tenant, fake, start, 6.4)
	// Neustart: der Puffer ist weg, dieselbe Viertelstunde wird erneut
	// abgefahren. Die Zeile darf sich nicht verdoppeln.
	a.resetEnergySampler("demo")
	runQuarterHAUSV428(t, a, tenant, fake, start, 6.4)

	intervals, _ := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	if len(intervals) != 1 {
		t.Fatalf("Viertelstunde wurde verdoppelt: %+v", intervals)
	}
	if math.Abs(intervals[0].AverageKW-6.4) > 1e-9 {
		t.Fatalf("Mittelwert = %v kW", intervals[0].AverageKW)
	}
}

func TestSamplerRecordsNothingWithoutConfirmedMappingHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	tenant := samplerTenantHAUSV428(a, "demo", fake)
	start := quarterAnchorHAUSV428()

	// Erkannt, aber nicht bestätigt: HAUSV misst nicht.
	if err := a.energyStore.UpsertMapping(energy.EntityMapping{
		ID: energy.NewID("mapping"), TenantSlug: "demo", EntityID: gridEntityHAUSV428,
		Metric: energy.MetricGridImportPower, Unit: "kW", Confirmed: false,
	}); err != nil {
		t.Fatal(err)
	}
	// Und eine bestätigte Zuordnung anderer Art zählt ebenfalls nicht.
	if err := a.energyStore.UpsertMapping(energy.EntityMapping{
		ID: energy.NewID("mapping"), TenantSlug: "demo", EntityID: "sensor.pv_power",
		Metric: energy.MetricPVPower, Unit: "kW", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	runQuarterHAUSV428(t, a, tenant, fake, start, 8.0)

	intervals, _ := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	if len(intervals) != 0 {
		t.Fatalf("ohne bestätigten Netzbezug darf nichts entstehen: %+v", intervals)
	}
	fake.mu.Lock()
	requests := len(fake.requests)
	fake.mu.Unlock()
	if requests != 0 {
		t.Fatalf("ohne bestätigte Zuordnung darf Home Assistant gar nicht gelesen werden (%d Abrufe)", requests)
	}
}

func TestSamplerKeepsHousesApartHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	addTestTenant(a, tenantConfig{Slug: "haus-a", Name: "Haus A"})
	firstHA := newFakeHomeAssistantHAUSV428(t)
	secondHA := newFakeHomeAssistantHAUSV428(t)
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	confirmGridImportHAUSV428(t, a, "haus-a", gridEntityHAUSV428)
	start := quarterAnchorHAUSV428()

	first := samplerTenantHAUSV428(a, "demo", firstHA)
	second := samplerTenantHAUSV428(a, "haus-a", secondHA)
	for i := 0; i < samplesPerQuarterHAUSV; i++ {
		at := start.Add(time.Duration(i) * sampleIntervalHAUSV428)
		firstHA.reading(3.0, at)
		secondHA.reading(12.0, at)
		a.sampleEnergyTenant(t.Context(), first, at)
		a.sampleEnergyTenant(t.Context(), second, at)
	}
	boundary := start.Add(15 * time.Minute)
	firstHA.reading(3.0, boundary)
	secondHA.reading(12.0, boundary)
	a.sampleEnergyTenant(t.Context(), first, boundary)
	a.sampleEnergyTenant(t.Context(), second, boundary)

	for _, item := range []struct {
		slug string
		kw   float64
	}{{"demo", 3.0}, {"haus-a", 12.0}} {
		intervals, _ := a.energyStore.ListIntervals(item.slug, time.Time{}, time.Time{})
		if len(intervals) != 1 {
			t.Fatalf("%s: Intervalle = %+v", item.slug, intervals)
		}
		if intervals[0].TenantSlug != item.slug {
			t.Fatalf("%s: Zeile trägt den Mandanten %q", item.slug, intervals[0].TenantSlug)
		}
		if math.Abs(intervals[0].AverageKW-item.kw) > 1e-9 {
			t.Fatalf("%s: Mittelwert = %v kW, erwartet %v kW", item.slug, intervals[0].AverageKW, item.kw)
		}
	}
}

func TestSamplerRecordsInObserveModeWithoutTouchingHomeAssistantHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	profile := energy.DefaultProfile("demo", time.Now())
	profile.OnboardingComplete = true
	if err := a.energyStore.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	stored, _, err := a.energyStore.Profile("demo")
	if err != nil || stored.OperatingMode != energy.ModeObserve {
		t.Fatalf("Betriebsmodus = %q err=%v", stored.OperatingMode, err)
	}
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	tenant := samplerTenantHAUSV428(a, "demo", fake)

	recorded := runQuarterHAUSV428(t, a, tenant, fake, quarterAnchorHAUSV428(), 5.0)
	if len(recorded) != 1 {
		t.Fatalf("im Beobachtungsmodus muss gemessen werden dürfen: %+v", recorded)
	}
	// Messen ist nicht Steuern: kein Dienstaufruf, kein Schreibzugriff.
	fake.auditReadOnly(t)
}

func TestSamplerHandlesWattAndRejectsUnknownUnitHAUSV428(t *testing.T) {
	mapping := energy.EntityMapping{Metric: energy.MetricGridImportPower, Unit: "W"}
	value, ok := energyGridImportKW(mapping, homeassistant.EntityState{State: "4810"})
	if !ok || math.Abs(value-4.81) > 1e-9 {
		t.Fatalf("Watt = %v ok=%v, erwartet 4,81 kW", value, ok)
	}
	// Einheit aus dem Attribut, wenn die Zuordnung keine trägt.
	value, ok = energyGridImportKW(energy.EntityMapping{Metric: energy.MetricGridImportPower},
		homeassistant.EntityState{State: "2.5", Attributes: map[string]any{"unit_of_measurement": "kW"}})
	if !ok || value != 2.5 {
		t.Fatalf("Attributeinheit = %v ok=%v", value, ok)
	}
	// Ohne bekannte Einheit wird nicht geraten.
	if _, ok := energyGridImportKW(energy.EntityMapping{Metric: energy.MetricGridImportPower},
		homeassistant.EntityState{State: "2.5"}); ok {
		t.Fatal("ohne Einheit darf keine Leistungsangabe entstehen")
	}
	// Ein vorzeichenbehafteter Netzsensor meldet Einspeisung negativ; Bezug ist
	// dann null, nicht negativ.
	value, ok = energyGridImportKW(mapping, homeassistant.EntityState{State: "-1200"})
	if !ok || value != 0 {
		t.Fatalf("negative Leistung = %v ok=%v, erwartet 0", value, ok)
	}
}

// TestCockpitStopsBeingEmptyAfterSamplingHAUSV428 ist der eigentliche Beweis:
// nach dem Messen rendert die Tarifkarte eine Spitze statt des Hinweises, dass
// für diesen Monat noch keine abgeschlossene Viertelstunde vorliegt.
func TestCockpitStopsBeingEmptyAfterSamplingHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	steps := []url.Values{
		{"action": {"profile"}, "household_name": {"Energiehaus"}, "home_type": {"house"}},
		{"action": {"assets"}, "assets": {"pv"}},
		{"action": {"mappings"}},
		{"action": {"finish"}},
	}
	for i, form := range steps {
		if response := authedFormRequest(t, a, "owner@example.com", "/demo/app/zuhause/onboarding", form); response.Code != http.StatusSeeOther {
			t.Fatalf("Onboarding-Schritt %d: status=%d", i+1, response.Code)
		}
	}
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)

	// Vor dem Messen ist die Tarifkarte leer — genau der gemeldete Zustand.
	before := authedRequest(t, a, "owner@example.com", "/demo/app/energie").Body.String()
	// Mit zugeordnetem Netzbezug ist der Leerzustand eine Wartezeit, keine
	// Aufforderung, eine Datei zu suchen — die Karte fuellt sich von selbst.
	if !strings.Contains(before, "Noch keine volle Viertelstunde") {
		t.Fatal("Ausgangslage verfehlt: die Tarifkarte war schon vorher gefüllt")
	}

	tenant := samplerTenantHAUSV428(a, "demo", fake)
	runQuarterHAUSV428(t, a, tenant, fake, quarterAnchorHAUSV428(), 16)

	after := authedRequest(t, a, "owner@example.com", "/demo/app/energie").Body.String()
	if strings.Contains(after, "Noch keine volle Viertelstunde") {
		t.Fatal("die Tarifkarte ist nach dem Messen immer noch leer")
	}
	block := billedBlockHAUSV425(t, after)
	for _, want := range []string{"Höchste Viertelstunde", "16\u00a0kW", "Verrechnet"} {
		if !strings.Contains(block, want) {
			t.Fatalf("Kennzahlenblock ohne %q, war:\n%s", want, block)
		}
	}
	// Und die Grundlage benennt die Quelle, damit niemand die Spitze für einen
	// Zählerexport hält.
	if !strings.Contains(after, "Quelle: Home Assistant") {
		t.Fatal("die Grundlage muss Home Assistant als Quelle benennen")
	}
	// Home-Assistant-Viertelstunden sind bewusst als geschätzt markiert. Sie
	// bleiben dennoch abgeschlossene, fuer den automatischen Beobachtungsweg
	// nutzbare Intervalle; sonst stünde dieser Haushalt dauerhaft bei 0 von 96.
	if !strings.Contains(after, "1 von 96 Viertelstunden") {
		t.Fatal("eine geschätzte Home-Assistant-Viertelstunde muss den Beobachtungsfortschritt erhöhen")
	}
}

// TestSmartMeterComparisonCanFireAfterSamplingHAUSV428 belegt, dass der
// Quellenvergleich aus energyComparisonForView jetzt überhaupt auslösen kann.
// Bisher gab es die zweite Quelle nie.
func TestSmartMeterComparisonCanFireAfterSamplingHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	tenant := samplerTenantHAUSV428(a, "demo", fake)
	start := quarterAnchorHAUSV428()

	// Referenz aus dem Smart-Meter-Export für denselben Monat.
	if err := a.energyStore.PutInterval(energy.Interval{
		TenantSlug: "demo",
		StartsAt:   start.Add(-15 * time.Minute),
		Duration:   15 * time.Minute,
		AverageKW:  8.0,
		ImportKWh:  2.0,
		Quality:    energy.QualityMeasured,
		Source:     energy.SourceSmartMeter,
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok := energyComparisonForView(intervalsForMonthHAUSV428(t, a), time.Now()); ok {
		t.Fatal("ohne Home-Assistant-Viertelstunde darf der Vergleich nicht auslösen")
	}

	runQuarterHAUSV428(t, a, tenant, fake, start, 8.4)

	comparison, ok := energyComparisonForView(intervalsForMonthHAUSV428(t, a), time.Now())
	if !ok {
		t.Fatal("der Quellenvergleich löst nach dem Messen immer noch nicht aus")
	}
	if comparison.Tone != "good" {
		t.Fatalf("Vergleich = %+v, erwartet plausible Quellen bei 8,0 gegen 8,4 kW", comparison)
	}
	raw, _ := energy.CompareMonthlyPeaks(intervalsForMonthHAUSV428(t, a), time.Now(), time.Local,
		energy.SourceSmartMeter, energy.SourceHomeAssistant)
	if raw.ReferencePeakKW != 8 || math.Abs(raw.ComparedPeakKW-8.4) > 1e-9 {
		t.Fatalf("Spitzen = %+v", raw)
	}
}

func intervalsForMonthHAUSV428(t *testing.T, a *app) []energy.Interval {
	t.Helper()
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	intervals, err := a.energyStore.ListIntervals("demo", monthStart.UTC(), time.Time{})
	if err != nil {
		t.Fatalf("Intervalle laden: %v", err)
	}
	return intervals
}

// TestSampledQuarterHoursAreCoveredByRetentionExportAndDeleteHAUSV428 hält fest,
// dass die neuen Zeilen denselben Lebenszyklus haben wie importierte: 13 Monate
// Aufbewahrung, vollständiger DSGVO-Export, vollständige Löschung.
func TestSampledQuarterHoursAreCoveredByRetentionExportAndDeleteHAUSV428(t *testing.T) {
	a := samplerAppHAUSV428(t)
	fake := newFakeHomeAssistantHAUSV428(t)
	confirmGridImportHAUSV428(t, a, "demo", gridEntityHAUSV428)
	tenant := samplerTenantHAUSV428(a, "demo", fake)
	runQuarterHAUSV428(t, a, tenant, fake, quarterAnchorHAUSV428(), 4.2)

	// Export: die Viertelstunde steht mit ihrer Quelle in der CSV.
	intervals, _ := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{})
	csv, err := energyIntervalsCSV(intervals)
	if err != nil {
		t.Fatalf("CSV: %v", err)
	}
	if !strings.Contains(string(csv), "home-assistant") || !strings.Contains(string(csv), "estimated") {
		t.Fatalf("Export ohne Quelle/Qualität:\n%s", csv)
	}

	// Aufbewahrung: dieselbe 13-Monats-Grenze wie für importierte Werte.
	_, intervalCutoff, _ := energyRetentionCutoffs(time.Now())
	if intervals[0].StartsAt.Before(intervalCutoff) {
		t.Fatalf("die Viertelstunde liegt außerhalb der Aufbewahrung: %s < %s", intervals[0].StartsAt, intervalCutoff)
	}
	if _, err := a.energyStore.PurgeExpired(time.Time{}, time.Now().UTC().AddDate(0, 1, 0), time.Time{}); err != nil {
		t.Fatalf("PurgeExpired: %v", err)
	}
	if remaining, _ := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{}); len(remaining) != 0 {
		t.Fatalf("die Aufbewahrungsfrist greift nicht: %+v", remaining)
	}

	// Löschung: der Messverlauf verschwindet vollständig.
	a.resetEnergySampler("demo")
	runQuarterHAUSV428(t, a, tenant, fake, quarterAnchorHAUSV428(), 4.2)
	summary, err := a.energyStore.DeleteMeasurementData("demo")
	if err != nil || summary.Intervals != 1 {
		t.Fatalf("Löschzusammenfassung = %+v err=%v", summary, err)
	}
	if remaining, _ := a.energyStore.ListIntervals("demo", time.Time{}, time.Time{}); len(remaining) != 0 {
		t.Fatalf("nach der Löschung blieb etwas übrig: %+v", remaining)
	}
}
