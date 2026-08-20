// Package homeconnector runs the local, outbound-only HAUSV Home connector.
// Home Assistant credentials never leave this process: the portal receives
// only a bounded energy catalog and then the read-only sensors selected by the
// portal.
package homeconnector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/version"
)

const (
	MaxEntityCount        = 50000
	MaxReadings           = 64
	DefaultInterval       = 30 * time.Second
	minimumInterval       = 15 * time.Second
	maxConnectorResponse  = 4 << 10
	maxHomeAssistantReply = 16 << 20
)

type Heartbeat struct {
	ConnectorVersion     string    `json:"connector_version"`
	HomeAssistantVersion string    `json:"home_assistant_version"`
	EntityCount          int       `json:"entity_count"`
	Readings             []Reading `json:"readings,omitempty"`
}

// Reading contains only the small energy-sensor subset allowed to leave the
// local network. Arbitrary Home Assistant attributes are intentionally absent.
type Reading struct {
	EntityID    string    `json:"entity_id"`
	State       string    `json:"state"`
	DisplayName string    `json:"display_name,omitempty"`
	Unit        string    `json:"unit"`
	DeviceClass string    `json:"device_class"`
	StateClass  string    `json:"state_class,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

type PairRequest struct {
	PairingCode string `json:"pairing_code"`
	Heartbeat
}

type PairResponse struct {
	Credential        string   `json:"credential"`
	SelectedEntityIDs []string `json:"selected_entity_ids,omitempty"`
}

type HeartbeatResponse struct {
	SelectedEntityIDs []string `json:"selected_entity_ids,omitempty"`
}

type Options struct {
	PortalURL      string
	PairingCode    string
	HAURL          string
	HATokenFile    string
	CredentialFile string
	Interval       time.Duration
	Once           bool
	HTTPClient     *http.Client
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("connector", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var options Options
	flags.StringVar(&options.PortalURL, "portal", "https://hausv.org", "HAUSV portal origin")
	flags.StringVar(&options.PairingCode, "pairing-code", "", "one-time code from the HAUSV setup page")
	flags.StringVar(&options.HAURL, "home-assistant-url", "", "local Home Assistant URL")
	flags.StringVar(&options.HATokenFile, "home-assistant-token-file", "", "local file containing the Home Assistant token")
	flags.StringVar(&options.CredentialFile, "credential-file", "hausv-connector.credential", "local connector credential file")
	flags.DurationVar(&options.Interval, "interval", DefaultInterval, "heartbeat interval")
	flags.BoolVar(&options.Once, "once", false, "pair and send one heartbeat, then exit")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unerwartete Argumente")
	}
	if err := RunWithOptions(ctx, options); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(stdout, "HAUSV Connector ist im Lesemodus verbunden.")
	return nil
}

func RunWithOptions(ctx context.Context, options Options) error {
	portalURL, err := normalizePortalURL(options.PortalURL)
	if err != nil {
		return err
	}
	haURL, err := normalizeHomeAssistantURL(options.HAURL)
	if err != nil {
		return err
	}
	if strings.TrimSpace(options.HATokenFile) == "" || strings.TrimSpace(options.CredentialFile) == "" {
		return errors.New("Token-Datei und lokale Connector-Datei sind erforderlich")
	}
	if options.Interval == 0 {
		options.Interval = DefaultInterval
	}
	if !options.Once && options.Interval < minimumInterval {
		return fmt.Errorf("das Intervall muss mindestens %s betragen", minimumInterval)
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	heartbeat, err := readHomeAssistant(ctx, client, haURL, options.HATokenFile)
	if err != nil {
		return err
	}
	credential, found, err := readCredential(options.CredentialFile)
	if err != nil {
		return err
	}
	selectedEntityIDs := []string{}
	if strings.TrimSpace(options.PairingCode) != "" {
		var paired PairResponse
		paired, err = pair(ctx, client, portalURL, options.PairingCode, heartbeat)
		if err != nil {
			return err
		}
		credential = paired.Credential
		selectedEntityIDs = paired.SelectedEntityIDs
		if err := writeCredential(options.CredentialFile, credential); err != nil {
			return err
		}
		found = true
	}
	if !found {
		return errors.New("Kopplungscode fehlt; erstellen Sie im Portal eine neue Kopplung")
	}
	selectedEntityIDs, err = sendHeartbeat(ctx, client, portalURL, credential, selectHeartbeatReadings(heartbeat, selectedEntityIDs))
	if err != nil {
		return err
	}
	if options.Once {
		return nil
	}

	ticker := time.NewTicker(options.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			heartbeat, err = readHomeAssistant(ctx, client, haURL, options.HATokenFile)
			if err != nil {
				return err
			}
			selectedEntityIDs, err = sendHeartbeat(ctx, client, portalURL, credential, selectHeartbeatReadings(heartbeat, selectedEntityIDs))
			if err != nil {
				return err
			}
		}
	}
}

func readHomeAssistant(ctx context.Context, client *http.Client, baseURL, tokenFile string) (Heartbeat, error) {
	token, err := readSecretFile(tokenFile, "Home-Assistant-Token")
	if err != nil {
		return Heartbeat{}, err
	}
	configBody, err := homeAssistantGET(ctx, client, baseURL+"/api/config", token, 1<<20)
	if err != nil {
		return Heartbeat{}, err
	}
	var config struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(configBody, &config); err != nil || !safeMetadata(config.Version) {
		return Heartbeat{}, errors.New("lokale Energiedaten haben ein unerwartetes Format")
	}
	statesBody, err := homeAssistantGET(ctx, client, baseURL+"/api/states", token, maxHomeAssistantReply)
	if err != nil {
		return Heartbeat{}, err
	}
	var states []struct {
		EntityID   string         `json:"entity_id"`
		State      string         `json:"state"`
		Attributes map[string]any `json:"attributes"`
		Updated    time.Time      `json:"last_updated"`
		Changed    time.Time      `json:"last_changed"`
	}
	if err := json.Unmarshal(statesBody, &states); err != nil || len(states) > MaxEntityCount {
		return Heartbeat{}, errors.New("lokale Energiedaten haben einen unerwarteten Umfang")
	}
	readings := make([]Reading, 0, MaxReadings)
	for _, state := range states {
		unit := connectorAttribute(state.Attributes, "unit_of_measurement")
		deviceClass := strings.ToLower(connectorAttribute(state.Attributes, "device_class"))
		displayName := connectorAttribute(state.Attributes, "friendly_name")
		if !allowedEnergyReading(state.EntityID, state.State, displayName, unit, deviceClass) {
			continue
		}
		updated := state.Updated
		if updated.IsZero() {
			updated = state.Changed
		}
		if updated.IsZero() {
			// Freshness is part of the value contract. Omitting a reading is
			// safer than inventing an update time that would make old data look
			// current in the portal.
			continue
		}
		readings = append(readings, Reading{
			EntityID: strings.ToLower(strings.TrimSpace(state.EntityID)), State: strings.TrimSpace(state.State),
			DisplayName: displayName, Unit: strings.TrimSpace(unit),
			DeviceClass: deviceClass, StateClass: strings.ToLower(connectorAttribute(state.Attributes, "state_class")),
			LastUpdated: updated.UTC(),
		})
	}
	sort.Slice(readings, func(i, j int) bool { return readings[i].EntityID < readings[j].EntityID })
	if len(readings) > MaxReadings {
		readings = readings[:MaxReadings]
	}
	return Heartbeat{
		ConnectorVersion:     version.Version,
		HomeAssistantVersion: strings.TrimSpace(config.Version),
		EntityCount:          len(states),
		Readings:             readings,
	}, nil
}

func connectorAttribute(attributes map[string]any, key string) string {
	value, _ := attributes[key].(string)
	return strings.TrimSpace(value)
}

func allowedEnergyReading(entityID, state, displayName, unit, deviceClass string) bool {
	entityID = strings.ToLower(strings.TrimSpace(entityID))
	if len(entityID) > 180 || len(strings.TrimSpace(state)) > 48 {
		return false
	}
	if IsVehicleSleepReading(entityID, state, displayName) {
		return true
	}
	if !strings.HasPrefix(entityID, "sensor.") {
		return false
	}
	normalizedUnit := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(unit), " ", ""))
	allowedUnit := normalizedUnit == "w" || normalizedUnit == "kw" || normalizedUnit == "mw" ||
		normalizedUnit == "wh" || normalizedUnit == "kwh" || normalizedUnit == "mwh" || normalizedUnit == "%"
	deviceClass = strings.ToLower(strings.TrimSpace(deviceClass))
	return allowedUnit && (deviceClass == "power" || deviceClass == "energy" || deviceClass == "battery")
}

// IsVehicleSleepReading recognizes only an explicit vehicle sleep signal.
// Requiring both the sleep marker and a vehicle marker keeps unrelated home
// state (for example bedroom or presence sensors) outside the connector's
// deliberately narrow disclosure boundary.
func IsVehicleSleepReading(entityID, state, displayName string) bool {
	entityID = strings.ToLower(strings.TrimSpace(entityID))
	if !strings.HasPrefix(entityID, "sensor.") && !strings.HasPrefix(entityID, "binary_sensor.") {
		return false
	}
	name := strings.ToLower(entityID + " " + strings.TrimSpace(displayName))
	if !strings.Contains(name, "sleep") && !strings.Contains(name, "asleep") &&
		!strings.Contains(name, "schlaf") && !strings.Contains(name, "schläf") {
		return false
	}
	normalizedName := strings.NewReplacer("_", " ", "-", " ", ".", " ", "/", " ").Replace(name)
	vehicleNamed := strings.Contains(normalizedName, "model x") || strings.Contains(normalizedName, "model y")
	for _, field := range strings.Fields(normalizedName) {
		switch field {
		case "vehicle", "fahrzeug", "auto", "car", "ev", "tesla", "enyaq", "etron",
			"ioniq", "kona", "leaf", "zoe", "taycan":
			vehicleNamed = true
		}
	}
	if !vehicleNamed {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "on", "off", "true", "false", "1", "0", "sleep", "sleeping", "asleep",
		"schläft", "awake", "online", "unknown", "unavailable":
		return true
	default:
		return false
	}
}

func selectHeartbeatReadings(heartbeat Heartbeat, selected []string) Heartbeat {
	if len(selected) == 0 {
		return heartbeat
	}
	wanted := make(map[string]bool, len(selected))
	for _, entityID := range selected {
		wanted[strings.ToLower(strings.TrimSpace(entityID))] = true
	}
	filtered := make([]Reading, 0, len(selected))
	for _, reading := range heartbeat.Readings {
		if wanted[reading.EntityID] || IsVehicleSleepReading(reading.EntityID, reading.State, reading.DisplayName) {
			filtered = append(filtered, reading)
		}
	}
	heartbeat.Readings = filtered
	return heartbeat
}

func homeAssistantGET(ctx context.Context, client *http.Client, endpoint, token string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("lokale Verbindung konnte nicht vorbereitet werden")
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "hausv-connector/"+version.Version)
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("lokale Energiedaten sind nicht erreichbar")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return nil, errors.New("lokale Energiedaten konnten nicht gelesen werden")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || int64(len(body)) > limit {
		return nil, errors.New("lokale Energiedaten haben einen unerwarteten Umfang")
	}
	return body, nil
}

func pair(ctx context.Context, client *http.Client, portalURL, pairingCode string, heartbeat Heartbeat) (PairResponse, error) {
	requestBody, _ := json.Marshal(PairRequest{PairingCode: strings.TrimSpace(pairingCode), Heartbeat: heartbeat})
	response, err := portalPOST(ctx, client, portalURL+"/api/home-connectors/pair", "", requestBody)
	if err != nil {
		return PairResponse{}, err
	}
	var result PairResponse
	if err := json.Unmarshal(response, &result); err != nil || !validSecret(result.Credential) {
		return PairResponse{}, errors.New("das Portal hat die Kopplung nicht bestätigt")
	}
	return result, nil
}

func sendHeartbeat(ctx context.Context, client *http.Client, portalURL, credential string, heartbeat Heartbeat) ([]string, error) {
	requestBody, _ := json.Marshal(heartbeat)
	response, err := portalPOST(ctx, client, portalURL+"/api/home-connectors/heartbeat", credential, requestBody)
	if err != nil {
		return nil, err
	}
	// Portal versions before measurement transport answered heartbeats with 204.
	// Accept that response so connector upgrades do not require a lockstep deploy.
	if len(bytes.TrimSpace(response)) == 0 {
		return nil, nil
	}
	var result HeartbeatResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, errors.New("Portalantwort ist ungültig")
	}
	return result.SelectedEntityIDs, nil
}

func portalPOST(ctx context.Context, client *http.Client, endpoint, credential string, body []byte) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("Portalverbindung konnte nicht vorbereitet werden")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "hausv-connector/"+version.Version)
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("Portal ist gerade nicht erreichbar")
	}
	defer response.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, maxConnectorResponse+1))
	if readErr != nil || len(responseBody) > maxConnectorResponse {
		return nil, errors.New("Portalantwort ist ungültig")
	}
	if response.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("Verbindung wurde widerrufen; starten Sie die Kopplung im Portal erneut")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, errors.New("Portal hat die Verbindung nicht angenommen")
	}
	return responseBody, nil
}

func readCredential(path string) (string, bool, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, errors.New("lokale Connector-Datei ist nicht lesbar")
	}
	info, err := os.Stat(path)
	if err != nil || runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return "", false, errors.New("lokale Connector-Datei muss nur für den Eigentümer lesbar sein")
	}
	credential := strings.TrimSpace(string(raw))
	if !validSecret(credential) {
		return "", false, errors.New("lokale Connector-Datei ist ungültig")
	}
	return credential, true, nil
}

func readSecretFile(path, label string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s-Datei ist nicht lesbar", label)
	}
	info, err := os.Stat(path)
	if err != nil || runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("%s-Datei muss nur für den Eigentümer lesbar sein", label)
	}
	secret := strings.TrimSpace(string(raw))
	if secret == "" || len(secret) > 4096 || strings.ContainsAny(secret, "\r\n") {
		return "", fmt.Errorf("%s-Datei ist ungültig", label)
	}
	return secret, nil
}

func writeCredential(path, credential string) error {
	if !validSecret(credential) {
		return errors.New("Connector-Zugang ist ungültig")
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return errors.New("lokaler Connector-Ordner konnte nicht erstellt werden")
	}
	temporary, err := os.CreateTemp(directory, ".hausv-connector-*")
	if err != nil {
		return errors.New("lokale Connector-Datei konnte nicht erstellt werden")
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return errors.New("lokale Connector-Datei konnte nicht geschützt werden")
	}
	if _, err := temporary.WriteString(credential + "\n"); err != nil {
		temporary.Close()
		return errors.New("lokaler Connector-Zugang konnte nicht gespeichert werden")
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return errors.New("lokaler Connector-Zugang konnte nicht gespeichert werden")
	}
	if err := temporary.Close(); err != nil {
		return errors.New("lokaler Connector-Zugang konnte nicht gespeichert werden")
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return errors.New("lokaler Connector-Zugang konnte nicht aktiviert werden")
	}
	return nil
}

func normalizePortalURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" && parsed.Path != "/" {
		return "", errors.New("Portaladresse ist ungültig")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && localHostname(parsed.Hostname())) {
		return "", errors.New("Portaladresse muss HTTPS verwenden")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func normalizeHomeAssistantURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" && parsed.Path != "/" {
		return "", errors.New("lokale Adresse ist ungültig")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("lokale Adresse muss HTTP oder HTTPS verwenden")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func localHostname(host string) bool {
	return strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
}

func safeMetadata(value string) bool {
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

func validSecret(value string) bool {
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
