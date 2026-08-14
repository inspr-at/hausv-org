package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/energy"
	"github.com/inspr-at/hausv-org/internal/homeconnector"
)

func TestHomeConnectorReadingsStayScopedAndDriveLiveEnergyHAUSV477(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	handler := a.handler()
	cookie := confirmedHomeSetupCookieHAUSV471(t, a, "private-home", "owner@example.com")
	pairingPage := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodPost, "/start/connector/pairing", cookie, url.Values{})
	pairingCode := extractPairingCodeHAUSV471(t, pairingPage.Body.String())
	now := time.Now().UTC().Truncate(time.Second)
	initial := []homeconnector.Reading{
		{EntityID: "sensor.grid_import_power", State: "1200", DisplayName: "Netzbezug", Unit: "W", DeviceClass: "power", StateClass: "measurement", LastUpdated: now},
		{EntityID: "sensor.pv_power", State: "2.4", DisplayName: "PV-Leistung", Unit: "kW", DeviceClass: "power", StateClass: "measurement", LastUpdated: now},
	}
	pairResponse := postHomeConnectorPayloadHAUSV477(t, handler, "/api/home-connectors/pair", "", homeconnector.PairRequest{
		PairingCode: pairingCode,
		Heartbeat:   homeconnector.Heartbeat{ConnectorVersion: "0.86.0", HomeAssistantVersion: "2026.8.1", EntityCount: 99, Readings: initial},
	})
	if pairResponse.Code != http.StatusCreated {
		t.Fatalf("pair status=%d body=%q", pairResponse.Code, pairResponse.Body.String())
	}
	var paired homeconnector.PairResponse
	if err := json.Unmarshal(pairResponse.Body.Bytes(), &paired); err != nil || paired.Credential == "" {
		t.Fatalf("pair response=%q err=%v", pairResponse.Body.String(), err)
	}
	readings, err := a.homeConnectorReadings.List("private-home")
	if err != nil || len(readings) != 2 {
		t.Fatalf("private readings=%+v err=%v", readings, err)
	}
	foreign, err := a.homeConnectorReadings.List("another-home")
	if err != nil || len(foreign) != 0 {
		t.Fatalf("foreign readings=%+v err=%v", foreign, err)
	}

	tenant := tenantConfig{Slug: "private-home", Name: "Privates Zuhause"}
	if err := a.saveSelectedEnergyMappings(context.Background(), tenant, []string{"sensor.grid_import_power"}); err != nil {
		t.Fatal(err)
	}
	mappings, err := a.energyStore.ListMappings("private-home")
	if err != nil || len(mappings) != 1 || mappings[0].Metric != energy.MetricGridImportPower {
		t.Fatalf("selected mappings=%+v err=%v", mappings, err)
	}
	heartbeatResponse := postHomeConnectorPayloadHAUSV477(t, handler, "/api/home-connectors/heartbeat", paired.Credential,
		homeconnector.Heartbeat{ConnectorVersion: "0.86.0", HomeAssistantVersion: "2026.8.1", EntityCount: 99, Readings: []homeconnector.Reading{
			{EntityID: "sensor.grid_import_power", State: "1800", DisplayName: "Netzbezug", Unit: "W", DeviceClass: "power", StateClass: "measurement", LastUpdated: now.Add(time.Second)},
			{EntityID: "sensor.pv_power", State: "9.9", DisplayName: "PV-Leistung", Unit: "kW", DeviceClass: "power", StateClass: "measurement", LastUpdated: now.Add(time.Second)},
		}})
	if heartbeatResponse.Code != http.StatusOK {
		t.Fatalf("heartbeat status=%d body=%q", heartbeatResponse.Code, heartbeatResponse.Body.String())
	}
	var heartbeatResult homeconnector.HeartbeatResponse
	if err := json.Unmarshal(heartbeatResponse.Body.Bytes(), &heartbeatResult); err != nil || len(heartbeatResult.SelectedEntityIDs) != 1 || heartbeatResult.SelectedEntityIDs[0] != "sensor.grid_import_power" {
		t.Fatalf("heartbeat result=%+v err=%v", heartbeatResult, err)
	}
	readings, _ = a.homeConnectorReadings.List("private-home")
	if readings[0].EntityID != "sensor.grid_import_power" || readings[0].State != "1800" || readings[1].State != "2.4" {
		t.Fatalf("filtered readings=%+v", readings)
	}

	metrics, status, latest := a.currentEnergyMetrics(context.Background(), tenant, mappings, energy.DefaultProfile("private-home", now))
	if len(metrics) != 1 || metrics[0].Numeric != 1800 || status == "Noch keine Messquelle verbunden" || latest.IsZero() {
		t.Fatalf("metrics=%+v status=%q latest=%v", metrics, status, latest)
	}

	revoked := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodPost, "/start/connector/revoke", cookie, url.Values{})
	if revoked.Code != http.StatusSeeOther {
		t.Fatalf("revoke status=%d", revoked.Code)
	}
	readings, err = a.homeConnectorReadings.List("private-home")
	if err != nil || len(readings) != 0 {
		t.Fatalf("readings after revoke=%+v err=%v", readings, err)
	}
}

func postHomeConnectorPayloadHAUSV477(t *testing.T, handler http.Handler, path, credential string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://hausv.org"+path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
