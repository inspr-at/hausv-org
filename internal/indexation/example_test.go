package indexation_test

import (
	"fmt"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

func ExampleApplyMieWeG() {
	snapshot, err := indexation.LoadSnapshot()
	if err != nil {
		panic(err)
	}
	base, _, err := snapshot.Data.Lookup(indexation.VPI2020, "2024-09")
	if err != nil {
		panic(err)
	}
	contract, err := indexation.Evaluate(indexation.Clause{
		Series: indexation.VPI2020, BaseMonth: "2024-09", BaseValue: base.Value,
		AmountCents: 100000, ThresholdKind: indexation.PercentThreshold,
		Threshold: 5 * indexation.Unit, PercentRounding: indexation.UnroundedPercent,
	}, snapshot.Data, "2025-12")
	if err != nil {
		panic(err)
	}
	// This immutable statutory anchor is NOT contract.NextClause.BaseMonth.
	ceiling, err := indexation.CapCurve("2024-09", 100000, snapshot.Annual, true, 2026)
	if err != nil {
		panic(err)
	}
	april := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	allowed, err := indexation.ApplyMieWeG(contract, indexation.StatutoryContext{
		Scope: indexation.ResidentialFullMRG, CurrentAmountCents: 100000,
		ContractEffectiveOn: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
		Ceiling:             ceiling,
	}, april)
	if err != nil {
		panic(err)
	}
	timing, err := indexation.Timing(indexation.TimingInput{
		TriggerMonth:     contract.TriggerMonth,
		FinalPublishedOn: time.Date(2026, time.February, 25, 0, 0, 0, 0, time.UTC),
		Mode:             indexation.MieWeGTiming, StatutoryEffectiveOn: allowed.EffectiveOn,
		RequiresMRGNotice: allowed.RequiresMRGNotice,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Vertrag (Cent):", allowed.CalculatedCents)
	fmt.Println("Begrenzt (Cent):", allowed.PermittedCents)
	fmt.Println("Frühester Zinstermin:", timing.DueOn.Format(time.DateOnly))
	// Output:
	// Vertrag (Cent): 105016
	// Begrenzt (Cent): 101735
	// Frühester Zinstermin: 2026-05-05
}
