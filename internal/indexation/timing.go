package indexation

import (
	"fmt"
	"time"
)

type TimingMode string

const (
	CautiousTiming    TimingMode = "cautious"
	ContractualTiming TimingMode = "contractual"
	MieWeGTiming      TimingMode = "mieweg"
)

const TimingSource = "https://www.wko.at/wirtschaftsrecht/wertsicherung-miet-und-pachtvertraege"

type TimingInput struct {
	ContractSchedule       bool // Fixed dates have no index publication prerequisite.
	TriggerMonth           Month
	FinalPublishedOn       time.Time // Actual publication date, not an estimated t+45.
	Mode                   TimingMode
	ContractualEffectiveOn time.Time // Required in contractual mode; legally reviewed by caller.
	StatutoryEffectiveOn   time.Time // April 1 from ApplyMieWeG; overrides cautious timing.
	RequiresMRGNotice      bool
	NoticeIssuedOn         time.Time // Both notice dates may be zero to plan earliest delivery.
	NoticeReceivedOn       time.Time
	DueDay                 int // Default 5; days 29..31 clamp to the month's last day.
}

type TimingResult struct {
	IndexEffectiveOn time.Time
	EarliestNoticeOn time.Time
	RentFrom         Month
	DueOn            time.Time
	NoticeDeadline   time.Time
	PlannedNotice    bool
	Mode             TimingMode
	Source           string
}

// Timing plans or checks notice dates. A premature letter is rejected; it is
// never silently moved forward to make an invalid demand appear valid.
// This helper does not decide MieWeG applicability or apply an April shift.
func Timing(in TimingInput) (TimingResult, error) {
	if !in.ContractSchedule && (!in.TriggerMonth.Valid() || in.FinalPublishedOn.IsZero() || monthOf(in.FinalPublishedOn) <= in.TriggerMonth) {
		return TimingResult{}, fmt.Errorf("trigger month and subsequent final publication date required")
	}
	day := in.DueDay
	if day == 0 {
		day = 5
	}
	if day < 1 || day > 31 {
		return TimingResult{}, fmt.Errorf("invalid rent due day")
	}
	final := civilDate(in.FinalPublishedOn)
	if in.ContractSchedule {
		if in.ContractualEffectiveOn.IsZero() || in.Mode == CautiousTiming {
			return TimingResult{}, fmt.Errorf("fixed schedule requires a contractual date and contractual or MieWeG timing")
		}
		final = civilDate(in.ContractualEffectiveOn)
	}
	var effective time.Time
	switch in.Mode {
	case CautiousTiming:
		effective = monthOf(final).date().AddDate(0, 2, 0)
	case ContractualTiming:
		if in.ContractualEffectiveOn.IsZero() || civilDate(in.ContractualEffectiveOn).Before(final) {
			return TimingResult{}, fmt.Errorf("contractual effective date must be explicit and not precede final publication")
		}
		effective = civilDate(in.ContractualEffectiveOn)
	case MieWeGTiming:
		if in.StatutoryEffectiveOn.IsZero() || in.StatutoryEffectiveOn.Year() < 2026 || in.StatutoryEffectiveOn.Month() != time.April || in.StatutoryEffectiveOn.Day() != 1 || civilDate(in.StatutoryEffectiveOn).Before(final) {
			return TimingResult{}, fmt.Errorf("MieWeG requires April 1 after final publication")
		}
		effective = civilDate(in.StatutoryEffectiveOn)
	default:
		return TimingResult{}, fmt.Errorf("timing mode must be explicit")
	}
	result := TimingResult{IndexEffectiveOn: effective, EarliestNoticeOn: effective, Mode: in.Mode, Source: TimingSource}
	if in.Mode == MieWeGTiming {
		result.Source = MieWeGNotesSource
	}
	if !in.RequiresMRGNotice {
		result.RentFrom = monthOf(effective)
		result.DueOn = rentDue(result.RentFrom, day)
		if result.DueOn.Before(effective) {
			result.RentFrom = result.RentFrom.next()
			result.DueOn = rentDue(result.RentFrom, day)
		}
		return result, nil
	}
	issued, received := in.NoticeIssuedOn, in.NoticeReceivedOn
	if issued.IsZero() != received.IsZero() {
		return TimingResult{}, fmt.Errorf("both notice issue and receipt dates required")
	}
	if issued.IsZero() {
		issued, received, result.PlannedNotice = effective, effective, true
	}
	issued, received = civilDate(issued), civilDate(received)
	if issued.Before(effective) || received.Before(effective) || received.Before(issued) {
		return TimingResult{}, fmt.Errorf("notice invalid: issued or received before effectiveness, or receipt before issue")
	}
	firstDue := received.AddDate(0, 0, 14)
	month := monthOf(firstDue)
	due := rentDue(month, day)
	if due.Before(firstDue) {
		month = month.next()
		due = rentDue(month, day)
	}
	result.RentFrom, result.DueOn, result.NoticeDeadline = month, due, due.AddDate(0, 0, -14)
	return result, nil
}

func rentDue(month Month, day int) time.Time {
	first := month.date()
	lastDay := first.AddDate(0, 1, -1).Day()
	return time.Date(first.Year(), first.Month(), min(day, lastDay), 0, 0, 0, 0, time.UTC)
}
