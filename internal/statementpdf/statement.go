// Package statementpdf renders working drafts from immutable statement runs.
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
	Measurements                    []string
}
type Document struct {
	UnitID, PartyID, UnitLabel string
	Header, Address, Basis     []string
	Costs                      []CostRow
	Total, Prepaid, Balance    string
	Excluded                   []string
	Contact                    string
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
	p := run.Input.Presentation
	org := p.Organisation
	if org == "" {
		org = "Hausverwaltung: Angabe fehlt"
	}
	d := Document{UnitID: unit.UnitID, PartyID: party.ID, UnitLabel: unit.Label,
		Header: []string{org, p.EstateName, p.EstateAddress, "Abrechnungsperiode: " + date(run.Input.Period.StartsOn) + " bis " + date(run.Input.Period.EndsOn), fmt.Sprintf("Lauf %s · Revision %d", run.ID, run.Revision), "Erstellt: " + timestamp(run.CreatedAt)},
		Total:  money(unit.AllocatedCents), Prepaid: money(unit.PrepaidCents), Contact: strings.Join(nonempty(p.ContactName, p.ContactEmail, p.ContactPhone, p.ContactAddress), " · ")}
	if d.Contact == "" {
		d.Contact = "Kontakt der Verwaltung fehlt"
	}
	role := "Wohnungseigentümer"
	if party.Renter {
		role = "Mietpartei"
		if party.Owner {
			role = "Wohnungseigentümer und Mietpartei"
		}
	}
	d.Address = append(d.Address, role)
	if party.Name != "" {
		d.Address = append(d.Address, party.Name)
	}
	d.Address = append(d.Address, party.ID)
	if strings.TrimSpace(party.Address) == "" {
		d.Address = append(d.Address, "Anschrift fehlt")
	} else {
		d.Address = append(d.Address, party.Address)
	}
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
				row.Measurements = append(row.Measurements, label+": "+timestamp(item.MeasuredAt)+" · "+measurement(item.ValueMicros, item.MeasurementUnit), "Quelle: "+item.SourceKind+" / "+item.SourceID)
			}
			if len(evidence) != 2 {
				row.Measurements = append(row.Measurements, "Grenzmessungen im gespeicherten Lauf unvollständig")
			}
		}
		d.Costs = append(d.Costs, row)
	}
	for _, cost := range run.Input.Structure.CostTypes {
		if !cost.Allocatable {
			d.Excluded = append(d.Excluded, cost.Name+": "+money(totals[cost.Key]))
		}
	}
	if len(d.Excluded) > 0 {
		d.Excluded = append(d.Excluded, "Gesamt nicht umlagefähig: "+money(run.Result.ExcludedCents))
	}
	return d
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
	return pdf.Pages(pages, pdf.Palette{Paper: [3]uint8{247, 243, 234}, Ink: [3]uint8{32, 37, 31}, Accent: [3]uint8{200, 153, 63}}), nil
}

func (d Document) Pages() []pdf.Page {
	lines := func(text string, style pdf.Style) []pdf.Line {
		var out []pdf.Line
		for _, line := range pdf.WrapText(text, 50) {
			out = append(out, pdf.Line{Text: line, Style: style})
		}
		return out
	}
	header := []pdf.Line{{Text: "Jahresabrechnung", Style: pdf.Heading}, {}}
	for _, text := range d.Header {
		header = append(header, lines(text, pdf.Body)...)
	}
	header = append(header, lines("Einheit: "+d.UnitLabel+" · Partei: "+d.PartyID, pdf.Body)...)
	header = append(header, pdf.Line{})
	var blocks [][]pdf.Line
	for _, text := range append(append([]string(nil), d.Address...), d.Basis...) {
		blocks = append(blocks, lines(text, pdf.Body))
	}
	blocks = append(blocks, []pdf.Line{{}})
	widths := []int{25, -18, 15, -11, -22}
	tableHeader := pdf.Columns([]string{"Kostenart", "Gesamtkosten Liegenschaft", "Verteiler-\nschlüssel", "Anteil", "Betrag Einheit"}, widths)
	for i, row := range d.Costs {
		rowLines := pdf.Columns([]string{row.Name, row.Total, row.Key, row.Share, row.Amount}, widths)
		if i == 0 {
			rowLines = append(append([]pdf.Line(nil), tableHeader...), rowLines...)
		}
		blocks = append(blocks, rowLines)
	}
	summary := []pdf.Line{{}}
	summary = append(summary, lines("Summe: "+d.Total, pdf.Strong)...)
	summary = append(summary, lines("Geleistete Akontozahlung: "+d.Prepaid, pdf.Body)...)
	summary = append(summary, lines(d.Balance, pdf.Strong)...)
	blocks = append(blocks, summary)
	for _, row := range d.Costs {
		if len(row.Measurements) > 0 {
			blocks = append(blocks, []pdf.Line{{}}, lines(row.Name+" · Messnachweis", pdf.Strong))
			for _, text := range row.Measurements {
				blocks = append(blocks, lines(text, pdf.Body))
			}
		}
	}
	if len(d.Excluded) > 0 {
		blocks = append(blocks, []pdf.Line{{}}, lines("Hinweis: Nicht umlagefähige Kosten", pdf.Strong))
		for _, text := range d.Excluded {
			blocks = append(blocks, lines(text, pdf.Body))
		}
	}
	contactLines := pdf.WrapText("Verwaltung: "+d.Contact, 62)
	if len(contactLines) > 3 {
		blocks = append(blocks, []pdf.Line{{}}, lines("Kontakt der Verwaltung", pdf.Strong), lines(d.Contact, pdf.Body))
		contactLines = append(contactLines[:2], "Weitere Kontaktdaten im Kontaktabschnitt.")
	}
	footer := append([]string{DraftNotice}, contactLines...)
	const maxLines = 43
	// Very long supplied addresses remain visible on continuation pages instead
	// of overflowing a fixed header. Only the title is then repeated.
	if len(header) > 16 {
		blocks = append([][]pdf.Line{header[2:]}, blocks...)
		header = header[:2]
	}
	current := append([]pdf.Line(nil), header...)
	var pages []pdf.Page
	flush := func() {
		pages = append(pages, pdf.Page{Lines: current, Footer: append([]string(nil), footer...)})
		current = append([]pdf.Line(nil), header...)
	}
	for _, block := range blocks {
		isTable := len(block) > 0 && block[0].Style == pdf.Table
		startsTable := isTable && block[0].Text == tableHeader[0].Text
		if len(current)+len(block) > maxLines && len(current) > len(header) {
			flush()
			if isTable && !startsTable {
				current = append(current, tableHeader...)
			}
		}
		for len(block) > 0 {
			n := min(len(block), maxLines-len(current))
			current = append(current, block[:n]...)
			block = block[n:]
			if len(block) > 0 {
				flush()
				if isTable {
					current = append(current, tableHeader...)
				}
			}
		}
	}
	if len(current) > len(header) {
		flush()
	}
	for i := range pages {
		pages[i].Footer = append(pages[i].Footer, fmt.Sprintf("Einheit %s · Seite %d / %d", d.UnitLabel, i+1, len(pages)))
	}
	return pages
}

func nonempty(values ...string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
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
