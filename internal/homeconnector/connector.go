// Package homeconnector runs the local, outbound-only HAUSV Home connector.
// Home Assistant credentials never leave this process: the portal receives
// only a compact reachability heartbeat.
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
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/version"
)

const (
	MaxEntityCount        = 50000
	DefaultInterval       = 30 * time.Second
	minimumInterval       = 15 * time.Second
	maxConnectorResponse  = 4 << 10
	maxHomeAssistantReply = 16 << 20
)

type Heartbeat struct {
	ConnectorVersion     string `json:"connector_version"`
	HomeAssistantVersion string `json:"home_assistant_version"`
	EntityCount          int    `json:"entity_count"`
}

type PairRequest struct {
	PairingCode string `json:"pairing_code"`
	Heartbeat
}

type PairResponse struct {
	Credential string `json:"credential"`
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
	if strings.TrimSpace(options.PairingCode) != "" {
		credential, err = pair(ctx, client, portalURL, options.PairingCode, heartbeat)
		if err != nil {
			return err
		}
		if err := writeCredential(options.CredentialFile, credential); err != nil {
			return err
		}
		found = true
	}
	if !found {
		return errors.New("Kopplungscode fehlt; erstellen Sie im Portal eine neue Kopplung")
	}
	if err := sendHeartbeat(ctx, client, portalURL, credential, heartbeat); err != nil {
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
			if err := sendHeartbeat(ctx, client, portalURL, credential, heartbeat); err != nil {
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
	var states []json.RawMessage
	if err := json.Unmarshal(statesBody, &states); err != nil || len(states) > MaxEntityCount {
		return Heartbeat{}, errors.New("lokale Energiedaten haben einen unerwarteten Umfang")
	}
	return Heartbeat{
		ConnectorVersion:     version.Version,
		HomeAssistantVersion: strings.TrimSpace(config.Version),
		EntityCount:          len(states),
	}, nil
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

func pair(ctx context.Context, client *http.Client, portalURL, pairingCode string, heartbeat Heartbeat) (string, error) {
	requestBody, _ := json.Marshal(PairRequest{PairingCode: strings.TrimSpace(pairingCode), Heartbeat: heartbeat})
	response, err := portalPOST(ctx, client, portalURL+"/api/home-connectors/pair", "", requestBody)
	if err != nil {
		return "", err
	}
	var result PairResponse
	if err := json.Unmarshal(response, &result); err != nil || !validSecret(result.Credential) {
		return "", errors.New("das Portal hat die Kopplung nicht bestätigt")
	}
	return result.Credential, nil
}

func sendHeartbeat(ctx context.Context, client *http.Client, portalURL, credential string, heartbeat Heartbeat) error {
	requestBody, _ := json.Marshal(heartbeat)
	_, err := portalPOST(ctx, client, portalURL+"/api/home-connectors/heartbeat", credential, requestBody)
	return err
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
