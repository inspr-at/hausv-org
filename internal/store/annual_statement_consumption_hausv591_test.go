package store

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	appdb "github.com/inspr-at/hausv-org/internal/db"
)

func annualStatementConsumptionBackends(t *testing.T) map[string]AnnualStatementConsumptionStorage {
	t.Helper()
	jsonStore, err := NewJSONAnnualStatementConsumptionStore(filepath.Join(t.TempDir(), "consumption.json"))
	if err != nil {
		t.Fatalf("json consumption store: %v", err)
	}
	_, lanes := testLanes(t)
	return map[string]AnnualStatementConsumptionStorage{
		"memory": NewMemoryAnnualStatementConsumptionStore(),
		"json":   jsonStore,
		"sql":    NewSQLAnnualStatementConsumptionStore(lanes),
	}
}

func mustViennaLocation(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		t.Fatalf("load Europe/Vienna: %v", err)
	}
	return location
}

func consumptionEvidence(unitID, costType, sourceID string, measuredAt time.Time, valueMicros int64, measurementUnit string) AnnualStatementConsumptionEvidence {
	return AnnualStatementConsumptionEvidence{
		UnitID: unitID, CostTypeKey: costType,
		SourceKind: ConsumptionSourceEntity, SourceID: sourceID,
		MeasuredAt: measuredAt, ValueMicros: valueMicros,
		MeasurementUnit: measurementUnit,
		ReceivedAt:      measuredAt.Add(time.Minute),
	}
}

func appendConsumption(t *testing.T, repository AnnualStatementConsumptionRepository, evidence AnnualStatementConsumptionEvidence) AnnualStatementConsumptionEvidence {
	t.Helper()
	stored, inserted, err := repository.Append(evidence)
	if err != nil || !inserted {
		t.Fatalf("append consumption evidence: inserted=%t evidence=%+v err=%v", inserted, stored, err)
	}
	return stored
}

func TestAnnualStatementConsumptionStorageParityAndPeriodVectors(t *testing.T) {
	location := mustViennaLocation(t)
	period := AnnualStatementPeriod{Year: 2026, StartsOn: "2026-04-01", EndsOn: "2027-03-31"}
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, location)
	endExclusive := time.Date(2027, 4, 1, 0, 0, 0, 0, location)

	for name, storage := range annualStatementConsumptionBackends(t) {
		t.Run(name, func(t *testing.T) {
			demo, ok := BindAnnualStatementConsumptionRepository(storage, testTenantRef("demo"))
			if !ok {
				t.Fatal("bind demo consumption repository")
			}
			other, ok := BindAnnualStatementConsumptionRepository(storage, testTenantRef("other"))
			if !ok {
				t.Fatal("bind other consumption repository")
			}
			if invalid, ok := BindAnnualStatementConsumptionRepository(storage, TenantRef{}); ok || invalid != nil {
				t.Fatal("invalid tenant must not bind")
			}

			// The same unit has two independent cumulative meters. Exact cost-type
			// selection must keep heating and hot water in separate vectors.
			appendConsumption(t, demo, consumptionEvidence(" top-2 ", "heizung", "sensor.top_2_heat", start, 5_000_000, " kWh "))
			appendConsumption(t, demo, consumptionEvidence("top-2", "heizung", "sensor.top_2_heat", endExclusive, 5_500_000, "kWh"))
			first := appendConsumption(t, demo, consumptionEvidence("TOP-1", "heizung", "SENSOR.TOP_1_HEAT", start, 10_000_000, "kWh"))
			appendConsumption(t, demo, consumptionEvidence("top-1", "heizung", "sensor.top_1_heat", start.Add(-time.Hour), 99_000_000, "kWh"))
			appendConsumption(t, demo, consumptionEvidence("top-1", "heizung", "sensor.top_1_heat", start.AddDate(0, 6, 0), 11_500_000, "kWh"))
			appendConsumption(t, demo, consumptionEvidence("top-1", "heizung", "sensor.top_1_heat", endExclusive, 13_000_000, "kWh"))
			appendConsumption(t, demo, consumptionEvidence("top-1", "heizung", "sensor.top_1_heat", endExclusive.Add(time.Hour), 1, "kWh"))
			appendConsumption(t, demo, consumptionEvidence("top-1", "warmwasser", "sensor.top_1_water", start, 700_000, "m3"))
			appendConsumption(t, demo, consumptionEvidence("top-1", "warmwasser", "sensor.top_1_water", endExclusive, 950_000, "m³"))

			// A source key is value-free and stable across delivery retries. The
			// first received-at timestamp wins; a retry never overwrites evidence.
			retry := consumptionEvidence("top-1", "heizung", "sensor.top_1_heat", start, 10_000_000, "kWh")
			retry.ReceivedAt = first.ReceivedAt.Add(time.Hour)
			stored, inserted, err := demo.Append(retry)
			if err != nil || inserted || stored.SourceKey != first.SourceKey || !stored.ReceivedAt.Equal(first.ReceivedAt) {
				t.Fatalf("idempotent retry = inserted=%t stored=%+v err=%v", inserted, stored, err)
			}
			changedValue := retry
			changedValue.ValueMicros++
			if _, _, err := demo.Append(changedValue); !errors.Is(err, ErrAnnualStatementConsumptionConflict) {
				t.Fatalf("conflicting duplicate error = %v, want typed conflict", err)
			}

			heating, err := demo.ConsumptionVector(period, "heizung", []string{"top-2", "top-1"}, location)
			if err != nil {
				t.Fatalf("heating vector: %v", err)
			}
			wantHeating := AnnualStatementConsumptionVector{
				PeriodYear: 2026, CostTypeKey: "heizung",
				Units: []AnnualStatementUnitConsumption{
					{UnitID: "top-1", ValueMicros: 3_000_000, MeasurementUnit: "kWh"},
					{UnitID: "top-2", ValueMicros: 500_000, MeasurementUnit: "kWh"},
				},
			}
			if !reflect.DeepEqual(heating, wantHeating) {
				t.Fatalf("heating vector = %+v, want %+v", heating, wantHeating)
			}
			hotWater, err := demo.ConsumptionVector(period, "warmwasser", []string{"top-1"}, location)
			if err != nil || len(hotWater.Gaps) != 0 || len(hotWater.Units) != 1 || hotWater.Units[0].ValueMicros != 250_000 || hotWater.Units[0].MeasurementUnit != "m³" {
				t.Fatalf("hot-water vector = %+v err=%v", hotWater, err)
			}

			// A caller cannot mutate persisted evidence through a returned vector.
			heating.Units[0].ValueMicros = 1
			rerun, err := demo.ConsumptionVector(period, "heizung", []string{"top-1", "top-2"}, location)
			if err != nil || !reflect.DeepEqual(rerun, wantHeating) {
				t.Fatalf("period re-run changed = %+v err=%v", rerun, err)
			}

			// The same value-free source identity is isolated by tenant. A foreign
			// value cannot collide with or leak into the demo vector.
			foreign := consumptionEvidence("top-1", "heizung", "sensor.top_1_heat", start, 99_000_000, "kWh")
			if _, inserted, err := other.Append(foreign); err != nil || !inserted {
				t.Fatalf("other tenant append: inserted=%t err=%v", inserted, err)
			}
			isolated, err := demo.ConsumptionVector(period, "heizung", []string{"top-1"}, location)
			if err != nil || isolated.Units[0].ValueMicros != 3_000_000 {
				t.Fatalf("foreign evidence leaked: %+v err=%v", isolated, err)
			}
		})
	}
}

func TestAnnualStatementConsumptionVectorsFailClosedWithValueFreeGapReasons(t *testing.T) {
	location := mustViennaLocation(t)
	period := AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31"}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, location)
	endExclusive := time.Date(2027, 1, 1, 0, 0, 0, 0, location)
	repository, _ := BindAnnualStatementConsumptionRepository(NewMemoryAnnualStatementConsumptionStore(), testTenantRef("demo"))

	// No evidence at all means no unit/source mapping exists.
	appendConsumption(t, repository, consumptionEvidence("no-start", "heizung", "sensor.no_start", start.Add(time.Hour), 10, "kWh"))
	appendConsumption(t, repository, consumptionEvidence("no-start", "heizung", "sensor.no_start", endExclusive, 20, "kWh"))
	appendConsumption(t, repository, consumptionEvidence("no-end", "heizung", "sensor.no_end", start, 10, "kWh"))
	appendConsumption(t, repository, consumptionEvidence("no-end", "heizung", "sensor.no_end", endExclusive.Add(-time.Hour), 20, "kWh"))
	appendConsumption(t, repository, consumptionEvidence("unitless", "heizung", "sensor.unitless", start, 10, ""))
	appendConsumption(t, repository, consumptionEvidence("unitless", "heizung", "sensor.unitless", endExclusive, 20, ""))
	appendConsumption(t, repository, consumptionEvidence("mixed-unit", "heizung", "sensor.mixed", start, 10, "kWh"))
	appendConsumption(t, repository, consumptionEvidence("mixed-unit", "heizung", "sensor.mixed", endExclusive, 20, "MWh"))
	appendConsumption(t, repository, consumptionEvidence("reset", "heizung", "sensor.reset", start, 100, "kWh"))
	appendConsumption(t, repository, consumptionEvidence("reset", "heizung", "sensor.reset", start.AddDate(0, 6, 0), 90, "kWh"))
	appendConsumption(t, repository, consumptionEvidence("reset", "heizung", "sensor.reset", endExclusive, 110, "kWh"))
	appendConsumption(t, repository, consumptionEvidence("ambiguous-source", "heizung", "sensor.one", start, 10, "kWh"))
	appendConsumption(t, repository, consumptionEvidence("ambiguous-source", "heizung", "sensor.one", endExclusive, 20, "kWh"))
	appendConsumption(t, repository, AnnualStatementConsumptionEvidence{
		UnitID: "ambiguous-source", CostTypeKey: "heizung", SourceKind: ConsumptionSourceAsset, SourceID: "meter-two",
		MeasuredAt: start, ValueMicros: 30, MeasurementUnit: "kWh", ReceivedAt: start.Add(time.Minute),
	})
	appendConsumption(t, repository, AnnualStatementConsumptionEvidence{
		UnitID: "ambiguous-source", CostTypeKey: "heizung", SourceKind: ConsumptionSourceAsset, SourceID: "meter-two",
		MeasuredAt: endExclusive, ValueMicros: 40, MeasurementUnit: "kWh", ReceivedAt: endExclusive.Add(time.Minute),
	})
	// This is newer than the end boundary and must never be substituted for it.
	appendConsumption(t, repository, consumptionEvidence("latest-only", "heizung", "sensor.latest", start, 10, "kWh"))
	appendConsumption(t, repository, consumptionEvidence("latest-only", "heizung", "sensor.latest", endExclusive.Add(time.Hour), 30, "kWh"))

	vector, err := repository.ConsumptionVector(period, "heizung", []string{
		"missing", "no-start", "no-end", "unitless", "mixed-unit", "reset", "ambiguous-source", "latest-only",
	}, location)
	if err != nil {
		t.Fatalf("gap vector: %v", err)
	}
	if len(vector.Units) != 0 {
		t.Fatalf("gapped vector published values: %+v", vector.Units)
	}
	want := []AnnualStatementConsumptionGap{
		{UnitID: "ambiguous-source", Reason: ConsumptionGapAmbiguousSource},
		{UnitID: "latest-only", Reason: ConsumptionGapMissingEndEvidence},
		{UnitID: "missing", Reason: ConsumptionGapMissingSourceMapping},
		{UnitID: "mixed-unit", Reason: ConsumptionGapAmbiguousMeasurementUnit},
		{UnitID: "no-end", Reason: ConsumptionGapMissingEndEvidence},
		{UnitID: "no-start", Reason: ConsumptionGapMissingStartEvidence},
		{UnitID: "reset", Reason: ConsumptionGapCounterReset},
		{UnitID: "unitless", Reason: ConsumptionGapAmbiguousMeasurementUnit},
	}
	if !reflect.DeepEqual(vector.Gaps, want) {
		t.Fatalf("gap reasons = %+v, want %+v", vector.Gaps, want)
	}

	for name, query := range map[string]func() error{
		"nil location": func() error {
			_, err := repository.ConsumptionVector(period, "heizung", []string{"missing"}, nil)
			return err
		},
		"duplicate expected unit": func() error {
			_, err := repository.ConsumptionVector(period, "heizung", []string{"top-1", " TOP-1 "}, location)
			return err
		},
		"invalid period": func() error {
			_, err := repository.ConsumptionVector(AnnualStatementPeriod{Year: 2026, StartsOn: "2026-12-31", EndsOn: "2026-01-01"}, "heizung", []string{"top-1"}, location)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := query(); !errors.Is(err, ErrAnnualStatementConsumptionInvalidQuery) {
				t.Fatalf("error = %v, want typed invalid query", err)
			}
		})
	}
}

func TestAnnualStatementConsumptionConcurrentDuplicateIngestion(t *testing.T) {
	location := mustViennaLocation(t)
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, location)
	for name, storage := range annualStatementConsumptionBackends(t) {
		t.Run(name, func(t *testing.T) {
			repository, _ := BindAnnualStatementConsumptionRepository(storage, testTenantRef("demo"))
			evidence := consumptionEvidence("top-1", "heizung", "sensor.concurrent", at, 123_000, "kWh")
			var inserted atomic.Int32
			var wg sync.WaitGroup
			for worker := 0; worker < 16; worker++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, created, err := repository.Append(evidence)
					if err != nil {
						t.Errorf("concurrent append: %v", err)
						return
					}
					if created {
						inserted.Add(1)
					}
				}()
			}
			wg.Wait()
			if got := inserted.Load(); got != 1 {
				t.Fatalf("inserted duplicates = %d, want exactly one", got)
			}

			conflicts := []AnnualStatementConsumptionEvidence{
				consumptionEvidence("top-1", "heizung", "sensor.concurrent-conflict", at, 1, "kWh"),
				consumptionEvidence("top-1", "heizung", "sensor.concurrent-conflict", at, 2, "kWh"),
			}
			inserted.Store(0)
			var rejected atomic.Int32
			for _, item := range conflicts {
				wg.Add(1)
				go func(item AnnualStatementConsumptionEvidence) {
					defer wg.Done()
					_, created, err := repository.Append(item)
					switch {
					case err == nil && created:
						inserted.Add(1)
					case errors.Is(err, ErrAnnualStatementConsumptionConflict):
						rejected.Add(1)
					default:
						t.Errorf("concurrent conflicting append: created=%t err=%v", created, err)
					}
				}(item)
			}
			wg.Wait()
			if inserted.Load() != 1 || rejected.Load() != 1 {
				t.Fatalf("concurrent conflict: inserted=%d rejected=%d, want 1/1", inserted.Load(), rejected.Load())
			}
		})
	}
}

func TestAnnualStatementConsumptionJSONAndSQLiteRoundTrips(t *testing.T) {
	location := mustViennaLocation(t)
	period := AnnualStatementPeriod{Year: 2026, StartsOn: "2026-01-01", EndsOn: "2026-12-31"}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, location)
	endExclusive := time.Date(2027, 1, 1, 0, 0, 0, 0, location)
	assertVector := func(t *testing.T, repository AnnualStatementConsumptionRepository) {
		t.Helper()
		vector, err := repository.ConsumptionVector(period, "heizung", []string{"top-1"}, location)
		if err != nil || len(vector.Gaps) != 0 || len(vector.Units) != 1 || vector.Units[0].ValueMicros != 250_000 {
			t.Fatalf("round-trip vector = %+v err=%v", vector, err)
		}
	}
	seed := func(t *testing.T, repository AnnualStatementConsumptionRepository) {
		t.Helper()
		appendConsumption(t, repository, consumptionEvidence("top-1", "heizung", "sensor.roundtrip", start, 1_000_000, "kWh"))
		appendConsumption(t, repository, consumptionEvidence("top-1", "heizung", "sensor.roundtrip", endExclusive, 1_250_000, "kWh"))
	}

	t.Run("json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "consumption.json")
		storage, err := NewJSONAnnualStatementConsumptionStore(path)
		if err != nil {
			t.Fatal(err)
		}
		repository, _ := BindAnnualStatementConsumptionRepository(storage, testTenantRef("demo"))
		seed(t, repository)
		if raw, err := os.ReadFile(path); err != nil || len(raw) == 0 {
			t.Fatalf("persisted JSON evidence: bytes=%d err=%v", len(raw), err)
		}
		reloaded, err := NewJSONAnnualStatementConsumptionStore(path)
		if err != nil {
			t.Fatalf("reload JSON evidence: %v", err)
		}
		repository, _ = BindAnnualStatementConsumptionRepository(reloaded, testTenantRef("demo"))
		assertVector(t, repository)
	})

	t.Run("sqlite", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "consumption.db")
		database, err := appdb.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		seedFixtureTenants(t, database)
		scoped, err := appdb.NewScoped(appdb.Config{Backend: appdb.BackendSQLite, DSN: path}, database)
		if err != nil {
			t.Fatal(err)
		}
		repository, _ := BindAnnualStatementConsumptionRepository(NewSQLAnnualStatementConsumptionStore(NewTenantDB(scoped)), testTenantRef("demo"))
		seed(t, repository)
		if err := scoped.Close(); err != nil {
			t.Fatal(err)
		}
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}

		database, err = appdb.Open(path)
		if err != nil {
			t.Fatalf("reopen migrated SQLite: %v", err)
		}
		defer database.Close()
		scoped, err = appdb.NewScoped(appdb.Config{Backend: appdb.BackendSQLite, DSN: path}, database)
		if err != nil {
			t.Fatal(err)
		}
		defer scoped.Close()
		repository, _ = BindAnnualStatementConsumptionRepository(NewSQLAnnualStatementConsumptionStore(NewTenantDB(scoped)), testTenantRef("demo"))
		assertVector(t, repository)
	})
}
