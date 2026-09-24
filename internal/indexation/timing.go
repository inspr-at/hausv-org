package indexation

import (
	"fmt"
	"time"
)

type TimingMode string

const (
	CautiousTiming    TimingMode = "wko"
	OEVITiming        TimingMode = "oevi"
	ContractualTiming TimingMode = "contract"
	MieWeGTiming      TimingMode = "mieweg"
)

const TimingSource = "https://www.wko.at/wirtschaftsrecht/wertsicherung-miet-und-pachtvertraege"

type TimingInput struct {
	ContractSchedule       bool // Fixed dates have no index publication prerequisite.
	TriggerMonth           Month
	FinalPublishedOn       time.Time // Actual publication date, not an estimated t+45.
	Mode                   TimingMode
	NotBefore              time.Time  // A later periodic contract date still applies in WKO/ÖVI mode.
	LegalFloorMode         TimingMode // Contract mode in full MRG: organisation reading; contract/empty falls back to WKO.
	ContractualEffectiveOn time.Time  // Required in contractual mode; legally reviewed by caller.
	StatutoryEffectiveOn   time.Time  // April 1 from ApplyMieWeG; overrides cautious timing.
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
		if in.ContractualEffectiveOn.IsZero() || (in.Mode != ContractualTiming && in.Mode != MieWeGTiming && in.Mode != "contractual") {
			return TimingResult{}, fmt.Errorf("fixed schedule requires a contractual date and contractual or MieWeG timing")
		}
		final = civilDate(in.ContractualEffectiveOn)
	}
	var effective time.Time
	// Retain readability of previously frozen calculations.
	if in.Mode == "cautious" {
		in.Mode = CautiousTiming
	}
	if in.Mode == "contractual" {
		in.Mode = ContractualTiming
	}
	switch in.Mode {
	case CautiousTiming:
		effective = monthOf(final).date().AddDate(0, 2, 0)
	case OEVITiming:
		effective = final
	case ContractualTiming:
		if in.ContractualEffectiveOn.IsZero() {
			return TimingResult{}, fmt.Errorf("contractual effective date must be explicit and not precede final publication")
		}
		effective = civilDate(in.ContractualEffectiveOn)
		floor := final
		if in.RequiresMRGNotice && !in.ContractSchedule && in.LegalFloorMode != OEVITiming {
			floor = monthOf(final).date().AddDate(0, 2, 0)
		}
		if effective.Before(floor) {
			effective = floor
		}
	case MieWeGTiming:
		if in.StatutoryEffectiveOn.IsZero() || in.StatutoryEffectiveOn.Year() < 2026 || in.StatutoryEffectiveOn.Month() != time.April || in.StatutoryEffectiveOn.Day() != 1 || civilDate(in.StatutoryEffectiveOn).Before(final) {
			return TimingResult{}, fmt.Errorf("MieWeG requires April 1 after final publication")
		}
		effective = civilDate(in.StatutoryEffectiveOn)
	default:
		return TimingResult{}, fmt.Errorf("timing mode must be explicit")
	}
	if (in.Mode == CautiousTiming || in.Mode == OEVITiming) && !in.NotBefore.IsZero() && civilDate(in.NotBefore).After(effective) {
		effective = civilDate(in.NotBefore)
	}
	result := TimingResult{IndexEffectiveOn: effective, EarliestNoticeOn: effective, Mode: in.Mode, Source: TimingSource}
	if in.Mode == OEVITiming || (in.Mode == ContractualTiming && in.LegalFloorMode == OEVITiming) {
		result.Source = "https://www.ovi.at/aktuelles/detailansicht/mietpreisbremse-3-milg-im-nationalrat-beschlossen"
	}
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
		return TimingResult{}, fmt.Errorf("Schreiben gesperrt: Versand und Zugang dürfen nicht vor dem angenommenen Wirksamwerden am %s liegen; der Zugang darf nicht vor dem Versand liegen", effective.Format("02.01.2006"))
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
