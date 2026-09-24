package indexation

import (
	"testing"
	"time"
)

func TestStaffelSelectionAndExactCompounding(t *testing.T) {
	amount := int64(10001)
	steps := []StaffelStep{{EffectiveOn: "2026-04-01", NetCents: &amount}, {EffectiveOn: "2027-04-01", Percent: "0.5"}, {EffectiveOn: "2028-04-01", Percent: "0.5"}}
	for _, tc := range []struct {
		date  string
		cents int64
		exact string
		count int
	}{
		{"2026-03-31", 10000, "10000", 0}, {"2026-04-01", 10001, "10001", 1},
		{"2027-03-31", 10001, "10001", 1}, {"2027-04-01", 10051, "2010201/200", 2},
		{"2028-04-01", 10101, "404050401/40000", 3},
	} {
		t.Run(tc.date, func(t *testing.T) {
			at, _ := time.Parse(time.DateOnly, tc.date)
			got, on, err := EvaluateStaffel(10000, 10000, steps, at)
			if err != nil || got.NewAmountCents != tc.cents || got.ExactAmountCents != tc.exact || len(got.Explanation) != tc.count || (tc.count == 0) != on.IsZero() {
				t.Fatalf("%+v %s %v", got, on, err)
			}
		})
	}
}

func TestStaffelInvalidSchedules(t *testing.T) {
	amount := int64(100000)
	for _, steps := range [][]StaffelStep{
		nil, {{EffectiveOn: "2026-04-01"}}, {{EffectiveOn: "2026-04-01", NetCents: &amount, Percent: "5"}},
		{{EffectiveOn: "2026-02-30", Percent: "5"}}, {{EffectiveOn: "2026-04-01", Percent: "-5"}},
		{{EffectiveOn: "2026-04-01", Percent: "0"}}, {{EffectiveOn: "2026-04-01", Percent: "1e9999"}},
		{{EffectiveOn: "2026-04-01", Percent: "5"}, {EffectiveOn: "2026-04-01", Percent: "6"}},
		{{EffectiveOn: "2027-04-01", Percent: "5"}, {EffectiveOn: "2026-04-01", Percent: "6"}},
	} {
		if err := ValidateStaffelSteps(steps); err == nil {
			t.Fatalf("accepted %+v", steps)
		}
	}
}

func TestStaffelTimingNeedsNoPublication(t *testing.T) {
	at := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	in := TimingInput{ContractSchedule: true, Mode: ContractualTiming, ContractualEffectiveOn: at, RequiresMRGNotice: true, DueDay: 5}
	got, err := Timing(in)
	if err != nil || got.DueOn.Format(time.DateOnly) != "2026-05-05" {
		t.Fatalf("%+v %v", got, err)
	}
	in.NoticeIssuedOn = at.AddDate(0, 0, -1)
	in.NoticeReceivedOn = in.NoticeIssuedOn
	if _, err := Timing(in); err == nil {
		t.Fatal("accepted premature notice")
	}
	in.ContractSchedule = false
	if _, err := Timing(in); err == nil {
		t.Fatal("index clause without publication accepted")
	}
}

func TestStaffelHalfCentUsesStatutoryRounding(t *testing.T) {
	at := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	contract, on, err := EvaluateStaffel(10000, 10000, []StaffelStep{{EffectiveOn: "2026-04-01", Percent: "0.005"}}, at)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyMieWeG(contract, StatutoryContext{Scope: ResidentialPartialMRG, CurrentAmountCents: 10000, ContractEffectiveOn: on, Ceiling: Ceiling{AmountCents: 10500, ExactAmountCents: "10500", EffectiveOn: at}}, at)
	if err != nil || got.AllowedNowCents != 10000 || contract.ExactAmountCents != "20001/2" {
		t.Fatalf("half-cent %+v %+v %v", contract, got, err)
	}
}
