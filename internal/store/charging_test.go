package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizeChargingControlSettingsDefaults(t *testing.T) {
	got := NormalizeChargingControlSettings(ChargingControlSettings{})
	want := ChargingControlSettings{
		StartSocPercent:  99,
		StopSocPercent:   95,
		StartFeedInW:     3300,
		StopFeedInW:      1500,
		StopDelayMinutes: 10,
		MinOnMinutes:     10,
		MinOffMinutes:    5,
	}
	if got != want {
		t.Fatalf("defaults = %+v, want %+v", got, want)
	}
	if got.Enabled || got.ShadowMode {
		t.Fatal("zero value must keep the controller off and shadow unset")
	}
}

func TestNormalizeChargingControlSettingsKeepsHysteresisOrdered(t *testing.T) {
	got := NormalizeChargingControlSettings(ChargingControlSettings{
		StartSocPercent: 90,
		StopSocPercent:  98,
		StartFeedInW:    2000,
		StopFeedInW:     4000,
	})
	if got.StopSocPercent > got.StartSocPercent {
		t.Fatalf("stop SOC %v above start SOC %v", got.StopSocPercent, got.StartSocPercent)
	}
	if got.StopFeedInW > got.StartFeedInW {
		t.Fatalf("stop feed-in %v above start feed-in %v", got.StopFeedInW, got.StartFeedInW)
	}
}

func TestSurplusRateDefaultsWhenUnset(t *testing.T) {
	if rate := SurplusRate(ParkingTariff{}); rate != DefaultSurplusRateEURPerKWh {
		t.Fatalf("rate = %v, want default %v", rate, DefaultSurplusRateEURPerKWh)
	}
	if rate := SurplusRate(ParkingTariff{SurplusRateEURPerKWh: 0.08}); rate != 0.08 {
		t.Fatalf("rate = %v, want 0.08", rate)
	}
}

func TestNormalizeChargingSessionsKeepsOpenDropsStale(t *testing.T) {
	now := time.Now().UTC()
	keepAfter := now.AddDate(-1, -1, 0)
	sessions := []ChargingSession{
		{ID: "old", Start: now.AddDate(-2, 0, 0), End: now.AddDate(-2, 0, 1), Mode: ChargingModeSurplus},
		{ID: "open-ancient", Start: now.AddDate(-2, 0, 0), Mode: ChargingModeSurplus},
		{ID: "fresh", Start: now.Add(-2 * time.Hour), End: now.Add(-1 * time.Hour), Mode: ChargingModeSurplus},
		{ID: "", Start: now},
		{ID: "backwards", Start: now, End: now.Add(-time.Hour)},
	}
	got := NormalizeChargingSessions(sessions, keepAfter)
	ids := map[string]bool{}
	for _, session := range got {
		ids[session.ID] = true
	}
	if !ids["open-ancient"] {
		t.Fatal("open session must never be trimmed")
	}
	if !ids["fresh"] {
		t.Fatal("recent closed session must survive")
	}
	if ids["old"] || ids[""] || ids["backwards"] {
		t.Fatalf("stale/malformed sessions must be dropped, got %v", ids)
	}
}

func TestChargingSessionLifecyclePersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "parking.json")
	s, err := NewParkingStore(path)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now().UTC().Add(-time.Hour)
	session, err := s.StartChargingSession("demo", ChargingSession{
		Start:         start,
		StartKWh:      2550.13,
		Mode:          ChargingModeSurplus,
		TriggerSource: ChargingTriggerAuto,
		StartedBy:     "system",
	}, ChargingControllerState{Phase: ChargingPhaseSurplus, LastSwitchAt: start})
	if err != nil {
		t.Fatal(err)
	}
	if session.ID == "" {
		t.Fatal("session must get an id")
	}

	// Reload from disk: open session and controller state must round-trip.
	reloaded, err := NewParkingStore(path)
	if err != nil {
		t.Fatal(err)
	}
	data := reloaded.TenantData("demo")
	if data.Charging.Phase != ChargingPhaseSurplus || data.Charging.ActiveSessionID != session.ID {
		t.Fatalf("controller state lost: %+v", data.Charging)
	}
	if len(data.ChargingSessions) != 1 || !data.ChargingSessions[0].End.IsZero() {
		t.Fatalf("open session lost: %+v", data.ChargingSessions)
	}

	// Meter going backwards is distrusted: energy clamps to zero, not negative.
	closed, found, err := reloaded.EndChargingSession("demo", session.ID, start.Add(time.Hour), 2549.0, "system", "feedin-low", ChargingControllerState{Phase: ChargingPhaseIdle})
	if err != nil || !found {
		t.Fatalf("end session: found=%v err=%v", found, err)
	}
	if closed.EndKWh != closed.StartKWh {
		t.Fatalf("backwards meter must clamp EndKWh to StartKWh, got %v", closed.EndKWh)
	}
	data = reloaded.TenantData("demo")
	if data.Charging.ActiveSessionID != "" || data.Charging.Phase != ChargingPhaseIdle {
		t.Fatalf("controller state after end: %+v", data.Charging)
	}

	// Ending twice must not find an open session again.
	_, found, err = reloaded.EndChargingSession("demo", session.ID, start.Add(2*time.Hour), 2551, "system", "manual", ChargingControllerState{Phase: ChargingPhaseIdle})
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("closed session must not be closable twice")
	}
}
