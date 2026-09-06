package store

import (
	"database/sql"
	"sort"
	"time"
)

// MeasuredUnits is display-only partial evidence. Only Vector may be allocated.
// BoundaryEvidence is bounded to two facts per expected unit, regardless of
// sampling frequency. Interior readings are checked for resets, never copied
// into each statement run.
type AnnualStatementConsumptionReport struct {
	Vector           AnnualStatementConsumptionVector
	MeasuredUnits    []AnnualStatementUnitConsumption
	BoundaryEvidence []AnnualStatementConsumptionEvidence
}

type annualConsumptionUnitState struct {
	first           AnnualStatementConsumptionEvidence
	previousValue   int64
	found           bool
	ambiguousSource bool
	ambiguousUnit   bool
	reset           bool
	start           *AnnualStatementConsumptionEvidence
	end             *AnnualStatementConsumptionEvidence
}
type annualConsumptionAccumulator struct {
	year       int
	cost       string
	expected   []string
	start, end time.Time
	units      map[string]*annualConsumptionUnitState
}

func newAnnualConsumptionAccumulator(year int, cost string, expected []string, start, end time.Time) *annualConsumptionAccumulator {
	a := &annualConsumptionAccumulator{year: year, cost: cost, expected: expected, start: start, end: end, units: map[string]*annualConsumptionUnitState{}}
	for _, id := range expected {
		a.units[id] = &annualConsumptionUnitState{}
	}
	return a
}

// add accepts readings ordered by unit, measurement time, then source key.
func (a *annualConsumptionAccumulator) add(item AnnualStatementConsumptionEvidence) {
	state, expected := a.units[item.UnitID]
	if !expected {
		return
	}
	if state.found {
		state.ambiguousSource = state.ambiguousSource || item.SourceKind != state.first.SourceKind || item.SourceID != state.first.SourceID
		state.ambiguousUnit = state.ambiguousUnit || item.MeasurementUnit != state.first.MeasurementUnit
		state.reset = state.reset || item.ValueMicros < state.previousValue
	} else {
		state.first = item
		state.found = true
	}
	state.previousValue = item.ValueMicros
	if item.MeasuredAt.Equal(a.start) {
		copy := item
		state.start = &copy
	}
	if item.MeasuredAt.Equal(a.end) {
		copy := item
		state.end = &copy
	}
}
func (a *annualConsumptionAccumulator) report() AnnualStatementConsumptionReport {
	out := AnnualStatementConsumptionReport{Vector: AnnualStatementConsumptionVector{PeriodYear: a.year, CostTypeKey: a.cost}}
	measurementUnits := map[string]bool{}
	for _, id := range a.expected {
		state := a.units[id]
		var reason AnnualStatementConsumptionGapReason
		switch {
		case !state.found:
			reason = ConsumptionGapMissingSourceMapping
		case state.ambiguousSource:
			reason = ConsumptionGapAmbiguousSource
		case state.ambiguousUnit || state.first.MeasurementUnit == "":
			reason = ConsumptionGapAmbiguousMeasurementUnit
		case state.start == nil:
			reason = ConsumptionGapMissingStartEvidence
		case state.end == nil:
			reason = ConsumptionGapMissingEndEvidence
		case state.reset:
			reason = ConsumptionGapCounterReset
		}
		if reason != "" {
			out.Vector.Gaps = append(out.Vector.Gaps, AnnualStatementConsumptionGap{UnitID: id, Reason: reason})
			continue
		}
		unit := AnnualStatementUnitConsumption{UnitID: id, ValueMicros: state.end.ValueMicros - state.start.ValueMicros, MeasurementUnit: state.start.MeasurementUnit}
		out.MeasuredUnits = append(out.MeasuredUnits, unit)
		out.BoundaryEvidence = append(out.BoundaryEvidence, *state.start, *state.end)
		measurementUnits[unit.MeasurementUnit] = true
	}
	if len(out.Vector.Gaps) == 0 {
		if len(measurementUnits) != 1 {
			for _, id := range a.expected {
				out.Vector.Gaps = append(out.Vector.Gaps, AnnualStatementConsumptionGap{UnitID: id, Reason: ConsumptionGapAmbiguousMeasurementUnit})
			}
		} else {
			out.Vector.Units = append([]AnnualStatementUnitConsumption(nil), out.MeasuredUnits...)
		}
	}
	return out
}
func buildAnnualStatementConsumptionReport(year int, cost string, expected []string, start, end time.Time, evidence []AnnualStatementConsumptionEvidence) AnnualStatementConsumptionReport {
	items := append([]AnnualStatementConsumptionEvidence(nil), evidence...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].UnitID != items[j].UnitID {
			return items[i].UnitID < items[j].UnitID
		}
		if !items[i].MeasuredAt.Equal(items[j].MeasuredAt) {
			return items[i].MeasuredAt.Before(items[j].MeasuredAt)
		}
		return items[i].SourceKey < items[j].SourceKey
	})
	accumulator := newAnnualConsumptionAccumulator(year, cost, expected, start, end)
	for _, item := range items {
		accumulator.add(item)
	}
	return accumulator.report()
}

type annualConsumptionQueryer interface {
	Query(string, ...any) (*sql.Rows, error)
}

func queryAnnualStatementConsumptionReport(query annualConsumptionQueryer, tenant TenantRef, year int, cost string, expected []string, start, end time.Time) (AnnualStatementConsumptionReport, error) {
	rows, err := query.Query(`SELECT source_key,unit_id,cost_type_key,source_kind,source_id,measured_at_ns,value_micros,measurement_unit,received_at_ns
 FROM annual_statement_consumption_evidence WHERE tenant_id=$1 AND cost_type_key=$2 AND measured_at_ns >= $3 AND measured_at_ns <= $4
 ORDER BY unit_id,measured_at_ns,source_key`, tenant.ID, cost, start.UnixNano(), end.UnixNano())
	if err != nil {
		return AnnualStatementConsumptionReport{}, err
	}
	defer rows.Close()
	accumulator := newAnnualConsumptionAccumulator(year, cost, expected, start, end)
	for rows.Next() {
		item, err := scanAnnualStatementConsumption(rows)
		if err != nil {
			return AnnualStatementConsumptionReport{}, err
		}
		accumulator.add(item)
	}
	if err := rows.Err(); err != nil {
		return AnnualStatementConsumptionReport{}, err
	}
	return accumulator.report(), nil
}
func (s *SQLAnnualStatementConsumptionStore) reportAnnualStatementConsumption(tenant TenantRef, year int, cost string, expected []string, start, end time.Time) (AnnualStatementConsumptionReport, error) {
	return queryAnnualStatementConsumptionReport(s.db.For(tenant), tenant, year, cost, expected, start, end)
}
