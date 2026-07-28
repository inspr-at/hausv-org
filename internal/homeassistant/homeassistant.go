// Package homeassistant talks to a Home Assistant instance: entity state,
// history and long-term statistics. Used only by the parking module.
//
// Display formatting deliberately stays OUT of here — that is a view concern.
package homeassistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/markus-barta/hausv-org/internal/store"
)

type Config struct {
	baseURL           string
	token             string
	meterEnergyEntity string
	powerEntity       string
	priceEntity       string
	plugSwitchEntity  string
	batterySocEntity  string
	gridFeedInEntity  string
}

type EntityState struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
	LastChanged time.Time      `json:"last_changed"`
	LastUpdated time.Time      `json:"last_updated"`
}

type HistoryState struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	LastChanged time.Time      `json:"last_changed"`
	LastUpdated time.Time      `json:"last_updated"`
	Attributes  map[string]any `json:"attributes"`
}

type Statistic struct {
	Start json.RawMessage `json:"start"`
	End   json.RawMessage `json:"end"`
	State *float64        `json:"state"`
	Sum   *float64        `json:"sum"`
	Mean  *float64        `json:"mean"`
	Min   *float64        `json:"min"`
	Max   *float64        `json:"max"`
}

func SamplesFromHistory(history []HistoryState) []store.ParkingNumericSample {
	out := make([]store.ParkingNumericSample, 0, len(history))
	for _, item := range history {
		value, err := ParseFloat(item.State)
		if err != nil {
			continue
		}
		at := item.timestamp()
		if at.IsZero() {
			continue
		}
		out = append(out, store.ParkingNumericSample{At: at.UTC(), Value: value})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	return out
}

func SamplesFromStatistics(stats []Statistic, fields ...string) []store.ParkingNumericSample {
	out := make([]store.ParkingNumericSample, 0, len(stats))
	for _, stat := range stats {
		at, err := ParseStatisticTime(stat.Start)
		if err != nil || at.IsZero() {
			continue
		}
		value, ok := StatisticValue(stat, fields...)
		if !ok {
			continue
		}
		out = append(out, store.ParkingNumericSample{At: at.UTC(), Value: value})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].At.Before(out[j].At)
	})
	return out
}

func StatisticValue(stat Statistic, fields ...string) (float64, bool) {
	for _, field := range fields {
		switch field {
		case "state":
			if stat.State != nil {
				return *stat.State, true
			}
		case "sum":
			if stat.Sum != nil {
				return *stat.Sum, true
			}
		case "mean":
			if stat.Mean != nil {
				return *stat.Mean, true
			}
		case "min":
			if stat.Min != nil {
				return *stat.Min, true
			}
		case "max":
			if stat.Max != nil {
				return *stat.Max, true
			}
		}
	}
	return 0, false
}

func ParseStatisticTime(raw json.RawMessage) (time.Time, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return time.Time{}, errors.New("missing statistic time")
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return time.Time{}, err
		}
		return time.Parse(time.RFC3339, value)
	}
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return time.Time{}, err
	}
	if value > 100000000000 {
		return time.UnixMilli(int64(value)), nil
	}
	return time.Unix(int64(value), 0), nil
}

// NewConfig builds the client config from explicit values. Reading os.Getenv in
// here would tie the package to the process environment and make it untestable;
// env parsing belongs in the composition root.
func NewConfig(baseURL, token, meterEnergyEntity, powerEntity, priceEntity string) Config {
	return Config{
		baseURL:           strings.TrimRight(baseURL, "/"),
		token:             strings.TrimSpace(token),
		meterEnergyEntity: meterEnergyEntity,
		powerEntity:       powerEntity,
		priceEntity:       priceEntity,
	}
}

// WithChargingEntities returns a copy that also knows the charging-control
// entities. Kept out of NewConfig so existing call sites stay untouched.
func (c Config) WithChargingEntities(plugSwitch, batterySoc, gridFeedIn string) Config {
	c.plugSwitchEntity = strings.TrimSpace(plugSwitch)
	c.batterySocEntity = strings.TrimSpace(batterySoc)
	c.gridFeedInEntity = strings.TrimSpace(gridFeedIn)
	return c
}

// CallService posts a service call for one entity, e.g. switch/turn_on.
// The response body is ignored on success: Home Assistant returns the list of
// changed states, but the controller confirms by reading the entity back.
func (c Config) CallService(ctx context.Context, domain, service, entityID string) error {
	if c.baseURL == "" || c.token == "" || domain == "" || service == "" || entityID == "" {
		return errors.New("home assistant not configured")
	}
	body := strings.NewReader(`{"entity_id":` + strconv.Quote(entityID) + `}`)
	endpoint := c.baseURL + "/api/services/" + url.PathEscape(domain) + "/" + url.PathEscape(service)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return errors.New("could not build home assistant service request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New("home assistant service request failed")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("home assistant returned %d", resp.StatusCode)
	}
	return nil
}

// SetSwitch turns the configured plug switch (or any switch entity) on or off.
func (c Config) SetSwitch(ctx context.Context, entityID string, on bool) error {
	service := "turn_off"
	if on {
		service = "turn_on"
	}
	return c.CallService(ctx, "switch", service, entityID)
}

func (c Config) State(ctx context.Context, entityID string) (EntityState, error) {
	if c.baseURL == "" || c.token == "" || entityID == "" {
		return EntityState{}, errors.New("home assistant not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/states/"+url.PathEscape(entityID), nil)
	if err != nil {
		return EntityState{}, errors.New("could not build home assistant request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return EntityState{}, errors.New("home assistant request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return EntityState{}, fmt.Errorf("home assistant returned %d", resp.StatusCode)
	}

	var state EntityState
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := dec.Decode(&state); err != nil {
		return EntityState{}, errors.New("home assistant returned invalid json")
	}
	return state, nil
}

// States returns the current entity catalogue. It is deliberately read-only
// and bounded: discovery may suggest measurement entities, but never exposes a
// service-call capability to the onboarding flow.
func (c Config) States(ctx context.Context) ([]EntityState, error) {
	if c.baseURL == "" || c.token == "" {
		return nil, errors.New("home assistant not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/states", nil)
	if err != nil {
		return nil, errors.New("could not build home assistant request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.New("home assistant request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("home assistant returned %d", resp.StatusCode)
	}

	var states []EntityState
	dec := json.NewDecoder(io.LimitReader(resp.Body, 16<<20))
	if err := dec.Decode(&states); err != nil {
		return nil, errors.New("home assistant returned invalid json")
	}
	if len(states) > 50000 {
		return nil, errors.New("home assistant returned too many entities")
	}
	sort.Slice(states, func(i, j int) bool { return states[i].EntityID < states[j].EntityID })
	return states, nil
}

func (c Config) History(ctx context.Context, start time.Time, end time.Time, entityIDs []string) (map[string][]HistoryState, error) {
	if c.baseURL == "" || c.token == "" || len(entityIDs) == 0 {
		return nil, errors.New("home assistant not configured")
	}
	endpoint := c.baseURL + "/api/history/period/" + url.PathEscape(start.UTC().Format(time.RFC3339))
	q := url.Values{}
	q.Set("end_time", end.UTC().Format(time.RFC3339))
	q.Set("filter_entity_id", strings.Join(entityIDs, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, errors.New("could not build home assistant history request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.New("home assistant history request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("home assistant history returned %d", resp.StatusCode)
	}

	var groups [][]HistoryState
	dec := json.NewDecoder(io.LimitReader(resp.Body, 8<<20))
	if err := dec.Decode(&groups); err != nil {
		return nil, errors.New("home assistant history returned invalid json")
	}
	out := map[string][]HistoryState{}
	for _, group := range groups {
		for _, item := range group {
			if item.EntityID == "" {
				continue
			}
			out[item.EntityID] = append(out[item.EntityID], item)
		}
	}
	for entityID := range out {
		sort.Slice(out[entityID], func(i, j int) bool {
			return out[entityID][i].timestamp().Before(out[entityID][j].timestamp())
		})
	}
	return out, nil
}

func (c Config) Statistics(ctx context.Context, start time.Time, end time.Time) ([]store.ParkingNumericSample, []store.ParkingNumericSample, error) {
	if c.baseURL == "" || c.token == "" || c.meterEnergyEntity == "" || c.priceEntity == "" {
		return nil, nil, errors.New("home assistant not configured")
	}
	wsURL, err := c.websocketURL()
	if err != nil {
		return nil, nil, err
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, nil, errors.New("home assistant websocket connection failed")
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
		_ = conn.SetWriteDeadline(deadline)
	}

	var authMessage struct {
		Type string `json:"type"`
	}
	if err := conn.ReadJSON(&authMessage); err != nil {
		return nil, nil, errors.New("home assistant websocket auth start failed")
	}
	if authMessage.Type == "auth_required" {
		if err := conn.WriteJSON(map[string]string{
			"type":         "auth",
			"access_token": c.token,
		}); err != nil {
			return nil, nil, errors.New("home assistant websocket auth failed")
		}
		if err := conn.ReadJSON(&authMessage); err != nil {
			return nil, nil, errors.New("home assistant websocket auth response failed")
		}
	}
	if authMessage.Type != "auth_ok" {
		return nil, nil, errors.New("home assistant websocket auth rejected")
	}

	const requestID = 1
	if err := conn.WriteJSON(map[string]any{
		"id":            requestID,
		"type":          "recorder/statistics_during_period",
		"start_time":    start.UTC().Format(time.RFC3339),
		"end_time":      end.UTC().Format(time.RFC3339),
		"statistic_ids": []string{c.meterEnergyEntity, c.priceEntity},
		"period":        "hour",
		"types":         []string{"state", "sum", "mean"},
	}); err != nil {
		return nil, nil, errors.New("home assistant websocket statistics request failed")
	}

	for {
		var response struct {
			ID      int                    `json:"id"`
			Type    string                 `json:"type"`
			Success bool                   `json:"success"`
			Error   map[string]any         `json:"error"`
			Result  map[string][]Statistic `json:"result"`
		}
		if err := conn.ReadJSON(&response); err != nil {
			return nil, nil, errors.New("home assistant websocket statistics response failed")
		}
		if response.ID != requestID {
			continue
		}
		if response.Type != "result" || !response.Success {
			return nil, nil, errors.New("home assistant websocket statistics rejected")
		}
		energySamples := SamplesFromStatistics(response.Result[c.meterEnergyEntity], "state", "sum")
		priceSamples := SamplesFromStatistics(response.Result[c.priceEntity], "state", "mean")
		return energySamples, priceSamples, nil
	}
}

func (c Config) websocketURL() (string, error) {
	parsed, err := url.Parse(c.baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid home assistant base url")
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	default:
		return "", errors.New("unsupported home assistant websocket scheme")
	}
	parsed.Path = "/api/websocket"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func (s HistoryState) timestamp() time.Time {
	if !s.LastChanged.IsZero() {
		return s.LastChanged
	}
	return s.LastUpdated
}

func ParseFloat(raw string) (float64, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if value == "" || strings.EqualFold(value, "unknown") || strings.EqualFold(value, "unavailable") {
		return 0, errors.New("state is not numeric")
	}
	return strconv.ParseFloat(value, 64)
}

// Accessors. The fields stay unexported: main read c.baseURL directly, which
// only worked while they shared a package, and exporting `baseURL` would collide
// with app.baseURL.

// Configured reports whether Home Assistant is reachable. Semantics preserved
// verbatim from main: BOTH the URL and the token must be set.
func (c Config) Configured() bool { return c.baseURL != "" && c.token != "" }

func (c Config) BaseURL() string           { return c.baseURL }
func (c Config) Token() string             { return c.token }
func (c Config) MeterEnergyEntity() string { return c.meterEnergyEntity }
func (c Config) PowerEntity() string       { return c.powerEntity }
func (c Config) PriceEntity() string       { return c.priceEntity }
func (c Config) PlugSwitchEntity() string  { return c.plugSwitchEntity }
func (c Config) BatterySocEntity() string  { return c.batterySocEntity }
func (c Config) GridFeedInEntity() string  { return c.gridFeedInEntity }

// ChargingConfigured reports whether the charging controller has everything it
// needs: a reachable instance plus all three control entities.
func (c Config) ChargingConfigured() bool {
	return c.Configured() && c.plugSwitchEntity != "" && c.batterySocEntity != "" && c.gridFeedInEntity != ""
}
