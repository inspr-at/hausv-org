package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/inspr-at/hausv-org/internal/homeconnector"
	"github.com/inspr-at/hausv-org/internal/store"
)

var homePairingCodePattern = regexp.MustCompile(`<code class="home-pairing-code">([A-Za-z0-9_-]+)</code>`)

func TestHomeConnectorPairRotateHeartbeatAndRevoke(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	handler := a.handler()
	setupCookie := confirmedHomeSetupCookieHAUSV471(t, a, "stadtpark-home", "owner@example.com")

	initial := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodGet, "/start/connector", setupCookie, nil)
	if initial.Code != http.StatusOK || !strings.Contains(initial.Body.String(), "Noch nicht gekoppelt") || !strings.Contains(initial.Body.String(), "Connector koppeln") {
		t.Fatalf("initial page status=%d body=%q", initial.Code, initial.Body.String())
	}

	pairingPage := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodPost, "/start/connector/pairing", setupCookie, url.Values{})
	if pairingPage.Code != http.StatusOK || pairingPage.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("pairing page status=%d cache=%q", pairingPage.Code, pairingPage.Header().Get("Cache-Control"))
	}
	pairingOne := extractPairingCodeHAUSV471(t, pairingPage.Body.String())
	for _, forbidden := range []string{"owner@example.com", `name="token"`, `name="base_url"`, `name="ha_token"`} {
		if strings.Contains(pairingPage.Body.String(), forbidden) {
			t.Fatalf("pairing page exposed %q", forbidden)
		}
	}
	credentialOne := pairConnectorHAUSV471(t, handler, pairingOne, http.StatusCreated)
	stored, found, err := a.homeConnectors.Get("stadtpark-home")
	if err != nil || !found || stored.Status != store.HomeConnectorConnected || stored.Generation != 1 || stored.EntityCount != 27 {
		t.Fatalf("stored connector = %+v found=%v err=%v", stored, found, err)
	}
	if bytes.Equal(stored.CredentialHash, []byte(credentialOne)) || bytes.Equal(stored.PairingHash, []byte(pairingOne)) {
		t.Fatal("connector secrets were stored verbatim")
	}
	pairConnectorHAUSV471(t, handler, pairingOne, http.StatusUnauthorized)

	heartbeat := heartbeatConnectorHAUSV471(t, handler, credentialOne, http.StatusNoContent)
	if heartbeat.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("heartbeat cache control = %q", heartbeat.Header().Get("Cache-Control"))
	}
	connected := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodGet, "/start/connector", setupCookie, nil)
	for _, want := range []string{"Verbunden · nur lesen", "2026.8.1", "27", "Zugang erneuern", "Verbindung widerrufen", "noch nicht übernommen"} {
		if !strings.Contains(connected.Body.String(), want) {
			t.Fatalf("connected page missing %q", want)
		}
	}
	if strings.Contains(connected.Body.String(), credentialOne) || strings.Contains(connected.Body.String(), pairingOne) {
		t.Fatal("connected page exposed a connector secret")
	}

	rotationPage := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodPost, "/start/connector/pairing", setupCookie, url.Values{})
	pairingTwo := extractPairingCodeHAUSV471(t, rotationPage.Body.String())
	if pairingTwo == pairingOne {
		t.Fatal("rotation reused the pairing code")
	}
	heartbeatConnectorHAUSV471(t, handler, credentialOne, http.StatusNoContent)
	credentialTwo := pairConnectorHAUSV471(t, handler, pairingTwo, http.StatusCreated)
	if credentialTwo == credentialOne {
		t.Fatal("rotation reused the connector credential")
	}
	heartbeatConnectorHAUSV471(t, handler, credentialOne, http.StatusUnauthorized)
	heartbeatConnectorHAUSV471(t, handler, credentialTwo, http.StatusNoContent)

	revoked := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodPost, "/start/connector/revoke", setupCookie, url.Values{})
	if revoked.Code != http.StatusSeeOther || revoked.Header().Get("Location") != "/start/connector?revoked=1" {
		t.Fatalf("revoke status=%d location=%q", revoked.Code, revoked.Header().Get("Location"))
	}
	heartbeatConnectorHAUSV471(t, handler, credentialTwo, http.StatusUnauthorized)
	revokedPage := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodGet, "/start/connector", setupCookie, nil)
	if !strings.Contains(revokedPage.Body.String(), "Verbindung widerrufen") || strings.Contains(revokedPage.Body.String(), "2026.8.1") {
		t.Fatalf("revoked page = %q", revokedPage.Body.String())
	}
}

func TestHomeConnectorRejectsExpiredForeignAndMalformedRequests(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	handler := a.handler()
	ownerCookie := confirmedHomeSetupCookieHAUSV471(t, a, "secure-home", "owner@example.com")
	foreignToken, _, err := a.homeSetupSessions.Put("other@example.com", "secure-home", authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	foreignCookie := &http.Cookie{Name: homeSetupCookieName, Value: foreignToken}
	foreign := homeConnectorSetupRequestHAUSV471(t, handler, http.MethodPost, "/start/connector/pairing", foreignCookie, url.Values{})
	if foreign.Code != http.StatusSeeOther || foreign.Header().Get("Location") != "/start" {
		t.Fatalf("foreign pairing status=%d location=%q", foreign.Code, foreign.Header().Get("Location"))
	}
	wrongOrigin := httptest.NewRequest(http.MethodPost, "http://hausv.org/start/connector/revoke", strings.NewReader(""))
	wrongOrigin.Header.Set("Origin", "https://example.test")
	wrongOrigin.AddCookie(ownerCookie)
	wrongOriginResponse := httptest.NewRecorder()
	handler.ServeHTTP(wrongOriginResponse, wrongOrigin)
	if wrongOriginResponse.Code != http.StatusForbidden {
		t.Fatalf("wrong origin status=%d", wrongOriginResponse.Code)
	}

	expiredCode := strings.Repeat("e", 43)
	point := time.Now()
	if _, err := a.homeConnectors.StartPairing("secure-home", homeConnectorHash(a.homeConnectorHashKey, expiredCode), point, point); err == nil {
		t.Fatal("store accepted already-expired pairing")
	}
	validCode := strings.Repeat("v", 43)
	past := time.Now().Add(-20 * time.Minute)
	if _, err := a.homeConnectors.StartPairing("secure-home", homeConnectorHash(a.homeConnectorHashKey, validCode), past.Add(10*time.Minute), past); err != nil {
		t.Fatal(err)
	}
	pairConnectorHAUSV471(t, handler, validCode, http.StatusUnauthorized)

	malformed := httptest.NewRequest(http.MethodPost, "http://hausv.org/api/home-connectors/pair", strings.NewReader(`{"pairing_code":"x","unknown":true}`))
	malformed.Header.Set("Content-Type", "application/json")
	malformedResponse := httptest.NewRecorder()
	handler.ServeHTTP(malformedResponse, malformed)
	if malformedResponse.Code != http.StatusBadRequest || strings.Contains(malformedResponse.Body.String(), validCode) {
		t.Fatalf("malformed status=%d body=%q", malformedResponse.Code, malformedResponse.Body.String())
	}
	oversized := httptest.NewRequest(http.MethodPost, "http://hausv.org/api/home-connectors/pair", strings.NewReader(`{"pairing_code":"`+strings.Repeat("x", maxHomeConnectorBody)+`"}`))
	oversized.Header.Set("Content-Type", "application/json")
	oversizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(oversizedResponse, oversized)
	if oversizedResponse.Code != http.StatusBadRequest {
		t.Fatalf("oversized status=%d", oversizedResponse.Code)
	}

	tenantAPI := httptest.NewRecorder()
	tenantRequest := httptest.NewRequest(http.MethodPost, "http://hausv.org/demo/api/home-connectors/heartbeat", strings.NewReader(`{}`))
	tenantRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(tenantAPI, tenantRequest)
	if tenantAPI.Code != http.StatusNotFound {
		t.Fatalf("tenant connector API status=%d", tenantAPI.Code)
	}
}

func TestHomeConnectorAPIsAreRateLimited(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	handler := a.handler()
	pairingCode := strings.Repeat("p", 43)
	for index := 0; index < homeConnectorPairAccountPolicy.limit; index++ {
		pairConnectorHAUSV471(t, handler, pairingCode, http.StatusUnauthorized)
	}
	limitedPair := pairConnectorHAUSV471Response(t, handler, pairingCode)
	if limitedPair.Code != http.StatusTooManyRequests || limitedPair.Header().Get("Retry-After") == "" || strings.Contains(limitedPair.Body.String(), pairingCode) {
		t.Fatalf("pair limit status=%d retry=%q body=%q", limitedPair.Code, limitedPair.Header().Get("Retry-After"), limitedPair.Body.String())
	}

	credential := strings.Repeat("c", 43)
	for index := 0; index < homeConnectorBeatAccountPolicy.limit; index++ {
		heartbeatConnectorHAUSV471(t, handler, credential, http.StatusUnauthorized)
	}
	limitedHeartbeat := heartbeatConnectorHAUSV471(t, handler, credential, http.StatusTooManyRequests)
	if limitedHeartbeat.Header().Get("Retry-After") == "" || strings.Contains(limitedHeartbeat.Body.String(), credential) {
		t.Fatalf("heartbeat limit retry=%q body=%q", limitedHeartbeat.Header().Get("Retry-After"), limitedHeartbeat.Body.String())
	}
}

func TestHomeConnectorDownloadsOnlyFixedArtifacts(t *testing.T) {
	a := newTestPortalApp(t, userProfile{Email: "admin@example.com", Role: roleAdmin, Tenants: []string{"demo"}, AuthMethods: defaultAuthMethods()})
	filename := homeConnectorDownloads["linux-amd64"]
	if err := os.WriteFile(filepath.Join(a.homeConnectorDownloadDir, filename), []byte("binary-fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	a.handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://hausv.org/downloads/"+filename, nil))
	if response.Code != http.StatusOK || response.Body.String() != "binary-fixture" || response.Header().Get("Content-Type") != "application/octet-stream" || !strings.Contains(response.Header().Get("Content-Disposition"), filename) {
		t.Fatalf("download status=%d type=%q disposition=%q body=%q", response.Code, response.Header().Get("Content-Type"), response.Header().Get("Content-Disposition"), response.Body.String())
	}
	for _, path := range []string{"/downloads/hausv-connector-linux-386", "/downloads/not-a-connector"} {
		blocked := httptest.NewRecorder()
		a.handler().ServeHTTP(blocked, httptest.NewRequest(http.MethodGet, "http://hausv.org"+path, nil))
		if blocked.Code != http.StatusNotFound {
			t.Fatalf("blocked download %s status=%d", path, blocked.Code)
		}
	}
}

func confirmedHomeSetupCookieHAUSV471(t *testing.T, a *app, slug, email string) *http.Cookie {
	t.Helper()
	if _, err := a.homeReservations.Reserve(store.HomeReservation{
		Slug: slug, HouseholdName: "Zuhause am Stadtpark", OwnerEmail: email, AuthorizationConfirmed: true,
	}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := a.homeReservations.Confirm(slug, email, time.Now()); err != nil || !ok {
		t.Fatalf("confirm: ok=%v err=%v", ok, err)
	}
	token, _, err := a.homeSetupSessions.Put(email, slug, authMethodEmail, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: homeSetupCookieName, Value: token}
}

func homeConnectorSetupRequestHAUSV471(t *testing.T, handler http.Handler, method, path string, cookie *http.Cookie, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	body := strings.NewReader("")
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	request := httptest.NewRequest(method, "http://hausv.org"+path, body)
	request.AddCookie(cookie)
	if method == http.MethodPost {
		request.Header.Set("Origin", "http://hausv.org")
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func extractPairingCodeHAUSV471(t *testing.T, body string) string {
	t.Helper()
	match := homePairingCodePattern.FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("pairing code missing from body")
	}
	return match[1]
}

func pairConnectorHAUSV471(t *testing.T, handler http.Handler, pairingCode string, wantStatus int) string {
	t.Helper()
	response := pairConnectorHAUSV471Response(t, handler, pairingCode)
	if response.Code != wantStatus {
		t.Fatalf("pair status=%d want=%d body=%q", response.Code, wantStatus, response.Body.String())
	}
	if wantStatus != http.StatusCreated {
		return ""
	}
	var result homeconnector.PairResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || !validConnectorSecret(result.Credential) {
		t.Fatalf("pair response valid=%v err=%v", validConnectorSecret(result.Credential), err)
	}
	if strings.Contains(response.Body.String(), pairingCode) {
		t.Fatal("pair response echoed the one-time code")
	}
	return result.Credential
}

func pairConnectorHAUSV471Response(t *testing.T, handler http.Handler, pairingCode string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(homeconnector.PairRequest{PairingCode: pairingCode, Heartbeat: homeconnector.Heartbeat{
		ConnectorVersion: "0.83.0", HomeAssistantVersion: "2026.8.1", EntityCount: 27,
	}})
	request := httptest.NewRequest(http.MethodPost, "http://hausv.org/api/home-connectors/pair", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func heartbeatConnectorHAUSV471(t *testing.T, handler http.Handler, credential string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(homeconnector.Heartbeat{ConnectorVersion: "0.83.0", HomeAssistantVersion: "2026.8.1", EntityCount: 27})
	request := httptest.NewRequest(http.MethodPost, "http://hausv.org/api/home-connectors/heartbeat", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+credential)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != wantStatus || strings.Contains(response.Body.String(), credential) {
		t.Fatalf("heartbeat status=%d want=%d body=%q", response.Code, wantStatus, response.Body.String())
	}
	return response
}
