package store

import (
	"testing"
	"time"
)

func TestAnnualStatementConsumptionReportBoundsSnapshotAndChecksInteriorReset(t *testing.T) {
	period := AnnualStatementPeriod{Year: 2025, StartsOn: "2025-01-01", EndsOn: "2025-12-31"}
	location := mustViennaLocation(t)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, location)
	end := start.AddDate(1, 0, 0)
	for name, storage := range annualStatementConsumptionBackends(t) {
		t.Run(name, func(t *testing.T) {
			repository, _ := BindAnnualStatementConsumptionRepository(storage, testTenantRef("demo"))
			appendConsumption(t, repository, consumptionEvidence("a", "heizung", "sensor.a", start, 100, "kWh"))
			for i := 1; i <= 100; i++ {
				appendConsumption(t, repository, consumptionEvidence("a", "heizung", "sensor.a", start.Add(time.Duration(i)*time.Hour), int64(100+i), "kWh"))
			}
			appendConsumption(t, repository, consumptionEvidence("a", "heizung", "sensor.a", end, 1000, "kWh"))
			report, err := repository.ConsumptionReport(period, "heizung", []string{"a", "b"}, location)
			if err != nil || len(report.MeasuredUnits) != 1 || len(report.BoundaryEvidence) != 2 || len(report.Vector.Units) != 0 || len(report.Vector.Gaps) != 1 {
				t.Fatalf("partial report=%+v %v", report, err)
			}
			appendConsumption(t, repository, consumptionEvidence("b", "heizung", "sensor.b", start, 100, "kWh"))
			appendConsumption(t, repository, consumptionEvidence("b", "heizung", "sensor.b", end, 200, "kWh"))
			report, err = repository.ConsumptionReport(period, "heizung", []string{"a", "b"}, location)
			if err != nil || len(report.Vector.Units) != 2 || len(report.BoundaryEvidence) != 4 {
				t.Fatalf("complete report=%+v %v", report, err)
			}
			// A late-arriving interior reset invalidates the vector despite both valid boundaries.
			appendConsumption(t, repository, consumptionEvidence("a", "heizung", "sensor.a", start.Add(110*time.Hour), 1, "kWh"))
			report, err = repository.ConsumptionReport(period, "heizung", []string{"a", "b"}, location)
			if err != nil || len(report.Vector.Units) != 0 || len(report.Vector.Gaps) != 1 || report.Vector.Gaps[0].Reason != ConsumptionGapCounterReset {
				t.Fatalf("reset hidden by compact snapshot: %+v %v", report, err)
			}
		})
	}
}
