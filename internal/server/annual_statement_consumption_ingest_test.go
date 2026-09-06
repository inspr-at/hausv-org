package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/homeconnector"
	"github.com/inspr-at/hausv-org/internal/store"
)

func seedAnnualConsumptionMapping(t *testing.T, a *app, slug, home, unit, kind, entity string, confirmed bool) {
	t.Helper()
	storage := a.energyStore.ForHome(home)
	profile := energy.DefaultProfileForHome(slug, home, time.Now())
	profile.UnitID = unit
	if err := storage.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	assetID := "asset-" + home + "-" + kind
	if err := storage.UpsertAsset(energy.Asset{ID: assetID, TenantSlug: slug, Kind: kind, Confirmed: confirmed}); err != nil {
		t.Fatal(err)
	}
	if err := storage.UpsertMapping(energy.EntityMapping{ID: "mapping-" + home + "-" + kind, TenantSlug: slug, EntityID: entity, DisplayName: entity, AssetID: assetID, Metric: energy.MetricConsumerEnergy, Confirmed: confirmed}); err != nil {
		t.Fatal(err)
	}
}

func TestAnnualConsumptionConnectorIngestion(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.homeConnectorReadings = store.NewMemoryHomeConnectorReadingStore()
	seedAnnualConsumptionMapping(t, a, "demo", "flat-1", "top-1", "heat-pump", "sensor.heat1", true)
	seedAnnualConsumptionMapping(t, a, "demo", "flat-2", "top-2", "heat-pump", "sensor.heat2", true)
	seedAnnualConsumptionMapping(t, a, "demo", "flat-1", "top-1", "hot-water", "sensor.water1", true)
	seedAnnualConsumptionMapping(t, a, "demo", "flat-2", "top-2", "hot-water", "sensor.water2", false)
	location, _ := time.LoadLocation("Europe/Vienna")
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, location)
	end := start.AddDate(1, 0, 0)
	for _, at := range []time.Time{start, end} {
		state1, state2 := "100", "100"
		if at.Equal(end) {
			state1, state2 = "101", "103"
		}
		readings := []homeconnector.Reading{}
		for _, pair := range []struct{ entity, state string }{{"sensor.heat1", state1}, {"sensor.heat2", state2}, {"sensor.water1", state1}, {"sensor.water2", state2}} {
			readings = append(readings, homeconnector.Reading{EntityID: pair.entity, State: pair.state, Unit: "kWh", DeviceClass: "energy", StateClass: "total_increasing", LastUpdated: at})
		}
		for i := 0; i < 2; i++ {
			if !a.persistHomeConnectorReadings("demo", readings, at.Add(time.Minute), false) {
				t.Fatal("ingestion failed")
			}
		}
	}
	repo := testRepositories(a, "demo").annualConsumption
	period := store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"}
	units := []store.Unit{{ID: "top-1"}, {ID: "top-2"}}
	vector, err := repo.ConsumptionVector(period, "heizung", []string{"top-1", "top-2"}, location)
	if err != nil {
		t.Fatal(err)
	}
	shares, ready := store.AnnualStatementConsumptionShares(vector, units)
	if !ready || shares[0].SharePPM != 250000 || shares[1].SharePPM != 750000 {
		t.Fatalf("shares=%+v, ready=%v", shares, ready)
	}
	water, _ := repo.ConsumptionVector(period, "warmwasser", []string{"top-1", "top-2"}, location)
	if len(water.Gaps) != 1 || water.Gaps[0].UnitID != "top-2" {
		t.Fatalf("unconfirmed mapping accepted: %+v", water)
	}
	other := a.repositoriesForTenant(testTenantRef("other")).annualConsumption
	foreign, _ := other.ConsumptionVector(period, "heizung", []string{"top-1"}, location)
	if len(foreign.Units) != 0 {
		t.Fatal("cross-tenant evidence")
	}
	selected := a.selectedHomeConnectorEntities("demo")
	if !slices.Contains(selected, "sensor.heat1") || !slices.Contains(selected, "sensor.heat2") {
		t.Fatalf("multi-home counters not selected: %v", selected)
	}
}

func TestAnnualConsumptionDoesNotInventEvidence(t *testing.T) {
	for _, variant := range []string{"ambiguous mapping", "power", "unknown", "measurement", "future", "missing timestamp"} {
		t.Run(variant, func(t *testing.T) {
			a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
			seedAnnualConsumptionMapping(t, a, "demo", "flat", "top-1", "heat-pump", "sensor.heat", true)
			at := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			reading := store.HomeConnectorReading{EntityID: "sensor.heat", State: "100", Unit: "kWh", DeviceClass: "energy", StateClass: "total_increasing", LastUpdated: at}
			switch variant {
			case "ambiguous mapping":
				seedAnnualConsumptionMapping(t, a, "demo", "other-flat", "top-2", "heat-pump", "sensor.heat", true)
			case "power":
				reading.Unit = "W"
			case "unknown":
				reading.State = "unavailable"
			case "measurement":
				reading.StateClass = "measurement"
			case "future":
				reading.LastUpdated = at.Add(time.Hour)
			case "missing timestamp":
				reading.LastUpdated = time.Time{}
			}
			if err := a.ingestAnnualStatementConsumption("demo", []store.HomeConnectorReading{reading}, at); err != nil {
				t.Fatal(err)
			}
			vector, err := testRepositories(a, "demo").annualConsumption.ConsumptionVector(store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"}, "heizung", []string{"top-1"}, time.UTC)
			if err != nil || len(vector.Gaps) != 1 || vector.Gaps[0].Reason != store.ConsumptionGapMissingSourceMapping {
				t.Fatalf("unexpected evidence: %+v %v", vector, err)
			}
		})
	}
}

func TestAnnualConsumptionEnergyMicros(t *testing.T) {
	for _, tc := range []struct {
		raw, unit string
		want      int64
		valid     bool
	}{
		{"1,25", "kWh", 1250000, true}, {"1250", "Wh", 1250000, true}, {"0.00125", "MWh", 1250000, true}, {"0", "kWh", 0, true},
		{"NaN", "kWh", 0, false}, {"-1", "kWh", 0, false}, {"1", "W", 0, false}, {"9223372036854775808", "kWh", 0, false}, {"0.0000001", "kWh", 0, false},
	} {
		if got, ok := annualStatementEnergyMicros(tc.raw, tc.unit); got != tc.want || ok != tc.valid {
			t.Errorf("%q %s = %d,%v", tc.raw, tc.unit, got, ok)
		}
	}
}

func TestAnnualConsumptionHASamplerWithoutGridMapping(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	seedAnnualConsumptionMapping(t, a, "demo", energy.DefaultHomeKey, "top-1", "heat-pump", "sensor.heat", true)
	at := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/states/sensor.heat" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"entity_id": "sensor.heat", "state": "100", "last_updated": at, "attributes": map[string]string{"unit_of_measurement": "kWh", "device_class": "energy", "state_class": "total_increasing"}})
	}))
	defer ha.Close()
	tenant := a.tenants["demo"]
	tenant.HA = homeassistant.NewConfig(ha.URL, "test-token", "", "", "")
	a.sampleEnergyTenant(context.Background(), tenant, at.Add(time.Minute))
	vector, err := testRepositories(a, "demo").annualConsumption.ConsumptionVector(store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"}, "heizung", []string{"top-1"}, time.UTC)
	if err != nil || len(vector.Gaps) != 1 || vector.Gaps[0].Reason != store.ConsumptionGapMissingEndEvidence {
		t.Fatalf("sample not persisted: %+v %v", vector, err)
	}
}

func TestAnnualConsumptionRemappingDoesNotRejectConnectorReadings(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.homeConnectorReadings = store.NewMemoryHomeConnectorReadingStore()
	seedAnnualConsumptionMapping(t, a, "demo", "flat", "top-1", "heat-pump", "sensor.heat", true)
	at := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	readings := []homeconnector.Reading{{EntityID: "sensor.heat", State: "100", Unit: "kWh", DeviceClass: "energy", StateClass: "total_increasing", LastUpdated: at}, {EntityID: "sensor.power", State: "1", Unit: "kW", DeviceClass: "power", StateClass: "measurement", LastUpdated: at}}
	if !a.persistHomeConnectorReadings("demo", readings, at, false) {
		t.Fatal("initial ingestion failed")
	}
	seedAnnualConsumptionMapping(t, a, "demo", "flat", "top-2", "heat-pump", "sensor.heat", true)
	readings[1].State = "2"
	if !a.persistHomeConnectorReadings("demo", readings, at.Add(time.Minute), false) {
		t.Fatal("historical mapping conflict rejected unrelated readings")
	}
	stored, err := a.homeConnectorReadings.List("demo")
	if err != nil {
		t.Fatal(err)
	}
	for _, reading := range stored {
		if reading.EntityID == "sensor.power" && reading.State == "2" {
			return
		}
	}
	t.Fatal("live reading was lost")
}

func TestAnnualConsumptionHASamplerRejectsCrossHomeDuplicateMeter(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	seedAnnualConsumptionMapping(t, a, "demo", "flat-1", "top-1", "heat-pump", "sensor.shared_heat", true)
	seedAnnualConsumptionMapping(t, a, "demo", "flat-2", "top-2", "heat-pump", "sensor.shared_heat", true)
	at := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	state := "100"
	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"entity_id": "sensor.shared_heat", "state": state, "last_updated": at, "attributes": map[string]string{"unit_of_measurement": "kWh", "device_class": "energy", "state_class": "total_increasing"}})
	}))
	defer ha.Close()
	tenant := a.tenants["demo"]
	tenant.HA = homeassistant.NewConfig(ha.URL, "test-token", "", "", "")
	for i := 0; i < 2; i++ {
		for _, home := range []string{"flat-1", "flat-2"} {
			a.sampleAnnualStatementConsumption(context.Background(), tenant, home, at.Add(time.Minute))
		}
		at, state = at.AddDate(1, 0, 0), "200"
	}
	vector, err := testRepositories(a, "demo").annualConsumption.ConsumptionVector(store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"}, "heizung", []string{"top-1", "top-2"}, time.UTC)
	if err != nil || len(vector.Units) != 0 || len(vector.Gaps) != 2 {
		t.Fatalf("shared physical meter created a complete allocation: %+v %v", vector, err)
	}
	for _, gap := range vector.Gaps {
		if gap.Reason != store.ConsumptionGapMissingSourceMapping {
			t.Fatalf("ambiguous source created evidence: %+v", gap)
		}
	}
}

func TestAnnualConsumptionConflictDoesNotDiscardFollowingCounters(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	a.homeConnectorReadings = store.NewMemoryHomeConnectorReadingStore()
	seedAnnualConsumptionMapping(t, a, "demo", "flat-1", "top-1", "heat-pump", "sensor.heat1", true)
	seedAnnualConsumptionMapping(t, a, "demo", "flat-2", "top-2", "heat-pump", "sensor.heat2", true)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	first := homeconnector.Reading{EntityID: "sensor.heat1", State: "100", Unit: "kWh", DeviceClass: "energy", StateClass: "total_increasing", LastUpdated: start}
	if !a.persistHomeConnectorReadings("demo", []homeconnector.Reading{first}, start, false) {
		t.Fatal("initial ingestion failed")
	}
	seedAnnualConsumptionMapping(t, a, "demo", "flat-1", "top-3", "heat-pump", "sensor.heat1", true)
	for i, at := range []time.Time{start, start.AddDate(1, 0, 0)} {
		second := homeconnector.Reading{EntityID: "sensor.heat2", State: []string{"100", "200"}[i], Unit: "kWh", DeviceClass: "energy", StateClass: "total_increasing", LastUpdated: at}
		if !a.persistHomeConnectorReadings("demo", []homeconnector.Reading{first, second}, at.Add(time.Minute), false) {
			t.Fatal("conflicting first row rejected the batch")
		}
	}
	repository := testRepositories(a, "demo").annualConsumption
	period := store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"}
	vector, err := repository.ConsumptionVector(period, "heizung", []string{"top-2"}, time.UTC)
	if err != nil || len(vector.Gaps) != 0 || len(vector.Units) != 1 || vector.Units[0].ValueMicros != 100_000_000 {
		t.Fatalf("following counter lost boundary evidence: %+v %v", vector, err)
	}
	original, err := repository.ConsumptionVector(period, "heizung", []string{"top-1", "top-3"}, time.UTC)
	if err != nil || len(original.Gaps) != 2 || original.Gaps[0].Reason != store.ConsumptionGapMissingEndEvidence || original.Gaps[1].Reason != store.ConsumptionGapMissingSourceMapping {
		t.Fatalf("conflicting mapping replaced the original fact: %+v %v", original, err)
	}
}

func TestAnnualConsumptionHASamplerContinuesAfterCounterTimeout(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "manager@example.com", Role: roleManager, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	seedAnnualConsumptionMapping(t, a, "demo", energy.DefaultHomeKey, "top-1", "heat-pump", "sensor.a_slow", true)
	seedAnnualConsumptionMapping(t, a, "demo", energy.DefaultHomeKey, "top-1", "hot-water", "sensor.b_fast", true)
	at := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/states/sensor.a_slow" {
			<-r.Context().Done()
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"entity_id": "sensor.b_fast", "state": "100", "last_updated": at, "attributes": map[string]string{"unit_of_measurement": "kWh", "device_class": "energy", "state_class": "total_increasing"}})
	}))
	defer ha.Close()
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	tenant := a.tenants["demo"]
	tenant.HA = homeassistant.NewConfig(ha.URL, "test-token", "", "", "")
	a.sampleAnnualStatementConsumption(context.Background(), tenant, energy.DefaultHomeKey, at.Add(time.Minute))
	vector, err := testRepositories(a, "demo").annualConsumption.ConsumptionVector(store.AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"}, "warmwasser", []string{"top-1"}, time.UTC)
	if err != nil || len(vector.Gaps) != 1 || vector.Gaps[0].Reason != store.ConsumptionGapMissingEndEvidence {
		t.Fatalf("first counter timeout discarded the later counter: %+v %v", vector, err)
	}
	if !strings.Contains(logs.String(), "confirmed counter could not be read") || !strings.Contains(logs.String(), "sensor.a_slow") {
		t.Fatal("unreadable confirmed counter was not identified in the log")
	}
}
