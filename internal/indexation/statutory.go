package indexation

import (
	"fmt"
	"math/big"
	"time"
)

const MieWeGSource = "https://www.ris.bka.gv.at/Dokumente/BgblAuth/BGBLA_2025_I_114/BGBLA_2025_I_114.pdf"
const MieWeGNotesSource = "https://www.parlament.gv.at/dokument/XXVIII/I/269/fnameorig_1718389.html"

type RentalScope string

const (
	ResidentialFullMRG    RentalScope = "residential_full_mrg"
	ResidentialPartialMRG RentalScope = "residential_partial_mrg"
	ResidentialExcluded   RentalScope = "residential_excluded_mrg"
	Commercial            RentalScope = "commercial"
	CommercialFullMRG     RentalScope = "commercial_full_mrg"
)

// AnnualModel describes one comparison year. Averages are the published VPI2020
// annual levels, never the rounded year-on-year percentages. FullMonths is
// explicit (0..12); the signing month does not count, even when signed on day 1.
type AnnualModel struct {
	Year            int
	BaseAmountCents int64
	PreviousAverage Decimal
	CurrentAverage  Decimal
	FullMonths      int
	PriceControlled bool // Includes angemessener Mietzins, Richtwert and Kategorie.
}

type AnnualStep struct {
	Year               int // Inflation year, not adjustment year.
	PreviousAverage    Decimal
	CurrentAverage     Decimal
	RawRatePercent     string // Exact integer ratio, not a rounded percentage.
	LimitedRatePercent string // After half-above-3 and special caps, before proration.
	FullMonths         int
	ExactAmountCents   string
	Source             string
}

type Ceiling struct {
	AmountCents      int64
	ExactAmountCents string // Integer ratio, retains fractions of a cent between years.
	EffectiveOn      time.Time
	PriceControlled  bool
	Years            []AnnualStep
	Explanation      []ExplanationStep
}

func annualFactor(year int, previous, current Decimal, restricted bool, months int) (*big.Rat, AnnualStep, error) {
	if previous <= 0 || current <= 0 || months < 0 || months > 12 {
		return nil, AnnualStep{}, fmt.Errorf("positive annual averages and 0..12 full months required")
	}
	rate := new(big.Rat).Mul(new(big.Rat).SetFrac64(int64(current-previous), int64(previous)), big.NewRat(100, 1))
	step := AnnualStep{Year: year, PreviousAverage: previous, CurrentAverage: current, RawRatePercent: rate.RatString(), FullMonths: months}
	if rate.Cmp(big.NewRat(3, 1)) > 0 {
		rate.Sub(rate, big.NewRat(3, 1)).Quo(rate, big.NewRat(2, 1)).Add(rate, big.NewRat(3, 1))
	}
	if restricted {
		capRate := int64(0)
		if year == 2025 {
			capRate = 1
		}
		if year == 2026 {
			capRate = 2
		}
		if capRate != 0 && rate.Cmp(big.NewRat(capRate, 1)) > 0 {
			rate.SetInt64(capRate)
		}
	}
	step.LimitedRatePercent = rate.RatString()
	rate.Mul(rate, big.NewRat(int64(months), 12))
	factor := new(big.Rat).Add(big.NewRat(1, 1), rate.Quo(rate, big.NewRat(100, 1)))
	return factor, step, nil
}

// AnnualCeiling is the one-year form of CapCurve. Historical/multi-year
// transitions should use CapCurve to avoid rounding at intermediate years.
func AnnualCeiling(model AnnualModel) (Ceiling, error) {
	if model.Year < 2026 || model.Year > 9999 || model.BaseAmountCents <= 0 {
		return Ceiling{}, fmt.Errorf("invalid annual statutory model")
	}
	factor, step, err := annualFactor(model.Year-1, model.PreviousAverage, model.CurrentAverage, model.PriceControlled, model.FullMonths)
	if err != nil {
		return Ceiling{}, err
	}
	amount := new(big.Rat).Mul(new(big.Rat).SetInt64(model.BaseAmountCents), factor)
	step.ExactAmountCents = amount.RatString()
	return finishCeiling(amount, model.Year, model.PriceControlled, []AnnualStep{step})
}

// CapCurve rebuilds the independent MieWeG ceiling from its immutable anchor.
// For old contracts, anchor is the index month of the last pre-2026 adjustment
// (December for an annual-average anchor), or conclusion when never adjusted.
// Only the final result is rounded. The statutory and contract curves must NEVER
// be reseeded from the lower prescribed rent. Missing/preliminary years fail.
func CapCurve(anchor Month, startCents int64, averages []AnnualValue, restricted bool, throughYear int) (Ceiling, error) {
	if !anchor.Valid() || startCents <= 0 || throughYear < 2026 || throughYear > 9999 || throughYear <= anchor.date().Year() {
		return Ceiling{}, fmt.Errorf("invalid cap anchor, amount or adjustment year")
	}
	byYear := make(map[int]AnnualValue)
	for _, v := range averages {
		if v.Series != VPI2020 {
			continue
		}
		if _, exists := byYear[v.Year]; exists {
			return Ceiling{}, fmt.Errorf("duplicate VPI2020 annual average: %d", v.Year)
		}
		byYear[v.Year] = v
	}
	amount := new(big.Rat).SetInt64(startCents)
	var steps []AnnualStep
	for year := anchor.date().Year(); year < throughYear; year++ {
		months := 12
		if year == anchor.date().Year() {
			months -= int(anchor.date().Month())
		}
		// December contributes zero. It still has a visible audit step, but needs no
		// fictitious annual observation or obsolete predecessor average.
		if months == 0 {
			steps = append(steps, AnnualStep{Year: year, FullMonths: 0, RawRatePercent: "0", LimitedRatePercent: "0", ExactAmountCents: amount.RatString()})
			continue
		}
		previous, pok := byYear[year-1]
		current, cok := byYear[year]
		if !pok || !cok || previous.Preliminary || current.Preliminary {
			return Ceiling{}, fmt.Errorf("final VPI2020 annual averages for %d and %d required", year-1, year)
		}
		factor, step, err := annualFactor(year, previous.Value, current.Value, restricted, months)
		if err != nil {
			return Ceiling{}, err
		}
		amount.Mul(amount, factor)
		step.ExactAmountCents, step.Source = amount.RatString(), current.Source
		steps = append(steps, step)
	}
	return finishCeiling(amount, throughYear, restricted, steps)
}

func finishCeiling(amount *big.Rat, year int, restricted bool, steps []AnnualStep) (Ceiling, error) {
	cents, err := roundRat(amount, false)
	if err != nil {
		return Ceiling{}, err
	}
	return Ceiling{AmountCents: cents, ExactAmountCents: amount.RatString(), EffectiveOn: time.Date(year, time.April, 1, 0, 0, 0, 0, time.UTC), PriceControlled: restricted, Years: steps, Explanation: []ExplanationStep{{Code: "statutory_parallel_curve", AmountCents: cents, ExactAmountCents: amount.RatString(), Source: MieWeGNotesSource}}}, nil
}

func FirstAdjustmentMonths(concluded Month, adjustmentYear int) (int, error) {
	if !concluded.Valid() || concluded.date().Year() < 2026 || adjustmentYear != concluded.date().Year()+1 {
		return 0, fmt.Errorf("new-contract first adjustment must be in the following year; use CapCurve for legacy transitions")
	}
	return 12 - int(concluded.date().Month()), nil
}

type StatutoryContext struct {
	Scope               RentalScope
	IsSublease          bool
	CurrentAmountCents  int64
	ContractEffectiveOn time.Time // Freeze contractual result at this date.
	Ceiling             Ceiling   // Separately computed, independent of the contract curve.
	MaxRentCents        *int64    // Optional manually reviewed § 16 maximum, not computed here.
}

type LimitedAdjustment struct {
	CalculatedCents   int64
	PermittedCents    int64
	AllowedNowCents   int64 // Index/cap timing only; Timing must still check notices.
	DeferredCents     int64 // Timing-only difference, NOT an arrears balance.
	CappedCents       int64 // Excess; no promise of future collectability.
	EffectiveOn       time.Time
	RequiresMRGNotice bool
	Explanation       []ExplanationStep
}

// ApplyMieWeG takes the lower exact curve, then rounds half down. It applies the
// April rule to increases only; favourable contractual decreases remain intact.
// WGG and unknown legal classifications are refused, not guessed as exempt.
func ApplyMieWeG(contract Evaluation, context StatutoryContext, asOf time.Time) (LimitedAdjustment, error) {
	if context.CurrentAmountCents <= 0 || contract.NewAmountCents < 0 || asOf.IsZero() || context.ContractEffectiveOn.IsZero() {
		return LimitedAdjustment{}, fmt.Errorf("amounts, contractual effective date and as-of date required")
	}
	covered := false
	switch context.Scope {
	case ResidentialFullMRG, ResidentialPartialMRG:
		covered = true
	case Commercial, CommercialFullMRG, ResidentialExcluded:
	default:
		return LimitedAdjustment{}, fmt.Errorf("unsupported rental scope (including unclassified WGG)")
	}
	full := context.Scope == ResidentialFullMRG || context.Scope == CommercialFullMRG
	result := LimitedAdjustment{CalculatedCents: contract.NewAmountCents, PermittedCents: contract.NewAmountCents, AllowedNowCents: context.CurrentAmountCents, EffectiveOn: civilDate(context.ContractEffectiveOn), RequiresMRGNotice: full && !context.IsSublease}
	if !contract.Crossed {
		result.PermittedCents = context.CurrentAmountCents
		result.Explanation = append(result.Explanation, ExplanationStep{Code: "no_contractual_jump", AmountCents: context.CurrentAmountCents})
		return result, nil
	}
	exact, err := exactCents(contract.ExactAmountCents, contract.NewAmountCents)
	if err != nil {
		return LimitedAdjustment{}, err
	}
	if covered && exact.Cmp(new(big.Rat).SetInt64(context.CurrentAmountCents)) > 0 {
		if result.EffectiveOn.Year() < 2026 {
			return LimitedAdjustment{}, fmt.Errorf("missed_pre2026: entitlement requires transition review")
		}
		ceiling := context.Ceiling
		if ceiling.PriceControlled != full {
			return LimitedAdjustment{}, fmt.Errorf("price-control ceiling must match full versus partial MRG classification")
		}
		if ceiling.EffectiveOn.IsZero() || ceiling.EffectiveOn.Year() < 2026 || ceiling.EffectiveOn.Month() != time.April || ceiling.EffectiveOn.Day() != 1 {
			return LimitedAdjustment{}, fmt.Errorf("verified April statutory ceiling required")
		}
		if civilDate(ceiling.EffectiveOn).Before(result.EffectiveOn) {
			return LimitedAdjustment{}, fmt.Errorf("ceiling must not predate contractual entitlement")
		}
		capExact, err := parseExactRatio(ceiling.ExactAmountCents)
		if err != nil {
			return LimitedAdjustment{}, err
		}
		capCents, err := roundRat(capExact, false)
		if err != nil || capCents != ceiling.AmountCents {
			return LimitedAdjustment{}, fmt.Errorf("inconsistent statutory ceiling")
		}
		if capExact.Cmp(exact) < 0 {
			exact = capExact
		}
		result.EffectiveOn = civilDate(ceiling.EffectiveOn)
		result.Explanation = append(result.Explanation, ceiling.Explanation...)
		result.PermittedCents, err = roundRat(exact, false)
		if err != nil {
			return LimitedAdjustment{}, err
		}
	} else if covered {
		result.PermittedCents, err = roundRat(exact, false)
		if err != nil {
			return LimitedAdjustment{}, err
		}
	}
	if context.MaxRentCents != nil {
		if !full || *context.MaxRentCents < 0 {
			return LimitedAdjustment{}, fmt.Errorf("invalid full-MRG rent maximum")
		}
		if result.PermittedCents > *context.MaxRentCents {
			result.PermittedCents = *context.MaxRentCents
			result.Explanation = append(result.Explanation, ExplanationStep{Code: "section_16_ceiling", AmountCents: result.PermittedCents})
		}
	}
	result.CappedCents = result.CalculatedCents - result.PermittedCents
	result.Explanation = append(result.Explanation, ExplanationStep{Code: "permitted_amount", AmountCents: result.PermittedCents, Source: MieWeGSource})
	if !civilDate(asOf).Before(result.EffectiveOn) {
		result.AllowedNowCents = result.PermittedCents
	}
	result.DeferredCents = result.PermittedCents - result.AllowedNowCents
	return result, nil
}
