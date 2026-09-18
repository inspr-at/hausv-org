package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSQLParkingStoreRoundTrip(t *testing.T) {
	sqlStore := NewSQLParkingStore(testStoreDB(t))
	if err := sqlStore.SetGridFee("demo", 0.21); err != nil {
		t.Fatalf("set grid fee: %v", err)
	}
	if got := sqlStore.TenantData("demo").Settings.GridFeeEURPerKWh; got != 0.21 {
		t.Fatalf("grid fee = %v, want 0.21", got)
	}
	if err := sqlStore.UpsertTariff("demo", ParkingTariff{EffectiveFrom: "2026-01-01", GridFeeEURPerKWh: 0.19, BaseFeeEUR: 12}); err != nil {
		t.Fatalf("upsert tariff: %v", err)
	}
	if err := sqlStore.SetMonthPaid("demo", "2026-03", true); err != nil {
		t.Fatalf("set month paid: %v", err)
	}
	session, err := sqlStore.StartChargingSession("demo", ChargingSession{
		Start: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), StartKWh: 10, Mode: ChargingModeManual, TriggerSource: ChargingTriggerWeb,
	}, ChargingControllerState{Phase: ChargingPhaseManualOn})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if session.ID == "" {
		t.Fatal("session id missing")
	}
	got := sqlStore.TenantData("demo")
	if !got.Months["2026-03"].Paid {
		t.Fatal("paid month not persisted")
	}
	if len(got.ChargingSessions) != 1 || got.ChargingSessions[0].ID != session.ID {
		t.Fatalf("sessions = %+v", got.ChargingSessions)
	}
}

func TestImportParkingIsNoopWhenSQLAlreadyHasRows(t *testing.T) {
	sqlStore := NewSQLParkingStore(testStoreDB(t))
	jsonStore, err := NewParkingStore(filepath.Join(t.TempDir(), "parking.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := jsonStore.SetGridFee("demo", 0.33); err != nil {
		t.Fatal(err)
	}
	if err := sqlStore.ImportParking(jsonStore); err != nil {
		t.Fatalf("first import: %v", err)
	}
	if err := jsonStore.SetGridFee("demo", 0.01); err != nil {
		t.Fatal(err)
	}
	if err := sqlStore.ImportParking(jsonStore); err != nil {
		t.Fatalf("second import: %v", err)
	}
	if got := sqlStore.TenantData("demo").Settings.GridFeeEURPerKWh; got != 0.33 {
		t.Fatalf("import overwrote live SQL: got %v", got)
	}
}
