package indexation

import (
	"fmt"
	"math/big"
	"time"
)

// StaffelStep is an explicit contractual step. Exactly one amount or percentage
// is required. Percentages apply to the previous exact contract value, never
// to the rent limited by MieWeG. Clause prose is not an input to this calculation.
type StaffelStep struct {
	EffectiveOn string `json:"effective_on"`
	NetCents    *int64 `json:"net_cents,omitempty"`
	Percent     string `json:"percent,omitempty"`
}

const MaxStaffelSteps = 120

func ValidateStaffelSteps(steps []StaffelStep) error {
	if len(steps) == 0 || len(steps) > MaxStaffelSteps {
		return fmt.Errorf("Staffelmietzins benötigt 1 bis %d vereinbarte Stufen", MaxStaffelSteps)
	}
	last := ""
	for _, step := range steps {
		on, err := time.Parse(time.DateOnly, step.EffectiveOn)
		if err != nil || on.Year() < 1 || step.EffectiveOn <= last {
			return fmt.Errorf("Staffeldaten müssen gültig, eindeutig und aufsteigend sein")
		}
		if (step.NetCents == nil) == (step.Percent == "") {
			return fmt.Errorf("je Staffel genau einen neuen HMZ netto oder einen Prozentsatz angeben")
		}
		if step.NetCents != nil && *step.NetCents <= 0 {
			return fmt.Errorf("Staffel-HMZ muss positiv sein")
		}
		if step.Percent != "" {
			p, err := ParseDecimal(step.Percent)
			if err != nil || p <= 0 {
				return fmt.Errorf("Staffelerhöhung muss ein positiver Prozentsatz mit höchstens sechs Nachkommastellen sein")
			}
		}
		last = step.EffectiveOn
	}
	return nil
}

// EvaluateStaffel rebuilds the contract curve from its original rent and the
// complete schedule. The caller supplies the contractual cutoff (April 1 for
// covered increases); later steps must never leak into an earlier April run.
func EvaluateStaffel(startCents, currentCents int64, steps []StaffelStep, through time.Time) (Evaluation, time.Time, error) {
	if startCents <= 0 || currentCents <= 0 || through.IsZero() {
		return Evaluation{}, time.Time{}, fmt.Errorf("Staffel-Ausgangsmiete und Stichtag erforderlich")
	}
	if err := ValidateStaffelSteps(steps); err != nil {
		return Evaluation{}, time.Time{}, err
	}
	amount := big.NewRat(startCents, 1)
	result := Evaluation{OldAmountCents: currentCents, NewAmountCents: startCents, ExactAmountCents: amount.RatString()}
	var effective time.Time
	for _, step := range steps {
		on, _ := time.Parse(time.DateOnly, step.EffectiveOn)
		if on.After(civilDate(through)) {
			break
		}
		var percent Decimal
		if step.NetCents != nil {
			amount.SetInt64(*step.NetCents)
		} else {
			percent, _ = ParseDecimal(step.Percent)
			factor := new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Quo(decimalRat(percent), big.NewRat(100, 1)))
			amount.Mul(amount, factor)
		}
		cents, err := roundRat(amount, true)
		if err != nil || len(amount.RatString()) > 2048 {
			return Evaluation{}, time.Time{}, fmt.Errorf("Staffelbetrag überschreitet den zulässigen Rechenbereich")
		}
		effective = on
		result.NewAmountCents, result.ExactAmountCents = cents, amount.RatString()
		result.Explanation = append(result.Explanation, ExplanationStep{Code: "staffel_step", Month: monthOf(on), Applied: percent, AmountCents: cents, ExactAmountCents: amount.RatString()})
	}
	// A sub-cent residue alone is not a new contractual adjustment. The exact
	// value remains available for MieWeG rounding and later percentage steps.
	result.Crossed = !effective.IsZero() && result.NewAmountCents != currentCents
	return result, effective, nil
}
