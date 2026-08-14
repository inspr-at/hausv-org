package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/homeconnector"
	"github.com/inspr-at/hausv-org/internal/store"
)

const (
	homeConnectorPairingTTL = 10 * time.Minute
	homeConnectorFreshFor   = 90 * time.Second
	maxHomeConnectorBody    = 8 << 10
)

var (
	homeConnectorPairSourcePolicy  = authRatePolicy{scope: "home-connector-pair-source", limit: 10, window: authRateWindow}
	homeConnectorPairAccountPolicy = authRatePolicy{scope: "home-connector-pair-code", limit: 5, window: authRateWindow}
	homeConnectorBeatSourcePolicy  = authRatePolicy{scope: "home-connector-beat-source", limit: 120, window: authRateWindow}
	homeConnectorBeatAccountPolicy = authRatePolicy{scope: "home-connector-beat-account", limit: 60, window: authRateWindow}
	homeConnectorDownloads         = map[string]string{
		"linux-amd64": "hausv-connector-linux-amd64",
		"linux-arm64": "hausv-connector-linux-arm64",
	}
)

func homeConnectorSecret(secret []byte) []byte {
	digest := sha256.Sum256(append([]byte("hausv-home-connector-v1\x00"), secret...))
	return digest[:]
}

func homeConnectorHash(key []byte, secret string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(strings.TrimSpace(secret)))
	return mac.Sum(nil)
}

func (a *app) startHomeConnectorPairing(w http.ResponseWriter, r *http.Request) {
	reservation, ok := a.homeSetupReservation(r)
	if !ok || a.homeConnectors == nil || len(a.homeConnectorHashKey) == 0 {
		http.Redirect(w, r, "/start", http.StatusSeeOther)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	pairingCode, err := randomToken(32)
	if err != nil {
		http.Error(w, "Kopplung konnte nicht vorbereitet werden", http.StatusInternalServerError)
		return
	}
	now := time.Now()
	connector, err := a.homeConnectors.StartPairing(reservation.Slug,
		homeConnectorHash(a.homeConnectorHashKey, pairingCode), now.Add(homeConnectorPairingTTL), now)
	if err != nil {
		logWarn("home connector pairing failed", "error_type", "store")
		http.Error(w, "Kopplung konnte nicht vorbereitet werden", http.StatusInternalServerError)
		return
	}
	a.renderHomeConnectorStart(w, reservation, connector, true, pairingCode, now)
}

func (a *app) revokeHomeConnector(w http.ResponseWriter, r *http.Request) {
	reservation, ok := a.homeSetupReservation(r)
	if !ok || a.homeConnectors == nil {
		http.Redirect(w, r, "/start", http.StatusSeeOther)
		return
	}
	if !sameOriginPost(r) {
		http.Error(w, "Bad request", http.StatusForbidden)
		return
	}
	if _, _, err := a.homeConnectors.Revoke(reservation.Slug, time.Now()); err != nil {
		logWarn("home connector revoke failed", "error_type", "store")
		http.Error(w, "Verbindung konnte nicht widerrufen werden", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/start/connector?revoked=1", http.StatusSeeOther)
}

func (a *app) pairHomeConnector(w http.ResponseWriter, r *http.Request) {
	if !a.isMarketingHost(r) || a.homeConnectors == nil || len(a.homeConnectorHashKey) == 0 {
		http.NotFound(w, r)
		return
	}
	var request homeconnector.PairRequest
	if !decodeHomeConnectorJSON(w, r, &request) {
		return
	}
	if !validHomeConnectorMetadata(request.Heartbeat) || !validConnectorSecret(request.PairingCode) {
		writeHomeConnectorUnauthorized(w)
		return
	}
	pairingHash := homeConnectorHash(a.homeConnectorHashKey, request.PairingCode)
	if !a.allowAuthRequest(w, r, homeConnectorPairSourcePolicy, homeConnectorPairAccountPolicy, fmt.Sprintf("%x", pairingHash)) {
		return
	}
	credential, err := randomToken(32)
	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	item, exchanged, err := a.homeConnectors.ExchangePairing(pairingHash,
		homeConnectorHash(a.homeConnectorHashKey, credential), store.HomeConnectorHeartbeat{
			ConnectorVersion: request.ConnectorVersion, HomeAssistantVersion: request.HomeAssistantVersion, EntityCount: request.EntityCount,
		}, time.Now())
	if err != nil {
		logWarn("home connector exchange failed", "error_type", "store")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	if !exchanged || item.Slug == "" {
		writeHomeConnectorUnauthorized(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(homeconnector.PairResponse{Credential: credential})
}

func (a *app) heartbeatHomeConnector(w http.ResponseWriter, r *http.Request) {
	if !a.isMarketingHost(r) || a.homeConnectors == nil || len(a.homeConnectorHashKey) == 0 {
		http.NotFound(w, r)
		return
	}
	credential, ok := connectorBearer(r.Header.Get("Authorization"))
	if !ok {
		writeHomeConnectorUnauthorized(w)
		return
	}
	credentialHash := homeConnectorHash(a.homeConnectorHashKey, credential)
	if !a.allowAuthRequest(w, r, homeConnectorBeatSourcePolicy, homeConnectorBeatAccountPolicy, fmt.Sprintf("%x", credentialHash)) {
		return
	}
	var heartbeat homeconnector.Heartbeat
	if !decodeHomeConnectorJSON(w, r, &heartbeat) {
		return
	}
	if !validHomeConnectorMetadata(heartbeat) {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	_, accepted, err := a.homeConnectors.Heartbeat(credentialHash, store.HomeConnectorHeartbeat{
		ConnectorVersion: heartbeat.ConnectorVersion, HomeAssistantVersion: heartbeat.HomeAssistantVersion, EntityCount: heartbeat.EntityCount,
	}, time.Now())
	if err != nil {
		logWarn("home connector heartbeat failed", "error_type", "store")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	if !accepted {
		writeHomeConnectorUnauthorized(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) downloadHomeConnector(w http.ResponseWriter, r *http.Request) {
	if !a.isMarketingHost(r) {
		http.NotFound(w, r)
		return
	}
	filename := r.PathValue("filename")
	allowed := false
	for _, candidate := range homeConnectorDownloads {
		allowed = allowed || filename == candidate
	}
	if !allowed || a.homeConnectorDownloadDir == "" {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(filepath.Join(a.homeConnectorDownloadDir, filename))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	http.ServeContent(w, r, filename, info.ModTime(), file)
}

func (a *app) homeSetupReservation(r *http.Request) (store.HomeReservation, bool) {
	if !a.isMarketingHost(r) || a.homeReservations == nil || a.homeSetupSessions == nil {
		return store.HomeReservation{}, false
	}
	cookie, err := r.Cookie(homeSetupCookieName)
	if err != nil {
		return store.HomeReservation{}, false
	}
	email, slug, _, ok := a.homeSetupSessions.Get(cookie.Value)
	if !ok {
		return store.HomeReservation{}, false
	}
	reservation, found, err := a.homeReservations.Get(slug)
	if err != nil || !found || reservation.OwnerEmail != normalizeEmail(email) ||
		(reservation.Status != store.HomeReservationEmailConfirmed && reservation.Status != store.HomeReservationActive) {
		return store.HomeReservation{}, false
	}
	return reservation, true
}

func decodeHomeConnectorJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/json") {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxHomeConnectorBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func validHomeConnectorMetadata(heartbeat homeconnector.Heartbeat) bool {
	return safeConnectorMetadata(heartbeat.ConnectorVersion) && safeConnectorMetadata(heartbeat.HomeAssistantVersion) &&
		heartbeat.EntityCount >= 0 && heartbeat.EntityCount <= homeconnector.MaxEntityCount
}

func safeConnectorMetadata(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if character < 0x20 || character > 0x7e {
			return false
		}
	}
	return true
}

func validConnectorSecret(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 32 || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if character != '-' && character != '_' && (character < 'A' || character > 'Z') && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
			return false
		}
	}
	return true
}

func connectorBearer(header string) (string, bool) {
	prefix, value, found := strings.Cut(strings.TrimSpace(header), " ")
	return value, found && strings.EqualFold(prefix, "Bearer") && validConnectorSecret(value)
}

func writeHomeConnectorUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
