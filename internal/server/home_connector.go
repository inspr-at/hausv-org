package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/homeconnector"
	"github.com/inspr-at/hausv-org/internal/store"
)

const (
	homeConnectorPairingTTL = 10 * time.Minute
	homeConnectorFreshFor   = 90 * time.Second
	maxHomeConnectorBody    = 64 << 10
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
	if a.homeConnectorReadings != nil {
		if err := a.homeConnectorReadings.Clear(reservation.Slug); err != nil {
			logWarn("home connector readings revoke failed", "error_type", "store")
			http.Error(w, "Verbindung konnte nicht widerrufen werden", http.StatusInternalServerError)
			return
		}
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
	now := time.Now()
	if !validHomeConnectorMetadata(request.Heartbeat) || !validConnectorSecret(request.PairingCode) || !validHomeConnectorReadingSet(request.Readings, now) {
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
		}, now)
	if err != nil {
		logWarn("home connector exchange failed", "error_type", "store")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	if !exchanged || item.Slug == "" {
		writeHomeConnectorUnauthorized(w)
		return
	}
	if !a.persistHomeConnectorReadings(item.Slug, request.Readings, now, true) {
		_, _, _ = a.homeConnectors.Revoke(item.Slug, now)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(homeconnector.PairResponse{Credential: credential, SelectedEntityIDs: a.selectedHomeConnectorEntities(item.Slug)})
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
	now := time.Now()
	if !validHomeConnectorMetadata(heartbeat) || !validHomeConnectorReadingSet(heartbeat.Readings, now) {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	item, accepted, err := a.homeConnectors.Heartbeat(credentialHash, store.HomeConnectorHeartbeat{
		ConnectorVersion: heartbeat.ConnectorVersion, HomeAssistantVersion: heartbeat.HomeAssistantVersion, EntityCount: heartbeat.EntityCount,
	}, now)
	if err != nil {
		logWarn("home connector heartbeat failed", "error_type", "store")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	if !accepted {
		writeHomeConnectorUnauthorized(w)
		return
	}
	selected := a.selectedHomeConnectorEntities(item.Slug)
	incoming := filterHomeConnectorReadings(heartbeat.Readings, selected)
	if len(selected) == 0 && !a.homeConnectorCatalogAllowed(item.Slug) {
		incoming = nil
		if a.homeConnectorReadings != nil {
			if err := a.homeConnectorReadings.Clear(item.Slug); err != nil {
				logWarn("home connector readings clear failed", "error_type", "store")
				http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
				return
			}
		}
	}
	if !a.persistHomeConnectorReadings(item.Slug, incoming, now, false) {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(homeconnector.HeartbeatResponse{SelectedEntityIDs: selected})
}

func (a *app) homeConnectorCatalogAllowed(slug string) bool {
	if a.energyStore == nil {
		return true
	}
	profile, exists, err := a.energyStore.Profile(slug)
	return err == nil && (!exists || !energyProfileUnclaimed(profile))
}

func filterHomeConnectorReadings(input []homeconnector.Reading, selected []string) []homeconnector.Reading {
	if len(selected) == 0 {
		return input
	}
	wanted := make(map[string]bool, len(selected))
	for _, entityID := range selected {
		wanted[strings.ToLower(strings.TrimSpace(entityID))] = true
	}
	out := make([]homeconnector.Reading, 0, len(selected))
	for _, reading := range input {
		if wanted[strings.ToLower(strings.TrimSpace(reading.EntityID))] ||
			homeconnector.IsVehicleSleepReading(reading.EntityID, reading.State, reading.DisplayName) {
			out = append(out, reading)
		}
	}
	return out
}

func (a *app) persistHomeConnectorReadings(slug string, input []homeconnector.Reading, now time.Time, replace bool) bool {
	if len(input) > homeconnector.MaxReadings || a.homeConnectorReadings == nil {
		return len(input) == 0 && a.homeConnectorReadings == nil
	}
	readings := make([]store.HomeConnectorReading, 0, len(input))
	seen := map[string]bool{}
	for _, reading := range input {
		entityID := strings.ToLower(strings.TrimSpace(reading.EntityID))
		if seen[entityID] || !validHomeConnectorReading(reading, now) {
			return false
		}
		seen[entityID] = true
		readings = append(readings, store.HomeConnectorReading{
			EntityID: entityID, State: strings.TrimSpace(reading.State), DisplayName: strings.TrimSpace(reading.DisplayName),
			Unit: strings.TrimSpace(reading.Unit), DeviceClass: strings.ToLower(strings.TrimSpace(reading.DeviceClass)),
			StateClass: strings.ToLower(strings.TrimSpace(reading.StateClass)), LastUpdated: reading.LastUpdated.UTC(),
		})
	}
	if replace {
		if err := a.homeConnectorReadings.Clear(slug); err != nil {
			logWarn("home connector reading reset failed", "error_type", "store")
			return false
		}
	}
	if err := a.homeConnectorReadings.Upsert(slug, readings, now); err != nil {
		logWarn("home connector readings persist failed", "error_type", "store")
		return false
	}
	return true
}

func validHomeConnectorReading(reading homeconnector.Reading, now time.Time) bool {
	entityID := strings.ToLower(strings.TrimSpace(reading.EntityID))
	state := strings.TrimSpace(reading.State)
	unit := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(reading.Unit), " ", ""))
	deviceClass := strings.ToLower(strings.TrimSpace(reading.DeviceClass))
	if len(entityID) > 180 || len(state) == 0 || len(state) > 48 ||
		len(reading.DisplayName) > 160 || len(reading.StateClass) > 32 || reading.LastUpdated.IsZero() ||
		reading.LastUpdated.After(now.Add(5*time.Minute)) {
		return false
	}
	if homeconnector.IsVehicleSleepReading(entityID, state, reading.DisplayName) {
		return true
	}
	if !strings.HasPrefix(entityID, "sensor.") {
		return false
	}
	allowedUnit := unit == "w" || unit == "kw" || unit == "mw" || unit == "wh" || unit == "kwh" || unit == "mwh" || unit == "%"
	value, parseErr := strconv.ParseFloat(strings.ReplaceAll(state, ",", "."), 64)
	allowedState := (parseErr == nil && !math.IsNaN(value) && !math.IsInf(value, 0)) || strings.EqualFold(state, "unknown") || strings.EqualFold(state, "unavailable")
	return allowedUnit && allowedState && (deviceClass == "power" || deviceClass == "energy" || deviceClass == "battery")
}

func validHomeConnectorReadingSet(readings []homeconnector.Reading, now time.Time) bool {
	if len(readings) > homeconnector.MaxReadings {
		return false
	}
	seen := map[string]bool{}
	for _, reading := range readings {
		entityID := strings.ToLower(strings.TrimSpace(reading.EntityID))
		if seen[entityID] || !validHomeConnectorReading(reading, now) {
			return false
		}
		seen[entityID] = true
	}
	return true
}

func (a *app) selectedHomeConnectorEntities(slug string) []string {
	if a.energyStore == nil {
		return nil
	}
	mappings, err := a.energyStore.ListMappings(slug)
	if err != nil {
		return nil
	}
	selected := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		entityID := strings.ToLower(strings.TrimSpace(mapping.EntityID))
		if mapping.Confirmed && (strings.HasPrefix(entityID, "sensor.") || strings.HasPrefix(entityID, "binary_sensor.")) {
			selected = append(selected, entityID)
		}
	}
	sort.Strings(selected)
	if len(selected) > homeconnector.MaxReadings {
		selected = selected[:homeconnector.MaxReadings]
	}
	return selected
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
