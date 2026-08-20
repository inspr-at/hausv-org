package homeconnector

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConnectorPairsLocallyWithoutSendingHomeAssistantSecrets(t *testing.T) {
	const homeAssistantToken = "local-home-assistant-secret"
	const pairingCode = "pairing_code_abcdefghijklmnopqrstuvwxyz123456"
	const connectorCredential = "connector_credential_abcdefghijklmnopqrstuvwxyz123456"
	var mu sync.Mutex
	var methods []string
	var pairBody, heartbeatBody []byte
	var heartbeatAuthorization string

	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		methods = append(methods, r.Method+" "+r.URL.Path)
		mu.Unlock()
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer "+homeAssistantToken {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/config":
			_, _ = io.WriteString(w, `{"version":"2026.8.1"}`)
		case "/api/states":
			_, _ = io.WriteString(w, `[{"entity_id":"sensor.private_power"},{"entity_id":"sensor.private_energy"}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ha.Close()

	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/api/home-connectors/pair":
			pairBody = body
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(PairResponse{Credential: connectorCredential})
		case "/api/home-connectors/heartbeat":
			heartbeatBody = body
			heartbeatAuthorization = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer portal.Close()

	directory := t.TempDir()
	tokenFile := filepath.Join(directory, "ha.token")
	credentialFile := filepath.Join(directory, "connector.credential")
	if err := os.WriteFile(tokenFile, []byte(homeAssistantToken), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RunWithOptions(context.Background(), Options{
		PortalURL: portal.URL, PairingCode: pairingCode, HAURL: ha.URL, HATokenFile: tokenFile,
		CredentialFile: credentialFile, Once: true, HTTPClient: portal.Client(),
	}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	gotMethods := append([]string(nil), methods...)
	mu.Unlock()
	if strings.Join(gotMethods, ",") != "GET /api/config,GET /api/states" {
		t.Fatalf("home assistant methods = %v", gotMethods)
	}
	for _, payload := range [][]byte{pairBody, heartbeatBody} {
		text := string(payload)
		for _, forbidden := range []string{homeAssistantToken, ha.URL, "sensor.private_power", "sensor.private_energy"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("portal payload exposed %q: %s", forbidden, text)
			}
		}
	}
	if !strings.Contains(string(pairBody), pairingCode) || !strings.Contains(string(pairBody), `"entity_count":2`) {
		t.Fatalf("pair body = %s", pairBody)
	}
	if heartbeatAuthorization != "Bearer "+connectorCredential {
		t.Fatalf("heartbeat authorization = %q", heartbeatAuthorization)
	}
	stored, err := os.ReadFile(credentialFile)
	if err != nil || strings.TrimSpace(string(stored)) != connectorCredential {
		t.Fatalf("stored credential valid=%v err=%v", strings.TrimSpace(string(stored)) == connectorCredential, err)
	}
	if info, err := os.Stat(credentialFile); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("credential mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestConnectorReusesCredentialAndReportsRevocationWithoutSecret(t *testing.T) {
	const homeAssistantToken = "local-home-assistant-secret"
	const connectorCredential = "connector_credential_abcdefghijklmnopqrstuvwxyz123456"
	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/config" {
			_, _ = io.WriteString(w, `{"version":"2026.8.1"}`)
			return
		}
		_, _ = io.WriteString(w, `[]`)
	}))
	defer ha.Close()
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/home-connectors/pair" {
			t.Fatal("existing credential unexpectedly started pairing")
		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer portal.Close()
	directory := t.TempDir()
	tokenFile := filepath.Join(directory, "ha.token")
	credentialFile := filepath.Join(directory, "connector.credential")
	if err := os.WriteFile(tokenFile, []byte(homeAssistantToken), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credentialFile, []byte(connectorCredential), 0o600); err != nil {
		t.Fatal(err)
	}
	err := RunWithOptions(context.Background(), Options{
		PortalURL: portal.URL, HAURL: ha.URL, HATokenFile: tokenFile, CredentialFile: credentialFile,
		Once: true, HTTPClient: portal.Client(),
	})
	if err == nil || !strings.Contains(err.Error(), "widerrufen") || strings.Contains(err.Error(), connectorCredential) || strings.Contains(err.Error(), homeAssistantToken) {
		t.Fatalf("revocation error = %v", err)
	}
}

func TestConnectorSendsCatalogThenOnlyPortalSelectedReadings(t *testing.T) {
	const token = "local-home-assistant-secret"
	const pairingCode = "pairing_code_abcdefghijklmnopqrstuvwxyz123456"
	const credential = "connector_credential_abcdefghijklmnopqrstuvwxyz123456"
	updated := time.Now().UTC().Format(time.RFC3339)
	ha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/config" {
			_, _ = io.WriteString(w, `{"version":"2026.8.1"}`)
			return
		}
		_, _ = io.WriteString(w, `[`+
			`{"entity_id":"sensor.grid_import_power","state":"1250","last_updated":"`+updated+`","attributes":{"friendly_name":"Netzbezug","unit_of_measurement":"W","device_class":"power","state_class":"measurement"}},`+
			`{"entity_id":"sensor.pv_power","state":"2.2","last_updated":"`+updated+`","attributes":{"friendly_name":"PV","unit_of_measurement":"kW","device_class":"power","state_class":"measurement"}},`+
			`{"entity_id":"sensor.no_timestamp_power","state":"900","attributes":{"friendly_name":"No timestamp","unit_of_measurement":"W","device_class":"power","state_class":"measurement"}},`+
			`{"entity_id":"sensor.bedroom_temperature","state":"21","last_updated":"`+updated+`","attributes":{"unit_of_measurement":"°C","device_class":"temperature"}},`+
			`{"entity_id":"switch.private_alarm","state":"on","last_updated":"`+updated+`","attributes":{}}]`)
	}))
	defer ha.Close()
	var pairRequest PairRequest
	var heartbeat Heartbeat
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/home-connectors/pair":
			_ = json.NewDecoder(r.Body).Decode(&pairRequest)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(PairResponse{Credential: credential, SelectedEntityIDs: []string{"sensor.grid_import_power"}})
		case "/api/home-connectors/heartbeat":
			_ = json.NewDecoder(r.Body).Decode(&heartbeat)
			_ = json.NewEncoder(w).Encode(HeartbeatResponse{SelectedEntityIDs: []string{"sensor.grid_import_power"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer portal.Close()
	directory := t.TempDir()
	tokenFile := filepath.Join(directory, "ha.token")
	credentialFile := filepath.Join(directory, "connector.credential")
	if err := os.WriteFile(tokenFile, []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RunWithOptions(context.Background(), Options{
		PortalURL: portal.URL, PairingCode: pairingCode, HAURL: ha.URL, HATokenFile: tokenFile,
		CredentialFile: credentialFile, Once: true, HTTPClient: portal.Client(),
	}); err != nil {
		t.Fatal(err)
	}
	if len(pairRequest.Readings) != 2 {
		t.Fatalf("pair catalog=%+v", pairRequest.Readings)
	}
	if len(heartbeat.Readings) != 1 || heartbeat.Readings[0].EntityID != "sensor.grid_import_power" {
		t.Fatalf("selected heartbeat=%+v", heartbeat.Readings)
	}
	encoded, _ := json.Marshal(pairRequest)
	for _, forbidden := range []string{token, ha.URL, "sensor.no_timestamp_power", "sensor.bedroom_temperature", "switch.private_alarm"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("catalog exposed %q: %s", forbidden, encoded)
		}
	}
}

func TestConnectorRejectsInsecurePortalAndLooseSecretFiles(t *testing.T) {
	if _, err := normalizePortalURL("http://example.com"); err == nil {
		t.Fatal("insecure public portal accepted")
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "secret")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 40)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readSecretFile(path, "Test"); err == nil {
		t.Fatal("world-readable secret file accepted")
	}
	if err := RunWithOptions(context.Background(), Options{Interval: time.Second}); err == nil {
		t.Fatal("invalid connector options accepted")
	}
}
