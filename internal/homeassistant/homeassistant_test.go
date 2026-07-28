package homeassistant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
