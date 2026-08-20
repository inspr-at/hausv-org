package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeassistant"
	"github.com/inspr-at/hausv-org/internal/homeconnector"
)

func energyConsumerHAUSV551(t *testing.T, cfg energyFlowConfig, id string) energyFlowConsumerConfig {
	t.Helper()
	for _, consumer := range cfg.Consumers {
		if consumer.ID == id {
			return consumer
		}
	}
	t.Fatalf("consumer %q missing from flow config", id)
	return energyFlowConsumerConfig{}
}

func TestConsumerReadingAgeAndStaleThresholdHAUSV551(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.Local)
	asset := energy.Asset{
		ID: "sauna", TenantSlug: "demo", Kind: "sauna", Name: "Sauna",
		Metadata: map[string]string{},
	}
	reading := energyMetricView{
		AssetID: asset.ID, Metric: energy.MetricConsumerPower,
		Numeric: 1200, Unit: "W", LastUpdated: now.Add(-2 * time.Minute),
	}

	fresh := energyConsumerHAUSV551(t,
		buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{asset}, nil, []energyMetricView{reading}, parkingLiveView{}, false, now),
		asset.ID,
	)
	if fresh.Age != "vor 2 Min." || fresh.DataStatus != "" || fresh.KW != 1.2 {
		t.Fatalf("fresh consumer reading = %+v", fresh)
	}

	reading.LastUpdated = now.Add(-11 * time.Minute)
	stale := energyConsumerHAUSV551(t,
		buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{asset}, nil, []energyMetricView{reading}, parkingLiveView{}, false, now),
		asset.ID,
	)
	if stale.Age != "vor 11 Min." || stale.DataStatus != "stale" || stale.DataLabel != "Veraltet" || stale.KW != 0 || stale.Active {
		t.Fatalf("stale consumer reading = %+v", stale)
	}

	asset.Metadata["stale_after_minutes"] = "30"
	overridden := energyConsumerHAUSV551(t,
		buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{asset}, nil, []energyMetricView{reading}, parkingLiveView{}, false, now),
		asset.ID,
	)
	if overridden.DataStatus != "" || overridden.KW != 1.2 || overridden.StaleAfterMinutes != 30 {
		t.Fatalf("consumer-specific stale threshold was not applied: %+v", overridden)
	}
}

func TestConsumerCardHierarchyKeepsEveryAvailableReading(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.Local)
	asset := energy.Asset{
		ID: "car", TenantSlug: "demo", Kind: "ev", Name: "Model X",
		Metadata: map[string]string{},
	}
	metrics := []energyMetricView{
		{AssetID: asset.ID, Metric: energy.MetricConsumerPower, Numeric: 0, Unit: "W", Value: formatEnergyReading(0, "W"), LastUpdated: now.Add(-9 * time.Hour)},
		{AssetID: asset.ID, Metric: energy.MetricBatterySOC, Numeric: 51, Unit: "%", Value: formatEnergyReading(51, "%")},
		{AssetID: asset.ID, Metric: energy.MetricConsumerEnergy, Numeric: 4890, Unit: "kWh", Value: formatEnergyReading(4890, "kWh")},
	}

	idle := energyConsumerHAUSV551(t,
		buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{asset}, nil, metrics, parkingLiveView{}, false, now),
		asset.ID,
	)
	if idle.State != "0\u00a0kW" || idle.Current == nil || idle.Current.Value != "0" || idle.Current.Unit != "kW" {
		t.Fatalf("idle current power was not kept as quiet card context: %+v", idle)
	}
	if idle.Primary == nil || idle.Primary.Value != "51" || idle.Primary.Unit != "%" || idle.Primary.Label != "Ladestand" {
		t.Fatalf("idle card did not promote the available state of charge: %+v", idle)
	}
	if len(idle.Metrics) != 1 || idle.Metrics[0].Label != "Energie" ||
		idle.Metrics[0].Value != "4.890" || idle.Metrics[0].Unit != "kWh" {
		t.Fatalf("idle card lost or misformatted its energy reading: %+v", idle)
	}
	if idle.DataLabel != "Veraltet" || idle.Age != "vor 9 Std." || strings.Contains(idle.State, "Home Assistant") {
		t.Fatalf("idle card did not use the approved quiet stale copy: %+v", idle)
	}

	metrics[0].Numeric = 800
	metrics[0].Value = formatEnergyReading(800, "W")
	metrics[0].LastUpdated = now.Add(-time.Minute)
	active := energyConsumerHAUSV551(t,
		buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{asset}, nil, metrics, parkingLiveView{}, false, now),
		asset.ID,
	)
	if active.Primary == nil || active.Primary.Value != "0,8" || active.Primary.Unit != "kW" || active.Primary.Label != "Jetzt" {
		t.Fatalf("flowing power was not promoted to the primary line: %+v", active)
	}
	if len(active.Metrics) != 2 || active.Metrics[0].Label != "Ladestand" || active.Metrics[1].Label != "Energie" {
		t.Fatalf("active card did not retain SOC and energy as quiet metrics: %+v", active)
	}

	boiler := energy.Asset{
		ID: "boiler", TenantSlug: "demo", Kind: "hot-water", Name: "Boiler - Warmwasser",
		Metadata: map[string]string{},
	}
	soleEnergy := energyConsumerHAUSV551(t,
		buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{boiler}, nil, []energyMetricView{{
			AssetID: boiler.ID, Metric: energy.MetricConsumerEnergy,
			Numeric: 4890.6, Unit: "kWh", Value: formatEnergyReading(4890.6, "kWh"),
		}}, parkingLiveView{}, false, now),
		boiler.ID,
	)
	if soleEnergy.Primary == nil || soleEnergy.Primary.Value != "4.890,6" || soleEnergy.Primary.Unit != "kWh" ||
		soleEnergy.Primary.Label != "Energie" || len(soleEnergy.Metrics) != 0 {
		t.Fatalf("sole kWh must appear once as the primary metric: %+v", soleEnergy)
	}
	payload, err := json.Marshal(soleEnergy)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"metrics":[]`) {
		t.Fatalf("structured empty metric row must reach the renderer: %s", payload)
	}

	parkingID := energyFlowNodeID("demo", "parking")
	parking := energy.Asset{
		ID: parkingID, TenantSlug: "demo", Kind: energyFlowParkingKind, Name: "Parkplatz 20",
		Metadata: map[string]string{},
	}
	parkingEnergy := energyConsumerHAUSV551(t,
		buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{parking}, nil, []energyMetricView{
			{AssetID: parkingID, Metric: energy.MetricConsumerPower, Numeric: 0, Unit: "W", Value: formatEnergyReading(0, "W"), LastUpdated: now.Add(-time.Minute)},
			{AssetID: parkingID, Metric: energy.MetricConsumerEnergy, Numeric: 2797.6, Unit: "kWh", Value: formatEnergyReading(2797.6, "kWh")},
		}, parkingLiveView{Available: true, Mode: "manual", ModeLabel: "Normalladen"}, false, now),
		parkingID,
	)
	if parkingEnergy.Primary == nil || parkingEnergy.Primary.Value != "2.797,6" ||
		parkingEnergy.Primary.Unit != "kWh" || parkingEnergy.Primary.Label != "Ladeenergie" ||
		len(parkingEnergy.Metrics) != 0 {
		t.Fatalf("parking sole kWh must appear once as the primary metric: %+v", parkingEnergy)
	}
}

func TestConsumerUnavailableAndUnknownStayDistinctHAUSV551(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.Local)
	for sourceState, want := range map[string]string{
		"unavailable": "Nicht verfügbar",
		"unknown":     "Unbekannt",
	} {
		t.Run(sourceState, func(t *testing.T) {
			asset := energy.Asset{ID: sourceState, Kind: "other", Name: sourceState, Metadata: map[string]string{}}
			reading := energyMetricView{
				AssetID: asset.ID, Metric: energy.MetricConsumerPower,
				SourceState: sourceState, LastUpdated: now.Add(-3 * time.Minute),
			}
			consumer := energyConsumerHAUSV551(t,
				buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{asset}, nil, []energyMetricView{reading}, parkingLiveView{}, false, now),
				asset.ID,
			)
			if consumer.State != want || consumer.DataStatus != sourceState || consumer.Age != "vor 3 Min." {
				t.Fatalf("%s consumer = %+v", sourceState, consumer)
			}
		})
	}
}

func TestSleepingVehicleUsesLastReadingClockTimeHAUSV551(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.Local)
	lastReading := time.Date(2026, 8, 20, 9, 17, 0, 0, time.Local)
	asset := energy.Asset{ID: "car", Kind: "ev", Name: "Auto", Metadata: map[string]string{}}
	metrics := []energyMetricView{
		{AssetID: asset.ID, Metric: energy.MetricConsumerPower, SourceState: "unavailable", LastUpdated: lastReading},
		{AssetID: asset.ID, Metric: energy.MetricConsumerSleep, SourceState: "sleep", LastUpdated: now.Add(-time.Minute)},
	}
	consumer := energyConsumerHAUSV551(t,
		buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{asset}, nil, metrics, parkingLiveView{}, false, now),
		asset.ID,
	)
	if consumer.State != "schläft · Stand 09:17" || consumer.DataStatus != "sleep" || consumer.Age != "" || consumer.KW != 0 {
		t.Fatalf("sleeping vehicle = %+v", consumer)
	}
}

func TestCurrentEnergyMetricsPreserveHANonNumericStatesHAUSV551(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := strings.TrimPrefix(r.URL.Path, "/api/states/sensor.")
		if state != "unavailable" && state != "unknown" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"entity_id":"sensor.%s","state":"%s","attributes":{"device_class":"power","unit_of_measurement":"W"},"last_updated":%q}`,
			state, state, now.Format(time.RFC3339Nano))
	}))
	t.Cleanup(ha.Close)

	tenant := tenantConfig{Slug: "demo", HA: homeassistant.NewConfig(ha.URL, "fixture", "", "", "")}
	mappings := []energy.EntityMapping{
		{EntityID: "sensor.unavailable", AssetID: "one", Metric: energy.MetricConsumerPower, Unit: "W", Confirmed: true},
		{EntityID: "sensor.unknown", AssetID: "two", Metric: energy.MetricConsumerPower, Unit: "W", Confirmed: true},
	}
	a := &app{}
	metrics, _, latest := a.currentEnergyMetrics(t.Context(), tenant, mappings, energy.HomeProfile{})
	if len(metrics) != 2 || !latest.Equal(now) {
		t.Fatalf("metrics=%+v latest=%s", metrics, latest)
	}
	got := map[string]string{}
	for _, metric := range metrics {
		got[metric.AssetID] = metric.SourceState
	}
	if got["one"] != "unavailable" || got["two"] != "unknown" {
		t.Fatalf("non-numeric Home Assistant states were lost: %+v", got)
	}
}

func TestConsumerStaleOverrideValidationHAUSV551(t *testing.T) {
	metadata := map[string]string{}
	if energyConsumerStaleAfter(energy.Asset{Metadata: metadata}) != 10*time.Minute {
		t.Fatal("default consumer stale threshold changed")
	}
	if !setEnergyConsumerStaleOverride(metadata, "45") || metadata["stale_after_minutes"] != "45" {
		t.Fatalf("valid override was not stored: %+v", metadata)
	}
	if setEnergyConsumerStaleOverride(metadata, "0") || setEnergyConsumerStaleOverride(metadata, "1441") {
		t.Fatalf("out-of-range override was accepted: %+v", metadata)
	}
	if !setEnergyConsumerStaleOverride(metadata, "") || metadata["stale_after_minutes"] != "" {
		t.Fatalf("empty override did not restore default: %+v", metadata)
	}
}

func TestPortalAcceptsExplicitVehicleSleepSignalHAUSV551(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	reading := homeconnector.Reading{
		EntityID: "binary_sensor.vehicle_model_x_asleep", State: "on",
		DisplayName: "Vehicle Model X Asleep", LastUpdated: now,
	}
	if !validHomeConnectorReading(reading, now) {
		t.Fatal("explicit vehicle sleep signal was rejected by portal validation")
	}
	reading.EntityID, reading.DisplayName = "binary_sensor.front_door", "Front door"
	if validHomeConnectorReading(reading, now) {
		t.Fatal("unrelated binary sensor was accepted as energy data")
	}
	reading.EntityID, reading.DisplayName = "binary_sensor.bedroom_sleeping", "Bedroom sleeping"
	if validHomeConnectorReading(reading, now) {
		t.Fatal("non-vehicle sleep sensor was accepted as energy data")
	}
}

func TestConnectorFilterRetainsVehicleSleepDiscoveryHAUSV551(t *testing.T) {
	input := []homeconnector.Reading{
		{EntityID: "sensor.house_power", State: "1200", DisplayName: "House power"},
		{EntityID: "binary_sensor.vehicle_model_x_asleep", State: "on", DisplayName: "Vehicle Model X Asleep"},
		{EntityID: "binary_sensor.bedroom_sleeping", State: "on", DisplayName: "Bedroom sleeping"},
	}
	got := filterHomeConnectorReadings(input, []string{"sensor.house_power"})
	if len(got) != 2 || got[0].EntityID != "sensor.house_power" || got[1].EntityID != "binary_sensor.vehicle_model_x_asleep" {
		t.Fatalf("connector discovery filter = %+v", got)
	}
}

func TestDeletingVehicleRemovesEveryMeasurementMappingHAUSV551(t *testing.T) {
	a := consumerAppHAUSV422(t)
	asset := energy.Asset{
		ID: "vehicle-delete", TenantSlug: "demo", Kind: "ev", Name: "Model X",
		Confirmed: true, Metadata: map[string]string{},
	}
	if err := a.energyStore.UpsertAsset(asset); err != nil {
		t.Fatal(err)
	}
	seeded := map[string]string{
		energy.MetricConsumerPower: "sensor.model_x_power",
		energy.MetricBatterySOC:    "sensor.model_x_soc",
		energy.MetricConsumerSleep: "binary_sensor.vehicle_model_x_asleep",
	}
	seededEntities := map[string]bool{}
	for metric, entityID := range seeded {
		seededEntities[entityID] = true
		if err := a.energyStore.UpsertMapping(energy.EntityMapping{
			ID: energy.NewID("mapping"), TenantSlug: "demo", AssetID: asset.ID,
			EntityID: entityID, Metric: metric, Confirmed: true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	response := authedFormRequest(t, a, "owner@example.com", "/demo/app/energie/verbraucher/entfernen", url.Values{"asset_id": {asset.ID}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("delete status=%d body=%s", response.Code, response.Body.String())
	}
	mappings, err := a.energyStore.ListMappings("demo")
	if err != nil {
		t.Fatal(err)
	}
	for _, mapping := range mappings {
		if seededEntities[mapping.EntityID] {
			t.Fatalf("vehicle mapping entity survived delete: %+v", mapping)
		}
	}
}

func TestConfiguredParkingReadingUsesAgeAndStaleThresholdHAUSV551(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.Local)
	parkingID := energyFlowNodeID("demo", "parking")
	asset := energy.Asset{
		ID: parkingID, TenantSlug: "demo", Kind: energyFlowParkingKind, Name: "Parkplatz",
		Metadata: map[string]string{},
	}
	charging := parkingLiveView{
		Available: true, Mode: "manual", ModeLabel: "Normalladen",
		PowerKW: 3.6, PowerEntity: "sensor.parking_power",
		PowerLastUpdated: now.Add(-11 * time.Minute),
	}
	consumer := energyConsumerHAUSV551(t,
		buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{asset}, nil, nil, charging, false, now),
		parkingID,
	)
	if consumer.Age != "vor 11 Min." || consumer.DataStatus != "stale" || consumer.DataLabel != "Veraltet" ||
		consumer.KW != 0 || consumer.Active {
		t.Fatalf("configured parking stale reading = %+v", consumer)
	}
}

func TestConsumerStaleOverrideReproPersistsIntoRenderedCardHAUSV551(t *testing.T) {
	updated := time.Now().Add(-12 * time.Minute).UTC().Truncate(time.Second)
	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/states/sensor.sauna_power" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"entity_id":"sensor.sauna_power","state":"1200","attributes":{"friendly_name":"Sauna Leistung","device_class":"power","unit_of_measurement":"W"},"last_updated":%q}`,
			updated.Format(time.RFC3339Nano))
	}))
	t.Cleanup(ha.Close)

	a := consumerAppHAUSV422(t)
	assets, err := a.energyStore.ListAssets("demo")
	if err != nil {
		t.Fatal(err)
	}
	var sauna energy.Asset
	for _, asset := range assets {
		if asset.Kind == "sauna" {
			sauna = asset
			break
		}
	}
	if sauna.ID == "" {
		t.Fatal("seeded sauna missing")
	}
	if err := a.energyStore.UpsertMapping(energy.EntityMapping{
		ID: energy.NewID("mapping"), TenantSlug: "demo", AssetID: sauna.ID,
		EntityID: "sensor.sauna_power", Metric: energy.MetricConsumerPower,
		Unit: "W", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}
	tenant := a.tenants["demo"]
	tenant.HA = homeassistant.NewConfig(ha.URL, "fixture", "", "", "")
	a.tenants["demo"] = tenant

	response := authedFormRequest(t, a, "owner@example.com", "/demo/app/energie/verbraucher", url.Values{
		"asset_id": {sauna.ID}, "name": {sauna.Name}, "kind": {"sauna"},
		"priority": {"1"}, "icon": {"flame"}, "flexibility": {"shift"},
		"stale_after_minutes": {"5"},
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("override save status=%d body=%s", response.Code, response.Body.String())
	}
	assets, err = a.energyStore.ListAssets("demo")
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range assets {
		if asset.ID == sauna.ID && asset.Metadata["stale_after_minutes"] != "5" {
			t.Fatalf("stale override did not persist: %+v", asset.Metadata)
		}
	}

	page := authedRequest(t, a, "owner@example.com", "/demo/app/energie")
	if page.Code != http.StatusOK {
		t.Fatalf("energy page status=%d body=%s", page.Code, page.Body.String())
	}
	body := page.Body.String()
	for _, want := range []string{
		`"dataStatus":"stale"`,
		`"dataLabel":"Veraltet"`,
		`"age":"vor 12 Min."`,
		`"staleAfterMinutes":5`,
		`energy-flow-big.data-stale .energy-flow-state-copy`,
		`energy-flow-data-label`,
		`Portal abgerufen vor 0&nbsp;s`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered stale consumer card missing %q", want)
		}
	}
}
