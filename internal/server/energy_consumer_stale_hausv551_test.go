package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
	if fresh.Age != "Stand vor 2 Min." || fresh.DataStatus != "" || fresh.KW != 1.2 {
		t.Fatalf("fresh consumer reading = %+v", fresh)
	}

	reading.LastUpdated = now.Add(-11 * time.Minute)
	stale := energyConsumerHAUSV551(t,
		buildEnergyFlowConfigAt("demo", energyLiveView{}, []energy.Asset{asset}, nil, []energyMetricView{reading}, parkingLiveView{}, false, now),
		asset.ID,
	)
	if stale.Age != "Stand vor 11 Min." || stale.DataStatus != "stale" || stale.DataLabel != "Veraltet" || stale.KW != 0 || stale.Active {
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
			if consumer.State != want || consumer.DataStatus != sourceState || consumer.Age != "Stand vor 3 Min." {
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
		EntityID: "binary_sensor.model_x_asleep", State: "on",
		DisplayName: "Model X Asleep", LastUpdated: now,
	}
	if !validHomeConnectorReading(reading, now) {
		t.Fatal("explicit vehicle sleep signal was rejected by portal validation")
	}
	reading.EntityID, reading.DisplayName = "binary_sensor.front_door", "Front door"
	if validHomeConnectorReading(reading, now) {
		t.Fatal("unrelated binary sensor was accepted as energy data")
	}
}
