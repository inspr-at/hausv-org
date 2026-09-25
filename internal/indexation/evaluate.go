package indexation

import (
	"fmt"
	"math/big"
)

type ThresholdKind string

const (
	PercentThreshold ThresholdKind = "percent"
	PointsThreshold  ThresholdKind = "points"
)

type CrossingRule string

const (
	Exceeds CrossingRule = "" // WKO default: movements up to and including the threshold are ignored.
	Reaches CrossingRule = "reaches"
)

type ChangeMode string

const (
	FullChange   ChangeMode = "" // WKO default: apply the entire change once crossed.
	ExcessChange ChangeMode = "excess"
)

type PercentRounding string

const (
	OneDecimalPercent PercentRounding = ""
	UnroundedPercent  PercentRounding = "unrounded"
)

// Clause contains the contractual base, not the amount after a statutory cap.
// A future lease model must retain both contractual and statutory histories.
type Clause struct {
	Series             Series
	BaseMonth          Month
	BaseValue          Decimal // Must match a final observation in the dataset.
	AmountCents        int64   // Net principal rent; no operating costs or tax.
	ThresholdKind      ThresholdKind
	Threshold          Decimal
	Crossing           CrossingRule
	ChangeMode         ChangeMode
	MinimumChangeCents int64 // Optional symmetric Bagatellgrenze; equality is enough.
	PercentRounding    PercentRounding
	ExactAmountCents   string // Optional exact curve state from NextClause, never the capped rent.
	AllowDerived       bool   // Explicit opt-in for flagged fallback chaining; published values preferred.
}

// ExplanationStep is machine-readable evidence, not a precomposed letter.
// Percentages are in millionths of one percent, amounts in cents.
type ExplanationStep struct {
	Code               string
	Month              Month
	Base               Decimal
	Index              Decimal
	Change             Decimal
	Applied            Decimal
	Threshold          Decimal
	AmountCents        int64
	Source             string
	ChainSource        string
	ExactAmountCents   string
	ExactChangePercent string
}

type Evaluation struct {
	Crossed          bool
	TriggerMonth     Month
	EvaluatedThrough Month
	PendingMonth     Month // First preliminary or unpublished month; never silently skipped.
	ChangePercent    Decimal
	AppliedPercent   Decimal
	OldAmountCents   int64
	NewAmountCents   int64
	NewBase          Decimal
	NextClause       Clause
	Explanation      []ExplanationStep
	ExactAmountCents string
}

// Revalue compares two indices using one-decimal indices and one-decimal percent
// changes, as in the Statistik Austria worked examples. Money rounds half up.
func Revalue(cents int64, base, current Decimal) (Decimal, int64, error) {
	base, err := roundTenth(base)
	if err != nil {
		return 0, 0, err
	}
	current, err = roundTenth(current)
	if err != nil {
		return 0, 0, err
	}
	change, err := percentChange(base, current)
	if err != nil {
		return 0, 0, err
	}
	amount, err := applyPercent(cents, change, true)
	return change, amount, err
}

// Evaluate finds the first qualifying final month after BaseMonth, up to asOf.
// It does not mutate the clause or dataset. To replay subsequent jumps, call it
// again with NextClause. Timing and MieWeG are deliberately separate operations.
func Evaluate(clause Clause, data Dataset, asOf Month) (Evaluation, error) {
	return evaluateClause(clause, data, asOf, "")
}

// EvaluateReference applies a reviewed periodic clause to its specified
// reference month. It shares all decimal, threshold and rounding logic with
// Evaluate; no synthetic intermediate index observations are introduced.
func EvaluateReference(clause Clause, data Dataset, reference Month) (Evaluation, error) {
	if !reference.Valid() || reference <= clause.BaseMonth {
		return Evaluation{}, fmt.Errorf("reference must follow base month")
	}
	return evaluateClause(clause, data, reference, reference)
}

func evaluateClause(clause Clause, data Dataset, asOf, reference Month) (Evaluation, error) {
	if !clause.Series.Valid() || !clause.BaseMonth.Valid() || !asOf.Valid() || asOf < clause.BaseMonth || clause.AmountCents <= 0 || clause.BaseValue <= 0 || clause.Threshold < 0 || clause.MinimumChangeCents < 0 {
		return Evaluation{}, fmt.Errorf("invalid clause or observation cutoff")
	}
	if clause.ThresholdKind != PercentThreshold && clause.ThresholdKind != PointsThreshold {
		return Evaluation{}, fmt.Errorf("unknown threshold kind")
	}
	if clause.Crossing != Exceeds && clause.Crossing != Reaches || clause.ChangeMode != FullChange && clause.ChangeMode != ExcessChange {
		return Evaluation{}, fmt.Errorf("unknown crossing rule or change mode")
	}
	if clause.PercentRounding != OneDecimalPercent && clause.PercentRounding != UnroundedPercent {
		return Evaluation{}, fmt.Errorf("unknown percentage rounding")
	}
	exactAmount, err := exactCents(clause.ExactAmountCents, clause.AmountCents)
	if err != nil {
		return Evaluation{}, err
	}
	baseObservation, found, err := data.Lookup(clause.Series, clause.BaseMonth)
	if err != nil {
		return Evaluation{}, err
	}
	if !found || baseObservation.Preliminary || baseObservation.Value != clause.BaseValue {
		return Evaluation{}, fmt.Errorf("base must match a final observation for %s %s", clause.Series, clause.BaseMonth)
	}
	if baseObservation.ChainSource != "" && !clause.AllowDerived {
		return Evaluation{}, fmt.Errorf("index_derived: base requires explicit review")
	}
	base, err := roundTenth(clause.BaseValue)
	if err != nil || base <= 0 {
		return Evaluation{}, fmt.Errorf("invalid rounded base")
	}
	result := Evaluation{OldAmountCents: clause.AmountCents, NewAmountCents: clause.AmountCents, NewBase: clause.BaseValue, NextClause: clause, EvaluatedThrough: clause.BaseMonth, ExactAmountCents: exactAmount.RatString()}
	result.Explanation = append(result.Explanation, ExplanationStep{Code: "base_final", Month: clause.BaseMonth, Base: base, Index: base, AmountCents: clause.AmountCents, Source: baseObservation.Source, ChainSource: baseObservation.ChainSource})
	latest := data.latest(clause.Series)
	for month := clause.BaseMonth.next(); month.Valid() && month <= asOf; month = month.next() {
		if reference != "" && month != reference {
			continue
		}
		v, found, err := data.Lookup(clause.Series, month)
		if err != nil {
			return Evaluation{}, err
		}
		if !found {
			if month <= latest {
				return Evaluation{}, MissingIndexError{Series: clause.Series, Period: string(month)}
			}
			result.PendingMonth = month
			result.Explanation = append(result.Explanation, ExplanationStep{Code: "not_published", Month: month})
			break
		}
		if v.Preliminary {
			result.PendingMonth = month
			result.Explanation = append(result.Explanation, ExplanationStep{Code: "preliminary_ignored", Month: month, Index: v.Value, Source: v.Source})
			break
		}
		if v.ChainSource != "" && !clause.AllowDerived {
			return Evaluation{}, fmt.Errorf("index_derived: %s requires explicit review", month)
		}
		index, err := roundTenth(v.Value)
		if err != nil {
			return Evaluation{}, err
		}
		change, err := percentChange(base, index)
		if err != nil {
			return Evaluation{}, err
		}
		changeExact := decimalRat(change)
		if clause.PercentRounding == UnroundedPercent {
			changeExact = new(big.Rat).Mul(new(big.Rat).SetFrac64(int64(index-base), int64(base)), big.NewRat(100, 1))
			change, err = displayDecimal(changeExact)
			if err != nil {
				return Evaluation{}, err
			}
		}
		result.EvaluatedThrough = month
		result.ChangePercent = change
		measure := new(big.Rat).Set(changeExact)
		if clause.ThresholdKind == PointsThreshold {
			measure = decimalRat(index - base)
		}
		magnitude := new(big.Rat).Abs(measure)
		cmp := magnitude.Cmp(decimalRat(clause.Threshold))
		crossed := cmp > 0 || clause.Crossing == Reaches && cmp == 0
		// An unchanged value is never a jump, including a zero threshold.
		crossed = crossed && measure.Sign() != 0
		step := ExplanationStep{Code: "within_threshold", Month: month, Base: base, Index: index, Change: change, Threshold: clause.Threshold, AmountCents: clause.AmountCents, Source: v.Source, ChainSource: v.ChainSource}
		if !crossed {
			result.Explanation = append(result.Explanation, step)
			continue
		}
		appliedExact := new(big.Rat).Set(changeExact)
		if clause.ChangeMode == ExcessChange {
			if clause.ThresholdKind == PercentThreshold {
				appliedExact.Sub(magnitude, decimalRat(clause.Threshold))
				if measure.Sign() < 0 {
					appliedExact.Neg(appliedExact)
				}
			} else {
				// Compare the excess points against the same base, symmetrically.
				adjusted := index - clause.Threshold
				if measure.Sign() < 0 {
					adjusted = index + clause.Threshold
				}
				applied, err := percentChange(base, adjusted)
				if err != nil {
					return Evaluation{}, err
				}
				appliedExact = decimalRat(applied)
				if clause.PercentRounding == UnroundedPercent {
					appliedExact = new(big.Rat).Mul(new(big.Rat).SetFrac64(int64(adjusted-base), int64(base)), big.NewRat(100, 1))
				}
			}
		}
		factor := new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Quo(appliedExact, big.NewRat(100, 1)))
		newExact := new(big.Rat).Mul(exactAmount, factor)
		amount, err := roundRat(newExact, true)
		if err != nil {
			return Evaluation{}, err
		}
		difference := amount - clause.AmountCents
		if difference < 0 {
			difference = -difference
		}
		applied, err := displayDecimal(appliedExact)
		if err != nil {
			return Evaluation{}, err
		}
		step.Applied, step.AmountCents = applied, amount
		step.ExactAmountCents, step.ExactChangePercent = newExact.RatString(), changeExact.RatString()
		if difference == 0 || difference < clause.MinimumChangeCents {
			step.Code = "below_minimum_change"
			result.Explanation = append(result.Explanation, step)
			continue
		}
		step.Code = "threshold_crossed"
		result.Explanation = append(result.Explanation, step)
		result.Crossed, result.TriggerMonth = true, month
		result.ChangePercent, result.AppliedPercent = change, applied
		result.NewAmountCents, result.NewBase = amount, v.Value
		result.ExactAmountCents = newExact.RatString()
		result.NextClause.BaseMonth, result.NextClause.BaseValue, result.NextClause.AmountCents = month, v.Value, amount
		result.NextClause.ExactAmountCents = newExact.RatString()
		result.Explanation = append(result.Explanation, ExplanationStep{Code: "base_reset", Month: month, Base: index, AmountCents: amount})
		return result, nil
	}
	return result, nil
}
