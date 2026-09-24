package indexation

import (
	"testing"
	"time"
)

func TestTiming(t *testing.T) {
	for _, tc := range []struct{ name, issued, received, want string }{
		{"WKO earliest", "", "", "2026-09-05"},
		{"exact deadline", "2026-08-22", "2026-08-22", "2026-09-05"},
		{"late receipt slips", "2026-08-22", "2026-08-23", "2026-10-05"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := TimingInput{TriggerMonth: "2026-04", FinalPublishedOn: date("2026-06-17"), Mode: CautiousTiming, RequiresMRGNotice: true}
			if tc.issued != "" {
				in.NoticeIssuedOn = date(tc.issued)
				in.NoticeReceivedOn = date(tc.received)
			}
			got, err := Timing(in)
			if err != nil || got.IndexEffectiveOn != date("2026-08-01") || got.DueOn != date(tc.want) || got.RentFrom != monthOf(date(tc.want)) {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
	for _, tc := range []struct {
		mode   TimingMode
		notice bool
		want   string
	}{
		{ContractualTiming, true, "2026-07-05"},
		{ContractualTiming, false, "2026-07-05"},
		{CautiousTiming, false, "2026-08-05"},
	} {
		got, err := Timing(TimingInput{TriggerMonth: "2026-04", FinalPublishedOn: date("2026-06-17"), ContractualEffectiveOn: date("2026-06-17"), Mode: tc.mode, RequiresMRGNotice: tc.notice})
		if err != nil || got.DueOn != date(tc.want) {
			t.Fatalf("contract timing: %+v %v", got, err)
		}
	}
	for _, tc := range []struct{ received, want string }{{"2026-04-21", "2026-05-05"}, {"2026-04-22", "2026-06-05"}, {"2026-06-20", "2026-07-05"}} {
		got, err := Timing(TimingInput{TriggerMonth: "2025-12", FinalPublishedOn: date("2026-02-25"), Mode: MieWeGTiming, StatutoryEffectiveOn: date("2026-04-01"), RequiresMRGNotice: true, NoticeIssuedOn: date(tc.received), NoticeReceivedOn: date(tc.received)})
		if err != nil || got.DueOn != date(tc.want) {
			t.Fatalf("April timing: %+v %v", got, err)
		}
	}
}

func TestTimingRejectsPrematureAndInvalidLetters(t *testing.T) {
	base := TimingInput{TriggerMonth: "2026-04", FinalPublishedOn: date("2026-06-17"), Mode: CautiousTiming, RequiresMRGNotice: true, NoticeIssuedOn: date("2026-08-01"), NoticeReceivedOn: date("2026-08-01")}
	for _, change := range []func(*TimingInput){
		func(in *TimingInput) { in.NoticeIssuedOn = date("2026-07-31") },
		func(in *TimingInput) { in.NoticeReceivedOn = date("2026-07-31") },
		func(in *TimingInput) { in.NoticeIssuedOn = time.Time{} },
		func(in *TimingInput) { in.Mode = "" },
		func(in *TimingInput) { in.DueDay = 32 },
		func(in *TimingInput) { in.FinalPublishedOn = date("2026-04-17") },
		func(in *TimingInput) { in.Mode = ContractualTiming },
		func(in *TimingInput) { in.Mode = MieWeGTiming; in.StatutoryEffectiveOn = date("2026-05-01") },
	} {
		in := base
		change(&in)
		if _, err := Timing(in); err == nil {
			t.Fatalf("invalid timing accepted: %+v", in)
		}
	}
}

func TestTimingCivilDatesYearBoundaryAndLeapYear(t *testing.T) {
	location, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Timing(TimingInput{TriggerMonth: "2025-09", FinalPublishedOn: time.Date(2025, 11, 17, 0, 0, 0, 0, location), Mode: CautiousTiming, RequiresMRGNotice: true})
	if err != nil || got.IndexEffectiveOn != date("2026-01-01") || got.DueOn != date("2026-02-05") {
		t.Fatalf("year boundary: %+v %v", got, err)
	}
	if got := rentDue("2028-02", 31); got != date("2028-02-29") {
		t.Fatal(got)
	}
}
