package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
	"github.com/inspr-at/hausv-org/internal/dbtest"
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

// Hold one writer after its read, then let an independent store contend for
// the same document. The old implementation lets both callbacks read the old
// value; the second commit then erases the first writer's payment.
func TestSQLParkingConcurrentUpdates(t *testing.T) {
	if dbtest.Backend() != appdb.BackendPostgres {
		t.Skip("requires PostgreSQL writer-lock observation")
	}
	for _, existing := range []bool{false, true} {
		name := "first-write"
		if existing {
			name = "existing-row"
		}
		t.Run(name, func(t *testing.T) {
			database, cfg := dbtest.OpenWithConfig(t)
			seedFixtureTenants(t, database)
			first := NewSQLParkingStore(lanesOver(t, database, cfg))
			second := NewSQLParkingStore(lanesOver(t, database, cfg))
			if existing {
				if err := first.SetGridFee("demo", 0.21); err != nil {
					t.Fatal(err)
				}
			}
			firstRead, secondRead := make(chan struct{}), make(chan struct{})
			releaseFirst, releaseSecond := make(chan struct{}), make(chan struct{})
			firstDone, secondDone := make(chan error, 1), make(chan error, 1)
			// Deferred releases also unblock workers if an assertion fails.
			defer func() {
				select {
				case <-releaseFirst:
				default:
					close(releaseFirst)
				}
				select {
				case <-releaseSecond:
				default:
					close(releaseSecond)
				}
			}()
			go func() {
				firstDone <- first.mutate("demo", func(mem *ParkingStore) error {
					close(firstRead)
					<-releaseFirst
					if err := mem.SetMonthPaid("demo", "2026-09", true); err != nil {
						return err
					}
					return mem.SetChargingState("demo", ChargingControllerState{Phase: ChargingPhaseManualOn})
				})
			}()
			select {
			case <-firstRead:
			case err := <-firstDone:
				t.Fatalf("first writer: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("first writer did not read")
			}
			sample := ParkingNumericSample{At: time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC), Value: 123}
			go func() {
				secondDone <- second.mutate("demo", func(mem *ParkingStore) error {
					close(secondRead)
					<-releaseSecond
					return mem.AppendReadings("demo", []ParkingNumericSample{sample}, nil)
				})
			}()
			// Wait for either an actual database lock wait (fixed code) or the
			// second read (old code). No scheduling delay is treated as proof.
			deadline := time.Now().Add(5 * time.Second)
			for {
				select {
				case <-secondRead:
					goto contended
				case err := <-secondDone:
					t.Fatalf("second writer: %v", err)
				default:
				}
				var waiting bool
				err := database.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_stat_activity
					WHERE datname=current_database() AND usename=current_user
					AND wait_event_type='Lock' AND query LIKE '%parking%'
					AND cardinality(pg_blocking_pids(pid)) > 0)`).Scan(&waiting)
				if err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("second writer neither read nor waited for a lock")
				}
				time.Sleep(10 * time.Millisecond)
			}
		contended:
			// Another tenant must remain writable while demo is locked.
			if err := second.SetMonthPaid("other", "2026-08", true); err != nil {
				t.Fatal(err)
			}
			close(releaseFirst)
			if err := <-firstDone; err != nil {
				t.Fatal(err)
			}
			close(releaseSecond)
			if err := <-secondDone; err != nil {
				t.Fatal(err)
			}
			got := first.TenantData("demo")
			if !got.Months["2026-09"].Paid {
				t.Error("concurrent reading update erased payment")
			}
			if got.Charging.Phase != ChargingPhaseManualOn {
				t.Error("concurrent reading update erased charging state")
			}
			if len(got.EnergySamples) != 1 || got.EnergySamples[0] != sample {
				t.Errorf("reading lost: %+v", got.EnergySamples)
			}
			if got.Months["2026-08"].Paid {
				t.Error("other tenant payment leaked into demo")
			}
			other := second.TenantData("other")
			if !other.Months["2026-08"].Paid || other.Months["2026-09"].Paid || len(other.EnergySamples) != 0 {
				t.Error("other tenant changed unexpectedly")
			}
		})
	}
}

func TestSQLParkingMutationRollback(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "first-write"
		if existing {
			name = "existing-row"
		}
		t.Run(name, func(t *testing.T) {
			database, lanes := testLanes(t)
			s := NewSQLParkingStore(lanes)
			if existing {
				if err := s.SetMonthPaid("demo", "2026-08", true); err != nil {
					t.Fatal(err)
				}
			}
			failed := errors.New("abort mutation")
			err := s.mutate("demo", func(mem *ParkingStore) error {
				if err := mem.SetMonthPaid("demo", "2026-09", true); err != nil {
					return err
				}
				return failed
			})
			if !errors.Is(err, failed) {
				t.Fatalf("mutation error = %v", err)
			}
			var count int
			if err := database.QueryRow(`SELECT count(*) FROM parking WHERE tenant_id=$1`, testTenantID("demo")).Scan(&count); err != nil {
				t.Fatal(err)
			}
			want := 0
			if existing {
				want = 1
			}
			if count != want {
				t.Fatalf("rollback left %d rows, want %d", count, want)
			}
			got := s.TenantData("demo")
			if got.Months["2026-09"].Paid || got.Months["2026-08"].Paid != existing {
				t.Fatal("failed mutation changed payment state")
			}
			if err := s.SetMonthPaid("demo", "2026-10", true); err != nil {
				t.Fatalf("writer lock not released: %v", err)
			}
		})
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
