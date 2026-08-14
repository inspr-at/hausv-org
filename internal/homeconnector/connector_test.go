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
