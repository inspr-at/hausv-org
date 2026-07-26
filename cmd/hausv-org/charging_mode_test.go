package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/markus-barta/hausv-org/internal/store"
)

func TestRunChargingModePreservesSettingsAndAudits(t *testing.T) {
	dir := t.TempDir()
	parkingPath := filepath.Join(dir, "parking.json")
	auditPath := filepath.Join(dir, "audit.jsonl")
	t.Setenv("PARKING_DATA_PATH", parkingPath)
	t.Setenv("AUDIT_DATA_PATH", auditPath)

	parking, err := store.NewParkingStore(parkingPath)
	if err != nil {
		t.Fatal(err)
	}
	settings := store.NormalizeChargingControlSettings(store.ChargingControlSettings{
		Enabled:    true,
		ShadowMode: true,
	})
	if err := parking.SetChargingControl("jhw22", settings); err != nil {
		t.Fatal(err)
	}

	if err := runChargingMode([]string{"--tenant", "jhw22", "--mode", "live"}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.NewParkingStore(parkingPath)
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.TenantData("jhw22").Settings.Charging
	if !got.Enabled || got.ShadowMode {
		t.Fatalf("expected live mode, got %+v", got)
	}
	if got.StartFeedInW != settings.StartFeedInW || got.MinOnMinutes != settings.MinOnMinutes {
		t.Fatalf("non-mode settings changed: got %+v want %+v", got, settings)
	}
	if raw, err := os.ReadFile(auditPath); err != nil || len(raw) == 0 {
		t.Fatalf("expected audit event, err=%v", err)
	}
}

func TestRunChargingModeRejectsInvalidMode(t *testing.T) {
	t.Setenv("PARKING_DATA_PATH", filepath.Join(t.TempDir(), "parking.json"))
	if err := runChargingMode([]string{"--tenant", "jhw22", "--mode", "invalid"}); err == nil {
		t.Fatal("expected invalid mode error")
	}
}
