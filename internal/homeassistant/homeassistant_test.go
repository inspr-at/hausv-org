package homeassistant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStatesIsReadOnlyAndSorted(t *testing.T) {
	var method, authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"entity_id":"sensor.z_power","state":"12","attributes":{"device_class":"power"}},
			{"entity_id":"sensor.a_energy","state":"4","attributes":{"device_class":"energy"}}
		]`))
	}))
	t.Cleanup(server.Close)

	cfg := NewConfig(server.URL, "test-token", "", "", "")
	states, err := cfg.States(context.Background())
	if err != nil {
		t.Fatalf("States: %v", err)
	}
	if method != http.MethodGet {
		t.Fatalf("method = %q, want GET", method)
	}
	if authorization != "Bearer test-token" {
		t.Fatalf("authorization header missing")
	}
	if len(states) != 2 || states[0].EntityID != "sensor.a_energy" || states[1].EntityID != "sensor.z_power" {
		t.Fatalf("states not sorted: %+v", states)
	}
}

func TestStatesRequiresConfiguration(t *testing.T) {
	if _, err := (Config{}).States(context.Background()); err == nil {
		t.Fatal("expected configuration error")
	}
}

func TestHistoryUsesBoundedResponseAndCarriesGroupEntity(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[[
			{"entity_id":"sensor.house_power","state":"1.2","last_changed":"2026-07-29T08:00:00Z"},
			{"state":"1.4","last_changed":"2026-07-29T08:15:00Z"}
		]]`))
	}))
	t.Cleanup(server.Close)

	cfg := NewConfig(server.URL, "test-token", "", "", "")
	start := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	history, err := cfg.History(context.Background(), start, start.Add(time.Hour), []string{"sensor.house_power"})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if !strings.Contains(query, "minimal_response=") || !strings.Contains(query, "no_attributes=") {
		t.Fatalf("history query is not bounded: %q", query)
	}
	items := history["sensor.house_power"]
	if len(items) != 2 || items[1].EntityID != "sensor.house_power" {
		t.Fatalf("minimal history group lost entity: %+v", items)
	}
}
