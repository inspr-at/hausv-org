package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markus-barta/hausv-org/internal/energy"
	"github.com/markus-barta/hausv-org/internal/homeassistant"
)

func TestEnergyLiveRefreshReturnsCurrentFlowWithoutPageReload(t *testing.T) {
	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/states/sensor.house_power" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"entity_id":"sensor.house_power","state":"4080","attributes":{"friendly_name":"Hausverbrauch","device_class":"power","unit_of_measurement":"W"}}`))
	}))
	t.Cleanup(ha.Close)

	a := consumerAppHAUSV422(t)
	tenant := a.tenants["jhw22"]
	tenant.HA = homeassistant.NewConfig(ha.URL, "fixture", "", "", "")
	a.tenants["jhw22"] = tenant
	if err := a.energyStore.UpsertMapping(energy.EntityMapping{
		ID: energy.NewID("mapping"), TenantSlug: "jhw22", EntityID: "sensor.house_power",
		Metric: energy.MetricLoadPower, DisplayName: "Hausverbrauch", Unit: "W", Confirmed: true,
	}); err != nil {
		t.Fatal(err)
	}

	response := authedRequest(t, a, "owner@example.com", "/app/energie/live")
	if response.Code != http.StatusOK {
		t.Fatalf("Live-Refresh: status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Live-Refresh darf nicht gecacht werden: %q", response.Header().Get("Cache-Control"))
	}
	var payload energyLiveRefreshResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Live-JSON lesen: %v", err)
	}
	if payload.Flow.Home.Value != "4,08" || payload.Flow.Home.Unit != "kW" || payload.UpdatedAt.IsZero() {
		t.Fatalf("unerwarteter Live-Stand: %+v", payload)
	}
}
