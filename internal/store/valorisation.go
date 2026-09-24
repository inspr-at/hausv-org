package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/indexation"
)

type ValorisationSettings struct {
	WirksamwerdenMode      string `json:"wirksamwerden_mode"`
	UnreviewedClausePolicy string `json:"unreviewed_clause_policy"`
	FourEyes               bool   `json:"four_eyes"`
	LetterSenderText       string `json:"letter_sender_text"`
}

func DefaultValorisationSettings() ValorisationSettings {
	return ValorisationSettings{WirksamwerdenMode: "cautious", UnreviewedClausePolicy: "block"}
}
func (s ValorisationSettings) Normalized() ValorisationSettings {
	if s.WirksamwerdenMode != "contractual" {
		s.WirksamwerdenMode = "cautious"
	}
	if s.UnreviewedClausePolicy != "warn" {
		s.UnreviewedClausePolicy = "block"
	}
	return s
}

type ValorisationIndex struct {
	Series, Period, Value, Status, PublishedOn, Source, PublicationSource string
	Used                                                                  bool
}
type ValorisationInput struct {
	EffectiveOn                           string
	Settings                              ValorisationSettings
	Leases                                []Lease
	UnitLabels                            map[string]string
	Prior                                 map[string]ValorisationItem
	Organisation, House, Address, Contact string
}
type ValorisationRun struct {
	ID, TenantSlug, OrgKey, EffectiveOn, Status, InputsSHA256 string
	Revision                                                  int
	CreatedBy, ApprovedBy, CancelReason, CancelledBy          string
	CreatedAt, ApprovedAt                                     time.Time
	Input                                                     ValorisationInput
	IndexVersion, IndexSHA256                                 string
	IndexSnapshot                                             []ValorisationIndex
	Items                                                     []ValorisationItem
}
type ValorisationItem struct {
	ID, LeaseID, ClauseID, UnitID                          string
	UnitLabel                                              string
	Group, Outcome, Reason                                 string
	Exceptions, Explanation                                []string
	Lease                                                  Lease
	Clause                                                 IndexClause
	Recipients                                             []LeaseParty
	OldCents, ContractCents, CapCents, NewCents            int64
	VATRateBP                                              int
	OldVATCents, NewVATCents, OldGrossCents, NewGrossCents int64
	CapAnchor                                              indexation.Month
	CapStartCents                                          int64
	Contract                                               indexation.Evaluation
	CalculationSteps                                       []indexation.ExplanationStep
	Ceiling                                                indexation.Ceiling
	Decision                                               indexation.LimitedAdjustment
	TimingInput                                            indexation.TimingInput
	Timing                                                 indexation.TimingResult
	NextClause                                             indexation.Clause
	Indices                                                []ValorisationIndex
	WirksamOn, CollectableFrom, NoticeDeadline             string
	RequiresMRGNotice, MieWeG, SpecialCap                  bool
	Excluded                                               bool
	ExcludeReason, OverrideReason, OverrideBy              string
	OverrideCents                                          *int64
	LetterDocumentID, LetterSHA256                         string
}

var ValorisationExceptionLabels = map[string]string{
	"no_clause":              "Keine Wertsicherungsklausel vorhanden.",
	"clause_invalid":         "Die Klausel ist ungültig oder nicht maschinell abbildbar.",
	"clause_unreviewed":      "Die Klausel muss geprüft werden.",
	"one_way_clause_risk":    "Einseitige Klausel bei einem Verbraucher: rechtliche Prüfung erforderlich.",
	"index_preliminary":      "Ein benötigter Indexwert ist noch vorläufig.",
	"index_missing":          "Ein benötigter Indexwert oder sein Veröffentlichungsnachweis fehlt.",
	"index_derived":          "Abgeleiteter Indexwert: veröffentlichte Messzahl erforderlich.",
	"base_value_mismatch":    "Die Basis stimmt nicht mit der veröffentlichten Messzahl überein.",
	"max_hmz_exceeded":       "Die berechnete Miete überschreitet die geprüfte Mietzinsobergrenze.",
	"lease_ended":            "Der Mietvertrag ist zum Stichtag beendet.",
	"lease_not_started":      "Der Mietvertrag ist zum Stichtag noch nicht aktiv.",
	"missed_pre2026":         "Nicht geltend gemachte Erhöhung vor 2026: gesonderte Prüfung erforderlich.",
	"staffel_anchor_missing": "Bisherige Mietzinsänderung: geprüfter MieWeG-Anker für die Staffel erforderlich.",
	"mixed_use_check":        "Nutzung oder rechtliche Einordnung muss gesondert geprüft werden.",
	"no_recipient":           "Eine gültige E-Mail-Adresse einer Vertragspartei fehlt.",
	"letter_too_early":       "Ein Schreiben darf erst ab Wirksamkeit ausgestellt werden.",
}

func (i *ValorisationItem) exception(code string) {
	for _, c := range i.Exceptions {
		if c == code {
			return
		}
	}
	i.Exceptions = append(i.Exceptions, code)
	i.Group = "exception"
	i.Reason = ValorisationExceptionLabels[code]
	i.Explanation = append(i.Explanation, i.Reason)
}

// PreviewValorisation is the I/O-free adapter. Every rent decision and both
// curves come from indexation; this adapter validates lease data and freezes
// human-readable evidence. A caller can invoke it for each authorised house.
func PreviewValorisation(input ValorisationInput, snapshot indexation.Snapshot, now time.Time) (ValorisationRun, error) {
	effective, err := time.Parse(time.DateOnly, input.EffectiveOn)
	if err != nil || effective.Year() < 2026 || now.IsZero() {
		return ValorisationRun{}, fmt.Errorf("Stichtag ab 2026 und Berechnungsdatum erforderlich")
	}
	input.Settings = input.Settings.Normalized()
	input.Leases = append([]Lease(nil), input.Leases...)
	sort.Slice(input.Leases, func(i, j int) bool { return input.Leases[i].ID < input.Leases[j].ID })
	run := ValorisationRun{EffectiveOn: input.EffectiveOn, Status: "draft", Input: input, IndexVersion: snapshot.Manifest.Version, IndexSHA256: snapshot.Manifest.DataSHA256}
	evidence := map[string]ValorisationIndex{}
	for _, lease := range input.Leases {
		item := evaluateValorisationLease(lease, input, snapshot, effective, now)
		item.UnitLabel = input.UnitLabels[lease.UnitID]
		run.Items = append(run.Items, item)
		for _, v := range item.Indices {
			evidence[v.Series+"/"+v.Period] = v
		}
	}
	for _, v := range evidence {
		run.IndexSnapshot = append(run.IndexSnapshot, v)
	}
	sort.Slice(run.IndexSnapshot, func(i, j int) bool {
		a, b := run.IndexSnapshot[i], run.IndexSnapshot[j]
		return a.Series+a.Period < b.Series+b.Period
	})
	raw, err := json.Marshal(struct {
		Input        ValorisationInput
		Version, SHA string
		Values       []ValorisationIndex
	}{input, run.IndexVersion, run.IndexSHA256, run.IndexSnapshot})
	if err != nil {
		return run, err
	}
	run.InputsSHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
	return run, nil
}

func evaluateValorisationLease(lease Lease, input ValorisationInput, snapshot indexation.Snapshot, effective, now time.Time) ValorisationItem {
	item := ValorisationItem{LeaseID: lease.ID, UnitID: lease.UnitID, Lease: lease, Group: "unchanged", Outcome: "unchanged", Reason: "Keine vertragliche Anpassung fällig."}
	date := effective.Format(time.DateOnly)
	if lease.EndsOn != "" && lease.EndsOn <= date {
		item.exception("lease_ended")
	}
	if lease.StartsOn > date || lease.Status == LeaseStatusDraft {
		item.exception("lease_not_started")
	}
	if lease.UseKind == UseKindSonstiges || lease.MRGScope == MRGWGG {
		item.exception("mixed_use_check")
	}
	class := ClassifyLease(lease)
	item.MieWeG, item.SpecialCap = class.MieWeG, class.SpecialCap
	item.Explanation = append(item.Explanation, class.MieWeGReason, class.CapReason)
	for _, party := range lease.Parties {
		if party.ValidFrom > date || (party.ValidTo != "" && party.ValidTo <= date) {
			continue
		}
		item.Recipients = append(item.Recipients, party)
		parsed, err := mail.ParseAddress(party.Email)
		if err != nil || parsed.Address != party.Email {
			item.exception("no_recipient")
		}
	}
	if len(item.Recipients) == 0 {
		item.exception("no_recipient")
	}
	for _, c := range lease.Components {
		if c.Kind == ComponentHMZ && c.ValidFrom <= date {
			if item.OldCents == 0 || c.ValidFrom >= latestHMZDate(lease.Components, date) {
				item.OldCents, item.VATRateBP = c.NetCents, c.VATRateBP
			}
		}
	}
	item.NewCents = item.OldCents
	for _, c := range lease.Clauses {
		if c.ComponentKind == ComponentHMZ && c.ValidFrom <= date && c.ValidFrom >= item.Clause.ValidFrom {
			item.Clause = c
		}
	}
	clause := item.Clause
	item.ClauseID = clause.ID
	switch {
	case clause.ID == "" || clause.ClauseType == ClauseNone:
		item.exception("no_clause")
	case clause.ReviewStatus == ReviewInvalid:
		item.exception("clause_invalid")
	case clause.ReviewStatus == ReviewDoubtful || (clause.ReviewStatus != ReviewOK && input.Settings.UnreviewedClausePolicy == "block"):
		item.exception("clause_unreviewed")
	}
	if !clause.TwoWay && lease.TenantIsConsumer {
		item.exception("one_way_clause_risk")
	}
	if item.OldCents <= 0 && lease.UseKind == UseKindGarage && (clause.ID == "" || clause.ClauseType == ClauseNone) && len(item.Exceptions) == 1 && item.Exceptions[0] == "no_clause" {
		item.Exceptions = nil
		item.Group = "unchanged"
		item.Reason = "Nur Stellplatzentgelt; kein Hauptmietzins zur Anpassung."
		item.Explanation = []string{item.Reason}
		return finishValorisationItem(item)
	}
	if item.OldCents <= 0 && len(item.Exceptions) == 0 {
		item.exception("clause_invalid")
	}
	if len(item.Exceptions) > 0 {
		return finishValorisationItem(item)
	}
	if clause.ReviewStatus != ReviewOK {
		item.Explanation = append(item.Explanation, "Warnung: Klausel ist noch ungeprüft; Organisationsrichtlinie erlaubt die Vorschau.")
	}
	prior, hasPrior := input.Prior[clause.ID]
	anchor := indexation.Month(lease.ConcludedOn[:7])
	start := item.OldCents
	if clause.ClauseType == ClauseStaffel {
		var err error
		start, err = staffelStartCents(lease, clause)
		if err != nil || indexation.ValidateStaffelSteps(clause.StaffelSteps) != nil {
			item.exception("clause_invalid")
			return finishValorisationItem(item)
		}
		legacy := clause.StaffelSteps[0].EffectiveOn < "2026-01-01"
		missingAnchor := !hasPrior && (clause.State == nil || clause.State.CapAnchorPeriod == "" || clause.State.CapValue == "")
		changedSincePrior := hasPrior && item.OldCents != prior.NewCents
		if item.MieWeG && ((missingAnchor && (legacy || item.OldCents != start)) || changedSincePrior) {
			contract, _, err := indexation.EvaluateStaffel(start, item.OldCents, clause.StaffelSteps, effective)
			if err != nil {
				item.exception("clause_invalid")
				return finishValorisationItem(item)
			}
			if contract.Crossed {
				if legacy && !hasPrior {
					item.exception("missed_pre2026")
				} else {
					item.exception("staffel_anchor_missing")
				}
				item.Explanation = append(item.Explanation, "Bisherige Mietzinsänderungen benötigen einen geprüften MieWeG-Anker samt damaligem Hauptmietzins. Bitte den bisherigen Anpassungsstand ergänzen.")
				return finishValorisationItem(item)
			}
		}
	}
	if clause.State != nil && clause.State.CapAnchorPeriod != "" {
		anchor = indexation.Month(clause.State.CapAnchorPeriod)
		var err error
		start, err = valorisationEuroCents(clause.State.CapValue)
		if err != nil {
			item.exception("index_missing")
			return finishValorisationItem(item)
		}
	}
	if hasPrior {
		anchor, start = prior.CapAnchor, prior.CapStartCents
	}
	item.CapAnchor, item.CapStartCents = anchor, start
	if item.MieWeG {
		if effective.Year() <= mustMonthDate(anchor).Year() {
			item.Reason = "Erste Anpassung frühestens am 1. April des Folgejahres."
			return finishValorisationItem(item)
		}
		for _, avg := range snapshot.Annual {
			if avg.Series == indexation.VPI2020 && avg.Year >= mustMonthDate(anchor).Year()-1 && avg.Year < effective.Year() {
				item.Indices = append(item.Indices, annualEvidence(avg))
				if avg.Preliminary {
					item.exception("index_preliminary")
				}
			}
		}
		var err error
		item.Ceiling, err = indexation.CapCurve(anchor, start, snapshot.Annual, item.SpecialCap, effective.Year())
		if err != nil {
			item.exception("index_missing")
			return finishValorisationItem(item)
		}
		item.CapCents = item.Ceiling.AmountCents
		if clause.ClauseType == ClauseStaffel && item.CapCents < item.OldCents {
			item.exception("staffel_anchor_missing")
			item.Explanation = append(item.Explanation, "Die Deckelkurve liegt unter dem geltenden Hauptmietzins. Bitte den bisherigen Anpassungsstand prüfen.")
			return finishValorisationItem(item)
		}
		for _, step := range item.Ceiling.Years {
			item.Explanation = append(item.Explanation, fmt.Sprintf("%d: Jahresmittel %s → %s; Änderung %s; nach Begrenzung %s; %d von 12 Monaten berücksichtigt. Deckelkurve: %s (gerundet angezeigt).", step.Year, ValorisationNumber(step.PreviousAverage.String(), 1), ValorisationNumber(step.CurrentAverage.String(), 1), ValorisationPercent(step.RawRatePercent, 5), ValorisationPercent(step.LimitedRatePercent, 5), step.FullMonths, ValorisationExactMoney(step.ExactAmountCents)))
		}
	}
	var contract indexation.Evaluation
	var contractualOn, published time.Time
	var trigger indexation.Month
	if clause.ClauseType == ClauseStaffel {
		base, err := staffelStartCents(lease, clause)
		if err != nil {
			item.exception("clause_invalid")
			return finishValorisationItem(item)
		}
		contract, contractualOn, err = indexation.EvaluateStaffel(base, item.OldCents, clause.StaffelSteps, effective)
		if err == nil && item.MieWeG && contract.NewAmountCents > item.OldCents && contractualOn.After(item.Ceiling.EffectiveOn) {
			// An increase after April cannot enter the preceding April curve.
			contract, contractualOn, err = indexation.EvaluateStaffel(base, item.OldCents, clause.StaffelSteps, item.Ceiling.EffectiveOn)
			item.Reason = "Weitere Staffelstufen werden frühestens am nächsten 1. April berücksichtigt."
		}
		if err != nil {
			item.exception("clause_invalid")
			return finishValorisationItem(item)
		}
		item.CalculationSteps = contract.Explanation
		for n, step := range contract.Explanation {
			scheduled := clause.StaffelSteps[n]
			basis := "neuer HMZ netto"
			if scheduled.Percent != "" {
				basis = "+" + strings.ReplaceAll(scheduled.Percent, ".", ",") + " % auf den vorigen Vertragsbetrag"
			}
			item.Explanation = append(item.Explanation, fmt.Sprintf("Staffelmietzins laut Vertrag ab %s: %s; vertraglich vereinbarter Hauptmietzins netto %s.", mustDate(scheduled.EffectiveOn).Format("02.01.2006"), basis, ValorisationExactMoney(step.ExactAmountCents)))
		}
		if !contractualOn.IsZero() && contractualOn.Year() < 2026 && (clause.State == nil || clause.State.LastEffectiveOn < contractualOn.Format(time.DateOnly)) && contract.Crossed {
			item.exception("missed_pre2026")
			return finishValorisationItem(item)
		}
	} else if clause.ClauseType == ClauseMieWeG {
		if !item.MieWeG {
			item.exception("clause_invalid")
			return finishValorisationItem(item)
		}
		contract = indexation.Evaluation{Crossed: item.CapCents != item.OldCents, OldAmountCents: item.OldCents, NewAmountCents: item.CapCents, ExactAmountCents: item.Ceiling.ExactAmountCents}
		contractualOn = item.Ceiling.EffectiveOn
		trigger = indexation.Month(fmt.Sprintf("%d-12", effective.Year()-1))
		published, _, _ = finalIndexPublication(trigger)
	} else {
		core, err := valorisationCoreClause(clause, item.OldCents)
		if hasPrior {
			core = prior.NextClause
			err = nil
		}
		if err != nil {
			item.exception("base_value_mismatch")
			return finishValorisationItem(item)
		}
		base, found, err := snapshot.Data.Lookup(core.Series, core.BaseMonth)
		if err != nil || !found {
			item.exception("index_missing")
			return finishValorisationItem(item)
		}
		evidence := indexEvidence(base)
		evidence.Used = true
		item.Indices = append(item.Indices, evidence)
		if base.Preliminary {
			item.exception("index_preliminary")
		}
		if base.ChainSource != "" {
			item.exception("index_derived")
		}
		if base.Value != core.BaseValue {
			item.exception("base_value_mismatch")
		}
		if len(item.Exceptions) > 0 {
			return finishValorisationItem(item)
		}
		cutoff := core.BaseMonth
		// Freeze the last observation whose evidenced contractual effect can fall
		// within V. A later preliminary publication is not a needed input.
		for _, v := range snapshot.Data.Values() {
			if v.Series != core.Series || v.Month <= core.BaseMonth {
				continue
			}
			pub, _, ok := finalIndexPublication(v.Month)
			if !ok {
				continue
			}
			on := pub
			if !item.MieWeG && lease.MRGScope == MRGVoll && input.Settings.WirksamwerdenMode == "cautious" {
				on = time.Date(pub.Year(), pub.Month()+2, 1, 0, 0, 0, 0, time.UTC)
			}
			if !on.After(effective) && !pub.After(now) && v.Month > cutoff {
				cutoff = v.Month
			}
		}
		type periodicReference struct {
			month     indexation.Month
			effective time.Time
		}
		var references []periodicReference
		periodic := clause.ClauseType == ClauseVPIPeriodic
		if periodic {
			if clause.PeriodicMonth < 1 || clause.ReferenceMonthOffset == nil || *clause.ReferenceMonthOffset >= 0 || *clause.ReferenceMonthOffset < -24 {
				item.exception("clause_invalid")
				item.Explanation = append(item.Explanation, "Periodische Klausel benötigt einen geprüften Referenzmonat (1 bis 24 Monate vor dem Anpassungsmonat).")
				return finishValorisationItem(item)
			}
			core.Threshold, core.ThresholdKind, core.ChangeMode = 0, indexation.PercentThreshold, indexation.FullChange
			for year := mustMonthDate(core.BaseMonth).Year(); year <= effective.Year(); year++ {
				on := time.Date(year, time.Month(clause.PeriodicMonth), 1, 0, 0, 0, 0, time.UTC)
				ref := indexation.Month(on.AddDate(0, *clause.ReferenceMonthOffset, 0).Format("2006-01"))
				if on.After(effective) || ref <= core.BaseMonth {
					continue
				}
				pub, _, ok := finalIndexPublication(ref)
				if !ok || pub.After(on) || pub.After(now) {
					item.exception("index_missing")
					return finishValorisationItem(item)
				}
				references = append(references, periodicReference{ref, on})
			}
		}
		contract = indexation.Evaluation{OldAmountCents: core.AmountCents, NewAmountCents: core.AmountCents, ExactAmountCents: core.ExactAmountCents, NextClause: core}
		referenceIndex := 0
		for {
			target := cutoff
			var periodicOn time.Time
			if periodic {
				if referenceIndex >= len(references) {
					break
				}
				target, periodicOn = references[referenceIndex].month, references[referenceIndex].effective
				referenceIndex++
			} else if core.BaseMonth >= cutoff {
				break
			}
			var result indexation.Evaluation
			var err error
			if periodic {
				result, err = indexation.EvaluateReference(core, snapshot.Data, target)
			} else {
				result, err = indexation.Evaluate(core, snapshot.Data, target)
			}
			if err != nil {
				code := "index_missing"
				if strings.Contains(err.Error(), "index_derived") {
					code = "index_derived"
				}
				item.exception(code)
				return finishValorisationItem(item)
			}
			item.CalculationSteps = append(item.CalculationSteps, result.Explanation...)
			for _, step := range result.Explanation {
				if v, ok, _ := snapshot.Data.Lookup(core.Series, step.Month); ok {
					evidence := indexEvidence(v)
					evidence.Used = step.Code == "base_final" || step.Code == "threshold_crossed" || (!result.Crossed && !contract.Crossed && step.Month == result.EvaluatedThrough)
					item.Indices = append(item.Indices, evidence)
				}
				if step.Code == "threshold_crossed" {
					item.Explanation = append(item.Explanation, valorisationContractExplanation(step, clause))
				}
			}
			if result.PendingMonth != "" {
				v, ok, _ := snapshot.Data.Lookup(core.Series, result.PendingMonth)
				if ok && v.Preliminary {
					item.exception("index_preliminary")
				} else {
					item.exception("index_missing")
				}
				return finishValorisationItem(item)
			}
			if !result.Crossed {
				if !contract.Crossed {
					contract = result
					item.Reason = fmt.Sprintf("Schwelle %s %s nicht erreicht: %s.", ValorisationNumber(clause.ThresholdValue, 2), ValorisationThresholdUnit(clause.ThresholdKind), ValorisationPercent(result.ChangePercent.String(), 2))
				}
				if periodic {
					continue
				}
				break
			}
			contract = result
			core = result.NextClause
			trigger = result.TriggerMonth
			published, _, _ = finalIndexPublication(trigger)
			if published.IsZero() {
				item.exception("index_missing")
				return finishValorisationItem(item)
			}
			contractualOn = time.Date(published.Year(), published.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			if periodic {
				contractualOn = periodicOn
			}
			if contractualOn.Year() < 2026 && (clause.State == nil || clause.State.LastEffectiveOn < contractualOn.Format(time.DateOnly)) {
				item.exception("missed_pre2026")
				return finishValorisationItem(item)
			}
		}
		item.NextClause = core
		// A capped contract curve remains independent. A subsequent April may
		// admit more of that same curve without a new threshold crossing.
		if !contract.Crossed && hasPrior && core.AmountCents > item.OldCents {
			contract.Crossed = true
			contract.NewAmountCents = core.AmountCents
			contract.ExactAmountCents = core.ExactAmountCents
			trigger = core.BaseMonth
			published, _, _ = finalIndexPublication(trigger)
			contractualOn = effective
		}
	}
	item.Contract = contract
	item.ContractCents = contract.NewAmountCents
	if !clause.TwoWay && contract.NewAmountCents < item.OldCents {
		item.Reason = "Die geprüfte einseitige Klausel sieht keine Senkung vor."
		return finishValorisationItem(item)
	}
	if !contract.Crossed {
		return finishValorisationItem(item)
	}
	if clause.ClauseType != ClauseStaffel && (published.IsZero() || published.After(now) || published.After(effective)) {
		item.exception("index_missing")
		return finishValorisationItem(item)
	}
	scope := indexation.Commercial
	if lease.UseKind == UseKindWohnung {
		scope = indexation.ResidentialExcluded
		if lease.MRGScope == MRGVoll {
			scope = indexation.ResidentialFullMRG
		}
		if lease.MRGScope == MRGTeil {
			scope = indexation.ResidentialPartialMRG
		}
	} else if lease.MRGScope == MRGVoll {
		scope = indexation.CommercialFullMRG
	}
	var err error
	item.Decision, err = indexation.ApplyMieWeG(contract, indexation.StatutoryContext{Scope: scope, IsSublease: lease.LeaseKind == LeaseKindUntermiete, CurrentAmountCents: item.OldCents, ContractEffectiveOn: contractualOn, Ceiling: item.Ceiling, MaxRentCents: lease.MaxHMZCents}, effective)
	if err != nil {
		code := "clause_invalid"
		if strings.Contains(err.Error(), "missed_pre2026") {
			code = "missed_pre2026"
		}
		item.exception(code)
		return finishValorisationItem(item)
	}
	item.NewCents = item.Decision.AllowedNowCents
	if clause.ClauseType == ClauseStaffel && item.MieWeG && item.Decision.CappedCents > 0 {
		item.Explanation = append(item.Explanation, fmt.Sprintf("MieWeG begrenzt die Staffel: %s bleiben oberhalb des zulässigen Betrags. Die Vertragskurve bleibt erhalten; der begrenzte Teil ist keine Nachforderung.", ValorisationExactMoney(fmt.Sprint(item.Decision.CappedCents))))
	}
	for _, step := range item.Decision.Explanation {
		if step.Code == "section_16_ceiling" {
			item.exception("max_hmz_exceeded")
		}
	}
	item.RequiresMRGNotice = item.Decision.RequiresMRGNotice && item.NewCents > item.OldCents
	mode := indexation.ContractualTiming
	if item.MieWeG && item.NewCents > item.OldCents {
		mode = indexation.MieWeGTiming
	} else if item.RequiresMRGNotice && clause.ClauseType != ClauseStaffel {
		mode = indexation.TimingMode(input.Settings.WirksamwerdenMode)
	}
	item.TimingInput = indexation.TimingInput{TriggerMonth: trigger, FinalPublishedOn: published, Mode: mode, ContractualEffectiveOn: contractualOn, StatutoryEffectiveOn: item.Decision.EffectiveOn, RequiresMRGNotice: item.RequiresMRGNotice, DueDay: lease.ZinsterminDay}
	item.TimingInput.ContractSchedule = clause.ClauseType == ClauseStaffel
	if item.TimingInput.ContractSchedule && item.MieWeG && item.Decision.PermittedCents > item.OldCents {
		item.TimingInput.Mode = indexation.MieWeGTiming
	}
	if mode == indexation.ContractualTiming && !item.MieWeG && clause.ClauseType != ClauseStaffel {
		item.TimingInput.ContractualEffectiveOn = effective
	}
	item.Timing, err = indexation.Timing(item.TimingInput)
	if err != nil {
		item.exception("index_missing")
		return finishValorisationItem(item)
	}
	item.WirksamOn = item.Timing.IndexEffectiveOn.Format(time.DateOnly)
	if item.WirksamOn > date {
		item.NewCents = item.OldCents
		item.Reason = "Anpassung wird erst später wirksam."
	}
	if item.NewCents != item.OldCents {
		if item.Group != "exception" {
			item.Group = "ready"
		}
		item.Outcome = "increase"
		item.Reason = "Erhöhung aus der vertraglichen Wertsicherung."
		if item.NewCents < item.OldCents {
			item.Outcome = "decrease"
			item.Reason = "Verminderung aus der vertraglichen Wertsicherung."
		}
		if now.Format(time.DateOnly) < item.WirksamOn {
			item.exception("letter_too_early")
		}
	}
	item.Explanation = append(item.Explanation, "Vertrags- und Deckelkurve werden unabhängig fortgeführt. Nur der Endbetrag wird gerundet; ein halber Cent wird bei MieWeG abgerundet.")
	return finishValorisationItem(item)
}

func latestHMZDate(items []RentComponent, date string) string {
	last := ""
	for _, c := range items {
		if c.Kind == ComponentHMZ && c.ValidFrom <= date && c.ValidFrom > last {
			last = c.ValidFrom
		}
	}
	return last
}
func mustMonthDate(m indexation.Month) time.Time { t, _ := time.Parse("2006-01", string(m)); return t }
func valorisationEuroCents(raw string) (int64, error) {
	d, err := indexation.ParseDecimal(raw)
	if err != nil || d < 0 || d%(indexation.Unit/100) != 0 {
		return 0, fmt.Errorf("invalid cent amount")
	}
	return int64(d / (indexation.Unit / 100)), nil
}
func valorisationCoreClause(c IndexClause, amount int64) (indexation.Clause, error) {
	out := indexation.Clause{Series: indexation.Series(strings.ToUpper(c.Series)), BaseMonth: indexation.Month(c.BasePeriod), AmountCents: amount, ThresholdKind: indexation.ThresholdKind(c.ThresholdKind), PercentRounding: indexation.UnroundedPercent}
	base, threshold := c.BaseValue, c.ThresholdValue
	if c.State != nil && c.State.ContractBasePeriod != "" {
		out.BaseMonth = indexation.Month(c.State.ContractBasePeriod)
		base = c.State.ContractBaseValue
		var err error
		out.AmountCents, err = valorisationEuroCents(c.State.ContractValue)
		if err != nil {
			return out, err
		}
	}
	var err error
	out.BaseValue, err = indexation.ParseDecimal(base)
	if err != nil {
		return out, err
	}
	if threshold == "" {
		threshold = "0"
	}
	out.Threshold, err = indexation.ParseDecimal(threshold)
	if c.ThresholdInclusive {
		out.Crossing = indexation.Reaches
	}
	if !c.FullChangeOnTrigger {
		out.ChangeMode = indexation.ExcessChange
	}
	if c.PctRounding == "one_decimal" {
		out.PercentRounding = indexation.OneDecimalPercent
	}
	return out, err
}
func finishValorisationItem(i ValorisationItem) ValorisationItem {
	seen := map[string]int{}
	indices := []ValorisationIndex{}
	for _, index := range i.Indices {
		key := index.Series + "/" + index.Period
		if pos, ok := seen[key]; ok {
			indices[pos].Used = indices[pos].Used || index.Used
		} else {
			seen[key] = len(indices)
			indices = append(indices, index)
		}
	}
	i.Indices = indices

	i.OldVATCents = ValorisationVAT(i.OldCents, i.VATRateBP)
	i.NewVATCents = ValorisationVAT(i.NewCents, i.VATRateBP)
	i.OldGrossCents = i.OldCents + i.OldVATCents
	i.NewGrossCents = i.NewCents + i.NewVATCents
	if !i.Timing.DueOn.IsZero() {
		i.CollectableFrom = i.Timing.DueOn.Format(time.DateOnly)
	}
	if !i.Timing.NoticeDeadline.IsZero() {
		i.NoticeDeadline = i.Timing.NoticeDeadline.Format(time.DateOnly)
	}
	return i
}

// VAT is commercial half-up cent rounding, independent of indexation rounding.
func ValorisationVAT(net int64, bp int) int64 {
	n := new(big.Int).Mul(big.NewInt(net), big.NewInt(int64(bp)))
	n.Add(n, big.NewInt(5000))
	n.Quo(n, big.NewInt(10000))
	return n.Int64()
}
func ValorisationLetterTiming(item ValorisationItem, at time.Time) (ValorisationItem, error) {
	if item.WirksamOn == "" || at.Format(time.DateOnly) < item.WirksamOn {
		return item, fmt.Errorf("Schreiben vor Wirksamkeit unzulässig")
	}
	in := item.TimingInput
	in.NoticeIssuedOn = at
	in.NoticeReceivedOn = at
	timing, err := indexation.Timing(in)
	if err != nil {
		return item, err
	}
	item.Timing = timing
	return finishValorisationItem(item), nil
}
