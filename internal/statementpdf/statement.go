// Package statementpdf renders drafts and approved immutable statement runs.
package statementpdf

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/inspr-at/hausv-org/internal/pdf"
	"github.com/inspr-at/hausv-org/internal/store"
	"github.com/inspr-at/hausv-org/internal/view"
)

const DraftNotice = "Entwurf zur Prüfung — keine Rechtsauskunft nach WEG/MRG"

var ErrNotFound = errors.New("statement document not found")

type CostRow struct {
	Name, Total, Key, Share, Amount string
	Net, Rate, VAT, Gross           string
	Measurements                    []string
}
type Document struct {
	UnitID, PartyID, UnitLabel string
	Sender, Address, Basis     []string
	Info                       []InfoField
	Reference, Timing          string
	Costs                      []CostRow
	ShowVAT                    bool
	VATSummary                 []string
	Reserve                    []string
	Total, Prepaid, Balance    string
	Excluded                   []string
	Contact                    string
	ApprovalNotice             string
	PaymentTerms               []string
	Proposals                  []string
	Inspection                 []string
	InspectionBasis            string
	Receipts                   []string
	Title                      string
}

// Documents selects only identities recorded in this run. Empty selectors mean
// all documents; a partial selector never broadens into a bulk download.
func Documents(run store.AnnualStatementRun, unitID, partyID string) ([]Document, error) {
	if (unitID == "") != (partyID == "") {
		return nil, ErrNotFound
	}
	units := store.AnnualStatementRunDisplayOrder(run)
	parties := append([]store.AnnualStatementRunParty(nil), run.Input.Parties...)
	sort.SliceStable(parties, func(i, j int) bool { return parties[i].ID < parties[j].ID })
	var out []Document
	for _, unit := range units {
		if unitID != "" && unit.UnitID != unitID {
			continue
		}
		found := false
		for _, party := range parties {
			if party.UnitID != unit.UnitID || (partyID != "" && party.ID != partyID) {
				continue
			}
			out = append(out, document(run, unit, party))
			found = true
		}
		// A bulk export must never silently omit units without recorded recipients.
		if unitID == "" && !found {
			return nil, ErrNotFound
		}
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return out, nil
}

func document(run store.AnnualStatementRun, unit store.AnnualStatementRunUnit, party store.AnnualStatementRunParty) Document {
	unit, vacancyNote := annualStatementPartyUnit(run, unit, party)
	d := letterDocument(run, unit.Label)
	d.UnitID, d.PartyID = unit.UnitID, party.ID
	d.Total, d.Prepaid = money(unit.AllocatedCents), money(unit.PrepaidCents)
	d.Timing = balanceTiming(run, unit)
	role := "Wohnungseigentümer"
	if party.Renter {
		role = "Mietpartei"
		if party.Owner {
			role = "Wohnungseigentümer und Mietpartei"
		}
	}
	d.Address = nonempty(role, party.Name, party.Address)
	d.Basis = []string{"Einheit: " + unit.Label}
	for _, basis := range run.Input.Structure.UnitBases {
		if basis.UnitID != unit.UnitID {
			continue
		}
		d.Basis = append(d.Basis, "Nutzwert / Miteigentumsanteil: "+view.FormatMiteigentumsanteil(basis.MiteigentumsanteilPPM))
		if basis.UsableAreaRecorded {
			d.Basis = append(d.Basis, "Nutzfläche: "+view.FormatDecimal(float64(basis.UsableAreaM2Hundredths)/100, 2)+" m²")
		}
		if basis.PersonsRecorded {
			d.Basis = append(d.Basis, fmt.Sprintf("Personen: %d", basis.Persons))
		}
		if vacancyNote != "" && basis.UnitID == unit.UnitID {
			d.Basis = append(d.Basis, vacancyNote)
		}
	}
	switch {
	case unit.BalanceCents > 0:
		d.Balance = "Nachzahlung " + money(unit.BalanceCents)
	case unit.BalanceCents < 0:
		d.Balance = "Guthaben " + money(-unit.BalanceCents)
	default:
		d.Balance = "Ausgeglichen 0,00 €"
	}
	totals := map[string]int64{}
	for _, receipt := range run.Input.Receipts {
		totals[receipt.CostTypeKey] += receipt.AmountCents
	}
	for _, cost := range unit.Costs {
		row := CostRow{Name: cost.Name, Total: money(totals[cost.CostTypeKey]), Key: allocationKey(cost.AllocationKey), Share: view.FormatDecimal(float64((cost.SharePPM+50)/100)/100, 2) + " %", Amount: money(cost.AmountCents)}
		if cost.AllocationKey == store.AllocationKeyVerbrauch {
			vector := run.Input.Consumption[cost.CostTypeKey]
			total := new(big.Int)
			var value int64
			var measurementUnit string
			for _, measured := range vector.Units {
				total.Add(total, big.NewInt(measured.ValueMicros))
				if measured.UnitID == unit.UnitID {
					value = measured.ValueMicros
					measurementUnit = measured.MeasurementUnit
				}
			}
			row.Measurements = append(row.Measurements, "Messbasis: "+measurement(value, measurementUnit)+" / "+measurementTotal(total, measurementUnit)+" gesamt")
			evidence := []store.AnnualStatementConsumptionEvidence{}
			for _, item := range run.Input.Evidence {
				if item.UnitID == unit.UnitID && item.CostTypeKey == cost.CostTypeKey {
					evidence = append(evidence, item)
				}
			}
			sort.SliceStable(evidence, func(i, j int) bool { return evidence[i].MeasuredAt.Before(evidence[j].MeasuredAt) })
			for i, item := range evidence {
				label := "Grenzmessung Beginn"
				if i > 0 {
					label = "Grenzmessung Ende"
				}
				row.Measurements = append(row.Measurements, label+": "+timestamp(item.MeasuredAt)+" · "+measurement(item.ValueMicros, item.MeasurementUnit), "Quelle: "+item.SourceKind)
			}
			if len(evidence) != 2 {
				row.Measurements = append(row.Measurements, "Grenzmessungen im gespeicherten Lauf unvollständig")
			}
		}
		if run.CalculationVersion >= 2 && run.Input.Structure.Legal.HeizKGApplies && store.IsAnnualHeatingCost(cost.CostTypeKey) {
			row.Key = "HeizKG"
			// A two-pool split has no single allocation share. Consumption and
			// area shares are explained separately in the measurement appendix.
			row.Share = "im Anhang"
			prepaid := run.Input.Structure.Legal.HeatingPrepayments[unit.UnitID][cost.CostTypeKey]
			if vacancyNote != "" && party.Owner && !party.Renter {
				prepaid = 0 // The occupied party keeps the recorded prepayments.
			}
			row.Measurements = append(row.Measurements, heatingDetails(run, unit, cost, prepaid)...)
		}
		if run.Input.Structure.Legal.ShowVAT {
			row.Net = money(cost.NetCents)
			row.Rate = fmt.Sprintf("%d %%", cost.VATRatePercent)
			row.VAT = money(cost.VATCents)
			row.Gross = money(cost.AmountCents)
			d.ShowVAT = true
		}
		d.Costs = append(d.Costs, row)
	}
	if d.ShowVAT {
		for _, group := range unit.VAT {
			d.VATSummary = append(d.VATSummary, fmt.Sprintf("%d %% · Netto %s · USt %s · Brutto %s", group.RatePercent, money(group.NetCents), money(group.VATCents), money(group.GrossCents)))
		}
	}
	for _, cost := range run.Input.Structure.CostTypes {
		if !cost.Allocatable {
			d.Excluded = append(d.Excluded, cost.Name+": "+money(totals[cost.Key]))
		}
	}
	if len(d.Excluded) > 0 {
		d.Excluded = append(d.Excluded, "Gesamt nicht umlagefähig: "+money(run.Result.ExcludedCents))
	}
	d.PaymentTerms = paymentTerms(run, unit)
	d.Reserve = ReserveLines(run, unit.UnitID)
	d.Proposals = proposalLines(run, unit.UnitID)
	d.Inspection, d.Receipts = inspectionAppendix(run)
	return d
}

// annualStatementPartyUnit gives the landlord the vacant-day slice and the
// tenant the occupied remainder. A party who is both sees both slices.
func annualStatementPartyUnit(run store.AnnualStatementRun, unit store.AnnualStatementRunUnit, party store.AnnualStatementRunParty) (store.AnnualStatementRunUnit, string) {
	var line *store.AnnualStatementVacancyLine
	for i := range run.Result.Vacancy {
		if run.Result.Vacancy[i].UnitID == unit.UnitID {
			line = &run.Result.Vacancy[i]
			break
		}
	}
	if line == nil {
		return unit, ""
	}
	note := fmt.Sprintf("%s: %s–%s (%d von %d Tagen)", store.AnnualStatementVacancyLabel, date(line.From), date(line.To), line.VacantDays, line.PeriodDays)
	if party.Owner && !party.Renter {
		owner := unit
		owner.Costs = append([]store.AnnualStatementRunCost(nil), line.Costs...)
		owner.VAT = line.VAT
		owner.AllocatedCents = line.AmountCents
		owner.PrepaidCents = 0
		owner.BalanceCents = line.AmountCents
		return owner, note
	}
	if party.Owner && party.Renter {
		combined := unit
		combined.Costs = append([]store.AnnualStatementRunCost(nil), unit.Costs...)
		for _, vacant := range line.Costs {
			for i := range combined.Costs {
				if combined.Costs[i].CostTypeKey == vacant.CostTypeKey {
					combined.Costs[i].AmountCents += vacant.AmountCents
					combined.Costs[i].NetCents += vacant.NetCents
					combined.Costs[i].VATCents += vacant.VATCents
					break
				}
			}
		}
		combined.VAT = append([]store.AnnualStatementVATGroup(nil), unit.VAT...)
		for _, vacant := range line.VAT {
			found := false
			for i := range combined.VAT {
				if combined.VAT[i].RatePercent == vacant.RatePercent {
					combined.VAT[i].NetCents += vacant.NetCents
					combined.VAT[i].VATCents += vacant.VATCents
					combined.VAT[i].GrossCents += vacant.GrossCents
					found = true
					break
				}
			}
			if !found {
				combined.VAT = append(combined.VAT, vacant)
			}
		}
		sort.Slice(combined.VAT, func(i, j int) bool { return combined.VAT[i].RatePercent < combined.VAT[j].RatePercent })
		combined.AllocatedCents += line.AmountCents
		combined.BalanceCents = combined.AllocatedCents - combined.PrepaidCents
		return combined, note
	}
	return unit, ""
}

func Render(run store.AnnualStatementRun, unitID, partyID string) ([]byte, error) {
	documents, err := Documents(run, unitID, partyID)
	if err != nil {
		return nil, err
	}
	var pages []pdf.Page
	for _, d := range documents {
		pages = append(pages, d.Pages()...)
	}
	return pdf.Pages(pages, statementPalette), nil
}

func nonempty(values ...string) []string {
	var out []string
	for _, value := range values {
		var lines []string
		for _, line := range strings.Split(value, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				lines = append(lines, line)
			}
		}
		if len(lines) > 0 {
			out = append(out, strings.Join(lines, "\n"))
		}
	}
	return out
}
func money(cents int64) string { return view.FormatEURCents(cents) }
func measurement(micros int64, unit string) string {
	return measurementTotal(big.NewInt(micros), unit)
}

// A vector's total can exceed int64 even though every stored counter fits.
// Preserve the six stored decimal places without floating-point rounding.
func measurementTotal(micros *big.Int, unit string) string {
	raw := new(big.Rat).SetFrac(micros, big.NewInt(1_000_000)).FloatString(6)
	parts := strings.SplitN(raw, ".", 2)
	digits := parts[0]
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "." + digits[i:]
	}
	return digits + "," + parts[1] + " " + unit
}
func date(raw string) string {
	at, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return raw
	}
	return at.Format(view.DeATDateLayout)
}
func timestamp(at time.Time) string {
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		loc = time.UTC
	}
	return view.FormatDateTimeIn(at, loc, view.DeATDateTimeLayout+" MST")
}
func allocationKey(key string) string {
	switch key {
	case store.AllocationKeyNutzwert:
		return "Nutzwert"
	case store.AllocationKeyFlaeche:
		return "Fläche"
	case store.AllocationKeyPersonen:
		return "Personen"
	case store.AllocationKeyVerbrauch:
		return "Verbrauch"
	default:
		return key
	}
}
