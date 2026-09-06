package store

import (
	"math"
	"reflect"
	"testing"
)

func TestAnnualStatementConsumptionShares(t *testing.T) {
	units := []Unit{{ID: "top-1"}, {ID: "top-2"}, {ID: "top-3"}}
	for _, test := range []struct {
		name   string
		values []int64
		want   []int
	}{
		{"equal with remainder", []int64{1, 1, 1}, []int{333334, 333333, 333333}},
		{"measured zero", []int64{0, 1, 3}, []int{0, 250000, 750000}},
		{"large counters", []int64{math.MaxInt64, math.MaxInt64, math.MaxInt64}, []int{333334, 333333, 333333}},
		{"all zero", []int64{0, 0, 0}, nil},
		{"negative", []int64{1, -1, 1}, nil},
		{"missing unit", []int64{1, 1}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			vector := AnnualStatementConsumptionVector{PeriodYear: 2026, CostTypeKey: "heizung"}
			for i, value := range test.values {
				vector.Units = append(vector.Units, AnnualStatementUnitConsumption{UnitID: units[i].ID, ValueMicros: value, MeasurementUnit: "kWh"})
			}
			shares, ready := AnnualStatementConsumptionShares(vector, units)
			if ready != (test.want != nil) {
				t.Fatalf("ready=%v want shares %v", ready, test.want)
			}
			var got []int
			for _, share := range shares {
				got = append(got, share.SharePPM)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("shares=%v want=%v", got, test.want)
			}
		})
	}
}

func TestAnnualStatementConsumptionSharesFailClosed(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*AnnualStatementConsumptionVector, []Unit)
	}{
		{"partial vector", func(v *AnnualStatementConsumptionVector, _ []Unit) {
			v.Gaps = []AnnualStatementConsumptionGap{{UnitID: "top-2", Reason: ConsumptionGapMissingEndEvidence}}
		}},
		{"duplicate measurement", func(v *AnnualStatementConsumptionVector, _ []Unit) { v.Units[1].UnitID = "top-1" }},
		{"foreign unit", func(v *AnnualStatementConsumptionVector, _ []Unit) { v.Units[1].UnitID = "foreign" }},
		{"duplicate expected unit", func(_ *AnnualStatementConsumptionVector, u []Unit) { u[1].ID = "top-1" }},
		{"mixed measurement units", func(v *AnnualStatementConsumptionVector, _ []Unit) { v.Units[1].MeasurementUnit = "m³" }},
		{"unknown measurement unit", func(v *AnnualStatementConsumptionVector, _ []Unit) {
			for i := range v.Units {
				v.Units[i].MeasurementUnit = "unknown"
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			units := []Unit{{ID: "top-1"}, {ID: "top-2"}}
			v := AnnualStatementConsumptionVector{Units: []AnnualStatementUnitConsumption{{UnitID: "top-1", ValueMicros: 1, MeasurementUnit: "kWh"}, {UnitID: "top-2", ValueMicros: 1, MeasurementUnit: "kWh"}}}
			test.mutate(&v, units)
			if shares, ready := AnnualStatementConsumptionShares(v, units); ready || shares != nil {
				t.Fatalf("invalid vector allocated: %+v", shares)
			}
		})
	}
}
